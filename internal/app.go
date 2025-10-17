package internal

import (
	"hypervisor/internal/db"
	"hypervisor/internal/env"
	"hypervisor/internal/events"
	"hypervisor/internal/gitcommits"
	"hypervisor/internal/githubhooks"
	"hypervisor/internal/hyperusers"
	releases_api "hypervisor/internal/releases/api"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func SetupApp(deployment string, envRoot string, appVersion string) *fiber.App {
	app := fiber.New()

	env.Init(envRoot, appVersion)

	deploy := strings.TrimSpace(deployment)

	if err := db.InitDB(deploy); err != nil {
		log.Fatal("Could not connect to MongoDB")
		return nil
	}

	if err := db.InitCache(); err != nil {
		log.Fatal("Could not connect to Redis")
		return nil
	}

	if db.Events != nil {
		events.Em = events.NewEmitter(db.Events, deploy)
	} else {
		events.Em = nil
	}

	hypervisor := app.Group("/hypervisor")

	hypervisor.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("PONG")
	})

	hypervisor.Get("/version", func(c fiber.Ctx) error {
		return c.SendString("v" + env.VERSION)
	})

	hyperusers.Routes(hypervisor)
	githubhooks.Routes(hypervisor)
	gitcommits.Routes(hypervisor)
	releases_api.Routes(hypervisor)

	return app
}
