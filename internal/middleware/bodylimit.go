package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit caps each request body at `maxBytes` to protect against
// denial-of-service through oversized payloads. The wrapped reader fails
// later reads (the ShouldBindJSON call in handlers) with an
// `http.MaxBytesError` once the cap is hit; the handler's standard
// validation-error path surfaces that as 400.
//
// This is a defense-in-depth measure — the reverse proxy / CDN in front
// of the server should also enforce a limit, but we don't rely on it.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
