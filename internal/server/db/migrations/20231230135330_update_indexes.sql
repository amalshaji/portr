-- migrate:up
CREATE UNIQUE INDEX IF NOT EXISTS idx_team_members_secret_key ON team_members (secret_key);

-- migrate:down
DROP INDEX IF EXISTS idx_team_members_secret_key;