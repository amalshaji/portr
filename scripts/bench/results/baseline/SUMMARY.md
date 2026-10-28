| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| direct-http-64k-c32 | 32656.6 | 930µs | 1.31ms | 2.1ms | 2140.18 | 0 |
| direct-http-tiny-c1 | 11598.9 | 80µs | 120µs | 170µs | 0.02 | 0 |
| direct-http-tiny-c32 | 62435.7 | 470µs | 750µs | 1.35ms | 0.12 | 0 |
| direct-ws-1k-c32 | 57643 | 460µs | 890µs | 1.37ms | 59.03 | 0 |
| loopback-http-1m-c8 | 50.3 | 158.89ms | 181.12ms | 207.16ms | 52.71 | 0 |
| loopback-http-64k-c32 | 756.7 | 41.69ms | 48.64ms | 56.27ms | 49.59 | 0 |
| loopback-http-hol-tiny-during-4m | 308.8 | 3.59ms | 73.75ms | 94.2ms | 0.00 | 0 |
| loopback-http-tiny-c1 | 452.1 | 1.97ms | 2.98ms | 6.08ms | 0.00 | 0 |
| loopback-http-tiny-c32 | 2557.1 | 6.08ms | 8.79ms | 19.61ms | 0.01 | 0 |
| loopback-ws-1k-c1 | 1897 | 490µs | 690µs | 1.22ms | 1.94 | 0 |
| loopback-ws-1k-c32 | 8304 | 3.51ms | 5.66ms | 8.87ms | 8.50 | 0 |
| loopback-ws-256k-c8 | 178 | 44.59ms | 54.65ms | 62.83ms | 46.66 | 0 |
| loopback-ws-handshake-c50 | - | 11.3ms | 22.54ms | 23.16ms | - | 0 |
| wan-http-1m-c8 | 1.8 | 4.49318s | 4.77955s | 5.03373s | 1.86 | 0 |
| wan-http-64k-c32 | 28.0 | 1.13235s | 1.13915s | 1.20196s | 1.84 | 0 |
| wan-http-hol-tiny-during-4m | 10.3 | 108.72ms | 2.25027s | 2.2662s | 0.00 | 0 |
| wan-http-tiny-c1 | 10.0 | 99.93ms | 106.4ms | 113.76ms | 0.00 | 0 |
| wan-http-tiny-c32 | 143.1 | 105.79ms | 129.99ms | 1.12044s | 0.00 | 0 |
| wan-ws-1k-c1 | 10 | 98.51ms | 102.97ms | 112.16ms | 0.01 | 0 |
| wan-ws-1k-c32 | 286 | 102.03ms | 111.92ms | 127ms | 0.29 | 0 |
| wan-ws-256k-c8 | 7 | 1.12504s | 1.13286s | 1.2479s | 1.81 | 0 |
| wan-ws-handshake-c50 | - | 129.89ms | 133.75ms | 134.28ms | - | 0 |

| drop scenario | result |
|---|---|
| loopback-drop-server-restart | recovery_seconds=0.86 visitor_responses=404x6 200x1  |
| wan-drop-5s-outage | recovery_seconds=2.90 visitor_responses=404x20 200x1  |
