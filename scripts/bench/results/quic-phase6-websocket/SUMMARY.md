| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| direct-http-64k-c32 | 32968.7 | 930µs | 1.19ms | 1.84ms | 2160.63 | 0 |
| direct-http-tiny-c1 | 12027.3 | 80µs | 110µs | 150µs | 0.02 | 0 |
| direct-http-tiny-c32 | 69442.7 | 420µs | 610µs | 1.22ms | 0.14 | 0 |
| direct-ws-1k-c32 | 58578 | 430µs | 910µs | 1.67ms | 59.98 | 0 |
| loopback-http-1m-c8 | 353.7 | 18.37ms | 23.43ms | 223.82ms | 370.86 | 0 |
| loopback-http-64k-c32 | 1881.0 | 9ms | 19.97ms | 208.44ms | 123.28 | 0 |
| loopback-http-hol-tiny-during-4m | 3894.0 | 1.24ms | 4.74ms | 7.2ms | 0.01 | 0 |
| loopback-http-tiny-c1 | 2043.0 | 470µs | 580µs | 790µs | 0.00 | 0 |
| loopback-http-tiny-c32 | 5805.2 | 2.98ms | 5.93ms | 10.14ms | 0.01 | 0 |
| loopback-ws-1k-c1 | 2697 | 360µs | 440µs | 660µs | 2.76 | 0 |
| loopback-ws-1k-c32 | 8923 | 2.33ms | 3.4ms | 4.85ms | 9.14 | 0 |
| loopback-ws-256k-c8 | 530 | 9.27ms | 14.06ms | 215.82ms | 139.03 | 0 |
| loopback-ws-handshake-c50 | - | 11.13ms | 11.51ms | 11.86ms | - | 0 |
| wan-http-1m-c8 | 2.4 | 3.47674s | 3.63368s | 3.84976s | 2.48 | 0 |
| wan-http-64k-c32 | 37.2 | 843.23ms | 846.94ms | 939.44ms | 2.44 | 0 |
| wan-http-hol-tiny-during-4m | 15.7 | 525.68ms | 634.78ms | 666.9ms | 0.00 | 0 |
| wan-http-tiny-c1 | 10.7 | 92.96ms | 100.24ms | 116.53ms | 0.00 | 0 |
| wan-http-tiny-c32 | 309.8 | 96.68ms | 101.53ms | 104.87ms | 0.00 | 0 |
| wan-ws-1k-c1 | 11 | 93.9ms | 102.09ms | 107.27ms | 0.01 | 0 |
| wan-ws-1k-c32 | 311 | 100.9ms | 108.26ms | 114.8ms | 0.32 | 0 |
| wan-ws-256k-c8 | 9 | 843.43ms | 848.41ms | 937.92ms | 2.41 | 0 |
| wan-ws-handshake-c50 | - | 107.34ms | 110.32ms | 110.67ms | - | 0 |

| drop scenario | result |
|---|---|
| loopback-drop-server-restart | recovery_seconds=0.66 visitor_responses=404x5 200x1  |
| wan-drop-5s-outage | recovery_seconds=3.02 visitor_responses=404x20 200x1  |
