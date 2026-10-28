| scenario | rate baseline-rerun → yamux | p50 baseline-rerun → yamux | p99 baseline-rerun → yamux | MB/s baseline-rerun → yamux |
|---|---|---|---|---|
| direct-http-64k-c32 | 34497.5 → 33836.3 | 900µs → 920µs | 1.7ms → 1.8ms | 2260.83 → 2217.49 |
| direct-http-tiny-c1 | 13113.2 → 11547.6 | 70µs → 80µs | 140µs → 190µs | 0.03 → 0.02 |
| direct-http-tiny-c32 | 68951.2 → 62374.4 | 420µs → 450µs | 1.28ms → 1.45ms | 0.14 → 0.12 |
| direct-ws-1k-c32 | 58534 → 56700 | 440µs → 450µs | 1.55ms → 1.44ms | 59.94 → 58.06 |
| loopback-http-1m-c8 | 52.1 → 313.6 | 153.77ms → 25.05ms | 200.35ms → 61.05ms | 54.58 → 328.85 |
| loopback-http-64k-c32 | 815.8 → 1504.2 | 38.33ms → 12.63ms | 51.59ms → 228.37ms | 53.46 → 98.58 |
| loopback-http-hol-tiny-during-4m | 315.9 → 1464.1 | 3.06ms → 4.21ms | 95.46ms → 13.36ms | 0.00 → 0.00 |
| loopback-http-tiny-c1 | 487.3 → 446.0 | 1.59ms → 1.73ms | 7.3ms → 10.36ms | 0.00 → 0.00 |
| loopback-http-tiny-c32 | 2631.7 → 2328.8 | 5.77ms → 5.92ms | 24.18ms → 34.23ms | 0.01 → 0.00 |
| loopback-ws-1k-c1 | 2148 → 2353 | 410µs → 390µs | 1.17ms → 930µs | 2.20 → 2.41 |
| loopback-ws-1k-c32 | 8269 → 10434 | 3.15ms → 2.67ms | 8.6ms → 6.27ms | 8.47 → 10.68 |
| loopback-ws-256k-c8 | 177 → 796 | 44.7ms → 9.41ms | 61.7ms → 20.77ms | 46.36 → 208.78 |
| loopback-ws-handshake-c50 | - → - | 13.36ms → 9.4ms | 18.36ms → 10.31ms | - → - |
| wan-http-1m-c8 | 1.8 → 2.3 | 4.49071s → 3.4974s | 5.019s → 3.78531s | 1.86 → 2.46 |
| wan-http-64k-c32 | 28.1 → 37.3 | 1.13025s → 848.06ms | 1.22569s → 950.64ms | 1.84 → 2.44 |
| wan-http-hol-tiny-during-4m | 10.3 → 32.0 | 111.53ms → 247.77ms | 2.26169s → 369.09ms | 0.00 → 0.00 |
| wan-http-tiny-c1 | 9.9 → 10.1 | 99.7ms → 98.01ms | 117.57ms → 116.98ms | 0.00 → 0.00 |
| wan-http-tiny-c32 | 268.7 → 217.3 | 108.99ms → 107.81ms | 293.7ms → 1.10198s | 0.00 → 0.00 |
| wan-ws-1k-c1 | 10 → 11 | 97.94ms → 91.87ms | 121.07ms → 135.08ms | 0.01 → 0.01 |
| wan-ws-1k-c32 | 288 → 287 | 104.83ms → 100.33ms | 124.88ms → 124.34ms | 0.30 → 0.29 |
| wan-ws-256k-c8 | 7 → 7 | 1.12468s → 1.05471s | 1.25393s → 1.09291s | 1.82 → 1.95 |
| wan-ws-handshake-c50 | - → - | 331.96ms → 109.42ms | 336.5ms → 112.94ms | - → - |

| drop scenario | baseline-rerun | yamux |
|---|---|---|
| loopback-drop-server-restart | recovery_seconds=1.03 visitor_responses=404x7 200x1  | recovery_seconds=1.06 visitor_responses=404x7 200x1  |
| wan-drop-5s-outage | recovery_seconds=3.26 visitor_responses=404x21 200x1  | recovery_seconds=2.98 visitor_responses=404x20 200x1  |
