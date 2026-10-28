// Command httpbench sends a fixed number of HTTP requests with a fixed
// concurrency and reports latency percentiles, throughput and failures.
//
// A fixed request count (rather than a duration) keeps the number of TCP
// connections bounded, so a tunnel that closes every connection cannot run the
// load generator out of ephemeral ports and turn the run into a port-exhaustion
// test.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	target := flag.String("url", "http://127.0.0.1:9100/tiny", "request url")
	host := flag.String("host", "", "Host header override")
	conns := flag.Int("c", 1, "concurrent workers")
	total := flag.Int("n", 1000, "total requests")
	timeout := flag.Duration("timeout", 30*time.Second, "per-request timeout")
	flag.Parse()

	transport := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		MaxIdleConns:        *conns,
		MaxIdleConnsPerHost: *conns,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}
	client := &http.Client{Transport: transport, Timeout: *timeout}

	var (
		next      atomic.Int64
		mu        sync.Mutex
		latencies []time.Duration
		bytesRead int64
		statuses  = map[string]int{}
		wg        sync.WaitGroup
	)

	start := time.Now()
	for range *conns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for next.Add(1) <= int64(*total) {
				request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, *target, nil)
				if err != nil {
					panic(err)
				}
				if *host != "" {
					request.Host = *host
				}
				sent := time.Now()
				response, err := client.Do(request)
				status := ""
				var read int64
				if err != nil {
					status = classify(err)
				} else {
					read, err = io.Copy(io.Discard, response.Body)
					_ = response.Body.Close()
					status = fmt.Sprint(response.StatusCode)
					if err != nil {
						status = "body:" + classify(err)
					}
				}
				elapsed := time.Since(sent)

				mu.Lock()
				statuses[status]++
				bytesRead += read
				if status == "200" {
					latencies = append(latencies, elapsed)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	failures := *total - statuses["200"]
	fmt.Printf("requests=%d conns=%d elapsed=%s failures=%d\n", *total, *conns, elapsed.Round(time.Millisecond), failures)
	fmt.Printf("latency %s\n", summarize(latencies))
	fmt.Printf("throughput %.1f req/s, %.2f MB/s\n", float64(len(latencies))/elapsed.Seconds(), float64(bytesRead)/elapsed.Seconds()/1e6)
	keys := make([]string, 0, len(statuses))
	for key := range statuses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, statuses[key]))
	}
	fmt.Printf("statuses %s\n", strings.Join(parts, " "))
	if failures > 0 {
		os.Exit(1)
	}
}

func classify(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "can't assign requested address"):
		return "err:ephemeral-ports"
	case strings.Contains(message, "connection refused"):
		return "err:refused"
	case strings.Contains(message, "connection reset"):
		return "err:reset"
	case strings.Contains(message, "EOF"):
		return "err:eof"
	case strings.Contains(message, "Timeout") || strings.Contains(message, "deadline"):
		return "err:timeout"
	default:
		return "err:other"
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
