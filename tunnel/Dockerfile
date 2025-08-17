FROM node:22-alpine AS web-builder

WORKDIR /app/web

COPY internal/server/admin/web/package.json internal/server/admin/web/pnpm-lock.yaml ./

RUN npm install -g pnpm && pnpm install

COPY internal/server/admin/web ./

RUN pnpm build

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

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
COPY --from=web-builder /app/static /app/internal/server/admin/static

RUN mkdir -p /app/data

EXPOSE 2222 8000 8001

ENTRYPOINT ["./portrd"]
