package router

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"gowoobro/global/config"
	"gowoobro/global/log"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// 관리자 토큰 유효기간. 하루면 한 번 로그인해 관리 작업을 마치기 충분하고,
// 토큰이 새어나가도 오래 살아남지 않는다.
const tokenTTL = 24 * time.Hour

// 예전에는 관리자 인증이 프론트에만 있었다. NEXT_PUBLIC_ADMIN_PASSWORD 를
// 브라우저에서 비교하는 방식이라 화면만 가릴 뿐, /api 는 URL 만 알면 누구나
// 호출할 수 있었다. 이제 서버가 판정한다.

// isPublicPath 는 토큰 없이 지나갈 수 있는 요청을 정한다.
//
// 기본은 "막는다"이고 여기 적힌 것만 열린다. 공개 사이트가 실제로 쓰는 것만
// 골라야 한다 — 특히 메인 페이지의 프로젝트 목록은 Next.js 서버가 SSR 중에
// 부르므로 토큰을 실을 수 없다. 그래서 반드시 공개여야 한다.
func isPublicPath(method, path string) bool {
	// 슬래시 하나 차이로 뚫리지 않도록 끝 슬래시를 정규화한다.
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}

	switch {
	// CORS 프리플라이트. 보통 cors 미들웨어가 먼저 끊지만 순서에 기대지 않는다.
	case method == http.MethodOptions:
		return true

	// 로그인 자체는 열려 있어야 한다.
	case method == http.MethodPost && path == "/api/login":
		return true

	// 메인 페이지(SSR)와 앱 상세 페이지의 프로젝트 조회.
	case method == http.MethodGet && path == "/api/projects":
		return true
	case method == http.MethodGet && strings.HasPrefix(path, "/api/projects/"):
		return true

	// 방문자의 질문 등록. 목록 조회(GET)는 관리자만 볼 수 있어야 하므로 제외.
	case method == http.MethodPost && path == "/api/questions":
		return true

	// FAB 의 답변 조회. address 로 걸러 자기 것만 본다.
	case method == http.MethodGet && path == "/api/answers":
		return true
	}

	return false
}

// adminClaims 는 토큰에 담기는 내용이다. 관리자는 한 명뿐이라 식별자는 두지 않는다.
type adminClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// authConfigured 는 관리자 인증에 필요한 값이 모두 있는지 본다. 하나라도 비어
// 있으면 로그인을 받지 않는다 — 빈 시크릿으로 토큰을 서명하면 누구나 위조할 수
// 있고, 빈 비밀번호는 아무 문자열이나 통과시킨다.
func authConfigured() bool {
	return config.JwtSecret != "" && config.AdminPassword != ""
}

func issueToken() (string, error) {
	now := time.Now()
	claims := adminClaims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.JwtSecret))
}

func parseToken(token string) (*adminClaims, error) {
	var claims adminClaims

	_, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
		// 알고리즘을 고정하지 않으면 alg=none 이나 RS256→HS256 혼동 공격이 통한다.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(config.JwtSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return nil, err
	}

	return &claims, nil
}

// Login 은 비밀번호를 확인하고 관리자 토큰을 내준다.
func Login(c *fiber.Ctx) error {
	var body struct {
		Password string `json:"password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "error", "message": "invalid request",
		})
	}

	if !authConfigured() {
		// 설정이 빠졌다는 사실을 클라이언트에 알려줄 이유는 없다. 로그로만 남긴다.
		log.Error().Msg("auth: JWT_SECRET 또는 ADMIN_PASSWORD 가 없어 로그인을 거부했다")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code": "error", "message": "invalid password",
		})
	}

	// 길이 차이로 정답을 좁히지 못하도록 상수 시간 비교를 쓴다.
	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(config.AdminPassword)) != 1 {
		log.Info().Str("address", clientIP(c)).Msg("auth: 관리자 로그인 실패")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code": "error", "message": "invalid password",
		})
	}

	token, err := issueToken()
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("auth: 토큰 발급 실패")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": "error", "message": "token error",
		})
	}

	log.Info().Str("address", clientIP(c)).Msg("auth: 관리자 로그인 성공")

	return c.JSON(fiber.Map{
		"code":      "ok",
		"token":     token,
		"expiresIn": int(tokenTTL.Seconds()),
	})
}

// JwtAuthRequired 는 isPublicPath 에 없는 모든 요청에 관리자 토큰을 요구한다.
func JwtAuthRequired(c *fiber.Ctx) error {
	if isPublicPath(c.Method(), c.Path()) {
		return c.Next()
	}

	deny := func() error {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code": "error", "message": "unauthorized",
		})
	}

	if !authConfigured() {
		return deny()
	}

	header := c.Get("Authorization")
	if header == "" {
		return deny()
	}

	// "Bearer <token>" 만 받는다.
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return deny()
	}

	claims, err := parseToken(parts[1])
	if err != nil || claims.Role != "admin" {
		return deny()
	}

	return c.Next()
}
