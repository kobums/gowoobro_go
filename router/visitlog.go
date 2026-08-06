package router

import (
	"gowoobro/global/log"
	"gowoobro/models"

	"github.com/gofiber/fiber/v2"
)

// recordVisit 는 iplog_tb 에 한 건 적재한다. 테스트에서 DB 없이 호출을 검증할 수
// 있도록 변수로 뒀다. 기본 구현은 요청 처리를 막지 않게 고루틴에서 돈다.
var recordVisit = func(address string, path string, agent string) {
	go func() {
		conn := models.NewConnection()
		if conn == nil || !conn.IsConnect() {
			log.Error().Str("address", address).Msg("visitlog: database connection error")
			return
		}
		defer conn.Close()

		manager := models.NewIplogManager(conn)
		item := models.Iplog{Address: address, Path: path, Agent: agent}
		if err := manager.Insert(&item); err != nil {
			// 방문 로그는 부가 기능이다. 실패해도 요청에는 영향을 주지 않고
			// 로그만 남긴다.
			log.Error().Str("error", err.Error()).Str("address", address).Msg("visitlog: insert failed")
		}
	}()
}

// VisitLog 는 요청마다 방문자 IP 를 iplog_tb 에 적재한다. 응답을 기다리게 하지
// 않도록 적재는 비동기로 한다.
func VisitLog(c *fiber.Ctx) error {
	// CORS preflight 는 브라우저 내부 동작이라 방문으로 치지 않는다.
	if c.Method() == fiber.MethodOptions {
		return c.Next()
	}

	if address := clientIP(c); address != "" {
		recordVisit(address, c.Path(), c.Get("User-Agent"))
	}

	return c.Next()
}
