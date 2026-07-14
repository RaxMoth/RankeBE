# Ranke Backend

Go + Gin + PostgreSQL API for the Ranke / Apex iOS app
([../RankeMobile](../RankeMobile)).

The wire contract — paths, JSON keys, error codes — is the source of truth
for the Flutter client. See `internal/handler/dto/` and the API map below.

## Stack

- Go 1.25, Gin
- PostgreSQL via `pgx/v5` + `pgxpool`
- Generated DB layer with [sqlc](https://sqlc.dev/) (`internal/db/sqlc`)
- JWT (HS256) access tokens + opaque refresh tokens stored in DB
- Sign in with Apple — verifies Apple JWKS server-side, accepts server-to-server revocation notifications
- Structured JSON logs via `log/slog`, per-request `X-Request-Id` correlation
- Per-IP rate limit on `/auth/*` (`golang.org/x/time/rate`)

## Layout

```
cmd/server/             bootstrap (config, pool, http.Server) — delegates router to internal/server
internal/
  apple/                Apple identity-token + notification verifier (JWKS cache)
  config/               env loader; production checks (strong secret, explicit CORS, bundle ID)
  db/
    migrations/         001_init.sql, 002_lists_and_entries_extensions.sql
    queries/            sqlc input — *.sql files
    sqlc/               generated query code (do NOT edit by hand)
  handler/              HTTP handlers (one file per resource)
    dto/                wire DTOs + sqlc-row mappers (camelCase)
  middleware/           Auth, RequestID, Logger, Recovery, BodyLimit, IPRateLimiter, role gates
  server/               router assembly + integration tests
  service/              business logic (transactions, validation)
sqlc.yaml               sqlc config
docker-compose.yml      one-command dev stack
```

The handler layer never serializes a sqlc row directly — every response
goes through a DTO mapper in `internal/handler/dto/`.

## Response envelope

Every successful response is wrapped:

```json
{ "data": <payload> }
```

Errors are `4xx`/`5xx` with:

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "..." } }
```

The mobile client (`lib/core/network/api_helpers.dart`) unwraps `data` and
maps `error` into its `ApiError` sealed class.

## Configuration

Copy `.env.example` to `.env` and fill in:

| Variable          | Required | Purpose                                       |
|-------------------|----------|-----------------------------------------------|
| `DATABASE_URL`    | yes      | Postgres connection string                    |
| `JWT_SECRET`      | yes      | HS256 signing secret for access tokens        |
| `JWT_ACCESS_TTL`  | no       | Access token TTL (default `15m`)              |
| `JWT_REFRESH_TTL` | no       | Refresh token TTL (default `720h`)            |
| `PORT`            | no       | HTTP port (default `8080`)                    |
| `APPLE_BUNDLE_ID` | no       | iOS bundle ID — required to enable `/auth/apple` |

If `APPLE_BUNDLE_ID` is empty the Apple Sign-In endpoint returns
`apple sign-in not configured`.

## Local development

One-command:

```bash
cp .env.example .env
# (edit .env — generate a real JWT_SECRET with `openssl rand -hex 32`)
make dev-up           # Postgres + API in docker compose
```

The first boot of the Postgres container auto-runs every migration via
`docker-entrypoint-initdb.d`. After editing a migration, `make dev-reset`
wipes the volume so the next `dev-up` re-applies.

Or step-by-step (your own Postgres, hot reload of the Go code):

```bash
make install
make migrate          # applies all migrations to $DATABASE_URL
make run              # foreground; reads .env
```

Health endpoints:

| Path        | Purpose                                    |
|-------------|--------------------------------------------|
| `GET /healthz` | Liveness — process is up (no deps)     |
| `GET /readyz`  | Readiness — pings Postgres, 503 on fail |
| `GET /health`  | Alias for `/healthz` (legacy)          |

Tests:

```bash
make test                 # unit tests, no DB required
make test-integration     # runs the full router against the dev Postgres
```

The integration test in `internal/server/server_test.go` truncates every
table before running, so don't point `TEST_DATABASE_URL` at a database
with data you care about.

## API map

All routes live under `/api/v1`. Auth routes are public; everything else
requires `Authorization: Bearer <accessToken>`.

### Auth

| Method | Path                | Body                                          |
|--------|---------------------|-----------------------------------------------|
| POST   | `/auth/register`    | `{ email, displayName, password }`            |
| POST   | `/auth/login`       | `{ email, password }`                         |
| POST   | `/auth/apple`       | `{ identityToken, fullName? }`                |
| POST   | `/auth/refresh`     | `{ refreshToken }`                            |
| POST   | `/auth/logout`      | `{ refreshToken? }` — auth required           |

`register`/`login`/`apple` return:

```json
{
  "user":         { "id": "...", "email": "...", "displayName": "...", "createdAt": "..." },
  "accessToken":  "...",
  "refreshToken": "...",
  "expiresIn":    900
}
```

### Users

| Method | Path                       | Notes                                  |
|--------|----------------------------|----------------------------------------|
| GET    | `/users/me`                | self profile (includes email)          |
| PATCH  | `/users/me`                | `{ displayName }`                      |
| DELETE | `/users/me`                | permanently delete account (App Store 5.1.1(v)) |
| GET    | `/users/:id/profile`       | public profile (no email, with public boards) |

**Account deletion cascade** — deleting a user wipes their refresh
tokens, entries, memberships, *and any list they own* (FK
`lists.owner_id` is `ON DELETE CASCADE`). Other members of an owned
list lose access. Future work: transfer ownership to the
next-most-senior admin before deletion.

### Apple notifications

| Method | Path                                | Notes                          |
|--------|-------------------------------------|--------------------------------|
| POST   | `/auth/apple/notifications`         | Apple server-to-server webhook |

Public endpoint; the signed JWT in the body is verified against Apple's
JWKS. On `consent-revoked` / `account-delete` we delete the local user.
Register the URL in Apple Developer → Services ID for it to fire.

### Lists

| Method | Path                              | Role        |
|--------|-----------------------------------|-------------|
| GET    | `/lists`                          | self        |
| GET    | `/lists/public?q=&category=&limit=&cursor=` | self  |
| POST   | `/lists`                          | self        |
| GET    | `/lists/:id`                      | self (public list) / member (private) |
| PATCH  | `/lists/:id`                      | owner/admin |
| DELETE | `/lists/:id`                      | owner       |
| POST   | `/lists/:id/join`                 | self (public lists only) |

`/lists/public` uses keyset pagination. `limit` defaults to 30 (max 100).
When more results remain, the response carries an opaque `X-Next-Cursor`
header; pass it back as `?cursor=` to fetch the next page. The body stays a
bare array of list summaries — the cursor lives only in the header. No header
means the last page was reached.

### Members

| Method | Path                                | Role        |
|--------|-------------------------------------|-------------|
| GET    | `/lists/:id/members`                | member      |
| PATCH  | `/lists/:id/members/:userId`        | owner       |
| DELETE | `/lists/:id/members/:userId`        | owner/admin |

### Invites

| Method | Path                                       | Role        |
|--------|--------------------------------------------|-------------|
| GET    | `/lists/:id/invite`                        | owner/admin |
| POST   | `/lists/:id/invite/regenerate`             | owner/admin |
| GET    | `/lists/invite/:token`                     | self (preview) |
| POST   | `/lists/invite/:token/join`                | self        |

### Entries

| Method | Path                                            | Role        |
|--------|-------------------------------------------------|-------------|
| PUT    | `/lists/:id/entries/me`                         | member      |
| DELETE | `/lists/:id/entries/me`                         | member      |
| PATCH  | `/lists/:id/entries/ranks`                      | owner/admin (bulk manual_rank for text lists) |
| GET    | `/lists/:id/entries/pending`                    | owner/admin |
| POST   | `/lists/:id/entries/:entryId/approve`           | owner/admin |
| POST   | `/lists/:id/entries/:entryId/reject`            | owner/admin |
| DELETE | `/lists/:id/entries/:entryId`                   | owner/admin |

`PUT /entries/me` body — exactly one of these three (matching the list's
`valueType`) plus an optional `note`:

```json
{ "valueNumber":     1234.5,  "note": "..." }
{ "valueDurationMs": 90000,   "note": "..." }
{ "valueText":       "lorem", "note": "..." }
```

## Schema

Two migrations:

- `001_init.sql` — `users`, `lists`, `list_members`, `entries`, `refresh_tokens`
- `002_lists_and_entries_extensions.sql` — adds `lists.category/locked` +
  chat links, adds `entries.status` (`pending`/`approved`/`rejected`) and
  `entries.previous_rank` for delta rendering. Existing rows default to
  `approved` so adding the column is non-breaking.

## sqlc

After editing anything in `internal/db/queries/`:

```bash
make sqlc
```

`sqlc.yaml` is configured to write generated code into `internal/db/sqlc/`.
Do not hand-edit those files.

## Docker

```bash
docker build -t ranke-be .
docker run -p 8080:8080 --env-file .env ranke-be
```
