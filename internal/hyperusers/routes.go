package hyperusers

import (
	"github.com/gofiber/fiber/v3"
	"hypervisor/internal/errmsg"
	"hypervisor/internal/models"
	"hypervisor/internal/utils"
)

func Routes(app fiber.Router) {
	group := app.Group("/hyperusers")

	group.Get("/ping", pingHandler)
	group.Post("/login", loginHandler)

	protected := group.Group("")
	protected.Use(models.HyperUserMiddleware)

	protected.Get("/whoami", whoamiHandler)
}

// pingHandler provides a simple heartbeat endpoint for hyperuser services.
// @Summary Hyperusers health check
// @Description Confirms the hyperusers service slice is online.
// @Tags Hyperusers Meta
// @Produce plain
// @Success 200 {string} string "PONG"
// @Router /hypervisor/hyperusers/ping [get]
func pingHandler(c fiber.Ctx) error {
	return c.SendString("PONG")
}

// whoamiHandler returns the authenticated hyperuser profile.
// @Summary Current hyperuser profile
// @Description Returns the hyperuser extracted from the bearer token.
// @Tags Hyperusers Meta
// @Security HyperUserAuth
// @Produce json
// @Success 200 {object} models.HyperUser
// @Failure 401 {object} errmsg._HyperUserNoToken
// @Router /hypervisor/hyperusers/whoami [get]
func whoamiHandler(c fiber.Ctx) error {
	var user models.HyperUser
	utils.GetLocals(c, "hyperuser", &user)
	if user.Username == "" {
		return utils.StatusError(c, errmsg.HyperUserNoToken)
	}
	user.Password = ""
	return c.JSON(user)
}
