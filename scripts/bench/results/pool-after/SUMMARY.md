| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| direct-http-64k-c32 | 32725.9 | 930µs | 1.32ms | 1.96ms | 2144.73 | 0 |
| direct-http-tiny-c1 | 11526.6 | 80µs | 120µs | 160µs | 0.02 | 0 |
| direct-http-tiny-c32 | 65328.0 | 440µs | 690µs | 1.3ms | 0.13 | 0 |
| direct-ws-1k-c32 | 57179 | 450µs | 810µs | 1.64ms | 58.55 | 0 |
| loopback-http-1m-c8 | 396.8 | 19.97ms | 24.42ms | 28.15ms | 416.10 | 0 |
| loopback-http-64k-c32 | 3692.5 | 8.33ms | 11.72ms | 19.94ms | 241.99 | 0 |
| loopback-http-hol-tiny-during-4m | 4039.3 | 1.25ms | 4.19ms | 6.26ms | 0.01 | 0 |
| loopback-http-tiny-c1 | 1888.3 | 490µs | 620µs | 1.36ms | 0.00 | 0 |
| loopback-http-tiny-c32 | 8473.2 | 3.35ms | 6.11ms | 8.57ms | 0.02 | 0 |
| loopback-ws-1k-c1 | 2398 | 400µs | 520µs | 790µs | 2.46 | 0 |
| loopback-ws-1k-c32 | 10260 | 2.43ms | 3.8ms | 5.26ms | 10.51 | 0 |
| loopback-ws-256k-c8 | 940 | 8.18ms | 11.34ms | 13.64ms | 246.54 | 0 |
| loopback-ws-handshake-c50 | - | 6.9ms | 11.1ms | 11.22ms | - | 0 |
| wan-http-1m-c8 | 13.0 | 632.68ms | 665.8ms | 696.83ms | 13.64 | 0 |
| wan-http-64k-c32 | 206.4 | 152.1ms | 156.76ms | 186.46ms | 13.53 | 0 |
| wan-http-hol-tiny-during-4m | 91.5 | 83.81ms | 104.65ms | 133.18ms | 0.00 | 0 |
| wan-http-tiny-c1 | 14.5 | 68ms | 75.4ms | 84.66ms | 0.00 | 0 |
| wan-http-tiny-c32 | 378.0 | 81.59ms | 93.15ms | 267.77ms | 0.00 | 0 |
| wan-ws-1k-c1 | 15 | 66.95ms | 74ms | 83.02ms | 0.02 | 0 |
| wan-ws-1k-c32 | 434 | 68.85ms | 80.42ms | 95.11ms | 0.44 | 0 |
| wan-ws-256k-c8 | 51 | 152.18ms | 155.89ms | 218ms | 13.42 | 0 |
| wan-ws-handshake-c50 | - | 81.41ms | 283.94ms | 284.41ms | - | 0 |

| drop scenario | result |
|---|---|
| loopback-drop-server-restart | recovery_seconds=1.05 visitor_responses=404x7 200x1  |
| wan-drop-5s-outage | recovery_seconds=2.54 visitor_responses=404x17 200x1  |
