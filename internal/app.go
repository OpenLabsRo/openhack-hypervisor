package internal

import (
	"hypervisor/internal/api"
	"hypervisor/internal/db"
	"hypervisor/internal/env"
	"hypervisor/internal/events"
	"hypervisor/internal/hyperusers"
	"hypervisor/internal/models"
	"hypervisor/internal/swagger"
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

	meta := hypervisor.Group("/meta")
	meta.Get("/ping", hypervisorPingHandler)
	meta.Get("/version", hypervisorVersionHandler)

	swagger.Register(hypervisor)

	hyperusers.Routes(hypervisor)

	hypervisor.Post("/stages", models.HyperUserMiddleware, api.CreateStageHandler)
	hypervisor.Get("/stages", models.HyperUserMiddleware, api.ListStagesHandler)
	hypervisor.Get("/stages/:stageId", models.HyperUserMiddleware, api.GetStageHandler)
	hypervisor.Post("/stages/:stageId/sessions", models.HyperUserMiddleware, api.CreateStageSessionHandler)
	hypervisor.Get("/stages/:stageId/sessions", models.HyperUserMiddleware, api.ListStageSessionsHandler)
	hypervisor.Post("/stages/:stageId/tests", models.HyperUserMiddleware, api.StartStageTestHandler)

	hypervisor.Post("/deployments", models.HyperUserMiddleware, api.CreateDeploymentHandler)

	hypervisor.Post("/sync", models.HyperUserMiddleware, api.SyncHandler)

	return app
}

// hypervisorPingHandler answers with a plain "PONG" for service uptime checks.
// @Summary Health check
// @Description Lightweight heartbeat used by load balancers to confirm the Hypervisor API is alive.
// @Tags Hypervisor Meta
// @Produce plain
// @Success 200 {string} string "PONG"
// @Router /hypervisor/meta/ping [get]
func hypervisorPingHandler(c fiber.Ctx) error {
	return c.SendString("PONG")
}

// hypervisorVersionHandler prints the current deployment version for observability.
// @Summary Current deployment version
// @Description Exposes the semantic version bundled with the running hypervisor service.
// @Tags Hypervisor Meta
// @Produce plain
// @Success 200 {string} string "v25.10.17.4"
// @Router /hypervisor/meta/version [get]
func hypervisorVersionHandler(c fiber.Ctx) error {
	return c.SendString("v" + env.VERSION)
}
