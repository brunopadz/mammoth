package main

import (
	"github.com/brunopadz/mammoth/cmd/api/routes"
	"github.com/brunopadz/mammoth/util/log"
	"github.com/gofiber/fiber/v2"
)

func main() {
	//r := redisConn()

	a := fiber.New()
	a.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.Send([]byte("mammoth api server"))
	})
	routes.UsersRouter(a)
	err := a.Listen(":8080")
	if err != nil {
		log.Fatal("Could not listen")
	}
}
