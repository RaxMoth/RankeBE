package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"ranke-be/internal/apple"
	"ranke-be/internal/config"
	db "ranke-be/internal/db/sqlc"
	"ranke-be/internal/handler"
	"ranke-be/internal/middleware"
	"ranke-be/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Connect to PostgreSQL
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping: %v", err)
	}

	queries := db.New(pool)

	// Apple Sign-In verifier (optional)
	var appleVerifier *apple.Verifier
	if cfg.AppleBundleID != "" {
		appleVerifier = apple.NewVerifier(cfg.AppleBundleID)
	}

	// Services — entry service depends on the list service for value-type /
	// rank-order context when computing previous_rank.
	authSvc := service.NewAuthService(queries, pool, cfg, appleVerifier)
	listSvc := service.NewListService(queries, pool)
	entrySvc := service.NewEntryService(queries, pool, listSvc)

	// Handlers
	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(queries, listSvc)
	listH := handler.NewListHandler(listSvc)
	entryH := handler.NewEntryHandler(entrySvc, listSvc)

	// Router
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")

	// Auth (public)
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/apple", authH.AppleSignIn)
		auth.POST("/refresh", authH.Refresh)
	}

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.AuthRequired(cfg.JWTSecret))
	{
		// Auth - logout requires auth
		protected.POST("/auth/logout", authH.Logout)

		// Users
		protected.GET("/users/me", userH.GetMe)
		protected.PATCH("/users/me", userH.UpdateMe)
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

		// List routes requiring owner or admin
		ownerAdmin := protected.Group("/lists/:id")
		ownerAdmin.Use(middleware.RequireListRole(queries, "owner", "admin"))
		{
			ownerAdmin.PATCH("", listH.Update)
			ownerAdmin.PATCH("/entries/ranks", listH.BulkUpdateRanks)
			ownerAdmin.DELETE("/members/:userId", listH.RemoveMember)

			// Invite link management
			ownerAdmin.GET("/invite", listH.GetInviteLink)
			ownerAdmin.POST("/invite/regenerate", listH.RegenerateInvite)

			// Moderation: pending feed + approve/reject
			ownerAdmin.GET("/entries/pending", entryH.GetPendingEntries)
			ownerAdmin.POST("/entries/:entryId/approve", entryH.ApproveEntry)
			ownerAdmin.POST("/entries/:entryId/reject", entryH.RejectEntry)
		}

		// List routes requiring owner only
		ownerOnly := protected.Group("/lists/:id")
		ownerOnly.Use(middleware.RequireListRole(queries, "owner"))
		{
			ownerOnly.DELETE("", listH.Delete)
			ownerOnly.PATCH("/members/:userId", listH.UpdateMemberRole)
		}

		// Members list (any member)
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

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}

	log.Println("Server stopped")
}
