| scenario | rate quic-phase6-websocket → quic-phase6-quic | p50 quic-phase6-websocket → quic-phase6-quic | p99 quic-phase6-websocket → quic-phase6-quic | MB/s quic-phase6-websocket → quic-phase6-quic |
|---|---|---|---|---|
| direct-http-64k-c32 | 32968.7 → 36065.3 | 930µs → 860µs | 1.84ms → 1.67ms | 2160.63 → 2363.57 |
| direct-http-tiny-c1 | 12027.3 → 12196.9 | 80µs → 70µs | 150µs → 190µs | 0.02 → 0.02 |
| direct-http-tiny-c32 | 69442.7 → 69309.0 | 420µs → 420µs | 1.22ms → 1.28ms | 0.14 → 0.14 |
| direct-ws-1k-c32 | 58578 → 71185 | 430µs → 420µs | 1.67ms → 890µs | 59.98 → 72.89 |
| loopback-http-1m-c8 | 353.7 → 84.3 | 18.37ms → 94.45ms | 223.82ms → 109.55ms | 370.86 → 88.34 |
| loopback-http-64k-c32 | 1881.0 → 951.6 | 9ms → 30.31ms | 208.44ms → 232.13ms | 123.28 → 62.37 |
| loopback-http-hol-tiny-during-4m | 3894.0 → 2508.6 | 1.24ms → 2.17ms | 7.2ms → 6.79ms | 0.01 → 0.01 |
| loopback-http-tiny-c1 | 2043.0 → 2087.4 | 470µs → 470µs | 790µs → 790µs | 0.00 → 0.00 |
| loopback-http-tiny-c32 | 5805.2 → 6100.1 | 2.98ms → 2.77ms | 10.14ms → 8.81ms | 0.01 → 0.01 |
| loopback-ws-1k-c1 | 2697 → 3028 | 360µs → 320µs | 660µs → 610µs | 2.76 → 3.10 |
| loopback-ws-1k-c32 | 8923 → 12046 | 2.33ms → 1.99ms | 4.85ms → 5.1ms | 9.14 → 12.33 |
| loopback-ws-256k-c8 | 530 → 196 | 9.27ms → 39.31ms | 215.82ms → 60.6ms | 139.03 → 51.28 |
| loopback-ws-handshake-c50 | - → - | 11.13ms → 7.53ms | 11.86ms → 1.00943s | - → - |
| wan-http-1m-c8 | 2.4 → 2.3 | 3.47674s → 3.48796s | 3.84976s → 3.5131s | 2.48 → 2.43 |
| wan-http-64k-c32 | 37.2 → 35.8 | 843.23ms → 889.25ms | 939.44ms → 1.07885s | 2.44 → 2.35 |
| wan-http-hol-tiny-during-4m | 15.7 → 66.1 | 525.68ms → 120.93ms | 666.9ms → 127.43ms | 0.00 → 0.00 |
| wan-http-tiny-c1 | 10.7 → 11.3 | 92.96ms → 87.78ms | 116.53ms → 102.99ms | 0.00 → 0.00 |
| wan-http-tiny-c32 | 309.8 → 334.3 | 96.68ms → 88.52ms | 104.87ms → 97.55ms | 0.00 → 0.00 |
| wan-ws-1k-c1 | 11 → 11 | 93.9ms → 87.81ms | 107.27ms → 100.78ms | 0.01 → 0.01 |
| wan-ws-1k-c32 | 311 → 362 | 100.9ms → 84.43ms | 114.8ms → 94.97ms | 0.32 → 0.37 |
| wan-ws-256k-c8 | 9 → 4 | 843.43ms → 1.99435s | 937.92ms → 3.81223s | 2.41 → 1.02 |
| wan-ws-handshake-c50 | - → - | 107.34ms → 92.57ms | 110.67ms → 94.07ms | - → - |

| drop scenario | quic-phase6-websocket | quic-phase6-quic |
|---|---|---|
| loopback-drop-server-restart | recovery_seconds=0.66 visitor_responses=404x5 200x1  | recovery_seconds=1.13 visitor_responses=404x8 200x1  |
| wan-drop-5s-outage | recovery_seconds=3.02 visitor_responses=404x20 200x1  | recovery_seconds=0.13 visitor_responses=200x1  |

Both runs use the same image (commit bdf05bd) on an M1 Pro, back to back.

- **wan head-of-line**: small requests during 4 MB downloads drop from 526 ms to
  121 ms p50 and from 667 ms to 127 ms p99. This is the case QUIC is for: each
  stream has its own flow and loss recovery instead of sharing one TCP
  connection.
- **wan outage**: a 5 s blackout no longer drops the tunnel. The QUIC
  connection outlives it (30 s idle timeout), so visitors see no 404s and
  recovery takes 0.13 s instead of 3 s.
- **wan small requests and websocket echo**: 5-15% lower latency.
- **wan bulk**: HTTP downloads stay at the upload cap on both transports.
- **wan-ws-256k-c8 is 2.4x slower on QUIC.** This is likely the harness:
  toxiproxy terminates TCP and shapes only payload bytes, so TCP ACKs never
  queue behind the upload cap, while udpdelay shapes every QUIC packet,
  ACKs included. The 256k echo is the only scenario that saturates both
  directions at once. A 2 MB udpdelay queue (`quic-phase6-quic-q2m`) leaves it
  unchanged, so it is not queue drops. Confirming needs shaping that treats
  both transports alike, such as netem on Linux.
- **loopback throughput is lower on QUIC** (1 MB responses 371 → 88 MB/s).
  Profiles show the time in per-packet UDP syscalls: macOS has no GSO/GRO,
  which quic-go uses on Linux. Still far above any WAN link.
- Stream windows of 2, 6 and 16 MiB made no difference on loopback; the 2 MiB
  stream / 32 MiB connection windows stay (see quicproto).
