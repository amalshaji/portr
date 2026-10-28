| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| wan-http-1m-c8 | 11.7 | 682.29ms | 692.7ms | 698.19ms | 12.28 | 0 |
| wan-http-64k-c32 | 206.9 | 152.21ms | 156.07ms | 314.68ms | 13.56 | 0 |
| wan-http-hol-tiny-during-4m | 61.9 | 116.33ms | 201.91ms | 263.88ms | 0.00 | 0 |
| wan-http-tiny-c1 | 14.8 | 66.56ms | 74.72ms | 86.08ms | 0.00 | 0 |
| wan-http-tiny-c32 | 344.7 | 68.58ms | 84.22ms | 1.06038s | 0.00 | 0 |
| wan-ws-1k-c1 | 16 | 62.33ms | 68.23ms | 77.17ms | 0.02 | 0 |
| wan-ws-1k-c32 | 459 | 65.26ms | 73.68ms | 81.99ms | 0.47 | 0 |
| wan-ws-256k-c8 | 51 | 151.97ms | 154.8ms | 221.59ms | 13.47 | 0 |
| wan-ws-handshake-c50 | - | 86.53ms | 272.32ms | 273.19ms | - | 0 |

| drop scenario | result |
|---|---|
| wan-drop-5s-outage | recovery_seconds=2.85 visitor_responses=404x20 200x1  |
