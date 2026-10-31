| scenario | req or msg/s | p50 | p90 | p99 | MB/s | failures |
|---|---|---|---|---|---|---|
| live-http-1m-c4 | 9.5 | 364.84ms | 648.46ms | 715.14ms | 9.93 | 0 |
| live-http-64k-c8 | 52.6 | 129.54ms | 165.75ms | 468.76ms | 3.45 | 0 |
| live-http-hol-tiny-during-4m | 34.0 | 101.17ms | 134.23ms | 283.98ms | 0.00 | 0 |
| live-http-tiny-c1 | 9.8 | 99.05ms | 106.44ms | 213.03ms | 0.00 | 0 |
| live-http-tiny-c16 | 144.4 | 98.38ms | 118.43ms | 240.4ms | 0.00 | 0 |
| live-ref-healthcheck-c1 | 18.4 | 49.69ms | 58.39ms | 160.13ms | 0.00 | 0 |
| live-ws-1k-c1 | 10 | 97.54ms | 115ms | 141.46ms | 0.01 | 0 |
| live-ws-1k-c16 | 132 | 95.76ms | 112.57ms | 151.5ms | 0.13 | 0 |
| live-ws-256k-c4 | 15 | 168.56ms | 482.6ms | 581.1ms | 4.05 | 0 |
| live-ws-handshake-c20 | - | 235.67ms | 260.4ms | 261.44ms | - | 0 |

| drop scenario | result |
|---|---|
| live-drop-client-health | unhealthy_events=0 reconnects=0  |
