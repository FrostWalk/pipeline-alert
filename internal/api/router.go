package api

import (
	applog "pipeline-horn/internal/log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter builds the HTTP router for the server.
func NewRouter(server *Server) *gin.Engine {
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20
	router.Use(loggerMiddleware(server.logger))

	router.POST("/auth/login", server.Login)

	apiGroup := router.Group("/api")
	apiGroup.Use(server.jwtAuthMiddleware())
	{
		apiGroup.GET("/integration/tokens", server.IntegrationTokens)
		apiGroup.GET("/pi/status", server.PiStatus)
		apiGroup.GET("/pi/sounds", server.PiListSounds)
		apiGroup.POST("/pi/sounds", server.PiUploadSound)
		apiGroup.POST("/pi/sounds/select", server.PiSelectSound)
		apiGroup.GET("/logs/server/stream", server.LogsServerStream)
		apiGroup.GET("/logs/pi/stream", server.LogsPiStream)
	}

	router.POST("/webhook", server.Webhook)
	router.GET("/ws", server.Websocket)

	router.NoRoute(spaHandler())

	return router
}

func loggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(applog.ContextWithLogger(c.Request.Context(), logger))
		c.Next()
	}
}
