// Package migrations owns the schema. The *.sql files in this directory are
// embedded into the binary and applied by Apply, so a deployed server carries
// its own migrations — no psql, no separate tooling, no files to ship
// alongside the container.
//
// Migrations are plain, forward-only SQL. Each file is applied exactly once,
// in lexical filename order (hence the zero-padded NNN_ prefix), and recorded
// in the schema_migrations table so re-runs are no-ops.
package migrations

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var FS embed.FS

// advisoryLockKey is an arbitrary constant. Apply grabs a session-level
// advisory lock on it so that two instances booting at once (rolling deploy,
// scaled replicas) serialize their migration runs instead of racing to apply
// the same file.
const advisoryLockKey int64 = 0x52414e4b45 // "RANKE"

// Apply runs every embedded migration not yet recorded in schema_migrations,
// in lexical filename order, and returns the versions it newly applied.
//
// Each migration runs in its own transaction together with the bookkeeping
// INSERT, so a migration and its version record commit atomically: a crash
// mid-run never leaves a half-applied file marked as done. A migration that
// fails aborts the run; migrations committed before it stay applied.
func Apply(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return nil, fmt.Errorf("acquire advisory lock: %w", err)
	}
	// Release on a fresh context so an already-cancelled ctx can't strand the
	// lock for the rest of the connection's session.
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockKey)

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return nil, fmt.Errorf("ensure schema_migrations: %w", err)
	}

	done, err := appliedVersions(ctx, conn)
	if err != nil {
		return nil, err
	}

	files, err := sqlFiles()
	if err != nil {
		return nil, err
	}

	var newly []string
	for _, name := range files {
		if done[name] {
			continue
		}
		body, err := FS.ReadFile(name)
		if err != nil {
			return newly, fmt.Errorf("read %s: %w", name, err)
		}
		if err := applyOne(ctx, conn, name, string(body)); err != nil {
			return newly, fmt.Errorf("apply %s: %w", name, err)
		}
		newly = append(newly, name)
	}
	return newly, nil
}

// applyOne runs a single migration file and records its version in the same
// transaction. Multi-statement SQL is fine: pgx sends an argument-less Exec
// over the simple query protocol, which executes every statement in the string.
func applyOne(ctx context.Context, conn *pgxpool.Conn, name, body string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, body); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", name); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func appliedVersions(ctx context.Context, conn *pgxpool.Conn) (map[string]bool, error) {
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("load applied versions: %w", err)
	}
	defer rows.Close()

	done := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		done[v] = true
	}
	return done, rows.Err()
}

// sqlFiles lists the embedded migration filenames in lexical order.
func sqlFiles() ([]string, error) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
