package hyperusers

import "github.com/gofiber/fiber/v3"

func Routes(app fiber.Router) {
	hyperusers := app.Group("/hyperusers")

	hyperusers.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("PONG")
	})

	hyperusers.Post("/login", loginHandler)
}
