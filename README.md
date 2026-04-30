# Ranke Backend

Go + Gin + PostgreSQL API for the Ranke / Apex iOS app
([../RankeMobile](../RankeMobile)).

The wire contract — paths, JSON keys, error codes — is the source of truth
for the Flutter client. See `internal/handler/dto/` and the API map below.

## Stack

- Go 1.24, Gin
- PostgreSQL via `pgx/v5` + `pgxpool`
- Generated DB layer with [sqlc](https://sqlc.dev/) (`internal/db/sqlc`)
- JWT (HS256) access tokens + opaque refresh tokens stored in DB
- Sign in with Apple — verifies Apple JWKS server-side

## Layout

```
cmd/server/             entry point + router wiring
internal/
  apple/                Apple identity-token verifier (JWKS cache)
  config/               env loader (DATABASE_URL, JWT_SECRET, ...)
  db/
    migrations/         001_init.sql, 002_lists_and_entries_extensions.sql
    queries/            sqlc input — *.sql files
    sqlc/               generated query code (do NOT edit by hand)
  handler/              HTTP handlers (one file per resource)
    dto/                wire DTOs + sqlc-row mappers (camelCase)
  middleware/           AuthRequired, RequireListRole, RequireListMember
  service/              business logic (transactions, validation)
sqlc.yaml               sqlc config
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

```bash
# install deps
make install

# bring up Postgres yourself, then apply migrations
make migrate

# run the server (reads .env)
make run
```

Hits `:8080` by default. `GET /health` returns `{"status":"ok"}`.

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
| GET    | `/users/:id/profile`       | public profile (no email, with public boards) |

### Lists

| Method | Path                              | Role        |
|--------|-----------------------------------|-------------|
| GET    | `/lists`                          | self        |
| GET    | `/lists/public?q=&category=`      | self        |
| POST   | `/lists`                          | self        |
| GET    | `/lists/:id`                      | self (public list) / member (private) |
| PATCH  | `/lists/:id`                      | owner/admin |
| DELETE | `/lists/:id`                      | owner       |
| POST   | `/lists/:id/join`                 | self (public lists only) |

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
