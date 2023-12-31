-- migrate:up
ALTER TABLE connections
ADD COLUMN team_id INTEGER NULL REFERENCES teams (id);

-- migrate:down
ALTER TABLE connections
DROP COLUMN team_id;