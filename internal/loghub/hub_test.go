package loghub

import (
	"context"
	"testing"
	"time"
)

func TestHubPublishSubscribe(t *testing.T) {
	t.Parallel()
	h := NewHub(4)
	ch, cancel := h.Subscribe()
	defer cancel()

	h.Publish(PingPayload{Timestamp: time.Now().UTC()})

	ctx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()

	select {
	case <-ctx.Done():
		t.Fatal("timeout waiting for message")
	case b := <-ch:
		if len(b) == 0 {
			t.Fatal("empty payload")
		}
	}
}
