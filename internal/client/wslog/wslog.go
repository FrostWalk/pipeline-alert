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
