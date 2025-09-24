package internal

import (
	"hypervisor/internal/db"
	"hypervisor/internal/env"
	"hypervisor/internal/events"
	"hypervisor/internal/hyperusers"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func SetupApp(deployment string, envRoot string, appVersion string) *fiber.App {
	app := fiber.New()

	env.Init(envRoot, appVersion)

	if err := db.InitDB(); err != nil {
		log.Fatal("Could not connect to MongoDB")
		return nil
	}

	if err := db.InitCache(); err != nil {
		log.Fatal("Could not connect to Redis")
		return nil
	}

	deploy := strings.TrimSpace(deployment)
	events.Em = events.NewEmitter(db.Events, deploy)

	hypervisor := app.Group("/hypervisor")

	hypervisor.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("PONG")
	})

	hyperusers.Routes(hypervisor)

	return app
}
