FROM oven/bun:1 as frontend-builder

WORKDIR /app

COPY internal/server/admin/web .

RUN bun i && bun run build

FROM golang:1.22.0 AS builder

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY . /app/

RUN CGO_ENABLED=1 go build -ldflags="-s -w -linkmode external -extldflags \"-static\"" -o portrd ./cmd/portrd

FROM alpine:3.19.1 as final

WORKDIR /app

COPY --from=builder /app/portrd /app/
COPY --from=builder /app/internal/server/admin/templates/ /app/internal/server/admin/templates/
COPY --from=builder /app/internal/server/admin/static/ /app/internal/server/admin/static/
COPY --from=frontend-builder /app/dist/ ./internal/server/admin/web/dist/

VOLUME [ "/app/configs" ]
VOLUME [ "/app/data" ]

ENTRYPOINT ["./portrd", "start"]
CMD [ "--config", "/app/config.yaml" ]
