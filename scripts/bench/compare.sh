#!/usr/bin/env bash
# Compares two benchmark runs side by side.
#
#   scripts/bench/compare.sh <before-label> <after-label>
set -euo pipefail
ROOT=$(cd "$(dirname "$0")" && pwd)/results
BEFORE=$ROOT/${1:?usage: compare.sh <before> <after>}/SUMMARY.md
AFTER=$ROOT/${2:?usage: compare.sh <before> <after>}/SUMMARY.md

echo "| scenario | rate $1 → $2 | p50 $1 → $2 | p99 $1 → $2 | MB/s $1 → $2 |"
echo "|---|---|---|---|---|"
awk -F'|' '
  function trim(s) { gsub(/^ +| +$/, "", s); return s }
  FNR == NR && $2 ~ /[a-z]-/ && $3 != "" { rate[trim($2)] = trim($3); p50[trim($2)] = trim($4); p99[trim($2)] = trim($6); mb[trim($2)] = trim($7); next }
  $2 ~ /[a-z]-/ && $3 != "" {
    name = trim($2)
    if (!(name in rate) || name ~ /drop/) next
    printf "| %s | %s → %s | %s → %s | %s → %s | %s → %s |\n", name, rate[name], trim($3), p50[name], trim($4), p99[name], trim($6), mb[name], trim($7)
  }
' "$BEFORE" "$AFTER"

echo
echo "| drop scenario | $1 | $2 |"
echo "|---|---|---|"
for file in "$ROOT/$2"/*-drop-*.txt; do
  name=$(basename "$file" .txt)
  printf '| %s | %s | %s |\n' "$name" "$(tr '\n' ' ' <"$ROOT/$1/$name.txt")" "$(tr '\n' ' ' <"$file")"
done
