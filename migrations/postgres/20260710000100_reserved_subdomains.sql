-- +goose Up
DROP INDEX IF EXISTS "idx_connection_active_subdomain_unique";

CREATE UNIQUE INDEX "idx_connection_active_subdomain_unique"
ON "connection" (LOWER("subdomain"))
WHERE "subdomain" IS NOT NULL AND "status" IN ('reserved', 'active');

CREATE TABLE "subdomain_reservation" (
    "id" SERIAL PRIMARY KEY,
    "subdomain" TEXT NOT NULL,
    "team_user_id" INTEGER NOT NULL REFERENCES "team_users" ("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX "idx_subdomain_reservation_name_unique"
ON "subdomain_reservation" (LOWER("subdomain"));

CREATE INDEX "idx_subdomain_reservation_team_user"
ON "subdomain_reservation" ("team_user_id");

-- +goose Down
DROP TABLE IF EXISTS "subdomain_reservation";

DROP INDEX IF EXISTS "idx_connection_active_subdomain_unique";

CREATE UNIQUE INDEX "idx_connection_active_subdomain_unique"
ON "connection" ("subdomain")
WHERE "subdomain" IS NOT NULL AND "status" IN ('reserved', 'active');
