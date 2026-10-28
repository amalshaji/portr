| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| live-http-1m-c4 | 8.1 | 393.15ms | 917.24ms | 1.12796s | 8.52 | 0 |
| live-http-64k-c8 | 50.7 | 136.97ms | 168.59ms | 471.45ms | 3.32 | 0 |
| live-http-hol-tiny-during-4m | 29.3 | 100.52ms | 193.62ms | 527.82ms | 0.00 | 0 |
| live-http-tiny-c1 | 9.4 | 102.46ms | 123.81ms | 221.97ms | 0.00 | 0 |
| live-http-tiny-c16 | 143.3 | 100.72ms | 120.85ms | 231.75ms | 0.00 | 0 |
| live-ref-healthcheck-c1 | 19.0 | 47.99ms | 59.24ms | 154.28ms | 0.00 | 0 |
| live-ws-1k-c1 | 10 | 96.46ms | 106.13ms | 119.98ms | 0.01 | 0 |
| live-ws-1k-c16 | 141 | 99.07ms | 109.33ms | 124.72ms | 0.14 | 0 |
| live-ws-256k-c4 | 14 | 198.95ms | 468.66ms | 582.09ms | 3.56 | 0 |
| live-ws-handshake-c20 | - | 220.03ms | 232.31ms | 251.56ms | - | 0 |

| drop scenario | result |
|---|---|
| live-drop-client-health | unhealthy_events=0 reconnects=0  |
