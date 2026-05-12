package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"pipeline-horn/internal/gitlab"
	applog "pipeline-horn/internal/log"

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
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !payload.IsFailedPipeline() {
		logger.Info(
			"webhook ignored: not a failed pipeline",
			zap.String("object_kind", payload.ObjectKind),
			zap.String("pipeline_status", payload.ObjectAttributes.Status),
		)
		c.Status(http.StatusOK)
		return
	}

	if !payload.UnderGroup(s.cfg.GroupPath) {
		logger.Info(
			"webhook ignored: project outside configured group",
			zap.String("project_path", payload.Project.PathWithNamespace),
			zap.String("group_path", s.cfg.GroupPath),
		)
		c.Status(http.StatusOK)
		return
	}

	logger.Info(
		"failed pipeline webhook accepted",
		zap.String("project_name", payload.Project.Name),
		zap.String("project_path", payload.Project.PathWithNamespace),
		zap.Int("pipeline_id", payload.ObjectAttributes.ID),
	)

	_ = s.dispatcher.Dispatch(payload.Project.PathWithNamespace, payload.ObjectAttributes.ID)
	c.Status(http.StatusOK)
}
