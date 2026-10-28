# Tunnel benchmark

End-to-end benchmark for the websocket tunnel transport: portrd runs in Docker,
the portr client and a Go origin run on the host, and `httpbench` / `wsbench`
generate load against portrd's proxy port.

```bash
docker build -t portr-bench:<label> .
SCALE=2 scripts/bench/run.sh <label>
```

Results go to `scripts/bench/results/<label>/`: one file per scenario,
`env.info`, and `SUMMARY.md`. Requires Docker, Go, `jq` and `curl`.

The client is built from the working tree. To measure an older server image,
pass a client binary that speaks its protocol version with `PORTR_BIN`.

Compare two runs:

```bash
scripts/bench/compare.sh baseline-rerun yamux
```

Runs on a laptop vary by roughly ±30% between identical runs, and the WAN
websocket handshake at 50 concurrent connections alternates between ~110 ms and
~330 ms on both protocol versions, so compare runs taken back to back.

## Deployed server

`live.sh` benchmarks a deployed server with an authenticated client config
(default `~/.portr/config.yaml`). The client tunnels a random `bench-*`
subdomain and load goes to its public URL, so both legs cross the internet.
Request counts are small because the server is shared, and there are no drop
scenarios; `live-drop-client-health` counts how often the client tore the
tunnel down during the run.

```bash
PORTR_BIN=path/to/portr CLIENT_SOURCE="main 209fa22" scripts/bench/live.sh live-before
scripts/bench/live.sh live-after
scripts/bench/compare.sh live-before live-after
```

Benchmark clients always run with a scratch `HOME`, so their request logs never
land in the real `~/.portr/db.sqlite`.

## Profiles

- `direct`: load generator straight to the origin, as a reference.
- `loopback`: tunnel client connects to portrd over localhost.
- `wan`: tunnel client connects through toxiproxy with 40 ms latency each way
  (~80 ms RTT) and a 2500 KB/s upload cap from client to server, like a home
  connection. Override with `WAN_LATENCY_MS` and `WAN_UPLOAD_KBPS`.

Visitor traffic always goes straight to portrd; only the tunnel leg is shaped.

## Scenarios

| name | what it measures |
|---|---|
| `http-tiny-c1` | per-request latency added by the tunnel |
| `http-tiny-c32` | many small parallel requests, like a dev-server page load |
| `http-64k-c32`, `http-1m-c8` | throughput for medium and large responses |
| `http-hol-tiny-during-4m` | small requests while 4 MB downloads share the tunnel (head-of-line blocking) |
| `ws-1k-c1`, `ws-1k-c32` | websocket echo round trip through the tunnel |
| `ws-256k-c8` | large websocket messages |
| `ws-handshake-c50` | websocket upgrade latency for 50 concurrent connects |
| `drop-server-restart` | seconds until visitors get 200 again after portrd restarts, and what they saw meanwhile |
| `drop-5s-outage` | same, after a 5 s network outage between client and server (wan only) |

Request counts are fixed rather than time-boxed. The tunnel closes every
connection, and on macOS each closed connection holds an ephemeral port in
TIME_WAIT for 30 s, so a time-boxed run turns into a port-exhaustion test. The
harness also waits for TIME_WAIT sockets to drain between scenarios.
