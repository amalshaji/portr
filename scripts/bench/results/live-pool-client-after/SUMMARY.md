| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| live-http-1m-c4 | 9.0 | 345.06ms | 719.06ms | 735.37ms | 9.40 | 0 |
| live-http-64k-c8 | 53.7 | 130.74ms | 153.19ms | 446.76ms | 3.52 | 0 |
| live-http-hol-tiny-during-4m | 33.4 | 104.46ms | 142.21ms | 358.31ms | 0.00 | 0 |
| live-http-tiny-c1 | 9.4 | 102.23ms | 117.14ms | 219.1ms | 0.00 | 0 |
| live-http-tiny-c16 | 144.5 | 99.24ms | 119.47ms | 238.95ms | 0.00 | 0 |
| live-ref-healthcheck-c1 | 14.5 | 49.77ms | 161.68ms | 359.6ms | 0.00 | 0 |
| live-ws-1k-c1 | 10 | 95.52ms | 99.49ms | 103.57ms | 0.01 | 0 |
| live-ws-1k-c16 | 137 | 96.08ms | 106.29ms | 129.84ms | 0.14 | 0 |
| live-ws-256k-c4 | 15 | 178.33ms | 478.34ms | 599.22ms | 3.99 | 0 |
| live-ws-handshake-c20 | - | 236.36ms | 254.12ms | 257.93ms | - | 0 |

| drop scenario | result |
|---|---|
| live-drop-client-health | unhealthy_events=0 reconnects=0  |
