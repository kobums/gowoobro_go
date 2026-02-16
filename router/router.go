package router

import (
	"gowoobro/router/routers"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
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

	// apiGroup.Use(JwtAuthRequired)


	// Setup domain-specific routes
	routers.SetupIpblockRoutes(apiGroup)
	routers.SetupQuestionsRoutes(apiGroup)
}