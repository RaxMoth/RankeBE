package middleware

import (
	"time"

	"gin-rest-template/pkg/logger"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		status := c.Writer.Status()

		// Log request details
		fields := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       path,
			"query":      query,
			"status":     status,
			"latency":    latency.String(),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		// Add error if exists
		if len(c.Errors) > 0 {
			fields["error"] = c.Errors.String()
		}

		// Log based on status code
		if status >= 500 {
			logger.Error("Server error", fields)
		} else if status >= 400 {
			logger.Warn("Client error", fields)
		} else {
			logger.Info("Request completed", fields)
		}
	}
}
