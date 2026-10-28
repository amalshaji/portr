| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| direct-http-64k-c32 | 35489.9 | 850µs | 1.15ms | 1.97ms | 2325.86 | 0 |
| direct-http-tiny-c1 | 10352.1 | 90µs | 130µs | 240µs | 0.02 | 0 |
| direct-http-tiny-c32 | 65412.9 | 440µs | 680µs | 1.31ms | 0.13 | 0 |
| direct-ws-1k-c32 | 54976 | 460µs | 1.07ms | 1.54ms | 56.30 | 0 |
| loopback-http-1m-c8 | 312.9 | 24.36ms | 31.19ms | 59.77ms | 328.14 | 0 |
| loopback-http-64k-c32 | 1176.9 | 14.1ms | 19.47ms | 42.35ms | 77.13 | 0 |
| loopback-http-hol-tiny-during-4m | 1824.9 | 3.52ms | 7.7ms | 11.11ms | 0.00 | 0 |
| loopback-http-tiny-c1 | 422.2 | 1.87ms | 3.58ms | 10.09ms | 0.00 | 0 |
| loopback-http-tiny-c32 | 1630.2 | 6.52ms | 11.26ms | 56.36ms | 0.00 | 1 (200=2999 err:eof=1) |
| loopback-ws-1k-c1 | 2088 | 430µs | 610µs | 1.13ms | 2.14 | 0 |
| loopback-ws-1k-c32 | 8975 | 3.07ms | 4.8ms | 7.52ms | 9.19 | 0 |
| loopback-ws-256k-c8 | 851 | 9.12ms | 12.11ms | 18.76ms | 223.05 | 0 |
| loopback-ws-handshake-c50 | - | 8.84ms | 10.37ms | 10.49ms | - | 0 |
| wan-http-1m-c8 | 13.0 | 610.99ms | 631.77ms | 677.82ms | 13.64 | 0 |
| wan-http-64k-c32 | 205.9 | 152.18ms | 157.57ms | 229.62ms | 13.49 | 0 |
| wan-http-hol-tiny-during-4m | 82.6 | 97.01ms | 119.34ms | 138.18ms | 0.00 | 0 |
| wan-http-tiny-c1 | 14.3 | 69.05ms | 77.39ms | 92.41ms | 0.00 | 0 |
| wan-http-tiny-c32 | 182.5 | 73.83ms | 280.5ms | 1.09542s | 0.00 | 0 |
| wan-ws-1k-c1 | 15 | 67.31ms | 75.07ms | 89.23ms | 0.02 | 0 |
| wan-ws-1k-c32 | 421 | 70.33ms | 82.32ms | 92.95ms | 0.43 | 0 |
| wan-ws-256k-c8 | 51 | 152.4ms | 156.86ms | 248.24ms | 13.37 | 0 |
| wan-ws-handshake-c50 | - | 78.83ms | 79.35ms | 79.59ms | - | 0 |

| drop scenario | result |
|---|---|
| loopback-drop-server-restart | recovery_seconds=0.81 visitor_responses=404x5 200x1  |
| wan-drop-5s-outage | recovery_seconds=2.96 visitor_responses=404x20 200x1  |
