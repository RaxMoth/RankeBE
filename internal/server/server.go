// Package server wires the HTTP router, middleware, and handlers
// together. Extracted from cmd/server/main.go so integration tests can
// stand up the same router without re-implementing the assembly.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"

	"ranke-be/internal/apple"
	"ranke-be/internal/config"
	db "ranke-be/internal/db/sqlc"
	"ranke-be/internal/handler"
	"ranke-be/internal/middleware"
	"ranke-be/internal/service"
)

// App is the assembled HTTP application — engine plus the dependencies
// callers may want to poke at directly in tests (Pool for truncating
// between cases, Logger for assertions on log output).
type App struct {
	Engine *gin.Engine
	Pool   *pgxpool.Pool
	Cfg    *config.Config
	Logger *slog.Logger
}

// New constructs the full router. The caller owns the pool and is
// responsible for closing it; New does not start a listener.
//
// `appleVerifier` may be nil — in that case the Apple-related endpoints
// degrade gracefully (sign-in returns an error, notifications 200s).
func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger, appleVerifier *apple.Verifier) *App {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	queries := db.New(pool)

	// Services
	authSvc := service.NewAuthService(queries, pool, cfg, appleVerifier)
	listSvc := service.NewListService(queries, pool)
	entrySvc := service.NewEntryService(queries, pool, listSvc)

	// Handlers
	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(queries, listSvc)
	listH := handler.NewListHandler(listSvc)
	entryH := handler.NewEntryHandler(entrySvc, listSvc)
	appleNotifyH := handler.NewAppleNotificationHandler(appleVerifier, queries, logger)

	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.BodyLimit(cfg.MaxBodyBytes),
	)
	router.Use(cors.New(buildCORSConfig(cfg)))

	// Liveness — cheap, no dependency check.
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	// Readiness — DB ping with a short ctx so a stuck pool doesn't
	// tie up the probe.
	router.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unready",
				"error":  "database unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	v1 := router.Group("/api/v1")

	// Per-IP rate limit on auth endpoints — humans retry on typos,
	// credential stuffers don't.
	authLimiter := middleware.NewIPRateLimiter(rate.Limit(5), 10, 10*time.Minute)

	// Auth — public, rate-limited. Logout lives here too because a
	// client whose access token has expired still needs to revoke its
	// refresh token; logout authenticates by presenting the refresh
	// token in the body, not by Bearer.
	auth := v1.Group("/auth")
	auth.Use(authLimiter.Middleware())
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/apple", authH.AppleSignIn)
		auth.POST("/refresh", authH.Refresh)
		auth.POST("/logout", authH.Logout)
	}

	// Apple server-to-server notifications — public, not rate-limited.
	// Apple's posts are infrequent, and a 429 would cause them to retry
	// in a tight loop.
	v1.POST("/auth/apple/notifications", appleNotifyH.Handle)

	// Authenticated routes
	protected := v1.Group("")
	protected.Use(middleware.AuthRequired(cfg.JWTSecret))
	{
		// Users
		protected.GET("/users/me", userH.GetMe)
		protected.PATCH("/users/me", userH.UpdateMe)
		protected.DELETE("/users/me", userH.DeleteMe) // App Store 5.1.1(v)
		protected.GET("/users/:id/profile", userH.GetUserProfile)

		// Lists — caller's lists + public discover + create
		protected.GET("/lists", listH.GetUserLists)
		protected.GET("/lists/public", listH.SearchPublic)
		protected.POST("/lists", listH.Create)

		// Invite routes (auth-only, no list membership required)
		protected.GET("/lists/invite/:token", listH.GetInvitePreview)
		protected.POST("/lists/invite/:token/join", listH.JoinByInvite)

		// Single list routes
		protected.GET("/lists/:id", listH.GetByID)
		protected.POST("/lists/:id/join", listH.JoinPublic)

		// Owner OR admin
		ownerAdmin := protected.Group("/lists/:id")
		ownerAdmin.Use(middleware.RequireListRole(queries, "owner", "admin"))
		{
			ownerAdmin.PATCH("", listH.Update)
			ownerAdmin.PATCH("/entries/ranks", listH.BulkUpdateRanks)
			ownerAdmin.DELETE("/members/:userId", listH.RemoveMember)
			ownerAdmin.GET("/invite", listH.GetInviteLink)
			ownerAdmin.POST("/invite/regenerate", listH.RegenerateInvite)
			ownerAdmin.GET("/entries/pending", entryH.GetPendingEntries)
			ownerAdmin.POST("/entries/:entryId/approve", entryH.ApproveEntry)
			ownerAdmin.POST("/entries/:entryId/reject", entryH.RejectEntry)
		}

		// Owner only
		ownerOnly := protected.Group("/lists/:id")
		ownerOnly.Use(middleware.RequireListRole(queries, "owner"))
		{
			ownerOnly.DELETE("", listH.Delete)
			ownerOnly.PATCH("/members/:userId", listH.UpdateMemberRole)
		}

		// Any member
		memberRoutes := protected.Group("/lists/:id")
		memberRoutes.Use(middleware.RequireListMember(queries))
		{
			memberRoutes.GET("/members", listH.GetMembers)
			memberRoutes.PUT("/entries/me", entryH.UpsertMyEntry)
			memberRoutes.DELETE("/entries/me", entryH.DeleteMyEntry)
		}

		// Delete any entry (owner/admin)
		entryAdmin := protected.Group("/lists/:id")
		entryAdmin.Use(middleware.RequireListRole(queries, "owner", "admin"))
		{
			entryAdmin.DELETE("/entries/:entryId", entryH.DeleteEntry)
		}
	}

	return &App{
		Engine: router,
		Pool:   pool,
		Cfg:    cfg,
		Logger: logger,
	}
}

// buildCORSConfig translates app config into a gin-contrib/cors.Config.
//
// Production: ALLOWED_ORIGINS must be an explicit list (config.validate
// already rejected wildcards), AllowCredentials safely on.
// Dev: no allowlist → echo origin back via AllowOriginFunc so credentials
// stay on without the gin-contrib wildcard validator tripping.
func buildCORSConfig(cfg *config.Config) cors.Config {
	c := cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Authorization", "Content-Type", middleware.RequestIDHeader,
		},
		ExposeHeaders:    []string{middleware.RequestIDHeader},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	if len(cfg.AllowedOrigins) == 0 {
		c.AllowOriginFunc = func(origin string) bool { return true }
		return c
	}
	c.AllowOrigins = cfg.AllowedOrigins
	return c
}
