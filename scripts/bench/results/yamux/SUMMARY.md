| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| direct-http-64k-c32 | 33836.3 | 920µs | 1.14ms | 1.8ms | 2217.49 | 0 |
| direct-http-tiny-c1 | 11547.6 | 80µs | 110µs | 190µs | 0.02 | 0 |
| direct-http-tiny-c32 | 62374.4 | 450µs | 800µs | 1.45ms | 0.12 | 0 |
| direct-ws-1k-c32 | 56700 | 450µs | 1.02ms | 1.44ms | 58.06 | 0 |
| loopback-http-1m-c8 | 313.6 | 25.05ms | 31.2ms | 61.05ms | 328.85 | 0 |
| loopback-http-64k-c32 | 1504.2 | 12.63ms | 19.98ms | 228.37ms | 98.58 | 0 |
| loopback-http-hol-tiny-during-4m | 1464.1 | 4.21ms | 7.76ms | 13.36ms | 0.00 | 0 |
| loopback-http-tiny-c1 | 446.0 | 1.73ms | 3.15ms | 10.36ms | 0.00 | 0 |
| loopback-http-tiny-c32 | 2328.8 | 5.92ms | 9.02ms | 34.23ms | 0.00 | 0 |
| loopback-ws-1k-c1 | 2353 | 390µs | 530µs | 930µs | 2.41 | 0 |
| loopback-ws-1k-c32 | 10434 | 2.67ms | 3.99ms | 6.27ms | 10.68 | 0 |
| loopback-ws-256k-c8 | 796 | 9.41ms | 13.38ms | 20.77ms | 208.78 | 0 |
| loopback-ws-handshake-c50 | - | 9.4ms | 10.12ms | 10.31ms | - | 0 |
| wan-http-1m-c8 | 2.3 | 3.4974s | 3.701s | 3.78531s | 2.46 | 0 |
| wan-http-64k-c32 | 37.3 | 848.06ms | 863.86ms | 950.64ms | 2.44 | 0 |
| wan-http-hol-tiny-during-4m | 32.0 | 247.77ms | 288.31ms | 369.09ms | 0.00 | 0 |
| wan-http-tiny-c1 | 10.1 | 98.01ms | 107.08ms | 116.98ms | 0.00 | 0 |
| wan-http-tiny-c32 | 217.3 | 107.81ms | 126.81ms | 1.10198s | 0.00 | 0 |
| wan-ws-1k-c1 | 11 | 91.87ms | 104.15ms | 135.08ms | 0.01 | 0 |
| wan-ws-1k-c32 | 287 | 100.33ms | 113.88ms | 124.34ms | 0.29 | 0 |
| wan-ws-256k-c8 | 7 | 1.05471s | 1.0858s | 1.09291s | 1.95 | 0 |
| wan-ws-handshake-c50 | - | 109.42ms | 112.45ms | 112.94ms | - | 0 |

| drop scenario | result |
|---|---|
| loopback-drop-server-restart | recovery_seconds=1.06 visitor_responses=404x7 200x1  |
| wan-drop-5s-outage | recovery_seconds=2.98 visitor_responses=404x20 200x1  |
