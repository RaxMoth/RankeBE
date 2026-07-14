// Command server is the entry point for the Ranke backend.
//
// Bootstrapping only — router assembly and route definitions live in
// internal/server so tests can stand up the same app without re-doing
// dependency wiring here.
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ranke-be/internal/apple"
	"ranke-be/internal/config"
	"ranke-be/internal/db/migrations"
	"ranke-be/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Structured JSON logger — single instance shared by middleware and
	// any future service-layer logging. JSON is friendly to log shippers
	// (Datadog, Loki, CloudWatch) without a parsing step.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Connect to PostgreSQL with explicit pool sizing. The defaults
	// (max=4) are too small for any real workload, and there's no way to
	// override them via DSN params — pgxpool wants them on the parsed
	// Config struct.
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("parse database url: %v", err)
	}
	poolCfg.MaxConns = cfg.DBMaxConns
	poolCfg.MinConns = cfg.DBMinConns
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.MaxConnLifetime = 1 * time.Hour
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := pool.Ping(pingCtx); err != nil {
		pingCancel()
		log.Fatalf("database ping: %v", err)
	}
	pingCancel()

	// Apply embedded schema migrations before serving. Idempotent — already
	// applied versions are skipped — and serialized via a Postgres advisory
	// lock so rolling deploys / scaled replicas don't race.
	migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 30*time.Second)
	applied, err := migrations.Apply(migrateCtx, pool)
	migrateCancel()
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if len(applied) > 0 {
		logger.Info("migrations applied", slog.Any("versions", applied))
	} else {
		logger.Info("migrations up to date")
	}

	// Apple Sign-In verifier (optional). config.validate already enforces
	// the bundle ID is set in production.
	var appleVerifier *apple.Verifier
	if cfg.AppleBundleID != "" {
		appleVerifier = apple.NewVerifier(cfg.AppleBundleID)
	}

	app := server.New(cfg, pool, logger, appleVerifier)

	// Start server with explicit timeouts so a slow client can't tie up a
	// connection indefinitely (slowloris). ReadHeaderTimeout in particular
	// matters because the request body can legitimately take a while
	// (uploads), but the headers should arrive quickly.
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Engine,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
		MaxHeaderBytes:    1 << 14, // 16 KiB — comfortable for any sane Bearer token
	}

	go func() {
		logger.Info("server starting", slog.String("addr", srv.Addr), slog.String("env", cfg.Env))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown — give in-flight requests up to 30s to finish.
	// Production deploys typically front this with a load balancer that
	// has already drained connections before SIGTERM lands, but we still
	// want to flush long-running handlers cleanly.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}

	logger.Info("server stopped")
}
