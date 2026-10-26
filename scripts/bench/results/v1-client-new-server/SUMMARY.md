| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| direct-http-64k-c32 | 35186.7 | 880µs | 1.1ms | 1.69ms | 2305.99 | 0 |
| direct-http-tiny-c1 | 13082.9 | 70µs | 100µs | 160µs | 0.03 | 0 |
| direct-http-tiny-c32 | 70060.5 | 410µs | 620µs | 1.31ms | 0.14 | 0 |
| direct-ws-1k-c32 | 56857 | 440µs | 870µs | 2ms | 58.22 | 0 |
| loopback-http-1m-c8 | 53.8 | 146.6ms | 169.95ms | 225.68ms | 56.40 | 0 |
| loopback-http-64k-c32 | 735.4 | 38.77ms | 46.31ms | 65.07ms | 48.20 | 0 |
| loopback-http-hol-tiny-during-4m | 319.9 | 3.36ms | 71.75ms | 89.66ms | 0.00 | 0 |
| loopback-http-tiny-c1 | 491.7 | 1.7ms | 2.92ms | 6.56ms | 0.00 | 0 |
| loopback-http-tiny-c32 | 2351.5 | 5.59ms | 9.12ms | 29.68ms | 0.00 | 0 |
| loopback-ws-1k-c1 | 1642 | 490µs | 850µs | 2.06ms | 1.68 | 0 |
| loopback-ws-1k-c32 | 7916 | 3.09ms | 5.4ms | 8.22ms | 8.11 | 0 |
| loopback-ws-256k-c8 | 179 | 44.44ms | 53.62ms | 64.49ms | 46.97 | 0 |
| loopback-ws-handshake-c50 | - | 12.83ms | 18.17ms | 18.51ms | - | 0 |
| wan-http-1m-c8 | 1.8 | 4.49034s | 4.52257s | 5.0125s | 1.86 | 0 |
| wan-http-64k-c32 | 28.1 | 1.13036s | 1.13573s | 1.23886s | 1.84 | 0 |
| wan-http-hol-tiny-during-4m | 10.3 | 107.13ms | 2.24885s | 2.26908s | 0.00 | 0 |
| wan-http-tiny-c1 | 10.1 | 97.47ms | 106.79ms | 123.91ms | 0.00 | 0 |
| wan-http-tiny-c32 | 120.6 | 114.95ms | 1.10286s | 1.124s | 0.00 | 0 |
| wan-ws-1k-c1 | 11 | 92.92ms | 99.55ms | 107.56ms | 0.01 | 0 |
| wan-ws-1k-c32 | 301 | 100.61ms | 113.07ms | 122.72ms | 0.31 | 0 |
| wan-ws-256k-c8 | 7 | 1.12584s | 1.1344s | 1.23945s | 1.82 | 0 |
| wan-ws-handshake-c50 | - | 112.51ms | 115.9ms | 116.14ms | - | 0 |

| drop scenario | result |
|---|---|
| loopback-drop-server-restart | recovery_seconds=0.92 visitor_responses=404x6 200x1  |
| wan-drop-5s-outage | recovery_seconds=2.78 visitor_responses=404x19 200x1  |
