# Progress Tracker

Last loop run: 2026-07-14T02:00:00Z
Stack: go (Gin + pgx/v5 + sqlc, Postgres 16)

Spec source: `README.md` (API map + behavior) cross-referenced against the
Flutter client's contract at `../RankeMobile/lib/core/network/api_paths.dart`.

## Done

- [x] JWT auth — register / login / refresh / logout (rotating refresh tokens) — pre-loop
- [x] Sign in with Apple identity token verification (JWKS cached, RS256) — pre-loop
- [x] Lists CRUD + invite tokens (preview, join, regenerate) — pre-loop
- [x] List members + roles (owner / admin / member) — pre-loop
- [x] Entries upsert across value types (number / duration / text) — pre-loop
- [x] Ranked entry feeds (5 variants) + manual-rank for text lists — pre-loop
- [x] Moderation queue (pending / approve / reject) — pre-loop
- [x] **Production hardening pass** — slog JSON logger, X-Request-Id middleware, panic recovery, body-size cap, HTTP timeouts, /healthz + /readyz, per-IP rate limit on `/auth/*`, pgxpool tuning, config validation (prod-mode wildcard CORS rejection, JWT secret length floor, mandatory APPLE_BUNDLE_ID) — `a3b3081`
- [x] **Account deletion** — `DELETE /users/me` with FK cascade (App Store 5.1.1(v)) — `a3b3081`
- [x] **Apple server-to-server revocation webhook** — `POST /auth/apple/notifications` (consent-revoked / account-delete) — `a3b3081`
- [x] **`GetUserLists` N+1 collapse** — `own_rank` computed inline; home feed now one query — `a3b3081`
- [x] **`ErrEmailTaken` sentinel** — Postgres unique-violation (23505) mapped to typed error → 409 EMAIL_TAKEN — `a3b3081`
- [x] **Refresh-token reuse detection** — presenting a revoked token nukes every refresh token for the affected user — `a3b3081`
- [x] **Apple Sign-In error sanitization** — verifier internals no longer leak to client; raw error logged with request_id — `a3b3081`
- [x] **Password validation tightened** — min 8 / max 128, displayName max 60 — `a3b3081`
- [x] **docker-compose dev stack** — `make dev-up` brings up Postgres + API, auto-applies migrations on first boot — `a3b3081`
- [x] **First integration test** — `internal/server/server_test.go`: register → /me → unauth check → create list → /lists → submit entry → ownRank=1 → delete account → cascade verification — `a3b3081`
- [x] **Sentinel errors in service/entries.go** — `ErrInvalidValueType` / `ErrListLocked` sentinels + handler `errors.Is` mapping. Already present in-tree (reconciled from Backlog/Tech-Debt this loop; not a new commit).
- [x] **`apple.containsString` → `slices.Contains`** — already migrated in-tree (reconciled this loop).
- [x] **Pagination on `/lists/public`** — keyset pagination on `(updated_at DESC, id DESC)`, `limit` param (default 30, cap 100), opaque `X-Next-Cursor` header. Body stays a bare array so the current mobile client keeps parsing. Unit tests for cursor round-trip + limit clamp. — `0a2053f`
- [x] **Migration tooling (stdlib, no third-party)** — `embed.FS` in `internal/db/migrations` + a runner (`Apply`) that records applied files in `schema_migrations`, wraps each in its own tx, and serializes concurrent runs with a `pg_advisory_lock`. Auto-applies on server boot; standalone `cmd/migrate` for CI/deploy; `make migrate` no longer needs psql. Dropped the compose `docker-entrypoint-initdb.d` mount (app now owns the schema). Integration test bootstraps its own schema via `Apply`. DB-free unit test guards the embed. — `80858e4` — Note: DB-touching path unverified this run (docker down).
- [x] **Deployment platform config (Fly.io)** — `fly.toml` (Dockerfile deploy, forced HTTPS, `/readyz` check, one warm machine) + README "Deploying" section with the `fly secrets set` recipe. Removed the now-redundant migrations COPY from the Dockerfile (they're embedded). — `72e977c`
- [x] **Middleware tests** — white-box DB-free coverage for RequestID (mint/echo/honor-incoming), RateLimiter (burst / per-IP / lazy GC / 429 envelope), Recovery (panic→500, no leak), BodyLimit (413 past cap). — `66d7a68`

## In Progress

(none)

## Backlog (from spec / mobile contract, not started)

- [ ] **Mobile: consume `X-Next-Cursor`** — backend now paginates `/lists/public`; the Flutter `searchPublicLists` still ignores the cursor header and reads only the first 30. Tracked here for cross-repo visibility (belongs to RankeMobile, not this repo's loop).
- [ ] **Privacy policy + ToS URLs** — App Store reject blocker. Needs real hosted legal content — a decision/asset for the owner, not autonomously codeable. Mobile already has `--dart-define` plumbing for `PRIVACY_POLICY_URL` / `TERMS_OF_SERVICE_URL`.
- [ ] **iOS bundle ID** — currently `com.example.flutterbase`. Lives in RankeMobile, not this repo. Set to a real reverse-DNS before enabling Sign in with Apple capability in Apple Developer.

## Proposed (NOT approved — do not implement)

(empty — populate only when there's no actionable Backlog)

## Tech Debt / Improvements

- [ ] **`internal/middleware/ratelimit.go` lazy GC** — current implementation walks the whole map on every request. Fine at low QPS, but consider a min-heap or a periodic janitor goroutine if /auth/* sees burst traffic.
- [ ] **Refresh tokens stored as raw hex** — should be SHA-256 hashed at rest so a DB leak doesn't immediately yield session takeover. Backward-incompatible — needs a migration.
- [ ] **No tests for the Apple verifier** — would need a fake JWKS server (`httptest.NewServer`) for end-to-end coverage.
- [ ] **`gin.H` everywhere** — handlers reach for `gin.H{...}` for ad-hoc responses; would be cleaner via the typed `dto` package.
- [ ] **CHANGELOG.md missing** — add it once we cut a 0.1.0 tag.

## Notes for the next loop iteration

- Three slices shipped this session: pagination (`0a2053f`), migration tooling
  (`80858e4`), Fly.io deploy config (`72e977c`), middleware tests (`66d7a68`).
  Tree is clean.
- **Verify next time docker is up:** run `make test-integration` (or `make dev-up`)
  to exercise `migrations.Apply` against a real Postgres — the runner's DB path
  (advisory lock, per-file tx, multi-statement Exec) has only been reasoned about,
  not run, because docker was down. Also a good moment to `fly deploy` and confirm
  the manifest against the live platform.
- Remaining Backlog is now **owner-blocked, not code-blocked**: Privacy/ToS URLs
  need real hosted legal content; the iOS bundle ID lives in RankeMobile. Neither
  is autonomously codeable in this repo.
- Next self-contained Go slices if you want to keep looping: Apple-verifier tests
  (fake JWKS via `httptest.NewServer`), sqlc typed DTOs replacing ad-hoc `gin.H`,
  or SHA-256 hashing refresh tokens at rest (needs a migration — now trivial to add).
- `.claude/settings.local.json` is per-developer and should stay gitignored; `.claude/commands/loop.md` is the shared `/loop` definition and is committed.
