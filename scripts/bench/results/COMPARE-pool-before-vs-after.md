| scenario | rate pool-before → pool-after | p50 pool-before → pool-after | p99 pool-before → pool-after | MB/s pool-before → pool-after |
|---|---|---|---|---|
| direct-http-64k-c32 | 35489.9 → 32725.9 | 850µs → 930µs | 1.97ms → 1.96ms | 2325.86 → 2144.73 |
| direct-http-tiny-c1 | 10352.1 → 11526.6 | 90µs → 80µs | 240µs → 160µs | 0.02 → 0.02 |
| direct-http-tiny-c32 | 65412.9 → 65328.0 | 440µs → 440µs | 1.31ms → 1.3ms | 0.13 → 0.13 |
| direct-ws-1k-c32 | 54976 → 57179 | 460µs → 450µs | 1.54ms → 1.64ms | 56.30 → 58.55 |
| loopback-http-1m-c8 | 312.9 → 396.8 | 24.36ms → 19.97ms | 59.77ms → 28.15ms | 328.14 → 416.10 |
| loopback-http-64k-c32 | 1176.9 → 3692.5 | 14.1ms → 8.33ms | 42.35ms → 19.94ms | 77.13 → 241.99 |
| loopback-http-hol-tiny-during-4m | 1824.9 → 4039.3 | 3.52ms → 1.25ms | 11.11ms → 6.26ms | 0.00 → 0.01 |
| loopback-http-tiny-c1 | 422.2 → 1888.3 | 1.87ms → 490µs | 10.09ms → 1.36ms | 0.00 → 0.00 |
| loopback-http-tiny-c32 | 1630.2 → 8473.2 | 6.52ms → 3.35ms | 56.36ms → 8.57ms | 0.00 → 0.02 |
| loopback-ws-1k-c1 | 2088 → 2398 | 430µs → 400µs | 1.13ms → 790µs | 2.14 → 2.46 |
| loopback-ws-1k-c32 | 8975 → 10260 | 3.07ms → 2.43ms | 7.52ms → 5.26ms | 9.19 → 10.51 |
| loopback-ws-256k-c8 | 851 → 940 | 9.12ms → 8.18ms | 18.76ms → 13.64ms | 223.05 → 246.54 |
| loopback-ws-handshake-c50 | - → - | 8.84ms → 6.9ms | 10.49ms → 11.22ms | - → - |
| wan-http-1m-c8 | 13.0 → 13.0 | 610.99ms → 632.68ms | 677.82ms → 696.83ms | 13.64 → 13.64 |
| wan-http-64k-c32 | 205.9 → 206.4 | 152.18ms → 152.1ms | 229.62ms → 186.46ms | 13.49 → 13.53 |
| wan-http-hol-tiny-during-4m | 82.6 → 91.5 | 97.01ms → 83.81ms | 138.18ms → 133.18ms | 0.00 → 0.00 |
| wan-http-tiny-c1 | 14.3 → 14.5 | 69.05ms → 68ms | 92.41ms → 84.66ms | 0.00 → 0.00 |
| wan-http-tiny-c32 | 182.5 → 378.0 | 73.83ms → 81.59ms | 1.09542s → 267.77ms | 0.00 → 0.00 |
| wan-ws-1k-c1 | 15 → 15 | 67.31ms → 66.95ms | 89.23ms → 83.02ms | 0.02 → 0.02 |
| wan-ws-1k-c32 | 421 → 434 | 70.33ms → 68.85ms | 92.95ms → 95.11ms | 0.43 → 0.44 |
| wan-ws-256k-c8 | 51 → 51 | 152.4ms → 152.18ms | 248.24ms → 218ms | 13.37 → 13.42 |
| wan-ws-handshake-c50 | - → - | 78.83ms → 81.41ms | 79.59ms → 284.41ms | - → - |

| drop scenario | pool-before | pool-after |
|---|---|---|
| loopback-drop-server-restart | recovery_seconds=0.81 visitor_responses=404x5 200x1  | recovery_seconds=1.05 visitor_responses=404x7 200x1  |
| wan-drop-5s-outage | recovery_seconds=2.96 visitor_responses=404x20 200x1  | recovery_seconds=2.54 visitor_responses=404x17 200x1  |
