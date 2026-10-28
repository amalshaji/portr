| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| live-http-1m-c4 | 7.6 | 434.87ms | 674.4ms | 761.95ms | 7.97 | 0 |
| live-http-64k-c8 | 48.8 | 141.79ms | 178.56ms | 470.4ms | 3.20 | 0 |
| live-http-hol-tiny-during-4m | 33.3 | 111.71ms | 137.06ms | 230.51ms | 0.00 | 0 |
| live-http-tiny-c1 | 8.5 | 116.18ms | 132.39ms | 221.6ms | 0.00 | 0 |
| live-http-tiny-c16 | 128.0 | 108.89ms | 151.31ms | 277.77ms | 0.00 | 0 |
| live-ref-healthcheck-c1 | 18.8 | 48.47ms | 60.39ms | 158.5ms | 0.00 | 0 |
| live-ws-1k-c1 | 9 | 101.38ms | 117.59ms | 139.72ms | 0.01 | 0 |
| live-ws-1k-c16 | 130 | 105.13ms | 131.85ms | 164.92ms | 0.13 | 0 |
| live-ws-256k-c4 | 9 | 382.34ms | 610.89ms | 681.21ms | 2.44 | 0 |
| live-ws-handshake-c20 | - | 223.17ms | 235.73ms | 239.78ms | - | 0 |

| drop scenario | result |
|---|---|
| live-drop-client-health | unhealthy_events=0 reconnects=0  |
