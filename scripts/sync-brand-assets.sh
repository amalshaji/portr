#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source_asset="$root/assets/brand/portr-mark.svg"
targets=(
  "$root/docs-v2/public/portr-mark.svg"
  "$root/internal/client/dashboard/ui-v2/public/portr-mark.svg"
  "$root/internal/client/dashboard/ui-v2/dist/static/portr-mark.svg"
  "$root/internal/server/admin/web-v2/public/portr-mark.svg"
  "$root/internal/server/admin/static/portr-mark.svg"
)

if [[ "${1:-}" == "--check" ]]; then
  status=0
  for target in "${targets[@]}"; do
    if ! cmp -s "$source_asset" "$target"; then
      echo "brand asset is out of sync: ${target#$root/}" >&2
      status=1
    fi
  done
  exit "$status"
fi

for target in "${targets[@]}"; do
  install -m 0644 "$source_asset" "$target"
done
