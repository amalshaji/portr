-- +goose Up
ALTER TABLE "team" ADD COLUMN "client_template" TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE "team" DROP COLUMN "client_template";
