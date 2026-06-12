package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns middleware that emits one structured log line per request
// using slog. The line includes status, method, path, latency in ms, the
// client IP, the request ID, and (when the handler stashed errors via
// gin's c.Error) the error chain.
//
// Pass the application slog.Logger; tests can swap in a no-op logger.
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		// Pick a level based on status — 5xx is an error, 4xx a warning,
		// everything else routine.
		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		attrs := []any{
			slog.String("request_id", GetRequestID(c)),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
			slog.String("ip", c.ClientIP()),
		}
		if errs := c.Errors.ByType(gin.ErrorTypePrivate).String(); errs != "" {
			attrs = append(attrs, slog.String("errors", errs))
		}

		log.LogAttrs(c.Request.Context(), level, "http_request", toAttrs(attrs)...)
	}
}

// toAttrs converts the variadic `any` slice (used to match slog's freeform
// API in the call site) into the typed []slog.Attr that LogAttrs expects.
// Skips anything that isn't already a slog.Attr.
func toAttrs(in []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(in))
	for _, v := range in {
		if a, ok := v.(slog.Attr); ok {
			out = append(out, a)
		}
	}
	return out
}

// Recovery returns middleware that catches panics, logs them with the
// request ID for correlation, and aborts the request with a 500. Replaces
// gin.Recovery() so the recovery log goes through our structured logger.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.LogAttrs(c.Request.Context(), slog.LevelError, "panic",
					slog.String("request_id", GetRequestID(c)),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.Any("panic", r),
				)
				c.AbortWithStatusJSON(500, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "internal server error",
					},
				})
			}
		}()
		c.Next()
	}
}
