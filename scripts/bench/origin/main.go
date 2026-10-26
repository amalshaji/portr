// Command origin is a minimal local HTTP service used as the tunnel target for
// benchmarks, so measured latency reflects the tunnel rather than the app.
package main

import (
	"bytes"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9100", "listen address")
	flag.Parse()

	payload := bytes.Repeat([]byte("portr-bench-"), 1<<20)

	mux := http.NewServeMux()
	mux.HandleFunc("/tiny", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	// /bytes?n=65536 returns n bytes of compressible text.
	mux.HandleFunc("/bytes", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if n <= 0 || n > len(payload) {
			n = 1024
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", strconv.Itoa(n))
		_, _ = w.Write(payload[:n])
	})
	// /delay?ms=50 simulates a slow handler.
	mux.HandleFunc("/delay", func(w http.ResponseWriter, r *http.Request) {
		ms, _ := strconv.Atoi(r.URL.Query().Get("ms"))
		time.Sleep(time.Duration(ms) * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	})
	// /ws echoes every message back unchanged.
	mux.Handle("/ws", websocket.Handler(func(conn *websocket.Conn) {
		conn.PayloadType = websocket.BinaryFrame
		for {
			var message []byte
			if err := websocket.Message.Receive(conn, &message); err != nil {
				return
			}
			if err := websocket.Message.Send(conn, message); err != nil {
				return
			}
		}
	}))

	log.Printf("origin listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
