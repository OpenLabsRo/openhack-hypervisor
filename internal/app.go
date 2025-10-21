package internal

import (
	"context"
	"hypervisor/internal/api"
	"hypervisor/internal/db"
	"hypervisor/internal/env"
	"hypervisor/internal/events"
	"hypervisor/internal/hyperusers"
	"hypervisor/internal/models"
	"hypervisor/internal/proxy"
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

	// Initialize proxy system
	if err := proxy.InitProxy(context.Background()); err != nil {
		log.Fatal("Could not initialize proxy system:", err)
		return nil
	}

	// Set up proxy routes (must be before API routes)
	proxy.GlobalRouteMap.SetupRoutes(app)

	hypervisor := app.Group("/hypervisor")

	meta := hypervisor.Group("/meta")
	meta.Get("/ping", hypervisorPingHandler)
	meta.Get("/version", hypervisorVersionHandler)

	swagger.Register(hypervisor)

	hyperusers.Routes(hypervisor)

	hypervisor.Post("/releases/sync", models.HyperUserMiddleware, api.SyncHandler)
	hypervisor.Get("/releases", models.HyperUserMiddleware, api.ListReleasesHandler)
	// hypervisor.Get("/releases/webhook", models.HyperUserMiddleware, api.ListReleasesHandler)  for the GitHub webhook integration

	hypervisor.Get("/env/template", models.HyperUserMiddleware, api.GetEnvTemplateHandler)
	hypervisor.Post("/env/template/validate", models.HyperUserMiddleware, api.UpdateEnvTemplateHandler)

	// creating and listing stages
	hypervisor.Post("/stages", models.HyperUserMiddleware, api.CreateStageHandler)
	hypervisor.Get("/stages", models.HyperUserMiddleware, api.ListStagesHandler)

	// getting and deleteing a certain stage
	hypervisor.Get("/stages/:stageId", models.HyperUserMiddleware, api.GetStageHandler)
	hypervisor.Delete("/stages/:stageId", models.HyperUserMiddleware, api.DeleteStageHandler)

	// getting and modifying a stage's environment
	hypervisor.Get("/stages/:stageId/env", models.HyperUserMiddleware, api.GetStageEnvHandler)
	hypervisor.Put("/stages/:stageId/env", models.HyperUserMiddleware, api.UpdateStageEnvHandler)

	// getting the list of tests, and starting a test
	hypervisor.Get("/stages/:stageId/tests", models.HyperUserMiddleware, api.ListTestsHandler)
	hypervisor.Post("/stages/:stageId/tests", models.HyperUserMiddleware, api.StartTestHandler)

	// cancelling a test
	hypervisor.Post("/stages/:stageId/tests/:sequence/cancel", models.HyperUserMiddleware, api.CancelTestHandler)

	// creating a deployment based on a stage ID
	hypervisor.Post("/deployments/:stageId", models.HyperUserMiddleware, api.CreateDeploymentHandler)

	// getting a list of deployments
	hypervisor.Get("/deployments", models.HyperUserMiddleware, api.ListDeploymentsHandler)

	// getting and deleting a certain deployment
	hypervisor.Get("/deployments/:deploymentId", models.HyperUserMiddleware, api.GetDeploymentHandler)
	hypervisor.Delete("/deployments/:deploymentId", models.HyperUserMiddleware, api.DeleteDeploymentHandler)

	// either promoting or shutting down a deployment
	hypervisor.Post("/deployments/:deploymentId/promote", models.HyperUserMiddleware, api.PromoteDeploymentHandler)

	// shutting down and starting a deployment
	hypervisor.Post("/deployments/:deploymentId/shutdown", models.HyperUserMiddleware, api.ShutdownDeploymentHandler)
	hypervisor.Post("/deployments/:deploymentId/start", models.HyperUserMiddleware, api.StartDeploymentHandler)

	// websockets for streaming test logs and deployment logs
	ws := hypervisor.Group("/ws")
	ws.Use(models.HyperUserMiddleware)
	ws.Get("/stages/:stageId/tests/:sequence", api.StreamTestLogs)
	ws.Get("/deployments/:deploymentId/logs", api.StreamDeploymentLogs)

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
