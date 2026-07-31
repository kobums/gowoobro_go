package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gowoobro/global/config"

	"github.com/gofiber/fiber/v2"
)

func withAuthConfig(t *testing.T, secret, password string) {
	t.Helper()

	prevSecret, prevPassword := config.JwtSecret, config.AdminPassword
	config.JwtSecret, config.AdminPassword = secret, password
	t.Cleanup(func() {
		config.JwtSecret, config.AdminPassword = prevSecret, prevPassword
	})
}

// newAuthApp 은 실제 router.go 와 같은 순서로 미들웨어를 얹는다. 라우트 핸들러는
// DB 를 타지 않는 더미라 인증 판정만 본다.
func newAuthApp() *fiber.App {
	app := fiber.New()

	seedEmptyBlocklist()

	api := app.Group("/api")
	api.Use(IpBlockGuard)
	api.Use(JwtAuthRequired)
	api.Post("/login", Login)

	ok := func(c *fiber.Ctx) error { return c.SendString("ok") }
	for _, p := range []string{"/projects", "/projects/:id", "/questions", "/answers", "/ipblock", "/upload/index"} {
		api.Get(p, ok)
		api.Post(p, ok)
		api.Put(p, ok)
		api.Delete(p, ok)
	}
	api.Post("/projects/count", ok)

	return app
}

func TestPublicPathsNeedNoToken(t *testing.T) {
	withAuthConfig(t, "test-secret", "test-password")
	app := newAuthApp()

	// 공개 사이트가 실제로 쓰는 호출들. 하나라도 401 이면 사이트가 깨진다.
	public := []struct{ method, path string }{
		{http.MethodGet, "/api/projects"},     // 메인 SSR
		{http.MethodGet, "/api/projects/3"},   // 앱 상세
		{http.MethodPost, "/api/questions"},   // 방문자 질문 등록
		{http.MethodGet, "/api/answers"},      // FAB 답변 조회
		{http.MethodOptions, "/api/projects"}, // CORS 프리플라이트
	}

	for _, tt := range public {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			res, err := app.Test(httptest.NewRequest(tt.method, tt.path, nil))
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode == http.StatusUnauthorized {
				t.Errorf("공개 경로가 401 로 막혔다 — 공개 사이트가 깨진다")
			}
		})
	}
}

func TestProtectedPathsRequireToken(t *testing.T) {
	withAuthConfig(t, "test-secret", "test-password")
	app := newAuthApp()

	protected := []struct{ method, path string }{
		{http.MethodGet, "/api/questions"},   // 관리자만 전체 질문 목록
		{http.MethodPost, "/api/answers"},    // 답변 작성
		{http.MethodPut, "/api/answers"},     // 답변 수정
		{http.MethodPost, "/api/projects"},   // 프로젝트 등록
		{http.MethodPut, "/api/projects"},    // 프로젝트 수정
		{http.MethodDelete, "/api/projects"}, // 프로젝트 삭제
		{http.MethodPost, "/api/projects/count"},
		{http.MethodPost, "/api/upload/index"}, // 파일 업로드
		{http.MethodGet, "/api/ipblock"},       // 차단 목록 열람
		{http.MethodDelete, "/api/questions"},
	}

	for _, tt := range protected {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			res, err := app.Test(httptest.NewRequest(tt.method, tt.path, nil))
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401 — 토큰 없이 통과하면 안 된다", res.StatusCode)
			}
		})
	}
}

func loginAndGetToken(t *testing.T, app *fiber.App, password string) (int, string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer res.Body.Close()

	var body struct {
		Code  string `json:"code"`
		Token string `json:"token"`
	}
	_ = json.NewDecoder(res.Body).Decode(&body)

	return res.StatusCode, body.Token
}

func TestLoginAndTokenAccess(t *testing.T) {
	withAuthConfig(t, "test-secret", "test-password")
	app := newAuthApp()

	t.Run("틀린 비밀번호는 401", func(t *testing.T) {
		code, token := loginAndGetToken(t, app, "wrong")
		if code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", code)
		}
		if token != "" {
			t.Errorf("토큰이 발급됐다: %q", token)
		}
	})

	t.Run("맞는 비밀번호는 토큰 발급", func(t *testing.T) {
		code, token := loginAndGetToken(t, app, "test-password")
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if token == "" {
			t.Fatal("토큰이 비었다")
		}

		req := httptest.NewRequest(http.MethodPost, "/api/projects", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		res, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("토큰을 넣었는데 status = %d", res.StatusCode)
		}
	})
}

func TestTokenRejectionCases(t *testing.T) {
	withAuthConfig(t, "test-secret", "test-password")
	app := newAuthApp()

	_, valid := loginAndGetToken(t, app, "test-password")

	// 다른 시크릿으로 서명한 토큰을 만든다.
	withAuthConfig(t, "other-secret", "test-password")
	_, foreign := loginAndGetToken(t, newAuthApp(), "test-password")
	withAuthConfig(t, "test-secret", "test-password")

	tests := []struct {
		name   string
		header string
	}{
		{"헤더 없음", ""},
		{"Bearer 접두사 없음", valid},
		{"엉뚱한 스킴", "Basic " + valid},
		{"깨진 토큰", "Bearer not-a-token"},
		{"서명 조작", "Bearer " + valid + "x"},
		{"다른 시크릿으로 서명", "Bearer " + foreign},
		{"alg=none", "Bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJyb2xlIjoiYWRtaW4ifQ."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/projects", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			res, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", res.StatusCode)
			}
		})
	}
}

// 설정이 비어 있으면 로그인도 토큰 검사도 통과시키면 안 된다.
func TestAuthDisabledWhenUnconfigured(t *testing.T) {
	for _, tt := range []struct{ name, secret, password string }{
		{"둘 다 없음", "", ""},
		{"시크릿 없음", "", "test-password"},
		{"비밀번호 없음", "test-secret", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			withAuthConfig(t, tt.secret, tt.password)
			app := newAuthApp()

			code, token := loginAndGetToken(t, app, tt.password)
			if code != http.StatusUnauthorized || token != "" {
				t.Errorf("설정이 없는데 로그인됐다: status=%d token=%q", code, token)
			}

			res, err := app.Test(httptest.NewRequest(http.MethodPost, "/api/projects", nil))
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusUnauthorized {
				t.Errorf("설정이 없는데 보호 경로가 status = %d", res.StatusCode)
			}
		})
	}
}
