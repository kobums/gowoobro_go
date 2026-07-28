package routers

import (

	"strconv"


	"gowoobro/controllers/rest"


	"github.com/gofiber/fiber/v2"
)

// SetupIpblockRoutes sets up routes for ipblock domain
//
// ipblock_tb 는 차단 목록이다. 차단 IP 는 DB 에 직접 INSERT 로 넣고, API 로는
// 조회만 한다 — 누구나 호출할 수 있는 쓰기 라우트를 열어두면 아무나 차단 목록을
// 고칠 수 있기 때문이다.
//
// 주의: 이 파일은 gomachine 생성기 산출물이다. 생성기를 다시 돌리면 아래에서
// 지운 POST/PUT/DELETE 라우트가 되살아나므로 다시 지워야 한다.
func SetupIpblockRoutes(group fiber.Router) {

	group.Get("/ipblock", func(c *fiber.Ctx) error {
		page_, _ := strconv.Atoi(c.Query("page"))
		pagesize_, _ := strconv.Atoi(c.Query("pagesize"))
		var controller rest.IpblockController
		controller.Init(c)
		controller.Index(page_, pagesize_)
		controller.Close()
		return c.JSON(controller.Result)
	})

	group.Get("/ipblock/:id", func(c *fiber.Ctx) error {
		id_, _ := strconv.ParseInt(c.Params("id"), 10, 64)
		var controller rest.IpblockController
		controller.Init(c)
		controller.Read(id_)
		controller.Close()
		return c.JSON(controller.Result)
	})

	group.Post("/ipblock/count", func(c *fiber.Ctx) error {

		var controller rest.IpblockController
		controller.Init(c)
		controller.Count()
		controller.Close()
		return c.JSON(controller.Result)
	})

}
