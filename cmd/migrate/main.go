// Command migrate applies the embedded schema migrations and exits.
//
// The server also migrates on boot, so this is for the times you want to run
// migrations on their own: a CI step, a deploy hook, or a manual check. It
// reads the same DATABASE_URL the server uses.
//
//	go run ./cmd/migrate
package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ranke-be/internal/config"
	"ranke-be/internal/db/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	applied, err := migrations.Apply(ctx, pool)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if len(applied) == 0 {
		log.Println("migrations up to date")
		return
	}
	for _, v := range applied {
		log.Printf("applied %s", v)
	}
}
