#!/usr/bin/env bash
set -euo pipefail

# ----------------------------
# Package name (argument > env > default)
# ----------------------------
PKG_NAME="${1:-${PKG_NAME:-portr}}"

echo "PKG_NAME=${PKG_NAME}"

# ----------------------------
# Optional dashboard build
# ----------------------------
if [[ "$PKG_NAME" == "portr" ]]; then
  echo "Building dashboard UI"

  pushd "./tunnel/internal/client/dashboard/ui" >/dev/null

  corepack enable >/dev/null 2>&1 || true
  pnpm install
  pnpm build

  rm -rf node_modules

  popd >/dev/null
else
  echo "Skipping dashboard build"
fi

# ----------------------------
# Go build
# ----------------------------
pushd "./tunnel" >/dev/null

go test ./...

PREFIX="${PREFIX:-$(pwd)}"
mkdir -p "$PREFIX/bin"
go build -trimpath -ldflags="-s -w" -o "$PREFIX/bin/${PKG_NAME}" "./cmd/${PKG_NAME}"

popd >/dev/null

echo "Build complete: $PREFIX/bin/${PKG_NAME}"