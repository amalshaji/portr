FROM node:20-alpine3.18 as frontend-builder

WORKDIR /app

COPY internal/server/admin/web .

RUN npm i -g pnpm && pnpm i && pnpm build

FROM golang:1.21.5 AS builder

WORKDIR /app

RUN apt update && apt install -y zip

COPY go.mod go.sum /app/

RUN go mod download

COPY . /app/

RUN CGO_ENABLED=1 go build -ldflags="-s -w -linkmode external -extldflags \"-static\"" -o localportd ./cmd/localportd

# Build client
RUN GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o localport cmd/localport/*.go && zip localport.zip localport

FROM alpine:3.18.4 

WORKDIR /app

COPY --from=builder /app/localportd /app/
COPY --from=builder /app/localport.zip /app/internal/server/admin/static/
COPY --from=builder /app/internal/server/admin/templates/ /app/internal/server/admin/templates/
COPY --from=builder /app/internal/server/admin/static/ /app/internal/server/admin/static/
COPY --from=frontend-builder /app/dist/ ./internal/server/admin/web/dist/

VOLUME [ "/app/configs" ]
VOLUME [ "/app/data" ]

ENTRYPOINT ["./localportd", "start"]
CMD [ "--config", "/app/config.yaml" ]
