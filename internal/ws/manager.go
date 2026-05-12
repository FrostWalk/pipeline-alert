package ws

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"pipeline-horn/internal/protocol"
)

var ErrNoClient = errors.New("no websocket client connected")

// Manager tracks one active client connection.
type Manager struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func NewManager() *Manager {
	return &Manager{}
}

// Replace stores a new connection and closes the previous one.
func (m *Manager) Replace(conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		_ = m.conn.Close()
	}
	m.conn = conn
}

// Clear removes the connection when it is still the active one.
func (m *Manager) Clear(conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn == conn {
		m.conn = nil
	}
}

// Notify sends the play-sound frame to the connected client.
func (m *Manager) Notify() error {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()

	if conn == nil {
		return ErrNoClient
	}

	deadline := time.Now().Add(5 * time.Second)
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return err
	}

	return conn.WriteMessage(websocket.BinaryMessage, []byte{protocol.PlaySound})
}
