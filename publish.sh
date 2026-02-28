#!/usr/bin/env bash
set -e

latest=$(git tag -l 'v*' --sort=-v:refname | head -n 1)

py=$(command -v python3 || command -v python)

if [ -z "$latest" ]; then
  next="v0.1.0"
else
  next=$($py - <<EOF
v="${latest#v}"
major, minor, patch = map(int, v.split("."))
print(f"v{major}.{minor}.{patch+1}")
EOF
)
fi

echo "Tagging $next"

git tag "$next"
git push origin "$next"