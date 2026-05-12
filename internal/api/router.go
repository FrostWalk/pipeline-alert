package api

import (
	applog "pipeline-horn/internal/log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter builds the HTTP router for the server.
func NewRouter(server *Server) *gin.Engine {
	router := gin.Default()
	router.Use(loggerMiddleware(server.logger))
	router.GET("/healthz", server.Healthz)
	router.POST("/webhook", server.Webhook)
	router.GET("/ws", server.Websocket)

	return router
}

func loggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(applog.ContextWithLogger(c.Request.Context(), logger))
		c.Next()
	}
}
