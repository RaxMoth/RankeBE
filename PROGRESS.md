# Progress Tracker

Last loop run: 2026-07-14T00:00:00Z
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
- [x] **Pagination on `/lists/public`** — keyset pagination on `(updated_at DESC, id DESC)`, `limit` param (default 30, cap 100), opaque `X-Next-Cursor` header. Body stays a bare array so the current mobile client keeps parsing. Unit tests for cursor round-trip + limit clamp. — `<pending>`

## In Progress

(none)

## Backlog (from spec / mobile contract, not started)

- [ ] **Migration tooling** — replace raw `psql` Makefile target with goose or golang-migrate; embed migrations via `embed.FS`. Current state: only the SQL files, applied manually or via compose's `docker-entrypoint-initdb.d` (which only fires on first boot).
- [ ] **Mobile: consume `X-Next-Cursor`** — backend now paginates `/lists/public`; the Flutter `searchPublicLists` still ignores the cursor header and reads only the first 30. Tracked here for cross-repo visibility (belongs to RankeMobile, not this repo's loop).
- [ ] **Deployment platform config** — pick Fly.io / Render / Cloud Run; write the platform manifest + secrets-loading recipe. Blocks shipping to TestFlight (needs HTTPS).
- [ ] **Privacy policy + ToS URLs** — App Store reject blocker. Mobile already has `--dart-define` plumbing for `PRIVACY_POLICY_URL` / `TERMS_OF_SERVICE_URL`.
- [ ] **iOS bundle ID** — currently `com.example.flutterbase`. Set to a real reverse-DNS before enabling Sign in with Apple capability in Apple Developer.

## Proposed (NOT approved — do not implement)

(empty — populate only when there's no actionable Backlog)

## Tech Debt / Improvements

- [ ] **`internal/middleware/ratelimit.go` lazy GC** — current implementation walks the whole map on every request. Fine at low QPS, but consider a min-heap or a periodic janitor goroutine if /auth/* sees burst traffic.
- [ ] **Refresh tokens stored as raw hex** — should be SHA-256 hashed at rest so a DB leak doesn't immediately yield session takeover. Backward-incompatible — needs a migration.
- [ ] **No tests for middleware** — RequestID, Logger, RateLimiter, Recovery all uncovered.
- [ ] **No tests for the Apple verifier** — would need a fake JWKS server (`httptest.NewServer`) for end-to-end coverage.
- [ ] **`gin.H` everywhere** — handlers reach for `gin.H{...}` for ad-hoc responses; would be cleaner via the typed `dto` package.
- [ ] **CHANGELOG.md missing** — add it once we cut a 0.1.0 tag.

## Notes for the next loop iteration

- The hardening pass is committed at `a3b3081` — future iterations start from a clean tree.
- Next priority: top of Backlog (Migration tooling OR sentinel errors in entries.go — both small slices, pick by current pain).
- `.claude/settings.local.json` is per-developer and should stay gitignored; `.claude/commands/loop.md` is the shared `/loop` definition and is committed.
