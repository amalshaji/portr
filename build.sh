#!/bin/bash
pushd ./tunnel/internal/client/dashboard/ui
corepack enable >/dev/null 2>&1 || true
pnpm install
pnpm build
popd

pushd ./tunnel
go test ./...
popd

pushd ./tunnel
go build -o ./bin/portr ./cmd/portr
go build -o ./bin/portrd ./cmd/portrd
popd