package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func init() { gin.SetMode(gin.TestMode) }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ── RequestID ─────────────────────────────────────────────────────────

func TestRequestIDMintsWhenAbsent(t *testing.T) {
	var captured string
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		captured = GetRequestID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(captured) {
		t.Errorf("minted request id = %q, want 32 hex chars", captured)
	}
	if got := w.Header().Get(RequestIDHeader); got != captured {
		t.Errorf("response %s = %q, want %q (echo of the context id)", RequestIDHeader, got, captured)
	}
}

func TestRequestIDHonorsIncomingHeader(t *testing.T) {
	const incoming = "upstream-correlation-id-123"
	var captured string
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		captured = GetRequestID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, incoming)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if captured != incoming {
		t.Errorf("request id = %q, want the incoming %q", captured, incoming)
	}
	if got := w.Header().Get(RequestIDHeader); got != incoming {
		t.Errorf("response %s = %q, want %q", RequestIDHeader, got, incoming)
	}
}

func TestGetRequestIDMissing(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if got := GetRequestID(c); got != "" {
		t.Errorf("GetRequestID without middleware = %q, want empty", got)
	}
}

// ── RateLimiter ───────────────────────────────────────────────────────

func TestRateLimiterAllowRespectsBurst(t *testing.T) {
	// rate.Every(time.Hour) → effectively no refill during the test, so the
	// bucket is exactly `burst` tokens and behaviour is deterministic.
	l := NewIPRateLimiter(rate.Every(time.Hour), 2, time.Minute)

	if !l.allow("1.2.3.4") || !l.allow("1.2.3.4") {
		t.Fatal("first two requests within burst should be allowed")
	}
	if l.allow("1.2.3.4") {
		t.Error("third request over burst should be denied")
	}
	// A different IP has its own bucket.
	if !l.allow("5.6.7.8") {
		t.Error("a fresh IP should be allowed despite another IP being throttled")
	}
}

func TestRateLimiterLazyGC(t *testing.T) {
	l := NewIPRateLimiter(rate.Every(time.Hour), 1, 10*time.Millisecond)

	l.allow("1.1.1.1")
	if len(l.clients) != 1 {
		t.Fatalf("expected 1 tracked client, got %d", len(l.clients))
	}

	// Wait past the ttl, then touch a different IP — the stale entry should be
	// swept during that lookup, leaving only the new one.
	time.Sleep(20 * time.Millisecond)
	l.allow("2.2.2.2")

	if _, stale := l.clients["1.1.1.1"]; stale {
		t.Error("stale IP should have been GC'd after ttl")
	}
	if len(l.clients) != 1 {
		t.Errorf("expected only the fresh client, got %d entries", len(l.clients))
	}
}

func TestRateLimiterMiddleware429(t *testing.T) {
	l := NewIPRateLimiter(rate.Every(time.Hour), 1, time.Minute)
	r := gin.New()
	r.Use(l.Middleware())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	first := httptest.NewRecorder()
	r.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request = %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	r.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request = %d, want 429", second.Code)
	}
	if !strings.Contains(second.Body.String(), "RATE_LIMITED") {
		t.Errorf("429 body = %q, want it to carry the RATE_LIMITED code", second.Body.String())
	}
}

// ── Recovery ──────────────────────────────────────────────────────────

func TestRecoveryConvertsPanicTo500(t *testing.T) {
	r := gin.New()
	r.Use(Recovery(discardLogger()))
	r.GET("/boom", func(c *gin.Context) { panic("kaboom") })

	w := httptest.NewRecorder()
	// If Recovery didn't catch the panic this call would crash the test.
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic response = %d, want 500", w.Code)
	}
	if !strings.Contains(w.Body.String(), "INTERNAL_ERROR") {
		t.Errorf("500 body = %q, want the INTERNAL_ERROR code (no panic detail leaked)", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "kaboom") {
		t.Error("panic message leaked into the response body")
	}
}

// ── BodyLimit ─────────────────────────────────────────────────────────

func TestBodyLimit(t *testing.T) {
	r := gin.New()
	r.Use(BodyLimit(10))
	r.POST("/", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	under := httptest.NewRecorder()
	r.ServeHTTP(under, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345")))
	if under.Code != http.StatusOK {
		t.Errorf("5-byte body = %d, want 200", under.Code)
	}

	over := httptest.NewRecorder()
	r.ServeHTTP(over, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 50))))
	if over.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("50-byte body over 10-byte cap = %d, want 413 (read past cap should error)", over.Code)
	}
}
