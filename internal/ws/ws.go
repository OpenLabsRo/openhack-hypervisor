package ws

import (
	"encoding/json"

	"hypervisor/internal/env"

	githubws "github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// Upgrader upgrades HTTP connections to WebSocket connections.
var Upgrader = githubws.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		// In drain mode, reject new WebSocket connections with 503
		if env.DRAIN_MODE {
			ctx.SetStatusCode(503)
			ctx.SetBodyString(`{"error": "Service is draining - please reconnect to active instance"}`)
			return false
		}
		return true
	},
}

// WriteStatus sends a status message to the websocket client.
func WriteStatus(conn *githubws.Conn, status string, message string) error {
	payload, err := json.Marshal(map[string]string{
		"type":    status,
		"message": message,
	})
	if err != nil {
		return err
	}
	return conn.WriteMessage(githubws.TextMessage, payload)
}

// WriteLog sends a log payload to the websocket client.
func WriteLog(conn *githubws.Conn, message []byte) error {
	payload, err := json.Marshal(map[string]string{
		"type":    "log",
		"message": string(message),
	})
	if err != nil {
		return err
	}
	return conn.WriteMessage(githubws.TextMessage, payload)
}
