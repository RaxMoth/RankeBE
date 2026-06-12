package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// RequestIDKey is the gin.Context key under which the request ID is stored.
const RequestIDKey = "requestID"

// RequestIDHeader is the HTTP header used both to ingest a caller-supplied
// request ID (e.g. from an upstream load balancer) and to surface ours back
// to the client.
const RequestIDHeader = "X-Request-Id"

// RequestID returns middleware that ensures every request has a stable ID
// used for log correlation. Honors an incoming X-Request-Id header so a
// caller-supplied ID can be threaded through a request that fans out to
// multiple services; otherwise mints a fresh 16-byte hex token.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Set(RequestIDKey, id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

// GetRequestID returns the request ID for the current request, or "" if
// the RequestID middleware did not run.
func GetRequestID(c *gin.Context) string {
	v, ok := c.Get(RequestIDKey)
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// newRequestID returns a 32-char hex string (16 random bytes). Not a UUID
// (no version/variant nibbles) — but for log correlation we only need
// collision resistance, not RFC 4122 conformance.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand reads cannot fail on supported platforms; treat as
		// programmer error.
		panic("requestid: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}
