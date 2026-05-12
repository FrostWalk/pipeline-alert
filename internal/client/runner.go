package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"pipeline-horn/internal/client/audio"
	"pipeline-horn/internal/config"
	"pipeline-horn/internal/protocol"
)

const (
	minReconnectDelay = time.Second
	maxReconnectDelay = time.Minute
	stableConnection  = 30 * time.Second
)

// Run maintains a persistent websocket connection and plays sounds on notify frames.
func Run(ctx context.Context, cfg config.ClientConfig, logger *zap.Logger) error {
	backoff := minReconnectDelay

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		stable, err := connectOnce(ctx, cfg, logger)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}

			delay := backoffWithJitter(backoff)
			logger.Warn("websocket connection lost", zap.Error(err), zap.Duration("retry_in", delay))
			if !sleep(ctx, delay) {
				return ctx.Err()
			}

			backoff = nextBackoff(backoff, stable)
			continue
		}

		backoff = minReconnectDelay
	}
}

func connectOnce(ctx context.Context, cfg config.ClientConfig, logger *zap.Logger) (bool, error) {
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 15 * time.Second,
	}

	if cfg.AcceptInvalidTLS {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicit operator opt-in
	}

	url, err := websocketURL(cfg)
	if err != nil {
		return false, err
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+cfg.WebsocketSecret)

	logger.Info("connecting websocket", zap.String("url", url))
	conn, _, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		return false, fmt.Errorf("dial websocket: %w", err)
	}
	defer conn.Close()

	logger.Info("websocket connected")
	connectedAt := time.Now()

	for {
		if err := ctx.Err(); err != nil {
			return time.Since(connectedAt) >= stableConnection, err
		}

		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return time.Since(connectedAt) >= stableConnection, fmt.Errorf("read websocket message: %w", err)
		}

		if messageType != websocket.BinaryMessage || !protocol.IsPlaySound(payload) {
			continue
		}

		logger.Info("play sound notification received")
		if err := audio.Play(cfg.SoundPath); err != nil {
			logger.Error("play sound failed", zap.Error(err))
		}
	}
}

func websocketURL(cfg config.ClientConfig) (string, error) {
	host := strings.TrimSpace(cfg.ServerURL)
	if host == "" {
		return "", fmt.Errorf("server_url is required")
	}

	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")

	scheme := "wss"
	if cfg.ServerPort == 80 {
		scheme = "ws"
	}

	return fmt.Sprintf("%s://%s:%d/ws", scheme, host, cfg.ServerPort), nil
}

func sleep(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func backoffWithJitter(delay time.Duration) time.Duration {
	jitter := time.Duration(rand.Int63n(int64(delay / 2)))
	return delay + jitter
}

func nextBackoff(current time.Duration, stable bool) time.Duration {
	if stable {
		return minReconnectDelay
	}

	next := current * 2
	if next > maxReconnectDelay {
		return maxReconnectDelay
	}
	return next
}
