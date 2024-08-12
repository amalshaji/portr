from tortoise import BaseDBAsyncClient


async def upgrade(db: BaseDBAsyncClient) -> str:
    return """
        CREATE TABLE IF NOT EXISTS "aerich" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "version" VARCHAR(255) NOT NULL,
    "app" VARCHAR(100) NOT NULL,
    "content" JSON NOT NULL
);
CREATE TABLE IF NOT EXISTS "user" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "created_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "email" VARCHAR(255) NOT NULL UNIQUE,
    "first_name" VARCHAR(255),
    "last_name" VARCHAR(255),
    "password" VARCHAR(255),
    "is_superuser" INT NOT NULL  DEFAULT 0
);
CREATE TABLE IF NOT EXISTS "session" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "created_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "token" VARCHAR(255) NOT NULL UNIQUE,
    "expires_at" TIMESTAMP NOT NULL,
    "user_id" INT NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS "idx_session_expires_823c67" ON "session" ("expires_at");
CREATE TABLE IF NOT EXISTS "githubuser" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "github_id" BIGINT NOT NULL UNIQUE,
    "github_access_token" VARCHAR(255) NOT NULL,
    "github_avatar_url" VARCHAR(255) NOT NULL,
    "user_id" INT NOT NULL UNIQUE REFERENCES "user" ("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS "idx_githubuser_github__f7df59" ON "githubuser" ("github_id");
CREATE TABLE IF NOT EXISTS "team" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "created_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(255) NOT NULL UNIQUE,
    "slug" VARCHAR(255) NOT NULL UNIQUE
);
CREATE INDEX IF NOT EXISTS "idx_team_slug_b2d3a8" ON "team" ("slug");
CREATE TABLE IF NOT EXISTS "team_users" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "created_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "secret_key" VARCHAR(42) NOT NULL UNIQUE,
    "role" VARCHAR(255) NOT NULL  DEFAULT 'member',
    "team_id" INT NOT NULL REFERENCES "team" ("id") ON DELETE CASCADE,
    "user_id" INT NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS "idx_team_users_secret__22c341" ON "team_users" ("secret_key");
CREATE TABLE IF NOT EXISTS "instancesettings" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "created_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "smtp_enabled" INT NOT NULL  DEFAULT 0,
    "smtp_host" VARCHAR(255),
    "smtp_port" INT,
    "smtp_username" VARCHAR(255),
    "smtp_password" BLOB,
    "from_address" VARCHAR(255),
    "add_user_email_subject" VARCHAR(255),
    "add_user_email_body" TEXT,
    "updated_by_id" INT REFERENCES "user" ("id") ON DELETE SET NULL
);
CREATE TABLE IF NOT EXISTS "connection" (
    "created_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "id" VARCHAR(26) NOT NULL  PRIMARY KEY,
    "type" VARCHAR(255) NOT NULL,
    "subdomain" VARCHAR(255),
    "port" INT,
    "status" VARCHAR(255) NOT NULL  DEFAULT 'reserved',
    "started_at" TIMESTAMP,
    "closed_at" TIMESTAMP,
    "created_by_id" INT NOT NULL REFERENCES "team_users" ("id") ON DELETE CASCADE,
    "team_id" INT NOT NULL REFERENCES "team" ("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS "idx_connection_status_7d03b9" ON "connection" ("status");"""


async def downgrade(db: BaseDBAsyncClient) -> str:
    return """
        """
