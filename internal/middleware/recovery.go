package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/edaptix/server/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				response.Error(c, http.StatusInternalServerError, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// notImplemented is a placeholder handler returning 501
func NotImplemented(c *gin.Context) {
	response.Error(c, http.StatusNotImplemented, "not implemented")
}

// placeholderHandler returns a 501 handler with a label for logging
func PlaceholderHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = fmt.Sprintf("handler %s not implemented", name)
		response.Error(c, http.StatusNotImplemented, "not implemented")
	}
}
