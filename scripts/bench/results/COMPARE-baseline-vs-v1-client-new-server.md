| scenario | rate baseline-rerun → v1-client-new-server | p50 baseline-rerun → v1-client-new-server | p99 baseline-rerun → v1-client-new-server | MB/s baseline-rerun → v1-client-new-server |
|---|---|---|---|---|
| direct-http-64k-c32 | 34497.5 → 35186.7 | 900µs → 880µs | 1.7ms → 1.69ms | 2260.83 → 2305.99 |
| direct-http-tiny-c1 | 13113.2 → 13082.9 | 70µs → 70µs | 140µs → 160µs | 0.03 → 0.03 |
| direct-http-tiny-c32 | 68951.2 → 70060.5 | 420µs → 410µs | 1.28ms → 1.31ms | 0.14 → 0.14 |
| direct-ws-1k-c32 | 58534 → 56857 | 440µs → 440µs | 1.55ms → 2ms | 59.94 → 58.22 |
| loopback-http-1m-c8 | 52.1 → 53.8 | 153.77ms → 146.6ms | 200.35ms → 225.68ms | 54.58 → 56.40 |
| loopback-http-64k-c32 | 815.8 → 735.4 | 38.33ms → 38.77ms | 51.59ms → 65.07ms | 53.46 → 48.20 |
| loopback-http-hol-tiny-during-4m | 315.9 → 319.9 | 3.06ms → 3.36ms | 95.46ms → 89.66ms | 0.00 → 0.00 |
| loopback-http-tiny-c1 | 487.3 → 491.7 | 1.59ms → 1.7ms | 7.3ms → 6.56ms | 0.00 → 0.00 |
| loopback-http-tiny-c32 | 2631.7 → 2351.5 | 5.77ms → 5.59ms | 24.18ms → 29.68ms | 0.01 → 0.00 |
| loopback-ws-1k-c1 | 2148 → 1642 | 410µs → 490µs | 1.17ms → 2.06ms | 2.20 → 1.68 |
| loopback-ws-1k-c32 | 8269 → 7916 | 3.15ms → 3.09ms | 8.6ms → 8.22ms | 8.47 → 8.11 |
| loopback-ws-256k-c8 | 177 → 179 | 44.7ms → 44.44ms | 61.7ms → 64.49ms | 46.36 → 46.97 |
| loopback-ws-handshake-c50 | - → - | 13.36ms → 12.83ms | 18.36ms → 18.51ms | - → - |
| wan-http-1m-c8 | 1.8 → 1.8 | 4.49071s → 4.49034s | 5.019s → 5.0125s | 1.86 → 1.86 |
| wan-http-64k-c32 | 28.1 → 28.1 | 1.13025s → 1.13036s | 1.22569s → 1.23886s | 1.84 → 1.84 |
| wan-http-hol-tiny-during-4m | 10.3 → 10.3 | 111.53ms → 107.13ms | 2.26169s → 2.26908s | 0.00 → 0.00 |
| wan-http-tiny-c1 | 9.9 → 10.1 | 99.7ms → 97.47ms | 117.57ms → 123.91ms | 0.00 → 0.00 |
| wan-http-tiny-c32 | 268.7 → 120.6 | 108.99ms → 114.95ms | 293.7ms → 1.124s | 0.00 → 0.00 |
| wan-ws-1k-c1 | 10 → 11 | 97.94ms → 92.92ms | 121.07ms → 107.56ms | 0.01 → 0.01 |
| wan-ws-1k-c32 | 288 → 301 | 104.83ms → 100.61ms | 124.88ms → 122.72ms | 0.30 → 0.31 |
| wan-ws-256k-c8 | 7 → 7 | 1.12468s → 1.12584s | 1.25393s → 1.23945s | 1.82 → 1.82 |
| wan-ws-handshake-c50 | - → - | 331.96ms → 112.51ms | 336.5ms → 116.14ms | - → - |

| drop scenario | baseline-rerun | v1-client-new-server |
|---|---|---|
| loopback-drop-server-restart | recovery_seconds=1.03 visitor_responses=404x7 200x1  | recovery_seconds=0.92 visitor_responses=404x6 200x1  |
| wan-drop-5s-outage | recovery_seconds=3.26 visitor_responses=404x21 200x1  | recovery_seconds=2.78 visitor_responses=404x19 200x1  |
