package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"pipeline-horn/internal/auth"
	"pipeline-horn/internal/loghub"
	"pipeline-horn/internal/piws"
	"pipeline-horn/internal/sounds"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken      string `json:"accessToken"`
	TokenType        string `json:"tokenType"`
	ExpiresInSeconds int64  `json:"expiresInSeconds"`
}

// Login exchanges credentials for JWT.
func (s *Server) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid JSON body", nil)
		return
	}
	tok, ttl, err := s.jwt.Login(req.Username, req.Password)
	if errors.Is(err, auth.ErrEmptyLoginFields) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	if errors.Is(err, auth.ErrInvalidCredentials) {
		writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL", "login failed", nil)
		return
	}
	c.JSON(http.StatusOK, loginResponse{
		AccessToken:      tok,
		TokenType:        "Bearer",
		ExpiresInSeconds: int64(ttl.Seconds()),
	})
}

func (s *Server) jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			if q := strings.TrimSpace(c.Query("accessToken")); q != "" {
				header = "Bearer " + q
			}
		}
		sub, err := s.jwt.ParseBearer(header)
		if err != nil {
			abortUnauthorized(c, "invalid or missing token")
			return
		}
		c.Set("jwtSubject", sub)
		c.Next()
	}
}

type piStatusResponse struct {
	IsConnected      bool       `json:"isConnected"`
	ConnectedSince   *time.Time `json:"connectedSince,omitempty"`
	LastSeenAt       *time.Time `json:"lastSeenAt,omitempty"`
	SelectedFileName string     `json:"selectedFileName"`
}

// PiStatus returns websocket + sound selection snapshot.
func (s *Server) PiStatus(c *gin.Context) {
	ok, since, last, _, _ := s.clients.Status()
	sel, err := s.sounds.Selected()
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL", "failed to read selection", nil)
		return
	}
	var sincePtr, lastPtr *time.Time
	if ok {
		sincePtr = new(since)
		lastPtr = new(last)
	}
	c.JSON(http.StatusOK, piStatusResponse{
		IsConnected:      ok,
		ConnectedSince:   sincePtr,
		LastSeenAt:       lastPtr,
		SelectedFileName: sel,
	})
}

type soundListResponse struct {
	Sounds           []soundInfo `json:"sounds"`
	SelectedFileName string      `json:"selectedFileName"`
}

type soundInfo struct {
	FileName    string    `json:"fileName"`
	SizeBytes   int64     `json:"sizeBytes"`
	ContentType string    `json:"contentType"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PiListSounds lists stored sounds.
func (s *Server) PiListSounds(c *gin.Context) {
	list, err := s.sounds.List()
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL", "failed to list sounds", nil)
		return
	}
	sel, err := s.sounds.Selected()
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL", "failed to read selection", nil)
		return
	}
	out := make([]soundInfo, 0, len(list))
	for _, x := range list {
		out = append(out, soundInfo{
			FileName:    x.FileName,
			SizeBytes:   x.SizeBytes,
			ContentType: x.ContentType,
			UpdatedAt:   x.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, soundListResponse{Sounds: out, SelectedFileName: sel})
}

type soundUploadResponse struct {
	FileName   string `json:"fileName"`
	SizeBytes  int64  `json:"sizeBytes"`
	SyncedToPi bool   `json:"syncedToPi"`
}

// PiUploadSound accepts multipart file upload.
func (s *Server) PiUploadSound(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, s.cfg.MaxSoundUploadBytes+1024)

	fh, err := c.FormFile("file")
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "file too large", nil)
			return
		}
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "missing or invalid multipart file field `file`", nil)
		return
	}

	src, err := fh.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "cannot read upload", nil)
		return
	}
	defer func() { _ = src.Close() }()

	baseName := filepath.Base(fh.Filename)
	n, err := s.sounds.SaveUploaded(baseName, src, s.cfg.MaxSoundUploadBytes)
	if errors.Is(err, sounds.ErrExists) {
		writeError(c, http.StatusConflict, "CONFLICT", "sound with this file name already exists", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	synced := false
	data, rerr := s.sounds.ReadFileBytes(baseName, s.cfg.MaxSoundUploadBytes)
	if rerr == nil {
		if err := piws.SendSound(func(v any) error { return s.clients.SendJSON(v) }, baseName, data); err == nil {
			synced = true
		}
	}

	c.JSON(http.StatusCreated, soundUploadResponse{
		FileName:   baseName,
		SizeBytes:  n,
		SyncedToPi: synced,
	})
}

type selectSoundRequest struct {
	FileName string `json:"fileName"`
}

type selectSoundResponse struct {
	SelectedFileName string `json:"selectedFileName"`
	AppliedToPi      bool   `json:"appliedToPi"`
}

// PiSelectSound sets active sound for notifications.
func (s *Server) PiSelectSound(c *gin.Context) {
	var req selectSoundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid JSON body", nil)
		return
	}
	name := strings.TrimSpace(req.FileName)
	if name == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "fileName is required", nil)
		return
	}
	ok, err := s.sounds.Has(name)
	if err != nil || !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "unknown sound file", nil)
		return
	}
	if err := s.sounds.SetSelected(name); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL", "failed to persist selection", nil)
		return
	}
	applied := false
	if err := s.clients.SendJSON(piws.SetActiveSound{
		Type:     piws.TypeSetActiveSound,
		FileName: name,
	}); err == nil {
		applied = true
	}
	c.JSON(http.StatusOK, selectSoundResponse{
		SelectedFileName: name,
		AppliedToPi:      applied,
	})
}

// LogsServerStream streams server logs via SSE.
func (s *Server) LogsServerStream(c *gin.Context) {
	s.sseLogStream(c, s.serverHub)
}

// LogsPiStream streams Raspberry Pi client logs via SSE.
func (s *Server) LogsPiStream(c *gin.Context) {
	s.sseLogStream(c, s.piHub)
}

func (s *Server) sseLogStream(c *gin.Context, hub *loghub.Hub) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		writeError(c, http.StatusInternalServerError, "INTERNAL", "streaming unsupported", nil)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	flusher.Flush()

	ch, cancel := hub.Subscribe()
	defer cancel()

	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ping.C:
			b, _ := json.Marshal(loghub.PingPayload{Timestamp: time.Now().UTC()})
			writeSSEData(c.Writer, flusher, "ping", b)
		case row, open := <-ch:
			if !open {
				return
			}
			writeSSEData(c.Writer, flusher, "log", row)
		}
	}
}

func writeSSEData(w http.ResponseWriter, f http.Flusher, event string, data []byte) {
	_, _ = fmt.Fprintf(w, "event: %s\n", event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	f.Flush()
}
