-- +goose Up
-- Create user table
CREATE TABLE
    IF NOT EXISTS "user" (
        "id" SERIAL PRIMARY KEY,
        "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "email" TEXT NOT NULL UNIQUE,
        "first_name" TEXT,
        "last_name" TEXT,
        "password" TEXT,
        "is_superuser" BOOLEAN NOT NULL DEFAULT FALSE
    );

-- Create session table
CREATE TABLE
    IF NOT EXISTS "session" (
        "id" SERIAL PRIMARY KEY,
        "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "token" TEXT NOT NULL UNIQUE,
        "expires_at" TIMESTAMP NOT NULL,
        "user_id" INTEGER NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE
    );

CREATE INDEX IF NOT EXISTS "idx_session_expires_at" ON "session" ("expires_at");

-- Create githubuser table
CREATE TABLE
    IF NOT EXISTS "githubuser" (
        "id" SERIAL PRIMARY KEY,
        "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "github_id" BIGINT NOT NULL UNIQUE,
        "github_access_token" TEXT NOT NULL,
        "github_avatar_url" TEXT NOT NULL,
        "user_id" INTEGER NOT NULL UNIQUE REFERENCES "user" ("id") ON DELETE CASCADE
    );

CREATE INDEX IF NOT EXISTS "idx_githubuser_github_id" ON "githubuser" ("github_id");

-- Create team table
CREATE TABLE
    IF NOT EXISTS "team" (
        "id" SERIAL PRIMARY KEY,
        "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "name" TEXT NOT NULL UNIQUE,
        "slug" TEXT NOT NULL UNIQUE
    );

CREATE INDEX IF NOT EXISTS "idx_team_slug" ON "team" ("slug");

-- Create team_users table
CREATE TABLE
    IF NOT EXISTS "team_users" (
        "id" SERIAL PRIMARY KEY,
        "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "secret_key" TEXT NOT NULL UNIQUE,
        "role" TEXT NOT NULL DEFAULT 'member',
        "team_id" INTEGER NOT NULL REFERENCES "team" ("id") ON DELETE CASCADE,
        "user_id" INTEGER NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE
    );

CREATE INDEX IF NOT EXISTS "idx_team_users_secret_key" ON "team_users" ("secret_key");

-- Create connection table
CREATE TABLE
    IF NOT EXISTS "connection" (
        "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "id" TEXT NOT NULL PRIMARY KEY,
        "type" TEXT NOT NULL,
        "subdomain" TEXT,
        "port" INTEGER,
        "status" TEXT NOT NULL DEFAULT 'reserved',
        "started_at" TIMESTAMP,
        "closed_at" TIMESTAMP,
        "created_by_id" INTEGER NOT NULL REFERENCES "team_users" ("id") ON DELETE CASCADE,
        "team_id" INTEGER NOT NULL REFERENCES "team" ("id") ON DELETE CASCADE
    );

CREATE INDEX IF NOT EXISTS "idx_connection_status" ON "connection" ("status");

-- +goose Down
DROP TABLE IF EXISTS "connection";

DROP TABLE IF EXISTS "team_users";

DROP TABLE IF EXISTS "team";

DROP TABLE IF EXISTS "githubuser";

DROP TABLE IF EXISTS "session";

DROP TABLE IF EXISTS "user";