package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	applog "pipeline-horn/internal/log"
	"pipeline-horn/internal/loghub"
	"pipeline-horn/internal/piws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

// Websocket upgrades authenticated clients to a persistent notify channel.
func (s *Server) Websocket(c *gin.Context) {
	logger := applog.LoggerFromContext(c.Request.Context())

	token, ok := bearerToken(c.GetHeader("Authorization"))
	if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.WebsocketSecret)) != 1 {
		logger.Warn("websocket rejected: invalid token")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	conn, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Warn("websocket upgrade failed", zap.Error(err))
		return
	}

	s.clients.Replace(conn)
	logger.Info("websocket client connected", zap.String("remote_addr", c.ClientIP()))
	defer func() {
		s.clients.Clear(conn)
		_ = conn.Close()
		logger.Info("websocket client disconnected", zap.String("remote_addr", c.ClientIP()))
	}()

	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		s.clients.TouchPong()
		return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	})

	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-pingDone:
				return
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			}
		}
	}()
	defer close(pingDone)

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warn("websocket read failed", zap.Error(err))
			}
			return
		}
		s.clients.TouchRead()
		if messageType == websocket.TextMessage {
			s.handlePiWSMessage(payload)
		}
	}
}

func (s *Server) handlePiWSMessage(payload []byte) {
	var msg piws.PiLog
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	if msg.Type != piws.TypePiLog {
		return
	}
	lvl := strings.TrimSpace(msg.Level)
	if lvl == "" {
		lvl = "info"
	}
	ev := loghub.PiLogEvent{
		Timestamp: time.Now().UTC(),
		Level:     lvl,
		Message:   msg.Message,
		Fields:    msg.Fields,
	}
	s.piHub.Publish(ev)
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}

	return token, true
}
