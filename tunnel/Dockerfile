FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY . /app/

RUN CGO_ENABLED=1 go build -ldflags="-s -w -linkmode external -extldflags \"-static\"" -o portrd ./cmd/portrd

FROM alpine:3.20 AS final

LABEL maintainer="Amal Shaji" \
    org.opencontainers.image.title="Portr Tunnel" \
    org.opencontainers.image.description="Tunnel server for Portr" \
    org.opencontainers.image.source="https://github.com/amalshaji/portr"

WORKDIR /app

COPY --from=builder /app/portrd /app/

ENTRYPOINT ["./portrd"]
