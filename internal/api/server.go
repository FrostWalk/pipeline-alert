package api

import (
	"time"

	"go.uber.org/zap"
	"pipeline-horn/internal/auth"
	"pipeline-horn/internal/config"
	"pipeline-horn/internal/loghub"
	"pipeline-horn/internal/notify"
	"pipeline-horn/internal/sounds"
	"pipeline-horn/internal/ws"
)

const cooldownInterval = 30 * time.Second

// Server holds shared runtime dependencies for HTTP handlers.
type Server struct {
	cfg        config.ServerConfig
	logger     *zap.Logger
	clients    *ws.Manager
	dispatcher *notify.Dispatcher

	jwt       *auth.JWT
	sounds    *sounds.Store
	serverHub *loghub.Hub
	piHub     *loghub.Hub
}

// NewServer builds handler dependencies from server config.
func NewServer(cfg config.ServerConfig, logger *zap.Logger, serverHub, piHub *loghub.Hub, soundStore *sounds.Store) *Server {
	clients := ws.NewManager()
	cooldown := notify.NewCooldown(cooldownInterval)

	return &Server{
		cfg:        cfg,
		logger:     logger,
		clients:    clients,
		dispatcher: notify.NewDispatcher(cooldown, clients, logger),
		jwt:        auth.NewJWT(cfg),
		sounds:     soundStore,
		serverHub:  serverHub,
		piHub:      piHub,
	}
}
