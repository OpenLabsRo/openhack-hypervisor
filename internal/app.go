package internal

import (
	"context"
	"fmt"
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
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func SetupApp(deployment string, envRoot string, appVersion string) *fiber.App {
	app := fiber.New()

	// Enable CORS for all origins
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
	}))

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
	meta.Get("/drain", hypervisorDrainStatusHandler)
	meta.Post("/drain", hypervisorDrainHandler)
	hypervisor.Get("/routing", api.GetRoutingMapHandler)

	swagger.Register(hypervisor)

	hyperusers.Routes(hypervisor)

	hypervisor.Post("/releases/sync", models.HyperUserMiddleware, api.SyncHandler)
	hypervisor.Get("/releases", models.HyperUserMiddleware, api.ListReleasesHandler)
	// hypervisor.Get("/releases/webhook", models.HyperUserMiddleware, api.ListReleasesHandler)  for the GitHub webhook integration

	hypervisor.Get("/env/template", models.HyperUserMiddleware, api.GetEnvTemplateHandler)
	hypervisor.Put("/env/template", models.HyperUserMiddleware, api.UpdateEnvTemplateHandler)

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
// @Failure 503 {string} string "Service is draining"
// @Router /hypervisor/meta/ping [get]
func hypervisorPingHandler(c fiber.Ctx) error {
	// In drain mode, return 503 to signal load balancers to stop routing traffic
	if env.DRAIN_MODE {
		return c.Status(503).SendString("Service is draining - please route to active instance")
	}
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

// hypervisorDrainHandler toggles drain mode for blue-green deployments.
// @Summary Toggle drain mode
// @Description Enables or disables drain mode, causing the service to reject new connections while keeping existing ones alive.
// @Tags Hypervisor Meta
// @Accept json
// @Produce json
// @Param request body map[string]bool true "Drain mode toggle"
// @Success 200 {object} map[string]interface{} "Drain mode status"
// @Failure 400 {object} errmsg._GeneralBadRequest
// @Router /hypervisor/meta/drain [post]
func hypervisorDrainHandler(c fiber.Ctx) error {
	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Note: This is a runtime toggle. In production, you might want to persist this
	// or use a more robust mechanism like updating a config file that gets reloaded
	env.DRAIN_MODE = req.Enabled

	return c.JSON(fiber.Map{
		"drain_mode": env.DRAIN_MODE,
		"message":    fmt.Sprintf("Drain mode %s", map[bool]string{true: "enabled", false: "disabled"}[env.DRAIN_MODE]),
	})
}

// hypervisorDrainStatusHandler returns the current drain mode status.
// @Summary Get drain mode status
// @Description Returns whether drain mode is currently enabled.
// @Tags Hypervisor Meta
// @Produce json
// @Success 200 {object} map[string]bool "Drain mode status"
// @Router /hypervisor/meta/drain [get]
func hypervisorDrainStatusHandler(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"drain_mode": env.DRAIN_MODE,
	})
}
