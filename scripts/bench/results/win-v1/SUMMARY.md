| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| wan-http-1m-c8 | 9.6 | 827.06ms | 837.51ms | 868.14ms | 10.10 | 0 |
| wan-http-64k-c32 | 155.7 | 203.04ms | 207.13ms | 223.04ms | 10.20 | 0 |
| wan-http-hol-tiny-during-4m | 45.9 | 76.74ms | 408.5ms | 414.2ms | 0.00 | 0 |
| wan-http-tiny-c1 | 15.1 | 65.51ms | 72.96ms | 81.36ms | 0.00 | 0 |
| wan-http-tiny-c32 | 297.5 | 69.87ms | 83.25ms | 1.08044s | 0.00 | 0 |
| wan-ws-1k-c1 | 16 | 61.98ms | 67.89ms | 77.95ms | 0.02 | 0 |
| wan-ws-1k-c32 | 456 | 67.02ms | 76.68ms | 91.56ms | 0.47 | 0 |
| wan-ws-256k-c8 | 39 | 201.95ms | 205.4ms | 250.18ms | 10.21 | 0 |
| wan-ws-handshake-c50 | - | 74.46ms | 75.54ms | 75.96ms | - | 0 |

| drop scenario | result |
|---|---|
| wan-drop-5s-outage | recovery_seconds=2.76 visitor_responses=404x20 200x1  |
