package api

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"pipeline-horn/internal/gitlab"
	applog "pipeline-horn/internal/log"
	"pipeline-horn/internal/loghub"
	"pipeline-horn/internal/notify"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Webhook accepts GitLab pipeline webhook payloads.
func (s *Server) Webhook(c *gin.Context) {
	logger := applog.LoggerFromContext(c.Request.Context())

	token := strings.TrimSpace(c.GetHeader(s.cfg.TokenHeader))
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.WebhookSecret)) != 1 {
		logger.Warn("webhook rejected: invalid token")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var payload gitlab.PipelineWebhook
	if err := c.ShouldBindJSON(&payload); err != nil {
		logger.Warn("webhook rejected: invalid payload", zap.Error(err))
		s.emitWebhookLog(c, payload, "parse_error", err.Error())
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !payload.IsFailedPipeline() {
		logger.Info(
			"webhook ignored: not a failed pipeline",
			zap.String("object_kind", payload.ObjectKind),
			zap.String("pipeline_status", payload.ObjectAttributes.Status),
		)
		s.emitWebhookLog(c, payload, "ignored_not_failed", "")
		c.Status(http.StatusOK)
		return
	}

	if !payload.UnderGroup(s.cfg.GroupPath) {
		logger.Info(
			"webhook ignored: project outside configured group",
			zap.String("project_path", payload.Project.PathWithNamespace),
			zap.String("group_path", s.cfg.GroupPath),
		)
		s.emitWebhookLog(c, payload, "ignored_outside_group", "")
		c.Status(http.StatusOK)
		return
	}

	logger.Info(
		"failed pipeline webhook accepted",
		zap.String("project_name", payload.Project.Name),
		zap.String("project_path", payload.Project.PathWithNamespace),
		zap.Int("pipeline_id", payload.ObjectAttributes.ID),
	)

	err := s.dispatcher.Dispatch(payload.Project.PathWithNamespace, payload.ObjectAttributes.ID)
	outcome := "dispatched"
	errMsg := ""
	switch {
	case err == nil:
	case errors.Is(err, notify.ErrSuppressed):
		outcome = "suppressed_cooldown"
	default:
		outcome = "notify_failed"
		errMsg = err.Error()
	}
	s.emitWebhookLog(c, payload, outcome, errMsg)

	c.Status(http.StatusOK)
}

func (s *Server) emitWebhookLog(c *gin.Context, payload gitlab.PipelineWebhook, outcome, errMsg string) {
	fields := map[string]any{
		"clientIp":         c.ClientIP(),
		"objectKind":       payload.ObjectKind,
		"pipelineStatus":   payload.ObjectAttributes.Status,
		"pipelineId":       payload.ObjectAttributes.ID,
		"projectPath":      payload.Project.PathWithNamespace,
		"projectName":      payload.Project.Name,
		"outcome":          outcome,
		"willNotifyBranch": payload.IsFailedPipeline() && payload.UnderGroup(s.cfg.GroupPath),
	}
	if errMsg != "" {
		fields["error"] = errMsg
	}
	level := "info"
	if outcome == "notify_failed" || outcome == "parse_error" {
		level = "warn"
	}
	ev := loghub.ServerLogEvent{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Logger:    "webhook",
		Message:   "webhook_received",
		EventType: "webhook_received",
		Fields:    fields,
	}
	s.serverHub.Publish(ev)
}
