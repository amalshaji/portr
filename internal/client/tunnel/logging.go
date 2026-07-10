package tunnel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/client/tui"
	"github.com/amalshaji/portr/internal/constants"
	"gorm.io/datatypes"
)

const maxCapturedBodyBytes = 1 << 20

type requestLogContextKey struct{}

type requestLogData struct {
	id        string
	request   *http.Request
	body      []byte
	startTime time.Time
	bodyMu    sync.RWMutex
}

func (d *requestLogData) setBody(body []byte) {
	d.bodyMu.Lock()
	d.body = body
	d.bodyMu.Unlock()
}

func (d *requestLogData) bodySnapshot() []byte {
	d.bodyMu.RLock()
	defer d.bodyMu.RUnlock()
	return bytes.Clone(d.body)
}

type loggingReadCloser struct {
	io.ReadCloser
	capture bytes.Buffer
	total   int64
	onDone  func([]byte, int64)
	once    sync.Once
	mu      sync.Mutex
}

func (r *loggingReadCloser) Read(payload []byte) (int, error) {
	n, err := r.ReadCloser.Read(payload)
	if n > 0 {
		r.capturePayload(payload[:n])
	}
	if err != nil {
		r.finish()
	}
	return n, err
}

func (r *loggingReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.finish()
	return err
}

func (r *loggingReadCloser) capturePayload(payload []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.total += int64(len(payload))
	remaining := maxCapturedBodyBytes - r.capture.Len()
	if remaining <= 0 {
		return
	}
	if len(payload) > remaining {
		payload = payload[:remaining]
	}
	_, _ = r.capture.Write(payload)
}

func (r *loggingReadCloser) finish() {
	r.once.Do(func() {
		r.mu.Lock()
		body := bytes.Clone(r.capture.Bytes())
		total := r.total
		r.mu.Unlock()
		r.onDone(body, total)
	})
}

func (s *Client) requestLoggingEnabled() bool {
	return s != nil && s.config.EnableRequestLogging
}

func redactHeaderValues(headers http.Header, redactNames []string) map[string][]string {
	if len(redactNames) == 0 {
		redactNames = config.DefaultRedactHeaders
	}

	redactSet := make(map[string]struct{}, len(redactNames))
	for _, name := range redactNames {
		redactSet[strings.ToLower(name)] = struct{}{}
	}

	redacted := make(map[string][]string, len(headers))
	for key, values := range headers {
		copiedValues := make([]string, len(values))
		if _, ok := redactSet[strings.ToLower(key)]; ok {
			for i := range copiedValues {
				copiedValues[i] = "[redacted]"
			}
		} else {
			copy(copiedValues, values)
		}
		redacted[key] = copiedValues
	}

	return redacted
}

func (s *Client) logHttpRequest(
	id string,
	request *http.Request,
	requestBody []byte,
	response *http.Response,
	responseBody []byte,
	durationMs int64,
) {
	if !s.requestLoggingEnabled() {
		return
	}

	if request.Header.Get("X-Portr-Ping-Request") == "true" {
		return
	}

	var replayedRequestId string
	var isReplayedRequest bool
	if replayedHeader := request.Header.Values("X-Portr-Replayed-Request-Id"); len(replayedHeader) > 0 {
		replayedRequestId = replayedHeader[0]
		isReplayedRequest = true
	}

	requestHeaders := redactHeaderValues(request.Header, s.config.RedactHeaders)
	delete(requestHeaders, "X-Portr-Replayed-Request-Id")

	requestHeadersBytes, err := json.Marshal(requestHeaders)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to marshal request headers", err)
		}
		return
	}

	responseHeaders := redactHeaderValues(response.Header, s.config.RedactHeaders)

	responseHeadersBytes, err := json.Marshal(responseHeaders)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to marshal request headers", err)
		}
		return
	}

	req := db.Request{
		ID:                 id,
		Host:               request.Host,
		Url:                request.URL.String(),
		Subdomain:          s.config.Tunnel.Subdomain,
		Localport:          s.config.Tunnel.Port,
		Method:             request.Method,
		Headers:            datatypes.JSON(requestHeadersBytes),
		Body:               requestBody,
		ResponseHeaders:    datatypes.JSON(responseHeadersBytes),
		ResponseBody:       responseBody,
		ResponseStatusCode: response.StatusCode,
		LoggedAt:           time.Now().UTC(),
		IsReplayed:         isReplayedRequest,
		ParentID:           replayedRequestId,
		DurationMs:         durationMs,
		BytesIn:            int64(len(requestBody)),
		BytesOut:           int64(len(responseBody)),
		Protocol:           request.Proto,
	}
	result := s.db.Conn.Create(&req)
	if result.Error != nil {
		if s.config.Debug {
			s.logDebug("Failed to log request", result.Error)
		}
		return
	}

	tunnelName := s.config.Tunnel.Name
	if tunnelName == "" {
		if s.config.Tunnel.Type == constants.Stub && s.config.Tunnel.Subdomain != "" {
			tunnelName = s.config.Tunnel.Subdomain
		} else {
			tunnelName = fmt.Sprintf("%d", s.config.Tunnel.Port)
		}
	}

	if s.tui != nil {
		s.tui.Send(tui.AddLogMsg{
			Time:   req.LoggedAt.Local().Format("15:04:05"),
			Name:   tunnelName,
			Method: req.Method,
			Status: req.ResponseStatusCode,
			URL:    req.Url,
		})
	} else if !s.config.DisableTerminalLogs {
		fmt.Printf("[%s] %s %s → %d\n",
			req.LoggedAt.Local().Format("15:04:05"),
			req.Method,
			req.Url,
			req.ResponseStatusCode)
	}
}
