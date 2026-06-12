package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter is a per-IP token-bucket limiter used to throttle auth
// endpoints. In-memory and single-instance: when we move to multiple
// replicas this needs to be backed by Redis (or fronted by a CDN/WAF that
// can enforce limits before the request reaches us).
//
// Old IP entries are GC'd lazily — every time we look up an IP we also
// drop anything not seen for `ttl`. That keeps memory bounded without a
// background goroutine.
type IPRateLimiter struct {
	rate  rate.Limit
	burst int
	ttl   time.Duration

	mu      sync.Mutex
	clients map[string]*ipClient
}

type ipClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter returns a limiter that allows `r` requests per second
// per IP with a burst of `burst`. Entries are forgotten after `ttl` of
// inactivity (keeps the map from growing unboundedly).
//
// Reasonable defaults for /auth/login: 5 rps with burst 10, ttl 10m.
func NewIPRateLimiter(r rate.Limit, burst int, ttl time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		rate:    r,
		burst:   burst,
		ttl:     ttl,
		clients: make(map[string]*ipClient),
	}
}

// Middleware returns gin middleware that rejects requests exceeding the
// per-IP rate with HTTP 429.
func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "too many requests, slow down",
				},
			})
			return
		}
		c.Next()
	}
}

func (l *IPRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Lazy GC: drop any client we haven't seen in `ttl`. Cheap because we
	// only iterate when a new request comes in anyway.
	for k, v := range l.clients {
		if now.Sub(v.lastSeen) > l.ttl {
			delete(l.clients, k)
		}
	}

	c, ok := l.clients[ip]
	if !ok {
		c = &ipClient{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.clients[ip] = c
	}
	c.lastSeen = now

	return c.limiter.Allow()
}
