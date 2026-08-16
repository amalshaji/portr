-- +goose Up
-- Remove duplicate memberships before enforcing uniqueness, keeping the oldest
-- row per (team_id, user_id) so existing deployments can run this migration.
DELETE FROM "team_users"
WHERE "id" NOT IN (
  SELECT MIN("id") FROM "team_users" GROUP BY "team_id", "user_id"
);

CREATE UNIQUE INDEX "idx_team_users_team_user_unique"
ON "team_users" ("team_id", "user_id");

-- +goose Down
DROP INDEX IF EXISTS "idx_team_users_team_user_unique";
