| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| direct-http-64k-c32 | 34497.5 | 900µs | 1.1ms | 1.7ms | 2260.83 | 0 |
| direct-http-tiny-c1 | 13113.2 | 70µs | 100µs | 140µs | 0.03 | 0 |
| direct-http-tiny-c32 | 68951.2 | 420µs | 630µs | 1.28ms | 0.14 | 0 |
| direct-ws-1k-c32 | 58534 | 440µs | 750µs | 1.55ms | 59.94 | 0 |
| loopback-http-1m-c8 | 52.1 | 153.77ms | 177.65ms | 200.35ms | 54.58 | 0 |
| loopback-http-64k-c32 | 815.8 | 38.33ms | 45.43ms | 51.59ms | 53.46 | 0 |
| loopback-http-hol-tiny-during-4m | 315.9 | 3.06ms | 76.71ms | 95.46ms | 0.00 | 0 |
| loopback-http-tiny-c1 | 487.3 | 1.59ms | 3.04ms | 7.3ms | 0.00 | 0 |
| loopback-http-tiny-c32 | 2631.7 | 5.77ms | 8.51ms | 24.18ms | 0.01 | 0 |
| loopback-ws-1k-c1 | 2148 | 410µs | 560µs | 1.17ms | 2.20 | 0 |
| loopback-ws-1k-c32 | 8269 | 3.15ms | 5.41ms | 8.6ms | 8.47 | 0 |
| loopback-ws-256k-c8 | 177 | 44.7ms | 55.18ms | 61.7ms | 46.36 | 0 |
| loopback-ws-handshake-c50 | - | 13.36ms | 17.73ms | 18.36ms | - | 0 |
| wan-http-1m-c8 | 1.8 | 4.49071s | 4.51609s | 5.019s | 1.86 | 0 |
| wan-http-64k-c32 | 28.1 | 1.13025s | 1.13437s | 1.22569s | 1.84 | 0 |
| wan-http-hol-tiny-during-4m | 10.3 | 111.53ms | 2.24777s | 2.26169s | 0.00 | 0 |
| wan-http-tiny-c1 | 9.9 | 99.7ms | 108.7ms | 117.57ms | 0.00 | 0 |
| wan-http-tiny-c32 | 268.7 | 108.99ms | 121.83ms | 293.7ms | 0.00 | 0 |
| wan-ws-1k-c1 | 10 | 97.94ms | 105.49ms | 121.07ms | 0.01 | 0 |
| wan-ws-1k-c32 | 288 | 104.83ms | 116.81ms | 124.88ms | 0.30 | 0 |
| wan-ws-256k-c8 | 7 | 1.12468s | 1.13138s | 1.25393s | 1.82 | 0 |
| wan-ws-handshake-c50 | - | 331.96ms | 335ms | 336.5ms | - | 0 |

| drop scenario | result |
|---|---|
| loopback-drop-server-restart | recovery_seconds=1.03 visitor_responses=404x7 200x1  |
| wan-drop-5s-outage | recovery_seconds=3.26 visitor_responses=404x21 200x1  |
