package notify

import (
	"errors"

	"go.uber.org/zap"
	"pipeline-horn/internal/ws"
)

var ErrSuppressed = errors.New("notification suppressed by cooldown")

// Dispatcher applies cooldown and forwards alerts to websocket clients.
type Dispatcher struct {
	cooldown *Cooldown
	clients  *ws.Manager
	logger   *zap.Logger
}

func NewDispatcher(cooldown *Cooldown, clients *ws.Manager, logger *zap.Logger) *Dispatcher {
	return &Dispatcher{
		cooldown: cooldown,
		clients:  clients,
		logger:   logger,
	}
}

// Dispatch attempts to notify the connected client.
func (d *Dispatcher) Dispatch(projectPath string, pipelineID int) error {
	if !d.cooldown.Allow() {
		d.logger.Info(
			"notification suppressed by cooldown",
			zap.String("project_path", projectPath),
			zap.Int("pipeline_id", pipelineID),
			zap.Duration("cooldown_remaining", d.cooldown.Remaining()),
		)
		return ErrSuppressed
	}

	if err := d.clients.Notify(); err != nil {
		d.logger.Warn(
			"notification not delivered",
			zap.String("project_path", projectPath),
			zap.Int("pipeline_id", pipelineID),
			zap.Error(err),
		)
		return err
	}

	d.logger.Info(
		"notification delivered",
		zap.String("project_path", projectPath),
		zap.Int("pipeline_id", pipelineID),
	)
	return nil
}
