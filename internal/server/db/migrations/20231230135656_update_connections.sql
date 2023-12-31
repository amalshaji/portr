-- migrate:up
ALTER TABLE connections
-- reserved, active, closed
ADD COLUMN status TEXT NOT NULL DEFAULT 'reserved';

ALTER TABLE connections
ADD COLUMN started_at TIMESTAMP NULL;

-- migrate:down
ALTER TABLE connections
DROP COLUMN status;

ALTER TABLE connections
DROP COLUMN started_at;