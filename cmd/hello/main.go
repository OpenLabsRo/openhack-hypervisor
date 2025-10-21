package main

import (
	"flag"

	"github.com/gofiber/fiber/v3"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")

	flag.Parse()

	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello from Port " + *port + "!\n")
	})

	app.Listen(":" + *port)
}
