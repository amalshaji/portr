| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| loopback-http-1m-c8 | 83.9 | 94.07ms | 103.49ms | 114.31ms | 87.98 | 0 |
| loopback-http-64k-c32 | 912.8 | 32.11ms | 39.53ms | 231.42ms | 59.82 | 0 |
| loopback-http-hol-tiny-during-4m | 2935.8 | 2.02ms | 3.33ms | 7.73ms | 0.01 | 0 |
| loopback-http-tiny-c1 | 1999.8 | 490µs | 590µs | 860µs | 0.00 | 0 |
| loopback-http-tiny-c32 | 6830.6 | 2.05ms | 4.66ms | 10.81ms | 0.01 | 0 |
| loopback-ws-1k-c1 | 3041 | 320µs | 400µs | 620µs | 3.11 | 0 |
| loopback-ws-1k-c32 | 12314 | 1.81ms | 2.61ms | 3.45ms | 12.61 | 0 |
| loopback-ws-256k-c8 | 197 | 40.47ms | 49.4ms | 55.47ms | 51.58 | 0 |
| loopback-ws-handshake-c50 | - | 6.04ms | 6.52ms | 7.32ms | - | 0 |
| wan-http-1m-c8 | 2.3 | 3.49597s | 3.65892s | 3.66413s | 2.43 | 0 |
| wan-http-64k-c32 | 35.8 | 920.04ms | 1.0719s | 1.0764s | 2.34 | 0 |
| wan-http-hol-tiny-during-4m | 24.4 | 346.86ms | 352.73ms | 354.32ms | 0.00 | 0 |
| wan-http-tiny-c1 | 11.3 | 88.27ms | 91.73ms | 106.93ms | 0.00 | 0 |
| wan-http-tiny-c32 | 332.8 | 89.88ms | 95.78ms | 99.71ms | 0.00 | 0 |
| wan-ws-1k-c1 | 11 | 87.64ms | 91.12ms | 101.26ms | 0.01 | 0 |
| wan-ws-1k-c32 | 364 | 83.93ms | 88.36ms | 101.3ms | 0.37 | 0 |
| wan-ws-256k-c8 | 4 | 1.79714s | 3.47456s | 3.72669s | 1.00 | 0 |
| wan-ws-handshake-c50 | - | 94.55ms | 95.72ms | 100.83ms | - | 0 |

| drop scenario | result |
|---|---|
| loopback-drop-server-restart | recovery_seconds=0.98 visitor_responses=404x7 200x1  |
| wan-drop-5s-outage | recovery_seconds=0.13 visitor_responses=200x1  |
