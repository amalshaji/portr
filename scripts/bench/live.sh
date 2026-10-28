#!/usr/bin/env bash
# Benchmarks a deployed portr server: the portr client and a Go origin run on
# this machine, and load goes to the public tunnel URL, so both the tunnel leg
# (this machine's uplink) and the visitor leg cross the internet.
#
#   [PORTR_BIN=path/to/portr CLIENT_SOURCE=desc] scripts/bench/live.sh <label> [client-config]
#
# client-config defaults to ~/.portr/config.yaml and must be authenticated
# against the server. Request counts are small because the server is shared.
# Results land in scripts/bench/results/<label>/.
set -euo pipefail

LABEL=${1:?usage: live.sh <label> [client-config]}
CONFIG=${2:-$HOME/.portr/config.yaml}
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OUT=$ROOT/scripts/bench/results/$LABEL
WORK=$(mktemp -d)
PIDS=()

cleanup() {
  for pid in "${PIDS[@]:-}"; do kill "$pid" 2>/dev/null || true; done
  rm -rf "$WORK"
}
trap cleanup EXIT

log() { printf '\n== %s\n' "$*"; }

mkdir -p "$OUT"
rm -f "$OUT"/*.txt "$OUT"/*.log

log "building tools"
if [ -n "${PORTR_BIN:-}" ]; then cp "$PORTR_BIN" "$WORK/portr"; else go build -o "$WORK/portr" "$ROOT/cmd/portr"; fi
go build -o "$WORK/origin" "$ROOT/scripts/bench/origin"
go build -o "$WORK/wsbench" "$ROOT/scripts/bench/wsbench"
go build -o "$WORK/httpbench" "$ROOT/scripts/bench/httpbench"

# The client logs every request to $HOME/.portr/db.sqlite; a scratch HOME keeps
# benchmark traffic out of the real request log.
mkdir -p "$WORK/home/.portr"
cfg=$WORK/home/.portr/config.yaml
grep -vE '^(disable_tui|disable_dashboard):' "$CONFIG" >"$cfg"
printf 'disable_tui: true\ndisable_dashboard: true\n' >>"$cfg"
chmod 600 "$cfg"
TUNNEL_DOMAIN=$(awk '/^tunnel_url:/ { print $2 }' "$cfg")
SUBDOMAIN=bench-$(openssl rand -hex 3)
BASE=https://$SUBDOMAIN.$TUNNEL_DOMAIN
WSBASE=wss://$SUBDOMAIN.$TUNNEL_DOMAIN

{
  echo "label=$LABEL server=$TUNNEL_DOMAIN subdomain=$SUBDOMAIN"
  if [ -n "${PORTR_BIN:-}" ]; then
    echo "client=$(basename "$PORTR_BIN") (PORTR_BIN, ${CLIENT_SOURCE:-source not recorded})"
  else
    echo "client source=$(git -C "$ROOT" rev-parse --short HEAD)"
  fi
  echo "server_version=$(curl -s "https://$TUNNEL_DOMAIN/api/v1/version")"
  echo "date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "host=$(uname -sm)"
} >"$OUT/env.info"

"$WORK/origin" -addr 127.0.0.1:9100 >"$WORK/origin.log" 2>&1 &
PIDS+=($!)
HOME="$WORK/home" "$WORK/portr" -c "$cfg" http 9100 -s "$SUBDOMAIN" >"$OUT/client.log" 2>&1 &
PIDS+=($!)
for _ in $(seq 150); do
  [ "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/tiny")" = 200 ] && break
  sleep 0.2
done
[ "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/tiny")" = 200 ] || { echo "tunnel did not come up" >&2; tail -5 "$OUT/client.log" >&2; exit 1; }

http() { "$WORK/httpbench" -url "$BASE$3" -c "$2" -n "$4" >"$OUT/$1.txt" 2>&1 || true; }
ws() { "$WORK/wsbench" -url "$WSBASE/ws" -c "$2" -n "$3" -size "$4" >"$OUT/$1.txt" 2>&1 || true; }

log "reference: server healthcheck without the tunnel"
"$WORK/httpbench" -url "https://$TUNNEL_DOMAIN/api/v1/healthcheck" -c 1 -n 30 >"$OUT/live-ref-healthcheck-c1.txt" 2>&1 || true

log "http"
http live-http-tiny-c1 1 /tiny 50
http live-http-tiny-c16 16 /tiny 320
http live-http-64k-c8 8 "/bytes?n=65536" 160
http live-http-1m-c4 4 "/bytes?n=1048576" 24
"$WORK/httpbench" -url "$BASE/bytes?n=4194304" -c 2 -n 6 >"$OUT/live-http-hol-tiny-during-4m.bulk4m.log" 2>&1 &
bulk=$!
sleep 0.5
http live-http-hol-tiny-during-4m 4 /tiny 120
wait $bulk 2>/dev/null || true

log "websocket"
ws live-ws-1k-c1 1 100 1024
ws live-ws-1k-c16 16 40 1024
ws live-ws-256k-c4 4 10 262144
ws live-ws-handshake-c20 20 1 16

# Unhealthy events mean the client tore the tunnel down and reconnected.
{
  echo "unhealthy_events=$(grep -c 'Tunnel unhealthy' "$OUT/client.log" || true)"
  echo "reconnects=$(grep -c 'Tunnel reconnected' "$OUT/client.log" || true)"
} >"$OUT/live-drop-client-health.txt"
# Only the connection lifecycle is worth keeping from the client log.
grep -E 'Tunnel|mismatch|rror' "$OUT/client.log" >"$OUT/client-events.log" || true
rm -f "$OUT/client.log"

log "summary"
"$ROOT/scripts/bench/summarize.sh" "$OUT" | tee "$OUT/SUMMARY.md"
