package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"hypervisor/internal/core"
	"hypervisor/internal/models"
	"hypervisor/internal/ws"

	githubws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

var errClientClosed = errors.New("websocket closed by client")

// StreamTestLogs upgrades the connection and continuously streams a test log.
func StreamTestLogs(c fiber.Ctx) error {
	stageID := c.Params("stageId") // Note: This is validated by the test.StageID check below
	testID := c.Params("testId")

	if stageID == "" || testID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing stage or test identifier")
	}

	type requestCtxProvider interface {
		RequestCtx() *fasthttp.RequestCtx
	}

	provider, ok := any(c).(requestCtxProvider)
	if !ok {
		return fiber.ErrInternalServerError
	}

	return ws.Upgrader.Upgrade(provider.RequestCtx(), func(conn *githubws.Conn) {
		defer conn.Close()

		ctx := context.Background()

		test, err := models.GetTestByID(ctx, testID)
		if err != nil {
			_ = ws.WriteStatus(conn, "error", fmt.Sprintf("test not found: %v", err))
			return
		}

		if test.StageID != stageID {
			_ = ws.WriteStatus(conn, "error", "test does not belong to the requested stage")
			return
		}

		if test.LogPath == "" {
			_ = ws.WriteStatus(conn, "error", "log path is not available for this test run")
			return
		}

		closed := make(chan struct{})
		var once sync.Once
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					once.Do(func() { close(closed) })
					return
				}
			}
		}()

		// Create a writer that sends log lines over the WebSocket connection.
		wsWriter := &websocketLogWriter{conn: conn}

		// Use a cancellable context tied to the client connection.
		streamCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			<-closed
			cancel()
		}()

		err = core.StreamLogFile(streamCtx, test.LogPath, test.ID, wsWriter, wsWriter)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, errClientClosed) {
			_ = ws.WriteStatus(conn, "error", fmt.Sprintf("log stream failed: %v", err))
		}

		_ = ws.WriteStatus(conn, "info", "log stream ended")
	})
}

// websocketLogWriter is an io.Writer that sends messages over a WebSocket.
type websocketLogWriter struct {
	conn *githubws.Conn
}

func (w *websocketLogWriter) Write(p []byte) (n int, err error) {
	if err := ws.WriteLog(w.conn, p); err != nil {
		// Use a specific error to signal that the client has disconnected.
		return 0, errClientClosed
	}
	return len(p), nil
}

func (w *websocketLogWriter) WriteStatus(level, message string) {
	_ = ws.WriteStatus(w.conn, level, message)
}
