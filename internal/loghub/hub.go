package loghub

import (
	"encoding/json"
	"sync"
	"time"
)

// Hub fans out JSON log lines to SSE subscribers (non-blocking publish).
type Hub struct {
	cap  int
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func NewHub(capacity int) *Hub {
	if capacity < 1 {
		capacity = 1
	}
	return &Hub{cap: capacity, subs: make(map[chan []byte]struct{})}
}

// Subscribe returns a channel of JSON-encoded events; caller must call cancel when done.
func (h *Hub) Subscribe() (ch <-chan []byte, cancel func()) {
	c := make(chan []byte, h.cap)
	h.mu.Lock()
	h.subs[c] = struct{}{}
	h.mu.Unlock()

	return c, func() {
		h.mu.Lock()
		delete(h.subs, c)
		h.mu.Unlock()
		close(c)
	}
}

// Publish sends one JSON object line; drops if subscriber buffer full.
func (h *Hub) Publish(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- b:
		default:
			// drop for slow consumer
		}
	}
}

// PingPayload is a minimal keepalive object for SSE `ping` events.
type PingPayload struct {
	Timestamp time.Time `json:"timestamp"`
}
