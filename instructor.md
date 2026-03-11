# INSTRUCTOR.md — Backend (Go)

## Purpose

This file is the single source of truth for implementing the Go backend for the Ranked Lists app.
Read it fully before writing any code. Every architectural decision is intentional.

---

## Tech Stack

| Concern          | Choice                                              |
| ---------------- | --------------------------------------------------- |
| Language         | Go 1.22+                                            |
| HTTP Framework   | Fiber v2 (`github.com/gofiber/fiber/v2`)            |
| Database         | PostgreSQL 15+                                      |
| Query layer      | sqlc (no ORM — raw SQL only)                        |
| Auth             | JWT (HS256), Apple Sign-In (server-side validation) |
| Password hashing | bcrypt (cost 12)                                    |
| UUID generation  | `pgcrypto` `gen_random_uuid()` in Postgres          |
| Deployment       | Fly.io or Render — single binary + managed Postgres |

---

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Fiber app wiring, middleware registration, graceful shutdown
├── internal/
│   ├── config/
│   │   └── config.go            # Load + validate env vars into a Config struct at startup
│   ├── db/
│   │   ├── migrations/          # Numbered .sql files (001_init.sql, 002_add_x.sql …)
│   │   ├── queries/             # .sql files consumed by sqlc
│   │   └── sqlc/                # Generated code — DO NOT EDIT
│   ├── handler/
│   │   ├── auth.go
│   │   ├── users.go
│   │   ├── lists.go
│   │   ├── entries.go
│   │   └── response.go          # Shared response helpers (Success, Fail, Err)
│   ├── middleware/
│   │   ├── auth.go              # JWT parse + inject userID into ctx
│   │   └── role.go              # Role guard factory: RequireRole("owner", "admin")
│   ├── service/
│   │   ├── auth.go
│   │   ├── lists.go
│   │   └── entries.go
│   └── apple/
│       └── verify.go            # Apple identity token validation
├── sqlc.yaml
├── go.mod
└── go.sum
```

### Rules

- Handlers are thin: parse input → call service → write response. No business logic.
- Services own all business logic. They receive a `context.Context` and the sqlc `*db.Queries` (or a `Store` wrapper).
- SQL lives only in `internal/db/queries/*.sql`. Never build query strings in Go.
- Generated sqlc code in `internal/db/sqlc/` is never edited by hand.

---

## Environment Variables

```
DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=require
JWT_SECRET=<256-bit random hex>
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h       # 30 days
PORT=8080
APPLE_BUNDLE_ID=com.yourcompany.rankapp
```

Parse all vars at startup in `config.go`. Fail fast with a clear message if any required var is missing.

---

## Database Schema

Run as migration `001_init.sql`.

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  email          TEXT        UNIQUE NOT NULL,
  display_name   TEXT        NOT NULL,
  auth_provider  TEXT        NOT NULL CHECK (auth_provider IN ('email', 'apple')),
  apple_sub      TEXT        UNIQUE,
  password_hash  TEXT,
  avatar_url     TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lists (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title        TEXT        NOT NULL,
  description  TEXT,
  value_type   TEXT        NOT NULL CHECK (value_type IN ('number', 'duration', 'text')),
  rank_order   TEXT        NOT NULL CHECK (rank_order IN ('asc', 'desc')),
  is_public    BOOLEAN     NOT NULL DEFAULT TRUE,
  invite_token UUID        NOT NULL DEFAULT gen_random_uuid(),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE list_members (
  list_id    UUID        NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
  user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role       TEXT        NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
  joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (list_id, user_id)
);

CREATE TABLE entries (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  list_id           UUID        NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
  user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  value_number      FLOAT8,
  value_duration_ms BIGINT,
  value_text        TEXT,
  manual_rank       INT,
  note              TEXT,
  submitted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (list_id, user_id)
);

CREATE TABLE refresh_tokens (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token       TEXT        UNIQUE NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  revoked     BOOLEAN     NOT NULL DEFAULT FALSE,   -- explicit revocation flag
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

> **Note on `refresh_tokens.revoked`:** Add this boolean column from day one. On refresh, mark the old token `revoked = TRUE` and issue a new one (rotation). On logout, mark it revoked. This prevents token reuse without needing to delete rows (preserves auditability).

---

## API Reference

All routes are prefixed `/api/v1`.  
Auth-required routes expect `Authorization: Bearer <access_token>`.

### Standardized Response Envelope

Every response — success or error — uses this shape:

```json
// Success
{ "data": { ... } }

// Error
{ "error": { "code": "LIST_NOT_FOUND", "message": "human readable" } }
```

Implement this in `handler/response.go`. Never write raw `c.JSON(...)` with ad-hoc shapes in handlers.

### Auth Endpoints

| Method | Path             | Auth | Notes                                                              |
| ------ | ---------------- | ---- | ------------------------------------------------------------------ |
| POST   | `/auth/register` | No   | Email + password                                                   |
| POST   | `/auth/login`    | No   | Email + password                                                   |
| POST   | `/auth/apple`    | No   | Apple identity token → upsert user → return JWT pair               |
| POST   | `/auth/refresh`  | No   | Body: `{ "refresh_token": "..." }` → rotate token, return new pair |
| POST   | `/auth/logout`   | Yes  | Body: `{ "refresh_token": "..." }` → mark revoked                  |

### Users Endpoints

| Method | Path        | Auth | Notes                                  |
| ------ | ----------- | ---- | -------------------------------------- |
| GET    | `/users/me` | Yes  | Returns current user profile           |
| PATCH  | `/users/me` | Yes  | Only `display_name` is updatable in v1 |

### Lists Endpoints

| Method | Path                         | Auth              | Notes                                                                                                             |
| ------ | ---------------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------- |
| GET    | `/lists`                     | Yes               | All lists the user owns or has joined. Each item includes `user_role` and `own_entry` (rank + value if submitted) |
| POST   | `/lists`                     | Yes               | Create list; auto-creates `list_members` row with role `owner`                                                    |
| GET    | `/lists/:id`                 | Yes\*             | List metadata + ranked entries. \*Public = any authed user; private = members only                                |
| PATCH  | `/lists/:id`                 | Yes (owner/admin) | Update title, description, is_public                                                                              |
| DELETE | `/lists/:id`                 | Yes (owner)       | Hard delete — cascades via FK                                                                                     |
| POST   | `/lists/:id/join`            | Yes               | Join a public list (creates `member` row)                                                                         |
| GET    | `/lists/invite/:token`       | Yes               | Resolve token → list preview (title, top 3 entries, member count)                                                 |
| POST   | `/lists/invite/:token/join`  | Yes               | Join via invite token (public or private)                                                                         |
| GET    | `/lists/:id/members`         | Yes (member)      | All members with roles                                                                                            |
| PATCH  | `/lists/:id/members/:userId` | Yes (owner)       | Update role: promote to `admin` or demote to `member`                                                             |
| DELETE | `/lists/:id/members/:userId` | Yes (owner/admin) | Remove member + their entry                                                                                       |
| PATCH  | `/lists/:id/entries/ranks`   | Yes (owner/admin) | **Bulk set `manual_rank`** for `text` type lists. Body: `[{ "entry_id": "...", "rank": 1 }, ...]`                 |

### Entries Endpoints

| Method | Path                          | Auth              | Notes                                                                                     |
| ------ | ----------------------------- | ----------------- | ----------------------------------------------------------------------------------------- |
| PUT    | `/lists/:id/entries/me`       | Yes (member)      | Upsert own entry. "me" = always the calling user. Uses `INSERT ... ON CONFLICT DO UPDATE` |
| DELETE | `/lists/:id/entries/me`       | Yes (member)      | Delete own entry                                                                          |
| DELETE | `/lists/:id/entries/:entryId` | Yes (owner/admin) | Delete any member's entry                                                                 |

> **Why `PUT .../entries/me` not `POST .../entries`?** Each user has exactly one entry per list — it's a singleton resource owned by "me". PUT communicates idempotent upsert clearly. No ambiguity about create vs update.

---

## Ranking Logic (SQL)

Implemented via `ROW_NUMBER()` window function in the entries query. Never computed in Go.

```sql
-- Example for 'number' type with rank_order = 'desc'
SELECT
  e.*,
  u.display_name,
  ROW_NUMBER() OVER (ORDER BY e.value_number DESC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1
ORDER BY rank ASC;
```

Adapt `ORDER BY` clause based on `value_type` and `rank_order`:

- `number` → `ORDER BY value_number ASC|DESC NULLS LAST`
- `duration` → `ORDER BY value_duration_ms ASC|DESC NULLS LAST`
- `text` → `ORDER BY manual_rank ASC NULLS LAST`

---

## Key Implementation Rules

### Apple Sign-In

Apple only sends `email` in the identity token on the **first** sign-in. On all subsequent logins, `email` will be null or absent. The upsert logic must:

1. Always use `apple_sub` as the lookup key (never email for Apple users).
2. On first sign-in: create user with `email` + `apple_sub`.
3. On subsequent sign-ins: update nothing — just return a JWT pair for the existing user.
4. Fetch Apple's public keys from `https://appleid.apple.com/auth/keys` and cache them (TTL ~1 hour). Validate the token's `aud` against `APPLE_BUNDLE_ID`.

### JWT

- Access token payload: `{ "sub": "<user_uuid>", "exp": ..., "iat": ... }`
- Never put roles or permissions in the JWT. Always resolve from DB in middleware.
- The `RequireRole` middleware reads membership from `list_members` using the `list_id` from the route param + `userID` from JWT context. Do not pass role as a JWT claim.

### Refresh Token Rotation

On `POST /auth/refresh`:

1. Look up token in `refresh_tokens` where `token = $1 AND revoked = FALSE AND expires_at > NOW()`.
2. If not found → 401.
3. Mark old token `revoked = TRUE`.
4. Issue new access token + new refresh token row.
5. Return both.

This is a single DB transaction.

### Entry Upsert

```sql
-- In queries/entries.sql
INSERT INTO entries (list_id, user_id, value_number, value_duration_ms, value_text, note, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (list_id, user_id)
DO UPDATE SET
  value_number      = EXCLUDED.value_number,
  value_duration_ms = EXCLUDED.value_duration_ms,
  value_text        = EXCLUDED.value_text,
  note              = EXCLUDED.note,
  updated_at        = NOW()
RETURNING *;
```

### Value Type Validation

On `PUT /lists/:id/entries/me`, after loading the list:

- If `value_type = 'number'`: only `value_number` may be set; reject if `value_duration_ms` or `value_text` is present.
- If `value_type = 'duration'`: only `value_duration_ms` may be set.
- If `value_type = 'text'`: only `value_text` may be set.

Return `400` with code `INVALID_VALUE_TYPE` if violated.

### Invite Token

- On `GET /lists/invite/:token`: no membership check — any authenticated user can preview.
- On `POST /lists/invite/:token/join`: check if user is already a member (return 200 idempotently, not 409).
- The `invite_token` is a UUID stored in the `lists` table. Never expose it in the `GET /lists` listing response — only in `GET /lists/:id` for owners/admins.

### Member Removal

`DELETE /lists/:id/members/:userId` must:

1. Refuse if `userId` is the list owner.
2. Delete the `list_members` row.
3. Delete the user's entry in this list (cascade or explicit delete).
4. Run both in a transaction.

---

## Error Codes (standardized)

Define these as constants in `handler/response.go`:

| Code                 | HTTP Status               |
| -------------------- | ------------------------- |
| `VALIDATION_ERROR`   | 400                       |
| `INVALID_VALUE_TYPE` | 400                       |
| `UNAUTHORIZED`       | 401                       |
| `FORBIDDEN`          | 403                       |
| `NOT_FOUND`          | 404                       |
| `LIST_NOT_FOUND`     | 404                       |
| `ALREADY_MEMBER`     | 200 (idempotent)          |
| `ALREADY_HAS_ENTRY`  | 200 (upsert, never error) |
| `INTERNAL_ERROR`     | 500                       |

---

## Middleware Stack (order matters in Fiber)

```
app.Use(recover middleware)
app.Use(logger middleware)
app.Use(CORS — allow your app's domains + localhost in dev)

/api/v1/auth/* → no auth middleware
/api/v1/* → JWTMiddleware (parse Bearer, inject userID into ctx)
```

For role-guarded routes, apply `RequireListRole("owner")` or `RequireListRole("owner", "admin")` as additional route-level middleware. This function should:

1. Extract `list_id` from route params.
2. Query `list_members` for `(list_id, userID)`.
3. Check role is in allowed set. Return 403 if not.

---

## sqlc Configuration

```yaml
# sqlc.yaml
version: "2"
sql:
    - engine: "postgresql"
      queries: "internal/db/queries/"
      schema: "internal/db/migrations/"
      gen:
          go:
              package: "db"
              out: "internal/db/sqlc"
              emit_json_tags: true
              emit_interface: true # generates Querier interface — useful for testing
              null_style: "sql" # use sql.NullString etc.
```

---

## Out of Scope for v1

- Push notifications
- Avatar upload / image storage
- Android / web
- External data verification (Strava, Apple Health)
- Premium tiers
- Soft deletes (add `deleted_at TIMESTAMPTZ` later if needed)
- Rate limiting (add as middleware when needed)
- Full-text search on lists

The data model is intentionally left extensible: nullable columns, role system, `avatar_url` column already present.
