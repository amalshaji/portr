-- +goose Up
CREATE TABLE
    IF NOT EXISTS "instance_settings" (
        "id" SERIAL PRIMARY KEY,
        "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "auto_signup_enabled" BOOLEAN NOT NULL DEFAULT FALSE
    );

CREATE TABLE
    IF NOT EXISTS "auto_signup_domains" (
        "id" SERIAL PRIMARY KEY,
        "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "domain" TEXT NOT NULL UNIQUE,
        "team_id" INTEGER NOT NULL REFERENCES "team" ("id") ON DELETE CASCADE
    );

CREATE INDEX IF NOT EXISTS "idx_auto_signup_domains_team_id" ON "auto_signup_domains" ("team_id");

INSERT INTO
    "instance_settings" (
        "id",
        "auto_signup_enabled"
    )
SELECT
    1,
    FALSE
WHERE
    NOT EXISTS (SELECT 1 FROM "instance_settings" WHERE "id" = 1);

SELECT setval(
    pg_get_serial_sequence('instance_settings', 'id'),
    COALESCE((SELECT MAX("id") FROM "instance_settings"), 1)
);

-- +goose Down
DROP TABLE IF EXISTS "auto_signup_domains";
DROP TABLE IF EXISTS "instance_settings";
