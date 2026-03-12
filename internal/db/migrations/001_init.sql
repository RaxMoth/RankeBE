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
  revoked     BOOLEAN     NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
