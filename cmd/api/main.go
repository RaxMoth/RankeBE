package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gin-rest-template/internal/config"
	"gin-rest-template/internal/handlers"
	"gin-rest-template/internal/middleware"
	"gin-rest-template/internal/repository"
	"gin-rest-template/internal/service"
	"gin-rest-template/pkg/logger"

	"github.com/gin-gonic/gin"
)

// @title           REST API Template
// @version         1.0
// @description     A production-ready REST API template with Gin framework
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger.Init(cfg.LogLevel)

	// Initialize repository based on database type
	var repo repository.Repository
	switch cfg.DatabaseType {
	case "mysql":
		repo, err = repository.NewMySQLRepository(cfg)
		if err != nil {
			logger.Fatal("Failed to initialize MySQL repository", "error", err)
		}
	case "firebase":
		repo, err = repository.NewFirebaseRepository(cfg)
		if err != nil {
			logger.Fatal("Failed to initialize Firebase repository", "error", err)
		}
	default:
		logger.Fatal("Invalid database type", "type", cfg.DatabaseType)
	}
	defer repo.Close()

	// Initialize service layer
	svc := service.NewService(repo, cfg)

	// Initialize handlers
	h := handlers.NewHandler(svc, cfg)

	// Setup router
	router := setupRouter(cfg, h)

	// Start server with graceful shutdown
	startServer(router, cfg)
}

func setupRouter(cfg *config.Config, h *handlers.Handler) *gin.Engine {
	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimiterMiddleware(cfg.RateLimitRequests, cfg.RateLimitDuration))

	// Health check
	router.GET("/health", h.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/refresh", h.RefreshToken)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			// Example resource (albums)
			albums := protected.Group("/albums")
			{
				albums.GET("", h.GetAlbums)
				albums.GET("/:id", h.GetAlbumByID)
				albums.POST("", h.CreateAlbum)
				albums.PUT("/:id", h.UpdateAlbum)
				albums.DELETE("/:id", h.DeleteAlbum)
			}

			// User routes
			users := protected.Group("/users")
			{
				users.GET("/me", h.GetCurrentUser)
				users.PUT("/me", h.UpdateCurrentUser)
			}
		}
	}

	// Swagger documentation
	if cfg.Environment != "production" {
		// router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		logger.Info("Swagger documentation available at http://localhost:" + cfg.Port + "/swagger/index.html")
	}

	return router
}

func startServer(router *gin.Engine, cfg *config.Config) {
	srv := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in a goroutine
	go func() {
		logger.Info(fmt.Sprintf("Starting server on port %s", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}
