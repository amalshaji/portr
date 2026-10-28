| scenario | rate live-before → live-after | p50 live-before → live-after | p99 live-before → live-after | MB/s live-before → live-after |
|---|---|---|---|---|
| live-http-1m-c4 | 7.6 → 7.6 | 400.73ms → 434.87ms | 819.26ms → 761.95ms | 8.00 → 7.97 |
| live-http-64k-c8 | 53.0 → 48.8 | 130.27ms → 141.79ms | 466.98ms → 470.4ms | 3.47 → 3.20 |
| live-http-hol-tiny-during-4m | 26.0 → 33.3 | 98.81ms → 111.71ms | 1.34893s → 230.51ms | 0.00 → 0.00 |
| live-http-tiny-c1 | 8.6 → 8.5 | 112.8ms → 116.18ms | 250.85ms → 221.6ms | 0.00 → 0.00 |
| live-http-tiny-c16 | 149.2 → 128.0 | 96.8ms → 108.89ms | 230.81ms → 277.77ms | 0.00 → 0.00 |
| live-ref-healthcheck-c1 | 18.1 → 18.8 | 48.97ms → 48.47ms | 161.12ms → 158.5ms | 0.00 → 0.00 |
| live-ws-1k-c1 | 10 → 9 | 99.06ms → 101.38ms | 142.96ms → 139.72ms | 0.01 → 0.01 |
| live-ws-1k-c16 | 144 → 130 | 96.82ms → 105.13ms | 110.43ms → 164.92ms | 0.15 → 0.13 |
| live-ws-256k-c4 | 11 → 9 | 210.97ms → 382.34ms | 700.35ms → 681.21ms | 2.88 → 2.44 |
| live-ws-handshake-c20 | - → - | 216.14ms → 223.17ms | 235.18ms → 239.78ms | - → - |

| drop scenario | live-before | live-after |
|---|---|---|
| live-drop-client-health | unhealthy_events=0 reconnects=0  | unhealthy_events=0 reconnects=0  |
