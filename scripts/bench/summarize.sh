#!/usr/bin/env bash
# Prints markdown tables from a scripts/bench/results/<label> directory.
set -euo pipefail
DIR=${1:?usage: summarize.sh <results-dir>}

echo "| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |"
echo "|---|---|---|---|---|---|---|"
for file in "$DIR"/*.txt; do
  name=$(basename "$file" .txt)
  case "$name" in
    *-drop-*) continue ;;
  esac
  awk -v name="$name" '
    /^(requests|conns)=/ { for (i = 1; i <= NF; i++) if ($i ~ /^failures=/) { split($i, f, "="); fails = f[2] } }
    /^(latency|echo rtt)/ { for (i = 1; i <= NF; i++) if ($i ~ /=/) { split($i, kv, "="); v[kv[1]] = kv[2] } }
    /^handshake/ { for (i = 2; i <= NF; i++) { split($i, kv, "="); h[kv[1]] = kv[2] } }
    /^throughput/ { rate = $2; for (i = 1; i <= NF; i++) if ($i == "MB/s" || $i == "MB/s,") mb = $(i - 1) }
    /^statuses/ { statuses = $0; sub(/^statuses /, "", statuses) }
    END {
      if (fails > 0 && statuses != "") fails = fails " (" statuses ")"
      if (name ~ /handshake/) printf "| %s | - | %s | %s | %s | - | %s |\n", name, h["p50"], h["p90"], h["p99"], fails
      else printf "| %s | %s | %s | %s | %s | %s | %s |\n", name, rate, v["p50"], v["p90"], v["p99"], mb, fails
    }' "$file"
done

echo
echo "| drop scenario | result |"
echo "|---|---|"
for file in "$DIR"/*-drop-*.txt; do
  [ -e "$file" ] || continue
  printf '| %s | %s |\n' "$(basename "$file" .txt)" "$(tr '\n' ' ' <"$file")"
done
