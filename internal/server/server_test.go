package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ranke-be/internal/config"
	"ranke-be/internal/db/migrations"
	"ranke-be/internal/server"
)

// TEST_DATABASE_URL points at a Postgres instance the test can wipe.
// When unset the test is skipped — keeps `go test ./...` fast in CI
// when no DB is available. Locally:
//
//	TEST_DATABASE_URL=postgres://ranke:ranke@localhost:5432/ranke?sslmode=disable \
//	  go test ./internal/server/...
const testDatabaseURLEnv = "TEST_DATABASE_URL"

// testApp boots a real router against the configured test DB. Each test
// gets a freshly-truncated schema so cases don't leak into each other.
//
// We don't use testcontainers here — the dev compose stack already runs
// Postgres locally, and pulling in the testcontainers/docker deps for one
// integration test isn't worth the build-time cost. CI can point
// TEST_DATABASE_URL at any throwaway Postgres.
func testApp(t *testing.T) (*server.App, func()) {
	t.Helper()
	dsn := os.Getenv(testDatabaseURLEnv)
	if dsn == "" {
		t.Skipf("set %s to run integration tests", testDatabaseURLEnv)
	}

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	poolCfg.MaxConns = 5
	poolCfg.MinConns = 1

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		t.Fatalf("ping: %v", err)
	}

	// Bring the schema up to date so a brand-new throwaway Postgres works
	// with no external provisioning — this also exercises the embedded
	// migration runner on every integration run.
	if _, err := migrations.Apply(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("migrate: %v", err)
	}

	// Wipe every table in dependency order so tests start from a known
	// blank state.
	if _, err := pool.Exec(context.Background(), `
		TRUNCATE TABLE refresh_tokens, entries, list_members, lists, users
		RESTART IDENTITY CASCADE
	`); err != nil {
		pool.Close()
		t.Fatalf("truncate: %v", err)
	}

	cfg := &config.Config{
		DatabaseURL:   dsn,
		JWTSecret:     "test-secret-must-be-at-least-32-chars-long-aaaaa",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 720 * time.Hour,
		Port:          "0",
		Env:           "development",
		MaxBodyBytes:  1 << 20,
		DBMaxConns:    5,
		DBMinConns:    1,
	}

	// Discard logs in tests — we don't want JSON noise across `go test`
	// output. Switch to NewTextHandler+os.Stderr if you need to debug a
	// failure.
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	app := server.New(cfg, pool, logger, nil)

	cleanup := func() {
		pool.Close()
	}
	return app, cleanup
}

// envelope is the success-response shape every handler returns. Tests
// decode into this and then narrow to whichever payload shape they need.
type envelope struct {
	Data json.RawMessage `json:"data"`
}

// do is the workhorse helper: marshals the body, sets the bearer token
// if any, fires the request through the test recorder, and returns the
// status + the envelope.Data slice for further decoding.
func do(t *testing.T, app *server.App, method, path, token string, body any) (int, []byte) {
	t.Helper()
	var buf io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		buf = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	app.Engine.ServeHTTP(rr, req)

	if rr.Code >= 400 {
		// Surface error bodies in the test log — debugging silent 500s
		// without this is painful.
		t.Logf("%s %s -> %d: %s", method, path, rr.Code, rr.Body.String())
	}

	var env envelope
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err == nil && len(env.Data) > 0 {
		return rr.Code, env.Data
	}
	return rr.Code, rr.Body.Bytes()
}

// TestEndToEnd_AuthAndLists exercises the happy path the iOS app
// depends on every launch: register → /users/me → create list →
// list lists → submit entry → fetch list → delete account.
//
// This is a single big test on purpose — splitting these would force
// us to maintain helper fixtures for "a user with a list with an
// entry", which is more code than the sequence we'd be testing. We
// can split when these scenarios start needing independent flakiness
// isolation.
func TestEndToEnd_AuthAndLists(t *testing.T) {
	app, cleanup := testApp(t)
	defer cleanup()

	// ── Register ─────────────────────────────────────────────────
	status, data := do(t, app, "POST", "/api/v1/auth/register", "", map[string]string{
		"email":       "alice@test.dev",
		"displayName": "Alice",
		"password":    "hunter2hunter2",
	})
	if status != http.StatusCreated {
		t.Fatalf("register: status=%d body=%s", status, data)
	}
	var reg struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatalf("decode register: %v: %s", err, data)
	}
	if reg.AccessToken == "" || reg.RefreshToken == "" || reg.User.ID == "" {
		t.Fatalf("register: missing fields: %+v", reg)
	}
	if reg.User.Email != "alice@test.dev" {
		t.Errorf("register: email mismatch: got %q", reg.User.Email)
	}

	// ── /users/me ────────────────────────────────────────────────
	status, data = do(t, app, "GET", "/api/v1/users/me", reg.AccessToken, nil)
	if status != http.StatusOK {
		t.Fatalf("users/me: status=%d body=%s", status, data)
	}
	if !strings.Contains(string(data), reg.User.ID) {
		t.Errorf("users/me: missing user ID in response: %s", data)
	}

	// ── /users/me without token must 401 ─────────────────────────
	status, _ = do(t, app, "GET", "/api/v1/users/me", "", nil)
	if status != http.StatusUnauthorized {
		t.Errorf("users/me without token: expected 401, got %d", status)
	}

	// ── Create list ──────────────────────────────────────────────
	status, data = do(t, app, "POST", "/api/v1/lists", reg.AccessToken, map[string]any{
		"title":     "World records 100m",
		"valueType": "duration",
		"rankOrder": "asc",
		"isPublic":  true,
	})
	if status != http.StatusCreated {
		t.Fatalf("create list: status=%d body=%s", status, data)
	}
	var list struct {
		ID              string `json:"id"`
		CurrentUserRole string `json:"currentUserRole"`
		MemberCount     int    `json:"memberCount"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatalf("decode list: %v: %s", err, data)
	}
	if list.ID == "" || list.CurrentUserRole != "owner" || list.MemberCount != 1 {
		t.Fatalf("create list: unexpected payload: %+v", list)
	}

	// ── GET /lists must contain the new list with ownRank = nil ─
	status, data = do(t, app, "GET", "/api/v1/lists", reg.AccessToken, nil)
	if status != http.StatusOK {
		t.Fatalf("list lists: status=%d body=%s", status, data)
	}
	var summaries []map[string]any
	if err := json.Unmarshal(data, &summaries); err != nil {
		t.Fatalf("decode lists: %v: %s", err, data)
	}
	if len(summaries) != 1 || summaries[0]["id"] != list.ID {
		t.Fatalf("list lists: unexpected: %+v", summaries)
	}
	if _, ok := summaries[0]["ownRank"]; ok {
		// omitempty means the key shouldn't appear at all when nil
		t.Errorf("ownRank should be omitted when user has no entry, got: %v", summaries[0]["ownRank"])
	}

	// ── Submit entry → ownRank becomes 1 ─────────────────────────
	status, data = do(t, app, "PUT",
		fmt.Sprintf("/api/v1/lists/%s/entries/me", list.ID),
		reg.AccessToken,
		map[string]any{"valueDurationMs": 9580, "note": "Bolt"},
	)
	if status != http.StatusOK {
		t.Fatalf("submit entry: status=%d body=%s", status, data)
	}

	status, data = do(t, app, "GET", "/api/v1/lists", reg.AccessToken, nil)
	if status != http.StatusOK {
		t.Fatalf("relist: status=%d body=%s", status, data)
	}
	if err := json.Unmarshal(data, &summaries); err != nil {
		t.Fatalf("decode lists 2: %v: %s", err, data)
	}
	if rank, ok := summaries[0]["ownRank"].(float64); !ok || int(rank) != 1 {
		t.Errorf("expected ownRank=1, got %v (raw: %v)", rank, summaries[0]["ownRank"])
	}

	// ── Account deletion cascades the list (5.1.1(v) behavior) ──
	status, _ = do(t, app, "DELETE", "/api/v1/users/me", reg.AccessToken, nil)
	if status != http.StatusOK {
		t.Fatalf("delete account: status=%d", status)
	}

	// New caller can't see the list anymore — register a second user
	// and check /lists/public is empty (the list was wiped via FK
	// cascade from users.id → lists.owner_id).
	status, data = do(t, app, "POST", "/api/v1/auth/register", "", map[string]string{
		"email":       "bob@test.dev",
		"displayName": "Bob",
		"password":    "hunter2hunter2",
	})
	if status != http.StatusCreated {
		t.Fatalf("bob register: status=%d body=%s", status, data)
	}
	var bob struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(data, &bob); err != nil {
		t.Fatalf("decode bob: %v", err)
	}
	status, data = do(t, app, "GET", "/api/v1/lists/public", bob.AccessToken, nil)
	if status != http.StatusOK {
		t.Fatalf("public lists: status=%d body=%s", status, data)
	}
	var public []map[string]any
	if err := json.Unmarshal(data, &public); err != nil {
		t.Fatalf("decode public: %v", err)
	}
	if len(public) != 0 {
		t.Errorf("expected empty public lists after owner deletion, got %d", len(public))
	}
}

// TestHealthz hits the liveness endpoint without a DB — proves the
// route is reachable even when the rest of the app is hosed.
func TestHealthz(t *testing.T) {
	app, cleanup := testApp(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	app.Engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("healthz: status=%d body=%s", rr.Code, rr.Body.String())
	}
}
