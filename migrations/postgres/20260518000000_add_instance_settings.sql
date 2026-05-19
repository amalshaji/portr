-- +goose Up
CREATE TABLE
    IF NOT EXISTS "instance_settings" (
        "id" SERIAL PRIMARY KEY,
        "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "auto_signup_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
        "auto_signup_allowed_domains" TEXT NOT NULL DEFAULT '',
        "auto_signup_team_id" INTEGER REFERENCES "team" ("id") ON DELETE SET NULL
    );

INSERT INTO
    "instance_settings" (
        "id",
        "auto_signup_enabled",
        "auto_signup_allowed_domains"
    )
SELECT
    1,
    FALSE,
    ''
WHERE
    NOT EXISTS (SELECT 1 FROM "instance_settings" WHERE "id" = 1);

SELECT setval(
    pg_get_serial_sequence('instance_settings', 'id'),
    COALESCE((SELECT MAX("id") FROM "instance_settings"), 1)
);

-- +goose Down
DROP TABLE IF EXISTS "instance_settings";
