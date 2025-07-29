FROM golang:1.24-alpine AS builder

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

RUN apk --no-cache add ca-certificates

RUN addgroup -S portr && adduser -S portr -G portr

WORKDIR /app

COPY --from=builder /app/portrd /app/
COPY --from=builder /app/migrations /app/migrations

RUN mkdir -p /app/data && chown portr:portr /app/data /app/portrd

RUN ls -l /app

USER portr

EXPOSE 2222 8000 8001

ENTRYPOINT ["./portrd"]
