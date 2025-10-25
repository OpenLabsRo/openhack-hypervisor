package models

import (
	"hypervisor/internal/errmsg"
	"hypervisor/internal/utils"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// HyperUserWebSocketMiddleware extracts the Authorization token from query parameters
// for WebSocket connections, since browsers don't allow custom headers in WebSocket upgrades.
// Usage: ws.Use(models.HyperUserWebSocketMiddleware)
// Expected query parameter: ?authorization=Bearer%20<token>
func HyperUserWebSocketMiddleware(c fiber.Ctx) error {
	// First try to get token from standard Authorization header (for non-browser clients)
	authHeader := strings.TrimSpace(c.Get("Authorization"))

	// If no header, try query parameter
	if authHeader == "" {
		authHeader = strings.TrimSpace(c.Query("authorization"))
	}

	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return utils.StatusError(c, errmsg.HyperUserNoToken)
	}

	tokens := strings.Fields(authHeader)
	if len(tokens) != 2 {
		return utils.StatusError(c, errmsg.HyperUserNoToken)
	}

	token := strings.TrimSpace(tokens[1])
	if token == "" {
		return utils.StatusError(c, errmsg.HyperUserNoToken)
	}

	var hyperuser HyperUser
	if err := hyperuser.ParseToken(token); err != nil {
		return utils.StatusError(c, errmsg.HyperUserNoToken)
	}

	utils.SetLocals(c, "hyperuser", hyperuser)

	return c.Next()
}
