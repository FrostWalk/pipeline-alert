package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Healthz reports whether the server process is ready to serve traffic.
func (s *Server) Healthz(c *gin.Context) {
	c.Status(http.StatusOK)
}
