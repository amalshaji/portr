#!/usr/bin/env bash
set -euo pipefail

# ----------------------------
# Defaults (can be overridden)
# ----------------------------

PKG_NAME="${PKG_NAME:-portr}"
PREFIX="${PREFIX:-tunnel}"

# ----------------------------
# Optional dashboard build
# ----------------------------

if [[ "$PKG_NAME" == "portr" ]]; then
  echo "Building dashboard UI (PKG_NAME=portr)"

  pushd tunnel/internal/client/dashboard/ui >/dev/null

  corepack enable >/dev/null 2>&1 || true
  pnpm install
  pnpm build

  # Prevent node_modules from leaking into the package
  rm -rf node_modules

  popd >/dev/null
else
  echo "Skipping dashboard build (PKG_NAME=$PKG_NAME)"
fi

# ----------------------------
# Go build
# ----------------------------

pushd tunnel >/dev/null

go test ./...

mkdir -p "$PREFIX/bin"
go build -o "$PREFIX/bin/$PKG_NAME" "./cmd/$PKG_NAME"

popd >/dev/null

echo "Build complete: $PREFIX/bin/$PKG_NAME"