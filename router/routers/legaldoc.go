package routers

import (
	"strconv"

	"gowoobro/controllers/rest"
	"gowoobro/models"

	"github.com/gofiber/fiber/v2"
)

// SetupLegaldocRoutes 는 약관·개인정보 처리방침 API 를 건다.
//
//	GET    /legaldocs/current?project=&type=&lang=  공개. 시행 중인 문서 + 버전 목록
//	GET    /legaldocs/:id                           공개. 특정 버전(시행된 것만)
//	GET    /legaldocs?project=                      관리자. 시행 전 포함 전체
//	POST   /legaldocs                               관리자. 새 버전 등록
//	PUT    /legaldocs                               관리자. 시행 전 버전 수정
//	DELETE /legaldocs                               관리자. 시행 전 버전 삭제
func SetupLegaldocRoutes(group fiber.Router) {

	respond := func(c *fiber.Ctx, controller *rest.LegaldocController) error {
		controller.Close()
		return c.Status(controller.Code).JSON(controller.Result)
	}

	parse := func(c *fiber.Ctx, controller *rest.LegaldocController) *models.Legaldoc {
		item := &models.Legaldoc{}
		if err := c.BodyParser(item); err != nil {
			controller.Code = fiber.StatusBadRequest
			controller.Error(err)
			return nil
		}
		return item
	}

	group.Get("/legaldocs", func(c *fiber.Ctx) error {
		project_, _ := strconv.ParseInt(c.Query("project"), 10, 64)
		var controller rest.LegaldocController
		controller.Init(c)
		controller.Index(project_)
		return respond(c, &controller)
	})

	// :id 보다 먼저 걸어야 "current" 가 id 로 해석되지 않는다.
	group.Get("/legaldocs/current", func(c *fiber.Ctx) error {
		project_, _ := strconv.ParseInt(c.Query("project"), 10, 64)
		var controller rest.LegaldocController
		controller.Init(c)
		controller.Current(project_, c.Query("type"), c.Query("lang"))
		return respond(c, &controller)
	})

	group.Get("/legaldocs/:id", func(c *fiber.Ctx) error {
		id_, _ := strconv.ParseInt(c.Params("id"), 10, 64)
		var controller rest.LegaldocController
		controller.Init(c)
		controller.Read(id_)
		return respond(c, &controller)
	})

	group.Post("/legaldocs", func(c *fiber.Ctx) error {
		var controller rest.LegaldocController
		controller.Init(c)
		if item_ := parse(c, &controller); item_ != nil {
			controller.Insert(item_)
		}
		return respond(c, &controller)
	})

	group.Put("/legaldocs", func(c *fiber.Ctx) error {
		var controller rest.LegaldocController
		controller.Init(c)
		if item_ := parse(c, &controller); item_ != nil {
			controller.Update(item_)
		}
		return respond(c, &controller)
	})

	group.Delete("/legaldocs", func(c *fiber.Ctx) error {
		var controller rest.LegaldocController
		controller.Init(c)
		if item_ := parse(c, &controller); item_ != nil {
			controller.Delete(item_.Id)
		}
		return respond(c, &controller)
	})
}
