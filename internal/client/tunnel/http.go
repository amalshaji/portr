package tunnel

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/amalshaji/portr/internal/utils"
	"github.com/charmbracelet/log"
	"github.com/oklog/ulid/v2"
)

func (s *Client) httpTunnel(src net.Conn, localEndpoint string) {
	if s.config.EnableHttpReverseProxy {
		s.httpTunnelReverseProxy(src, localEndpoint)
		return
	}

	s.httpTunnelLegacy(src, localEndpoint)
}

func (s *Client) httpTunnelReverseProxy(src net.Conn, localEndpoint string) {
	defer src.Close()

	target := &url.URL{
		Scheme: "http",
		Host:   localEndpoint,
	}

	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, localEndpoint)
		},
	}
	defer transport.CloseIdleConnections()

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport

	defaultDirector := proxy.Director
	proxy.Director = func(request *http.Request) {
		host := request.Host
		defaultDirector(request)
		request.Host = host
	}

	proxy.ModifyResponse = func(response *http.Response) error {
		if response.StatusCode == http.StatusSwitchingProtocols {
			return nil
		}

		if strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
			return nil
		}

		logData, ok := response.Request.Context().Value(requestLogContextKey{}).(*requestLogData)
		if !ok || logData == nil || logData.request == nil {
			return nil
		}

		if !s.requestLoggingEnabled() {
			return nil
		}

		response.Body = &loggingReadCloser{
			ReadCloser: response.Body,
			onDone: func(responseBody []byte, _ int64) {
				durationMs := time.Since(logData.startTime).Milliseconds()
				s.logHttpRequest(logData.id, logData.request, logData.body, response, responseBody, durationMs)
			},
		}
		return nil
	}

	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		if s.config.Debug {
			s.logDebug("HTTP reverse proxy failed", err)
		}

		htmlContent := utils.LocalServerNotOnline(localEndpoint)
		writer.Header().Set("X-Portr-Error", "true")
		writer.Header().Set("X-Portr-Error-Reason", "local-server-not-online")
		writer.Header().Set("Content-Type", "text/html")
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte(htmlContent))
	}

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Portr-Ping-Request") == "true" {
			writer.WriteHeader(http.StatusOK)
			return
		}

		if isWebSocketUpgrade(request) {
			hijacker, ok := writer.(http.Hijacker)
			if !ok {
				http.Error(writer, "websocket proxy unsupported", http.StatusInternalServerError)
				return
			}

			conn, rw, err := hijacker.Hijack()
			if err != nil {
				if s.config.Debug {
					s.logDebug("Failed to hijack websocket connection", err)
				}
				return
			}
			defer conn.Close()

			if err := s.handleWebSocketRequest(conn, rw.Reader, rw.Writer, request, localEndpoint); err != nil && s.config.Debug {
				s.logDebug("Failed to proxy websocket request", err)
			}
			return
		}

		requestBody, err := io.ReadAll(request.Body)
		if err != nil {
			if s.config.Debug {
				s.logDebug("Failed to read request body for reverse proxy logging", err)
			}
			http.Error(writer, "Bad Request", http.StatusBadRequest)
			return
		}
		request.Body.Close()
		request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

		requestForLog := request.Clone(context.Background())
		requestForLog.Header = request.Header.Clone()
		requestForLog.Host = request.Host
		if request.URL != nil {
			clonedURL := *request.URL
			requestForLog.URL = &clonedURL
		}

		logCtx := context.WithValue(request.Context(), requestLogContextKey{}, &requestLogData{
			id:        ulid.Make().String(),
			request:   requestForLog,
			body:      requestBody,
			startTime: time.Now(),
		})

		proxy.ServeHTTP(writer, request.WithContext(logCtx))
	})

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}

	listener := &singleConnListener{conn: src}
	err := server.Serve(listener)
	if err != nil && err != net.ErrClosed {
		if s.config.Debug {
			s.logDebug("Reverse proxy tunnel closed with error", err)
		}
	}
}

func (s *Client) httpTunnelLegacy(src net.Conn, localEndpoint string) {
	var dst net.Conn

	defer src.Close()

	srcReader := bufio.NewReader(src)
	srcWriter := bufio.NewWriter(src)

	request, err := http.ReadRequest(srcReader)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to read request", err)
		}
		return
	}

	if request.Header.Get("X-Portr-Ping-Request") == "true" {
		response := &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewBufferString("")),
		}

		err = response.Write(srcWriter)
		if err != nil {
			if s.config.Debug {
				log.Error("Failed to write health check response", "error", err)
			}
			return
		}
		srcWriter.Flush()
		return
	}

	if isWebSocketUpgrade(request) {
		if err := s.handleWebSocketRequest(src, srcReader, srcWriter, request, localEndpoint); err != nil && s.config.Debug {
			s.logDebug("Failed to proxy websocket request", err)
		}
		return
	}

	// Connect to the local endpoint only after filtering internal health checks.
	dst, err = net.Dial("tcp", localEndpoint)
	if err != nil {
		// serve local html if the local server is not available
		// change this to a beautiful template
		htmlContent := utils.LocalServerNotOnline(localEndpoint)
		fmt.Fprintf(src, "HTTP/1.1 503 Service Unavailable\r\n")
		fmt.Fprintf(src, "Content-Length: %d\r\n", len(htmlContent))
		fmt.Fprintf(src, "Content-Type: text/html\r\n")
		fmt.Fprintf(src, "X-Portr-Error: true\r\n")
		fmt.Fprintf(src, "X-Portr-Error-Reason: local-server-not-online\r\n\r\n")
		fmt.Fprint(src, htmlContent)
		return
	}
	defer dst.Close()

	dstReader := bufio.NewReader(dst)
	dstWriter := bufio.NewWriter(dst)

	// read and replace request body
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to read request body", err)
		}
		return
	}
	defer request.Body.Close()
	request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	err = request.Write(dstWriter)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to tunnel request to local", err)
		}
		return
	}
	dstWriter.Flush()

	response, err := http.ReadResponse(dstReader, request)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to read response", err)
		}
		return
	}

	// Handle WebSocket upgrades and SSE streams with TCP tunneling
	if response.StatusCode == http.StatusSwitchingProtocols {
		// WebSocket upgrade - write response headers and switch to TCP tunneling
		err = response.Write(srcWriter)
		if err != nil {
			if s.config.Debug {
				s.logDebug("Failed to write WebSocket upgrade response", err)
			}
			return
		}
		srcWriter.Flush()

		// Drain any bytes already buffered post-handshake to avoid loss when switching to raw TCP
		if n := dstReader.Buffered(); n > 0 {
			buf := make([]byte, n)
			if _, err := io.ReadFull(dstReader, buf); err == nil {
				if _, err := srcWriter.Write(buf); err != nil {
					if s.config.Debug {
						s.logDebug("Failed to flush buffered server bytes on WS upgrade", err)
					}
					return
				}
				srcWriter.Flush()
			}
		}

		if n := srcReader.Buffered(); n > 0 {
			buf := make([]byte, n)
			if _, err := io.ReadFull(srcReader, buf); err == nil {
				if _, err := dstWriter.Write(buf); err != nil {
					if s.config.Debug {
						s.logDebug("Failed to flush buffered client bytes on WS upgrade", err)
					}
					return
				}
				dstWriter.Flush()
			}
		}

		s.tcpTunnel(src, dst)
		return
	}

	// Check for SSE (Server-Sent Events) streams
	contentType := response.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		// Ensure SSE response body is closed when streaming finishes or on error
		defer response.Body.Close()

		// SSE stream - copy the response body in real-time without buffering
		// Write status line and headers first
		fmt.Fprintf(srcWriter, "%s %s\r\n", response.Proto, response.Status)

		// Write headers, excluding Content-Length and Transfer-Encoding
		// as we'll be streaming the body directly
		for key, values := range response.Header {
			if key == "Content-Length" || key == "Transfer-Encoding" {
				continue
			}
			for _, value := range values {
				fmt.Fprintf(srcWriter, "%s: %s\r\n", key, value)
			}
		}

		// Empty line to end headers
		fmt.Fprintf(srcWriter, "\r\n")
		srcWriter.Flush()

		// Stream the body with immediate flushing for real-time delivery
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := response.Body.Read(buf)
			if n > 0 {
				_, writeErr := srcWriter.Write(buf[:n])
				if writeErr != nil {
					if s.config.Debug {
						s.logDebug("Failed to write SSE data", writeErr)
					}
					return
				}
				// Flush immediately to ensure real-time streaming
				srcWriter.Flush()
			}
			if err != nil {
				if err != io.EOF && s.config.Debug {
					s.logDebug("SSE stream ended", err)
				}
				break
			}
		}
		return
	}

	// read and replace response body for regular HTTP responses
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to read response body", err)
		}
		return
	}
	defer response.Body.Close()
	response.Body = io.NopCloser(bytes.NewBuffer(responseBody))

	err = response.Write(srcWriter)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to write response to remote", err)
		}
		return
	}
	srcWriter.Flush()

	s.logHttpRequest(ulid.Make().String(), request, requestBody, response, responseBody, 0)
}
