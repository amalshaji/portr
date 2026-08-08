-- +goose Up
CREATE UNIQUE INDEX "idx_team_users_team_user_unique"
ON "team_users" ("team_id", "user_id");

-- +goose Down
DROP INDEX IF EXISTS "idx_team_users_team_user_unique";
