-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS "idx_connection_active_subdomain_unique"
ON "connection" ("subdomain")
WHERE "subdomain" IS NOT NULL AND "status" IN ('reserved', 'active');

-- +goose Down
DROP INDEX IF EXISTS "idx_connection_active_subdomain_unique";
