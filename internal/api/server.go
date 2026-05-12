package api

import (
	"time"

	"go.uber.org/zap"
	"pipeline-horn/internal/config"
	"pipeline-horn/internal/notify"
	"pipeline-horn/internal/ws"
)

const cooldownInterval = 30 * time.Second

// Server holds shared runtime dependencies for HTTP handlers.
type Server struct {
	cfg        config.ServerConfig
	logger     *zap.Logger
	clients    *ws.Manager
	dispatcher *notify.Dispatcher
}

// NewServer builds handler dependencies from server config.
func NewServer(cfg config.ServerConfig, logger *zap.Logger) *Server {
	clients := ws.NewManager()
	cooldown := notify.NewCooldown(cooldownInterval)

	return &Server{
		cfg:        cfg,
		logger:     logger,
		clients:    clients,
		dispatcher: notify.NewDispatcher(cooldown, clients, logger),
	}
}
