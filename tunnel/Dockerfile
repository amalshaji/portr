FROM golang:1.22.0 AS builder

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY . /app/

RUN CGO_ENABLED=1 go build -ldflags="-s -w -linkmode external -extldflags \"-static\"" -o portrd ./cmd/portrd

FROM alpine:3.19.1 as final

WORKDIR /app

COPY --from=builder /app/portrd /app/

ENTRYPOINT ["./portrd"]
