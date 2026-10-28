#!/usr/bin/env bash
# Benchmarks the websocket tunnel end to end: portrd in Docker, the portr client
# and a Go origin on the host, httpbench and wsbench as load generators.
#
#   [PORTR_BIN=path/to/portr] scripts/bench/run.sh <label> [image]
#
# HTTP and websocket scenarios run over two network profiles:
#   loopback  client <-> server over localhost
#   wan       client <-> server through toxiproxy (RTT and upload limit below)
# Visitor traffic always hits portrd directly; only the tunnel leg is shaped.
# Results land in scripts/bench/results/<label>/.
set -euo pipefail

LABEL=${1:?usage: run.sh <label> [image]}
IMAGE=${2:-portr-bench:$LABEL}
SCALE=${SCALE:-1}                          # multiplies request counts
WAN_LATENCY_MS=${WAN_LATENCY_MS:-40}      # each direction, so RTT ~= 2x
WAN_UPLOAD_KBPS=${WAN_UPLOAD_KBPS:-2500}  # client -> server, KB/s (~20 Mbit/s)

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OUT=$ROOT/scripts/bench/results/$LABEL
WORK=$(mktemp -d)
mkdir -p "$OUT"
rm -f "$OUT"/*.txt "$OUT"/*.log
{
  echo "label=$LABEL image=$IMAGE scale=$SCALE"
  echo "commit=$(git -C "$ROOT" rev-parse --short HEAD)$(git -C "$ROOT" diff --quiet -- cmd internal || echo -dirty)"
  echo "date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "host=$(uname -sm) cpu=$(sysctl -n machdep.cpu.brand_string 2>/dev/null || nproc) docker=$(docker version -f '{{.Server.Version}}')"
  echo "wan=${WAN_LATENCY_MS}ms each way, upload ${WAN_UPLOAD_KBPS} KB/s"
} >"$OUT/env.info"

HOST=bench.localhost:8001
PIDS=()

cleanup() {
  for pid in "${PIDS[@]:-}"; do kill "$pid" 2>/dev/null || true; done
  docker rm -f portr-bench toxiproxy-bench >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

log() { printf '\n== %s\n' "$*"; }

wait_for() {
  local url=$1 host=${2:-}
  for _ in $(seq 100); do
    if [ -n "$host" ]; then
      [ "$(curl -s -o /dev/null -w '%{http_code}' -H "Host: $host" "$url")" = 200 ] && return 0
    else
      curl -sf -o /dev/null "$url" && return 0
    fi
    sleep 0.2
  done
  echo "timed out waiting for $url" >&2
  return 1
}

log "building tools"
# PORTR_BIN pins the client binary, so an older server image can be measured
# with the client that speaks its protocol version.
if [ -n "${PORTR_BIN:-}" ]; then cp "$PORTR_BIN" "$WORK/portr"; else go build -o "$WORK/portr" "$ROOT/cmd/portr"; fi
go build -o "$WORK/origin" "$ROOT/scripts/bench/origin"
go build -o "$WORK/wsbench" "$ROOT/scripts/bench/wsbench"
go build -o "$WORK/httpbench" "$ROOT/scripts/bench/httpbench"

log "starting portrd ($IMAGE)"
docker rm -f portr-bench toxiproxy-bench >/dev/null 2>&1 || true
docker run -d --name portr-bench -p 8000:8000 -p 8001:8001 \
  -e PORTR_DB_URL=sqlite:///app/data/db.sqlite3 \
  -e PORTR_DOMAIN=localhost:8000 -e PORTR_SERVER_URL=http://localhost:8000 \
  -e PORTR_TRANSPORT=websocket -e PORTR_WS_URL=localhost:8001 \
  -e PORTR_ADMIN_PORT=8000 -e PORTR_PROXY_PORT=8001 \
  -e PORTR_AUTO_MIGRATE=true -e PORTR_TUNNEL_USE_LOCALHOST=true \
  "$IMAGE" start all >/dev/null
wait_for http://localhost:8000/api/v1/healthcheck

# The first login creates the superuser; its default team carries the secret key.
JAR=$WORK/cookies
PASSWORD=$(openssl rand -hex 12)
curl -sf -c "$JAR" -b "$JAR" -H 'content-type: application/json' \
  -d "{\"email\":\"bench@example.com\",\"password\":\"$PASSWORD\"}" \
  http://localhost:8000/api/v1/auth/login >/dev/null
SECRET=$(curl -sf -b "$JAR" -H 'X-Team-Slug: default-team' http://localhost:8000/api/v1/user/me | jq -r .secret_key)

log "starting toxiproxy"
docker run -d --name toxiproxy-bench -p 8474:8474 -p 18001:18001 \
  --add-host host.docker.internal:host-gateway ghcr.io/shopify/toxiproxy:2.12.0 >/dev/null
wait_for http://localhost:8474/version
TOXI=http://localhost:8474
curl -sf -X POST $TOXI/proxies -d '{"name":"tunnel","listen":"0.0.0.0:18001","upstream":"host.docker.internal:8001"}' >/dev/null
curl -sf -X POST $TOXI/proxies/tunnel/toxics -d "{\"name\":\"lat_down\",\"type\":\"latency\",\"stream\":\"downstream\",\"attributes\":{\"latency\":$WAN_LATENCY_MS}}" >/dev/null
curl -sf -X POST $TOXI/proxies/tunnel/toxics -d "{\"name\":\"lat_up\",\"type\":\"latency\",\"stream\":\"upstream\",\"attributes\":{\"latency\":$WAN_LATENCY_MS}}" >/dev/null
curl -sf -X POST $TOXI/proxies/tunnel/toxics -d "{\"name\":\"bw_up\",\"type\":\"bandwidth\",\"stream\":\"upstream\",\"attributes\":{\"rate\":$WAN_UPLOAD_KBPS}}" >/dev/null

"$WORK/origin" -addr 127.0.0.1:9100 >"$WORK/origin.log" 2>&1 &
PIDS+=($!)
wait_for http://127.0.0.1:9100/tiny

CLIENT_PID=
start_client() {
  local ws_url=$1 cfg=$WORK/client-$1.yaml
  cat >"$cfg" <<EOF
server_url: localhost:8000
ws_url: $ws_url
tunnel_url: localhost:8001
transport: websocket
secret_key: $SECRET
use_localhost: true
disable_tui: true
disable_dashboard: true
EOF
  # The client logs every request to $HOME/.portr/db.sqlite; a scratch HOME
  # keeps benchmark traffic out of the real request log.
  mkdir -p "$WORK/home-$1"
  HOME="$WORK/home-$1" "$WORK/portr" -c "$cfg" http 9100 -s bench >"$WORK/client-$1.log" 2>&1 &
  CLIENT_PID=$!
  PIDS+=($CLIENT_PID)
  wait_for http://127.0.0.1:8001/tiny "$HOST"
}

stop_client() {
  kill "$CLIENT_PID" 2>/dev/null || true
  wait "$CLIENT_PID" 2>/dev/null || true
  # Let the server drop the session before the next profile registers.
  sleep 2
}

# Connections the tunnel closes linger in TIME_WAIT on this host; wait for them
# to drain so one scenario cannot starve the next of ephemeral ports.
drain() {
  for _ in $(seq 60); do
    [ "$(netstat -an -p tcp | grep -c TIME_WAIT)" -lt 1000 ] && return 0
    sleep 1
  done
}

# http <file> <conns> <requests> <path> [target]
http() {
  local name=$1 conns=$2 requests=$(( $3 * SCALE )) path=$4 target=${5:-tunnel}
  local args=(-url "http://127.0.0.1:8001$path" -host "$HOST")
  if [ "$target" = direct ]; then args=(-url "http://127.0.0.1:9100$path"); fi
  drain
  "$WORK/httpbench" "${args[@]}" -c "$conns" -n "$requests" >"$OUT/$name.txt" 2>&1 || true
}

# ws <file> <conns> <messages> <size> [target]
ws() {
  local name=$1 conns=$2 messages=$3 size=$4 target=${5:-tunnel}
  local args=(-url ws://127.0.0.1:8001/ws -host "$HOST")
  if [ "$target" = direct ]; then args=(-url ws://127.0.0.1:9100/ws); fi
  drain
  "$WORK/wsbench" "${args[@]}" -c "$conns" -n "$messages" -size "$size" >"$OUT/$name.txt" 2>&1 || true
}

# Head-of-line: small requests measured while 4 MB downloads share the tunnel.
# hol <file> <bulk requests> <tiny requests>
hol() {
  local name=$1
  drain
  "$WORK/httpbench" -url "http://127.0.0.1:8001/bytes?n=4194304" -host "$HOST" -c 4 -n $(( $2 * SCALE )) >"$OUT/$name.bulk4m.log" 2>&1 &
  local bulk=$!
  sleep 0.5
  "$WORK/httpbench" -url http://127.0.0.1:8001/tiny -host "$HOST" -c 8 -n $(( $3 * SCALE )) >"$OUT/$name.txt" 2>&1 || true
  wait $bulk 2>/dev/null || true
}

# recover <file> <trigger...>: time from trigger completing until the tunnel
# serves 200 again, counting what visitors saw meanwhile.
recover() {
  local name=$1; shift
  local codes=$WORK/codes
  : >"$codes"
  "$@"
  local start
  start=$(python3 -c 'import time; print(time.time())')
  local deadline=$(( $(date +%s) + 60 ))
  while :; do
    code=$(curl -s -o /dev/null -m 2 -w '%{http_code}' -H "Host: $HOST" http://127.0.0.1:8001/tiny || true)
    echo "$code" >>"$codes"
    [ "$code" = 200 ] && break
    if [ "$(date +%s)" -gt "$deadline" ]; then echo "gave up after 60s" >>"$OUT/$name.txt"; break; fi
    sleep 0.1
  done
  python3 - "$start" "$codes" >>"$OUT/$name.txt" <<'EOF'
import collections, sys, time
start, path = float(sys.argv[1]), sys.argv[2]
codes = [line.strip() for line in open(path) if line.strip()]
print(f"recovery_seconds={time.time() - start:.2f}")
print("visitor_responses=" + " ".join(f"{code}x{count}" for code, count in collections.Counter(codes).items()))
EOF
}

toxi_outage() {
  curl -sf -X POST $TOXI/proxies/tunnel -d '{"enabled":false}' >/dev/null
  sleep 5
  curl -sf -X POST $TOXI/proxies/tunnel -d '{"enabled":true}' >/dev/null
}

server_restart() {
  docker restart -t 2 portr-bench >/dev/null
  wait_for http://localhost:8000/api/v1/healthcheck
}

# run_profile <profile> <request multiplier>: the wan profile is far slower, so
# it gets smaller request counts to keep each scenario around ten seconds.
run_profile() {
  local profile=$1 k=$2
  log "$profile: http"
  http "$profile-http-tiny-c1" 1 $(( 1000 / k )) /tiny
  http "$profile-http-tiny-c32" 32 $(( 3000 / k )) /tiny
  http "$profile-http-64k-c32" 32 $(( 2000 / k )) "/bytes?n=65536"
  http "$profile-http-1m-c8" 8 $(( 400 / k )) "/bytes?n=1048576"
  hol "$profile-http-hol-tiny-during-4m" $(( 80 / k )) $(( 2000 / k ))
  log "$profile: websocket"
  ws "$profile-ws-1k-c1" 1 $(( 2000 / k )) 1024
  ws "$profile-ws-1k-c32" 32 $(( 400 / k )) 1024
  ws "$profile-ws-256k-c8" 8 $(( 80 / k )) 262144
  ws "$profile-ws-handshake-c50" 50 1 16
}

log "direct origin reference"
http direct-http-tiny-c1 1 1000 /tiny direct
http direct-http-tiny-c32 32 3000 /tiny direct
http direct-http-64k-c32 32 2000 "/bytes?n=65536" direct
ws direct-ws-1k-c32 32 200 1024 direct

start_client localhost:8001
run_profile loopback 1
log "loopback: server restart"
recover loopback-drop-server-restart server_restart
stop_client

start_client localhost:18001
run_profile wan 10
log "wan: 5s network outage"
recover wan-drop-5s-outage toxi_outage
stop_client

log "summary"
"$ROOT/scripts/bench/summarize.sh" "$OUT" | tee "$OUT/SUMMARY.md"
