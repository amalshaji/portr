-- +goose Up
-- Remove duplicate memberships before enforcing uniqueness, keeping the oldest
-- row per (team_id, user_id) so existing deployments can run this migration.
DELETE FROM "team_users" a
USING "team_users" b
WHERE a."team_id" = b."team_id"
  AND a."user_id" = b."user_id"
  AND a."id" > b."id";

CREATE UNIQUE INDEX "idx_team_users_team_user_unique"
ON "team_users" ("team_id", "user_id");

-- +goose Down
DROP INDEX IF EXISTS "idx_team_users_team_user_unique";
