| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| live-http-1m-c4 | 7.6 | 400.73ms | 756.19ms | 819.26ms | 8.00 | 0 |
| live-http-64k-c8 | 53.0 | 130.27ms | 150.82ms | 466.98ms | 3.47 | 0 |
| live-http-hol-tiny-during-4m | 26.0 | 98.81ms | 228.41ms | 1.34893s | 0.00 | 0 |
| live-http-tiny-c1 | 8.6 | 112.8ms | 124.61ms | 250.85ms | 0.00 | 0 |
| live-http-tiny-c16 | 149.2 | 96.8ms | 116.41ms | 230.81ms | 0.00 | 0 |
| live-ref-healthcheck-c1 | 18.1 | 48.97ms | 65.42ms | 161.12ms | 0.00 | 0 |
| live-ws-1k-c1 | 10 | 99.06ms | 107.85ms | 142.96ms | 0.01 | 0 |
| live-ws-1k-c16 | 144 | 96.82ms | 102.37ms | 110.43ms | 0.15 | 0 |
| live-ws-256k-c4 | 11 | 210.97ms | 481.09ms | 700.35ms | 2.88 | 0 |
| live-ws-handshake-c20 | - | 216.14ms | 228.04ms | 235.18ms | - | 0 |

| drop scenario | result |
|---|---|
| live-drop-client-health | unhealthy_events=0 reconnects=0  |
