package api

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	applog "pipeline-horn/internal/log"
	"pipeline-horn/internal/loghub"
	"pipeline-horn/internal/piws"
	"pipeline-horn/internal/sounds"

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
	session := &soundSyncSession{}
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
			s.handlePiWSMessage(payload, session)
		}
	}
}

type soundSyncSession struct {
	uploadName      string
	uploadSizeBytes int64
	uploadSHA256    string
	uploadIsDefault bool
	uploadOffset    int64
	uploadBuf       []byte
}

func (s *Server) handlePiWSMessage(payload []byte, syncSession *soundSyncSession) {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &probe); err != nil {
		return
	}
	switch probe.Type {
	case piws.TypePiLog:
		s.handlePiLog(payload)
	case piws.TypeSoundInventory:
		s.handleSoundInventory(payload)
	case piws.TypeSoundUploadStart:
		s.handleSoundUploadStart(payload, syncSession)
	case piws.TypeSoundUploadChunk:
		s.handleSoundUploadChunk(payload, syncSession)
	default:
		return
	}
}

func (s *Server) handlePiLog(payload []byte) {
	var msg piws.PiLog
	if err := json.Unmarshal(payload, &msg); err != nil {
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

func (s *Server) handleSoundInventory(payload []byte) {
	var msg piws.SoundInventory
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	clientByName := make(map[string]piws.SoundInventoryItem, len(msg.Sounds))
	for _, item := range msg.Sounds {
		name, ok := normalizeSoundFileName(item.FileName)
		if !ok {
			continue
		}
		item.FileName = name
		item.SHA256 = strings.ToLower(strings.TrimSpace(item.SHA256))
		clientByName[name] = item
	}

	serverSounds, err := s.sounds.List()
	if err != nil {
		return
	}
	serverByName := make(map[string]sounds.Info, len(serverSounds))
	for _, snd := range serverSounds {
		serverByName[snd.FileName] = snd
	}

	for _, snd := range serverSounds {
		clientItem, ok := clientByName[snd.FileName]
		if ok && strings.EqualFold(clientItem.SHA256, snd.SHA256) {
			continue
		}
		data, err := s.sounds.ReadFileBytes(snd.FileName, s.cfg.MaxSoundUploadBytes)
		if err != nil {
			continue
		}
		_ = piws.SendSound(func(v any) error { return s.clients.SendJSON(v) }, snd.FileName, data)
	}

	for _, item := range clientByName {
		serverItem, ok := serverByName[item.FileName]
		if ok && strings.EqualFold(serverItem.SHA256, item.SHA256) {
			continue
		}
		_ = s.clients.SendJSON(piws.SoundRequestUpload{
			Type:     piws.TypeSoundRequestUpload,
			FileName: item.FileName,
		})
	}
}

func (s *Server) handleSoundUploadStart(payload []byte, syncSession *soundSyncSession) {
	var msg piws.SoundUploadStart
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	fileName, ok := normalizeSoundFileName(msg.FileName)
	if !ok {
		s.resetSyncSession(syncSession)
		return
	}
	if msg.SizeBytes < 0 || msg.SizeBytes > s.cfg.MaxSoundUploadBytes {
		s.resetSyncSession(syncSession)
		return
	}

	syncSession.uploadName = fileName
	syncSession.uploadSizeBytes = msg.SizeBytes
	syncSession.uploadSHA256 = strings.ToLower(strings.TrimSpace(msg.SHA256))
	syncSession.uploadIsDefault = msg.IsDefault
	syncSession.uploadOffset = 0
	if msg.SizeBytes == 0 {
		_, _ = s.sounds.UpsertFromClient(fileName, nil, s.cfg.MaxSoundUploadBytes, syncSession.uploadSHA256, syncSession.uploadIsDefault)
		s.resetSyncSession(syncSession)
		return
	}
	syncSession.uploadBuf = make([]byte, msg.SizeBytes)
}

func (s *Server) handleSoundUploadChunk(payload []byte, syncSession *soundSyncSession) {
	if syncSession.uploadName == "" || syncSession.uploadBuf == nil {
		return
	}
	var msg piws.SoundUploadChunk
	if err := json.Unmarshal(payload, &msg); err != nil {
		s.resetSyncSession(syncSession)
		return
	}
	fileName, ok := normalizeSoundFileName(msg.FileName)
	if !ok || fileName != syncSession.uploadName {
		s.resetSyncSession(syncSession)
		return
	}
	if msg.Offset != syncSession.uploadOffset {
		s.resetSyncSession(syncSession)
		return
	}
	chunk, err := base64.StdEncoding.DecodeString(msg.DataB64)
	if err != nil {
		s.resetSyncSession(syncSession)
		return
	}
	if int64(len(chunk)) == 0 || msg.Offset+int64(len(chunk)) > syncSession.uploadSizeBytes {
		s.resetSyncSession(syncSession)
		return
	}
	copy(syncSession.uploadBuf[msg.Offset:], chunk)
	syncSession.uploadOffset += int64(len(chunk))
	if syncSession.uploadOffset != syncSession.uploadSizeBytes {
		return
	}

	if _, err := s.sounds.UpsertFromClient(
		syncSession.uploadName,
		syncSession.uploadBuf,
		s.cfg.MaxSoundUploadBytes,
		syncSession.uploadSHA256,
		syncSession.uploadIsDefault,
	); err != nil {
		s.resetSyncSession(syncSession)
		return
	}
	s.resetSyncSession(syncSession)
}

func (s *Server) resetSyncSession(syncSession *soundSyncSession) {
	syncSession.uploadName = ""
	syncSession.uploadSizeBytes = 0
	syncSession.uploadSHA256 = ""
	syncSession.uploadIsDefault = false
	syncSession.uploadOffset = 0
	syncSession.uploadBuf = nil
}

func normalizeSoundFileName(fileName string) (string, bool) {
	base := filepath.Base(strings.TrimSpace(fileName))
	if base == "" || base == "." || base == ".." || strings.HasPrefix(base, ".") || base != strings.TrimSpace(fileName) {
		return "", false
	}
	return base, true
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
