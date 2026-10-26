// Command wsbench measures websocket handshake latency and echo round trips
// against the bench origin's /ws endpoint, directly or through a tunnel.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"slices"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	target := flag.String("url", "ws://127.0.0.1:9100/ws", "websocket url")
	host := flag.String("host", "", "Host header override")
	conns := flag.Int("c", 10, "concurrent connections")
	messages := flag.Int("n", 200, "messages per connection")
	size := flag.Int("size", 1024, "message size in bytes")
	flag.Parse()

	payload := bytes.Repeat([]byte("w"), *size)
	var (
		mu         sync.Mutex
		handshakes []time.Duration
		rtts       []time.Duration
		failures   int
		wg         sync.WaitGroup
	)

	start := time.Now()
	for range *conns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			config, err := websocket.NewConfig(*target, "http://localhost/")
			if err != nil {
				log.Fatal(err)
			}
			dialAddr := config.Location.Host
			if *host != "" {
				config.Location.Host = *host
			}
			dialStart := time.Now()
			conn, err := dial(config, dialAddr)
			if err != nil {
				mu.Lock()
				failures++
				mu.Unlock()
				return
			}
			defer conn.Close()
			handshake := time.Since(dialStart)
			conn.PayloadType = websocket.BinaryFrame

			local := make([]time.Duration, 0, *messages)
			for range *messages {
				sent := time.Now()
				if err := websocket.Message.Send(conn, payload); err != nil {
					break
				}
				var reply []byte
				if err := websocket.Message.Receive(conn, &reply); err != nil || len(reply) != len(payload) {
					break
				}
				local = append(local, time.Since(sent))
			}

			mu.Lock()
			handshakes = append(handshakes, handshake)
			rtts = append(rtts, local...)
			if len(local) < *messages {
				failures++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("conns=%d messages=%d size=%dB elapsed=%s failures=%d\n", *conns, *messages, *size, elapsed.Round(time.Millisecond), failures)
	fmt.Printf("handshake %s\n", summarize(handshakes))
	fmt.Printf("echo rtt  %s\n", summarize(rtts))
	if elapsed > 0 {
		fmt.Printf("throughput %.0f msg/s, %.2f MB/s each way\n",
			float64(len(rtts))/elapsed.Seconds(),
			float64(len(rtts)*(*size))/elapsed.Seconds()/1e6)
	}
	if failures > 0 {
		os.Exit(1)
	}
}

func summarize(samples []time.Duration) string {
	if len(samples) == 0 {
		return "no samples"
	}
	slices.Sort(samples)
	pick := func(p float64) time.Duration {
		return samples[min(len(samples)-1, int(float64(len(samples))*p))].Round(10 * time.Microsecond)
	}
	return fmt.Sprintf("p50=%s p90=%s p99=%s max=%s n=%d", pick(0.50), pick(0.90), pick(0.99), samples[len(samples)-1].Round(10*time.Microsecond), len(samples))
}

// dial connects to addr but handshakes with config's Host, so tunnel
// subdomains work without DNS for *.localhost.
func dial(config *websocket.Config, addr string) (*websocket.Conn, error) {
	raw, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	conn, err := websocket.NewClient(config, raw)
	if err != nil {
		_ = raw.Close()
		return nil, err
	}
	return conn, nil
}
