| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| wan-http-1m-c8 | 13.1 | 607.86ms | 630.53ms | 663.13ms | 13.71 | 0 |
| wan-http-64k-c32 | 207.8 | 152.05ms | 155.47ms | 173.49ms | 13.62 | 0 |
| wan-http-hol-tiny-during-4m | 86.2 | 90.49ms | 113.08ms | 147.69ms | 0.00 | 0 |
| wan-http-tiny-c1 | 15.4 | 64.76ms | 69.68ms | 80.96ms | 0.00 | 0 |
| wan-http-tiny-c32 | 284.9 | 70.51ms | 90.14ms | 1.07278s | 0.00 | 0 |
| wan-ws-1k-c1 | 16 | 62.08ms | 68.96ms | 79.37ms | 0.02 | 0 |
| wan-ws-1k-c32 | 452 | 65.54ms | 75.79ms | 86.14ms | 0.46 | 0 |
| wan-ws-256k-c8 | 51 | 152.13ms | 155.11ms | 207.49ms | 13.45 | 0 |
| wan-ws-handshake-c50 | - | 272.05ms | 278.33ms | 278.47ms | - | 0 |

| drop scenario | result |
|---|---|
| wan-drop-5s-outage | recovery_seconds=2.63 visitor_responses=404x18 200x1  |
