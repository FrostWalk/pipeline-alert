package client

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"pipeline-horn/internal/client/audio"
	"pipeline-horn/internal/client/wslog"
	"pipeline-horn/internal/config"
	"pipeline-horn/internal/piws"
	"pipeline-horn/internal/protocol"
)

const (
	minReconnectDelay = time.Second
	maxReconnectDelay = time.Minute
	stableConnection  = 30 * time.Second
)

type selectedFile struct {
	FileName string `json:"fileName"`
}

// Run maintains a persistent websocket connection and plays sounds on notify frames.
func Run(ctx context.Context, cfg config.ClientConfig, logger *zap.Logger) error {
	if err := config.EnsureSoundDir(cfg.SoundDir); err != nil {
		return fmt.Errorf("sound dir: %w", err)
	}

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
			wslog.PiLog("warn", fmt.Sprintf("websocket connection lost: %v", err))
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
	wslog.PiLog("info", "connecting websocket "+url)

	conn, _, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		return false, fmt.Errorf("dial websocket: %w", err)
	}
	defer conn.Close()
	wslog.SetConn(conn)
	defer wslog.Clear(conn)

	logger.Info("websocket connected")
	wslog.PiLog("info", "websocket connected")

	connectedAt := time.Now()

	var syncName string
	var syncTotal int64
	var syncBuf []byte

	for {
		if err := ctx.Err(); err != nil {
			return time.Since(connectedAt) >= stableConnection, err
		}

		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return time.Since(connectedAt) >= stableConnection, fmt.Errorf("read websocket message: %w", err)
		}

		switch messageType {
		case websocket.BinaryMessage:
			if protocol.IsPlaySound(payload) {
				logger.Info("play sound notification received")
				wslog.PiLog("info", "play sound notification received")
				playPath := resolvePlayPath(cfg)
				if err := audio.Play(playPath); err != nil {
					logger.Error("play sound failed", zap.Error(err), zap.String("path", playPath))
					wslog.PiLog("error", fmt.Sprintf("play sound failed: %v path=%s", err, playPath))
				}
			}
		case websocket.TextMessage:
			if err := handleTextControl(cfg, logger, payload, &syncName, &syncTotal, &syncBuf); err != nil {
				logger.Warn("control message error", zap.Error(err))
				wslog.PiLog("warn", fmt.Sprintf("control message error: %v", err))
			}
		default:
			continue
		}
	}
}

func handleTextControl(cfg config.ClientConfig, logger *zap.Logger, payload []byte, syncName *string, syncTotal *int64, syncBuf *[]byte) error {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &probe); err != nil {
		return err
	}
	switch probe.Type {
	case piws.TypeSoundSyncStart:
		var m piws.SoundSyncStart
		if err := json.Unmarshal(payload, &m); err != nil {
			return err
		}
		if m.SizeBytes < 0 || m.SizeBytes > 50<<20 {
			return fmt.Errorf("invalid sync size")
		}
		base := filepath.Base(m.FileName)
		if base != m.FileName || strings.HasPrefix(base, ".") {
			return fmt.Errorf("invalid file name")
		}
		*syncName = base
		*syncTotal = m.SizeBytes
		if m.SizeBytes == 0 {
			dst := filepath.Join(cfg.SoundDir, base)
			if err := os.WriteFile(dst, nil, 0o640); err != nil {
				return err
			}
			logger.Info("sound sync complete", zap.String("file", base))
			wslog.PiLog("info", fmt.Sprintf("sound sync complete file=%s", base))
			*syncName = ""
			*syncTotal = 0
			*syncBuf = nil
			return nil
		}
		*syncBuf = make([]byte, m.SizeBytes)
		logger.Info("sound sync start", zap.String("file", base), zap.Int64("bytes", m.SizeBytes))
		wslog.PiLog("info", fmt.Sprintf("sound sync start file=%s bytes=%d", base, m.SizeBytes))

	case piws.TypeSoundSyncChunk:
		var m piws.SoundSyncChunk
		if err := json.Unmarshal(payload, &m); err != nil {
			return err
		}
		base := filepath.Base(m.FileName)
		if *syncName == "" || base != *syncName || *syncBuf == nil {
			return fmt.Errorf("chunk for unexpected file")
		}
		if m.Offset < 0 || m.Offset > *syncTotal {
			return fmt.Errorf("invalid offset")
		}
		chunk, err := base64.StdEncoding.DecodeString(m.DataB64)
		if err != nil {
			return err
		}
		if m.Offset+int64(len(chunk)) > *syncTotal {
			return fmt.Errorf("chunk overflows declared size")
		}
		copy((*syncBuf)[m.Offset:], chunk)

		if m.Offset+int64(len(chunk)) == *syncTotal {
			dst := filepath.Join(cfg.SoundDir, *syncName)
			if err := os.WriteFile(dst, *syncBuf, 0o640); err != nil {
				return err
			}
			logger.Info("sound sync complete", zap.String("file", *syncName))
			wslog.PiLog("info", fmt.Sprintf("sound sync complete file=%s", *syncName))
			*syncName = ""
			*syncTotal = 0
			*syncBuf = nil
		}

	case piws.TypeSetActiveSound:
		var m piws.SetActiveSound
		if err := json.Unmarshal(payload, &m); err != nil {
			return err
		}
		base := filepath.Base(m.FileName)
		if base != m.FileName || strings.HasPrefix(base, ".") {
			return fmt.Errorf("invalid file name")
		}
		if err := writeSelectedName(cfg.SoundDir, base); err != nil {
			return err
		}
		logger.Info("active sound updated", zap.String("file", base))
		wslog.PiLog("info", fmt.Sprintf("active sound updated file=%s", base))

	default:
		return nil
	}
	return nil
}

func resolvePlayPath(cfg config.ClientConfig) string {
	name, err := readSelectedName(cfg.SoundDir)
	if err != nil || strings.TrimSpace(name) == "" {
		return cfg.SoundPath
	}
	candidate := filepath.Join(cfg.SoundDir, filepath.Base(name))
	if fi, err := os.Stat(candidate); err == nil && fi.Mode().IsRegular() {
		return candidate
	}
	return cfg.SoundPath
}

func readSelectedName(dir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(dir, ".selected"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var st selectedFile
	if err := json.Unmarshal(b, &st); err != nil {
		return "", err
	}
	return strings.TrimSpace(st.FileName), nil
}

func writeSelectedName(dir, fileName string) error {
	st := selectedFile{FileName: fileName}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, ".selected.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, ".selected"))
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
