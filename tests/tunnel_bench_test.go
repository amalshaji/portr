package tests_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"
)

// The benchmarks compare transports end to end through the data-flow harness.
// Loopback has no latency or loss, so for a WAN picture run them through
// scripts/bench-transports.sh on Linux, which adds netem delay and loss.

var benchPayload = bytes.Repeat([]byte("x"), 32<<20)

func startBenchBackend(b *testing.B) *httptest.Server {
	b.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bulk" {
			_, _ = w.Write(benchPayload)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	b.Cleanup(backend.Close)
	return backend
}

func benchGet(client *http.Client, harness *tunnelHarness, path string) (int64, error) {
	request, err := http.NewRequest(http.MethodGet, harness.publicServer.URL+path, nil)
	if err != nil {
		return 0, err
	}
	request.Host = harness.tunnelHost
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	return io.Copy(io.Discard, response.Body)
}

func BenchmarkTunnelDownload(b *testing.B) {
	for _, transport := range dataFlowTransports {
		b.Run(string(transport), func(b *testing.B) {
			harness := startHTTPTunnelHarness(b, transport, startBenchBackend(b).URL)
			client := &http.Client{Transport: &http.Transport{}}
			// Idle visitor connections hold tunnel streams open, so close them
			// before the harness cleanup shuts the tunnel down.
			b.Cleanup(client.CloseIdleConnections)

			b.SetBytes(int64(len(benchPayload)))
			for b.Loop() {
				n, err := benchGet(client, harness, "/bulk")
				if err != nil || n != int64(len(benchPayload)) {
					b.Fatalf("download got %d bytes: %v", n, err)
				}
			}
		})
	}
}

// BenchmarkTunnelSmallRequestsDuringDownload measures small-request latency
// while bulk downloads share the tunnel, the case where one multiplexed
// connection can delay unrelated streams.
func BenchmarkTunnelSmallRequestsDuringDownload(b *testing.B) {
	for _, transport := range dataFlowTransports {
		b.Run(string(transport), func(b *testing.B) {
			harness := startHTTPTunnelHarness(b, transport, startBenchBackend(b).URL)

			stop := make(chan struct{})
			var downloads sync.WaitGroup
			for range 2 {
				downloads.Go(func() {
					client := &http.Client{Transport: &http.Transport{}}
					defer client.CloseIdleConnections()
					for {
						select {
						case <-stop:
							return
						default:
						}
						if _, err := benchGet(client, harness, "/bulk"); err != nil {
							return
						}
					}
				})
			}

			client := &http.Client{Transport: &http.Transport{}}
			b.Cleanup(client.CloseIdleConnections)
			var latencies []time.Duration
			for b.Loop() {
				started := time.Now()
				if _, err := benchGet(client, harness, "/small"); err != nil {
					b.Fatalf("small request: %v", err)
				}
				latencies = append(latencies, time.Since(started))
			}
			b.StopTimer()
			close(stop)
			downloads.Wait()

			slices.Sort(latencies)
			b.ReportMetric(percentileMillis(latencies, 0.50), "p50-ms")
			b.ReportMetric(percentileMillis(latencies, 0.99), "p99-ms")
		})
	}
}

func percentileMillis(sorted []time.Duration, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * p)
	return float64(sorted[index].Microseconds()) / 1000
}
