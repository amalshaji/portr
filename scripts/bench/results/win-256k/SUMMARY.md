| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| wan-http-1m-c8 | 12.7 | 642.63ms | 700.76ms | 771.61ms | 13.35 | 0 |
| wan-http-64k-c32 | 206.6 | 152.17ms | 155.72ms | 215.34ms | 13.54 | 0 |
| wan-http-hol-tiny-during-4m | 100.5 | 75.78ms | 95.75ms | 141.13ms | 0.00 | 0 |
| wan-http-tiny-c1 | 15.1 | 65.57ms | 71.4ms | 82.88ms | 0.00 | 0 |
| wan-http-tiny-c32 | 300.7 | 66.86ms | 78.77ms | 1.07376s | 0.00 | 0 |
| wan-ws-1k-c1 | 16 | 60.84ms | 66.33ms | 76.75ms | 0.02 | 0 |
| wan-ws-1k-c32 | 466 | 65.48ms | 73.5ms | 84.48ms | 0.48 | 0 |
| wan-ws-256k-c8 | 29 | 269.02ms | 289.2ms | 309.65ms | 7.68 | 0 |
| wan-ws-handshake-c50 | - | 281.3ms | 282.05ms | 1.07006s | - | 0 |

| drop scenario | result |
|---|---|
| wan-drop-5s-outage | recovery_seconds=2.65 visitor_responses=404x19 200x1  |
