#!/bin/sh
set -eu

case "${1:-}" in
  "")
    go list ./...
    ;;
  --build)
    go list -f '{{if .GoFiles}}{{.ImportPath}}{{end}}' ./...
    ;;
  *)
    echo "usage: $0 [--build]" >&2
    exit 2
    ;;
esac | grep -v '/node_modules/' | grep .
