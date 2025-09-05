FROM oven/bun:1 AS web-builder

WORKDIR /app/web

COPY internal/server/admin/web-v2/package.json internal/server/admin/web-v2/bun.lock ./

RUN bun install --frozen-lockfile

COPY internal/server/admin/web-v2 ./

RUN bun run build

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

COPY --from=web-builder /app/static /app/internal/server/admin/static

ARG VERSION=dev

RUN CGO_ENABLED=1 go build \
    -ldflags="-s -w -linkmode external -extldflags \"-static\" -X main.version=${VERSION}" \
    -o portrd ./cmd/portrd

FROM alpine:3.20 AS final

LABEL maintainer="Amal Shaji" \
    org.opencontainers.image.title="Portr Server" \
    org.opencontainers.image.description="Server for Portr" \
    org.opencontainers.image.source="https://github.com/amalshaji/portr"

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /app/portrd /app/
COPY --from=builder /app/migrations /app/migrations

RUN mkdir -p /app/data

EXPOSE 2222 8000 8001

ENTRYPOINT ["./portrd"]
