| scenario | rate live-pool-client-before → live-pool-client-after | p50 live-pool-client-before → live-pool-client-after | p99 live-pool-client-before → live-pool-client-after | MB/s live-pool-client-before → live-pool-client-after |
|---|---|---|---|---|
| live-http-1m-c4 | 9.5 → 9.0 | 364.84ms → 345.06ms | 715.14ms → 735.37ms | 9.93 → 9.40 |
| live-http-64k-c8 | 52.6 → 53.7 | 129.54ms → 130.74ms | 468.76ms → 446.76ms | 3.45 → 3.52 |
| live-http-hol-tiny-during-4m | 34.0 → 33.4 | 101.17ms → 104.46ms | 283.98ms → 358.31ms | 0.00 → 0.00 |
| live-http-tiny-c1 | 9.8 → 9.4 | 99.05ms → 102.23ms | 213.03ms → 219.1ms | 0.00 → 0.00 |
| live-http-tiny-c16 | 144.4 → 144.5 | 98.38ms → 99.24ms | 240.4ms → 238.95ms | 0.00 → 0.00 |
| live-ref-healthcheck-c1 | 18.4 → 14.5 | 49.69ms → 49.77ms | 160.13ms → 359.6ms | 0.00 → 0.00 |
| live-ws-1k-c1 | 10 → 10 | 97.54ms → 95.52ms | 141.46ms → 103.57ms | 0.01 → 0.01 |
| live-ws-1k-c16 | 132 → 137 | 95.76ms → 96.08ms | 151.5ms → 129.84ms | 0.13 → 0.14 |
| live-ws-256k-c4 | 15 → 15 | 168.56ms → 178.33ms | 581.1ms → 599.22ms | 4.05 → 3.99 |
| live-ws-handshake-c20 | - → - | 235.67ms → 236.36ms | 261.44ms → 257.93ms | - → - |

| drop scenario | live-pool-client-before | live-pool-client-after |
|---|---|---|
| live-drop-client-health | unhealthy_events=0 reconnects=0  | unhealthy_events=0 reconnects=0  |
