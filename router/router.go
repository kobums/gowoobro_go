package router

import (
	"strconv"
	"strings"
	"time"

	"gowoobro/global/log"
	"gowoobro/router/routers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func getArrayCommal(name string) []int64 {
	values := strings.Split(name, ",")

	var items []int64
	for _, item := range values {
		n, _ := strconv.ParseInt(item, 10, 64)
		items = append(items, n)
	}

	return items
}

func getArrayCommai(name string) []int {
	values := strings.Split(name, ",")

	var items []int
	for _, item := range values {
		n, _ := strconv.Atoi(item)
		items = append(items, n)
	}

	return items
}

func SetRouter(r *fiber.App) {

	// r.Get("/api/jwt", func(c *fiber.Ctx) error {
	// 	loginid := c.Query("loginid")
	//     passwd := c.Query("passwd")
	//     return c.JSON(JwtAuth(c, loginid, passwd))
	// })

	apiGroup := r.Group("/api")

	// ipblock_tb 에 등록된 IP 는 여기서 403 으로 끊는다. 차단된 IP 는 로그인
	// 시도조차 못 하도록 인증보다 앞에 둔다.
	apiGroup.Use(IpBlockGuard)

	// 관리자 토큰 검사. isPublicPath 에 적힌 것만 통과하고 나머지는 401 이다.
	apiGroup.Use(JwtAuthRequired)

	// 로그인은 서버가 비밀번호를 판정하므로 무차별 대입이 가능하다. 특히 짧은
	// 비밀번호면 전수 조사가 금방 끝난다. IP 당 시도 횟수를 묶어둔다.
	apiGroup.Post("/login", limiter.New(limiter.Config{
		Max:        10,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return clientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			log.Info().Str("address", clientIP(c)).Msg("auth: 로그인 시도 제한 초과")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"code": "error", "message": "too many attempts",
			})
		},
	}), Login)

	// Setup domain-specific routes
	routers.SetupIpblockRoutes(apiGroup)
	routers.SetupQuestionsRoutes(apiGroup)
	routers.SetupAnswersRoutes(apiGroup)
	routers.SetupProjectsRoutes(apiGroup)
	routers.SetupUploadRoutes(apiGroup)
}
