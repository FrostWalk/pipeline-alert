package ws

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"pipeline-horn/internal/protocol"
)

var ErrNoClient = errors.New("no websocket client connected")

// Manager tracks one active client connection.
type Manager struct {
	mu sync.RWMutex

	conn        *websocket.Conn
	writeMu     sync.Mutex
	connectedAt time.Time
	lastSeen    time.Time
	lastPong    time.Time
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
	now := time.Now().UTC()
	m.connectedAt = now
	m.lastSeen = now
	m.lastPong = now
}

// Clear removes the connection when it is still the active one.
func (m *Manager) Clear(conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn == conn {
		m.conn = nil
	}
}

// TouchRead marks client activity from a successful read.
func (m *Manager) TouchRead() {
	m.mu.Lock()
	m.lastSeen = time.Now().UTC()
	m.mu.Unlock()
}

// TouchPong marks client activity from pong handler.
func (m *Manager) TouchPong() {
	m.mu.Lock()
	m.lastSeen = time.Now().UTC()
	m.lastPong = time.Now().UTC()
	m.mu.Unlock()
}

// Status returns connection snapshot for management API.
func (m *Manager) Status() (isConnected bool, connectedSince time.Time, lastSeen time.Time, hasPong bool, lastPong time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.conn == nil {
		return false, time.Time{}, time.Time{}, false, time.Time{}
	}
	return true, m.connectedAt, m.lastSeen, !m.lastPong.IsZero(), m.lastPong
}

// Notify sends the play-sound frame to the connected client.
func (m *Manager) Notify() error {
	m.mu.RLock()
	conn := m.conn
	m.mu.RUnlock()

	if conn == nil {
		return ErrNoClient
	}

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	deadline := time.Now().Add(5 * time.Second)
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return err
	}

	return conn.WriteMessage(websocket.BinaryMessage, []byte{protocol.PlaySound})
}

// SendJSON sends a single text JSON websocket message to the client.
func (m *Manager) SendJSON(v any) error {
	m.mu.RLock()
	conn := m.conn
	m.mu.RUnlock()
	if conn == nil {
		return ErrNoClient
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	deadline := time.Now().Add(60 * time.Second)
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, b)
}
