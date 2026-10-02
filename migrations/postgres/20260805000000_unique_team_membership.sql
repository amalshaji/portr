-- +goose Up
-- Duplicate memberships are removed below, keeping the oldest row per
-- (team_id, user_id). Records owned by a duplicate row are re-pointed at the
-- surviving membership first: connection.created_by_id and
-- subdomain_reservation.team_user_id both cascade on delete, so deleting a
-- duplicate without reassigning them would destroy user data.
UPDATE "connection"
SET "created_by_id" = (
    SELECT MIN(tu2."id")
    FROM "team_users" tu
    JOIN "team_users" tu2
      ON tu2."team_id" = tu."team_id" AND tu2."user_id" = tu."user_id"
    WHERE tu."id" = "connection"."created_by_id"
)
WHERE "created_by_id" IN (
    SELECT "id" FROM "team_users" tu
    WHERE "id" > (
        SELECT MIN("id") FROM "team_users" tu2
        WHERE tu2."team_id" = tu."team_id" AND tu2."user_id" = tu."user_id"
    )
);

UPDATE "subdomain_reservation"
SET "team_user_id" = (
    SELECT MIN(tu2."id")
    FROM "team_users" tu
    JOIN "team_users" tu2
      ON tu2."team_id" = tu."team_id" AND tu2."user_id" = tu."user_id"
    WHERE tu."id" = "subdomain_reservation"."team_user_id"
)
WHERE "team_user_id" IN (
    SELECT "id" FROM "team_users" tu
    WHERE "id" > (
        SELECT MIN("id") FROM "team_users" tu2
        WHERE tu2."team_id" = tu."team_id" AND tu2."user_id" = tu."user_id"
    )
);

DELETE FROM "team_users"
WHERE "id" NOT IN (
    SELECT MIN("id") FROM "team_users" GROUP BY "team_id", "user_id"
);

CREATE UNIQUE INDEX "idx_team_users_team_user_unique"
ON "team_users" ("team_id", "user_id");

-- +goose Down
DROP INDEX IF EXISTS "idx_team_users_team_user_unique";
