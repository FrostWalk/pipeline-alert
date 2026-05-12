package notify

import (
	"sync"
	"time"
)

// Cooldown limits how often notifications may pass through.
type Cooldown struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

func NewCooldown(interval time.Duration) *Cooldown {
	return &Cooldown{interval: interval}
}

// Allow reports whether a notification is allowed now and records it when true.
func (c *Cooldown) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if !c.last.IsZero() && now.Sub(c.last) < c.interval {
		return false
	}

	c.last = now
	return true
}

// Remaining returns time left in the active cooldown window.
func (c *Cooldown) Remaining() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.last.IsZero() {
		return 0
	}

	remaining := c.interval - time.Since(c.last)
	if remaining < 0 {
		return 0
	}
	return remaining
}
