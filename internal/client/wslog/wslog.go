package wslog

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"pipeline-horn/internal/piws"
)

var (
	mu  sync.Mutex
	cur *websocket.Conn
)

// SetConn registers the active websocket for uplink logs.
func SetConn(c *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()
	cur = c
}

// Clear drops the active websocket if it still matches conn.
func Clear(conn *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()
	if cur == conn {
		cur = nil
	}
}

// PiLog sends a structured log line to the server (best-effort).
func PiLog(level, message string) {
	msg := piws.PiLog{
		Type:    piws.TypePiLog,
		Level:   level,
		Message: message,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}

	mu.Lock()
	c := cur
	if c == nil {
		mu.Unlock()
		return
	}
	_ = c.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = c.WriteMessage(websocket.TextMessage, b)
	mu.Unlock()
	if err != nil {
		return
	}
}

// SendJSON writes a JSON text message on the active websocket.
func SendJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	if cur == nil {
		return websocket.ErrCloseSent
	}
	_ = cur.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return cur.WriteMessage(websocket.TextMessage, b)
}
