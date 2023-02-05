package routes

import (
	"context"
	"github.com/brunopadz/mammoth/util/log"
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"net/http"
)

type User struct {
	User string
}

func UsersRouter(app fiber.Router) {

	ctx := context.Background()

	app.Get("/users", func(c *fiber.Ctx) error {
		r := redisConn()
		d, err := r.SMembers(context.Background(), "users").Result()
		if err != nil {
			log.Fatal("aaa")
		}
		return c.JSON(d)
	})
	app.Put("/users", func(c *fiber.Ctx) error {
		var d User
		r := redisConn()
		err := c.BodyParser(&d)
		if err != nil {
			c.Status(http.StatusBadRequest)
		}
		// Check user exists if not, add user
		r.SAdd(ctx, "users", &d)

		// uncomment below
		//return c.Send([]byte("to-do"))
	})
	app.Delete("/users", func(c *fiber.Ctx) error {
		return c.Send([]byte("to-do"))
	})
}

func redisConn() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

}
