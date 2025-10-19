package ws

import (
	"encoding/json"

	githubws "github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// Upgrader upgrades HTTP connections to WebSocket connections.
var Upgrader = githubws.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
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
