#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
assets=(
  "portr-mark.svg"
  "portr-mark-on-light.svg"
)
target_directories=(
  "$root/docs-v2/public"
  "$root/internal/client/dashboard/ui-v2/public"
  "$root/internal/client/dashboard/ui-v2/dist/static"
  "$root/internal/server/admin/web-v2/public"
  "$root/internal/server/admin/static"
)

if [[ "${1:-}" == "--check" ]]; then
  status=0
  for asset in "${assets[@]}"; do
    source_asset="$root/assets/brand/$asset"
    for target_directory in "${target_directories[@]}"; do
      target="$target_directory/$asset"
      if ! cmp -s "$source_asset" "$target"; then
        echo "brand asset is out of sync: ${target#$root/}" >&2
        status=1
      fi
    done
  done
  exit "$status"
fi

for asset in "${assets[@]}"; do
  source_asset="$root/assets/brand/$asset"
  for target_directory in "${target_directories[@]}"; do
    install -m 0644 "$source_asset" "$target_directory/$asset"
  done
done
