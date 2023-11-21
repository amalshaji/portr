FROM golang:1.21.4 AS builder

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY . /app/

RUN CGO_ENABLED=1 go build -ldflags="-s -w -linkmode external -extldflags \"-static\"" -o localportd ./cmd/localportd

FROM alpine:3.18.4 

WORKDIR /app

COPY --from=builder /app/localportd /app/

VOLUME [ "/app/configs" ]
VOLUME [ "/app/data" ]

ENTRYPOINT ["./localportd", "start"]
CMD [ "--config", "/app/config.yaml" ]