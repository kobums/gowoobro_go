package routers

import (

	"strconv"

	"gowoobro/global/log"


	"gowoobro/controllers/rest"


	"gowoobro/models"
	"github.com/gofiber/fiber/v2"
)

// SetupIpblockRoutes sets up routes for ipblock domain
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

	group.Post("/ipblock", func(c *fiber.Ctx) error {
		item_ := &models.Ipblock{}
		err := c.BodyParser(item_)
		if err != nil {
		    log.Error().Msg(err.Error())
		}
		var controller rest.IpblockController
		controller.Init(c)
		controller.Insert(item_)
		controller.Close()
		return c.JSON(controller.Result)
	})

	group.Post("/ipblock/batch", func(c *fiber.Ctx) error {
		var items_ *[]models.Ipblock
		items__ref := &items_
		err := c.BodyParser(items__ref)
		if err != nil {
		    log.Error().Msg(err.Error())
		}
		var controller rest.IpblockController
		controller.Init(c)
		controller.Insertbatch(items_)
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

	group.Put("/ipblock", func(c *fiber.Ctx) error {
		item_ := &models.Ipblock{}
		err := c.BodyParser(item_)
		if err != nil {
		    log.Error().Msg(err.Error())
		}
		var controller rest.IpblockController
		controller.Init(c)
		controller.Update(item_)
		controller.Close()
		return c.JSON(controller.Result)
	})

	group.Delete("/ipblock", func(c *fiber.Ctx) error {
		item_ := &models.Ipblock{}
		err := c.BodyParser(item_)
		if err != nil {
		    log.Error().Msg(err.Error())
		}
		var controller rest.IpblockController
		controller.Init(c)
		controller.Delete(item_)
		controller.Close()
		return c.JSON(controller.Result)
	})

	group.Delete("/ipblock/batch", func(c *fiber.Ctx) error {
		item_ := &[]models.Ipblock{}
		err := c.BodyParser(item_)
		if err != nil {
		    log.Error().Msg(err.Error())
		}
		var controller rest.IpblockController
		controller.Init(c)
		controller.Deletebatch(item_)
		controller.Close()
		return c.JSON(controller.Result)
	})

}