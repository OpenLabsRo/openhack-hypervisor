package ws

import (
	"context"
	"errors"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

var errClientClosed = errors.New("websocket closed by client")

// WebsocketLogWriter is an io.Writer that sends messages over a WebSocket.
type WebsocketLogWriter struct {
	conn *websocket.Conn
}

func (w *WebsocketLogWriter) Write(p []byte) (n int, err error) {
	if err := WriteLog(w.conn, p); err != nil {
		// Use a specific error to signal that the client has disconnected.
		return 0, errClientClosed
	}
	return len(p), nil
}

func (w *WebsocketLogWriter) WriteStatus(level, message string) {
	_ = WriteStatus(w.conn, level, message)
}

// StreamWebSocket upgrades to WebSocket and streams using the provided streamer function.
func StreamWebSocket(c fiber.Ctx, streamer func(ctx context.Context, writer *WebsocketLogWriter) error) error {
	type requestCtxProvider interface {
		RequestCtx() *fasthttp.RequestCtx
	}

	provider, ok := any(c).(requestCtxProvider)
	if !ok {
		return fiber.ErrInternalServerError
	}

	return Upgrader.Upgrade(provider.RequestCtx(), func(conn *websocket.Conn) {
		defer conn.Close()

		ctx := context.Background()

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
		wsWriter := &WebsocketLogWriter{conn: conn}

		// Use a cancellable context tied to the client connection.
		streamCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			<-closed
			cancel()
		}()

		err := streamer(streamCtx, wsWriter)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, errClientClosed) {
			_ = WriteStatus(conn, "error", "log stream failed")
		}

		_ = WriteStatus(conn, "info", "log stream ended")
	})
}
