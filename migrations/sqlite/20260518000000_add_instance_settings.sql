-- +goose Up
CREATE TABLE
    IF NOT EXISTS "instance_settings" (
        "id" INTEGER PRIMARY KEY AUTOINCREMENT,
        "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "smtp_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
        "smtp_host" TEXT NOT NULL DEFAULT '',
        "smtp_port" INTEGER NOT NULL DEFAULT 587,
        "smtp_username" TEXT NOT NULL DEFAULT '',
        "smtp_password" TEXT NOT NULL DEFAULT '',
        "from_address" TEXT NOT NULL DEFAULT '',
        "add_user_email_subject" TEXT NOT NULL DEFAULT 'Welcome to Portr!',
        "add_user_email_body" TEXT NOT NULL DEFAULT 'You have been added to a Portr team. Please set up your account using the temporary password provided.',
        "auto_signup_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
        "auto_signup_allowed_domains" TEXT NOT NULL DEFAULT '',
        "auto_signup_team_id" INTEGER REFERENCES "team" ("id") ON DELETE SET NULL
    );

INSERT INTO
    "instance_settings" (
        "id",
        "smtp_port",
        "add_user_email_subject",
        "add_user_email_body",
        "auto_signup_enabled",
        "auto_signup_allowed_domains"
    )
SELECT
    1,
    587,
    'Welcome to Portr!',
    'You have been added to a Portr team. Please set up your account using the temporary password provided.',
    FALSE,
    ''
WHERE
    NOT EXISTS (SELECT 1 FROM "instance_settings" WHERE "id" = 1);

-- +goose Down
DROP TABLE IF EXISTS "instance_settings";
