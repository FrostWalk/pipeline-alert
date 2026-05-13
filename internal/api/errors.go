package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorBody matches OpenAPI error envelope.
type ErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details any    `json:"details,omitempty"`
	} `json:"error"`
}

func writeError(c *gin.Context, status int, code, message string, details any) {
	var body ErrorBody
	body.Error.Code = code
	body.Error.Message = message
	body.Error.Details = details
	c.JSON(status, body)
}

func abortUnauthorized(c *gin.Context, message string) {
	writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
	c.Abort()
}

func abortValidation(c *gin.Context, message string, details any) {
	writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message, details)
	c.Abort()
}
