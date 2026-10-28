| scenario | rate live-v1-client-after-deploy → live-after | p50 live-v1-client-after-deploy → live-after | p99 live-v1-client-after-deploy → live-after | MB/s live-v1-client-after-deploy → live-after |
|---|---|---|---|---|
| live-http-1m-c4 | 8.1 → 7.6 | 393.15ms → 434.87ms | 1.12796s → 761.95ms | 8.52 → 7.97 |
| live-http-64k-c8 | 50.7 → 48.8 | 136.97ms → 141.79ms | 471.45ms → 470.4ms | 3.32 → 3.20 |
| live-http-hol-tiny-during-4m | 29.3 → 33.3 | 100.52ms → 111.71ms | 527.82ms → 230.51ms | 0.00 → 0.00 |
| live-http-tiny-c1 | 9.4 → 8.5 | 102.46ms → 116.18ms | 221.97ms → 221.6ms | 0.00 → 0.00 |
| live-http-tiny-c16 | 143.3 → 128.0 | 100.72ms → 108.89ms | 231.75ms → 277.77ms | 0.00 → 0.00 |
| live-ref-healthcheck-c1 | 19.0 → 18.8 | 47.99ms → 48.47ms | 154.28ms → 158.5ms | 0.00 → 0.00 |
| live-ws-1k-c1 | 10 → 9 | 96.46ms → 101.38ms | 119.98ms → 139.72ms | 0.01 → 0.01 |
| live-ws-1k-c16 | 141 → 130 | 99.07ms → 105.13ms | 124.72ms → 164.92ms | 0.14 → 0.13 |
| live-ws-256k-c4 | 14 → 9 | 198.95ms → 382.34ms | 582.09ms → 681.21ms | 3.56 → 2.44 |
| live-ws-handshake-c20 | - → - | 220.03ms → 223.17ms | 251.56ms → 239.78ms | - → - |

| drop scenario | live-v1-client-after-deploy | live-after |
|---|---|---|
| live-drop-client-health | unhealthy_events=0 reconnects=0  | unhealthy_events=0 reconnects=0  |
