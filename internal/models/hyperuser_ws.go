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
// Expected query parameter: ?authorization=<token>
func HyperUserWebSocketMiddleware(c fiber.Ctx) error {
	// First try to get token from standard Authorization header (for non-browser clients)
	authHeader := strings.TrimSpace(c.Get("Authorization"))

	var token string
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		// Extract from Bearer header
		tokens := strings.Fields(authHeader)
		if len(tokens) == 2 {
			token = strings.TrimSpace(tokens[1])
		}
	} else {
		// Try query parameter without Bearer prefix
		token = strings.TrimSpace(c.Query("authorization"))
	}

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
