package main

import (
	"hypervisor/internal/db"
	"hypervisor/internal/hyperusers"
	"log"

	"github.com/gofiber/fiber/v3"
)

func main() {
	app := fiber.New()

	err := db.InitCache()
	if err != nil {
		log.Fatal(err)
	}

	err = db.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	hypervisor := app.Group("/hypervisor")

	hypervisor.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("PONG")
	})

	hyperusers.Routes(hypervisor)

	app.Listen(":8080")
}
