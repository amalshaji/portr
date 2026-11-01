| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| direct-http-64k-c32 | 36065.3 | 860µs | 1.16ms | 1.67ms | 2363.57 | 0 |
| direct-http-tiny-c1 | 12196.9 | 70µs | 120µs | 190µs | 0.02 | 0 |
| direct-http-tiny-c32 | 69309.0 | 420µs | 600µs | 1.28ms | 0.14 | 0 |
| direct-ws-1k-c32 | 71185 | 420µs | 550µs | 890µs | 72.89 | 0 |
| loopback-http-1m-c8 | 84.3 | 94.45ms | 101.86ms | 109.55ms | 88.34 | 0 |
| loopback-http-64k-c32 | 951.6 | 30.31ms | 36.81ms | 232.13ms | 62.37 | 0 |
| loopback-http-hol-tiny-during-4m | 2508.6 | 2.17ms | 3.84ms | 6.79ms | 0.01 | 0 |
| loopback-http-tiny-c1 | 2087.4 | 470µs | 560µs | 790µs | 0.00 | 0 |
| loopback-http-tiny-c32 | 6100.1 | 2.77ms | 5.53ms | 8.81ms | 0.01 | 0 |
| loopback-ws-1k-c1 | 3028 | 320µs | 410µs | 610µs | 3.10 | 0 |
| loopback-ws-1k-c32 | 12046 | 1.99ms | 3.02ms | 5.1ms | 12.33 | 0 |
| loopback-ws-256k-c8 | 196 | 39.31ms | 50.02ms | 60.6ms | 51.28 | 0 |
| loopback-ws-handshake-c50 | - | 7.53ms | 10.22ms | 1.00943s | - | 0 |
| wan-http-1m-c8 | 2.3 | 3.48796s | 3.51211s | 3.5131s | 2.43 | 0 |
| wan-http-64k-c32 | 35.8 | 889.25ms | 1.02069s | 1.07885s | 2.35 | 0 |
| wan-http-hol-tiny-during-4m | 66.1 | 120.93ms | 126.64ms | 127.43ms | 0.00 | 0 |
| wan-http-tiny-c1 | 11.3 | 87.78ms | 91.66ms | 102.99ms | 0.00 | 0 |
| wan-http-tiny-c32 | 334.3 | 88.52ms | 96.07ms | 97.55ms | 0.00 | 0 |
| wan-ws-1k-c1 | 11 | 87.81ms | 90.87ms | 100.78ms | 0.01 | 0 |
| wan-ws-1k-c32 | 362 | 84.43ms | 89.14ms | 94.97ms | 0.37 | 0 |
| wan-ws-256k-c8 | 4 | 1.99435s | 3.76153s | 3.81223s | 1.02 | 0 |
| wan-ws-handshake-c50 | - | 92.57ms | 93.87ms | 94.07ms | - | 0 |

| drop scenario | result |
|---|---|
| loopback-drop-server-restart | recovery_seconds=1.13 visitor_responses=404x8 200x1  |
| wan-drop-5s-outage | recovery_seconds=0.13 visitor_responses=200x1  |
