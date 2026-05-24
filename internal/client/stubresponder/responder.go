package stubresponder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	placeholderPattern           = regexp.MustCompile(`\{\{\s*([^{}|\s]+)\s*(?:\|\s*(int|bool|float)\s*)?\}\}`)
	quotedPlaceholderPattern     = regexp.MustCompile(`"\{\{\s*([^{}|\s]+)\s*\}\}"`)
	quotedCastPlaceholderPattern = regexp.MustCompile(`"\{\{\s*([^{}|\s]+)\s*\|\s*(int|bool|float)\s*\}\}"`)
)

type Route struct {
	Subdomain        string
	ResponseFormat   string
	ResponseTemplate string
}

type Responder struct {
	server   *http.Server
	listener net.Listener
	mu       sync.RWMutex
	routes   map[string]Route
}

func New() *Responder {
	responder := &Responder{
		routes: make(map[string]Route),
	}
	responder.server = &http.Server{
		Handler:           responder,
		ReadHeaderTimeout: 15 * time.Second,
	}
	return responder
}

func (r *Responder) Start() error {
	if r.listener != nil {
		return nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start stub responder: %w", err)
	}
	r.listener = listener

	go func() {
		if err := r.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			// The tunnel worker health checks surface unreachable local endpoints;
			// there is no caller to return this asynchronous server error to.
		}
	}()

	return nil
}

func (r *Responder) Addr() string {
	if r.listener == nil {
		return ""
	}
	return r.listener.Addr().String()
}

func (r *Responder) Port() int {
	if r.listener == nil {
		return 0
	}
	if addr, ok := r.listener.Addr().(*net.TCPAddr); ok {
		return addr.Port
	}
	return 0
}

func (r *Responder) Register(route Route) error {
	subdomain := strings.ToLower(strings.TrimSpace(route.Subdomain))
	if subdomain == "" {
		return fmt.Errorf("subdomain is required")
	}
	if strings.TrimSpace(route.ResponseFormat) == "" {
		return fmt.Errorf("response format is required")
	}
	if strings.TrimSpace(route.ResponseTemplate) == "" {
		return fmt.Errorf("response template is required")
	}

	route.Subdomain = subdomain
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.routes[subdomain]; exists {
		return fmt.Errorf("stub route for subdomain %q already exists", subdomain)
	}
	r.routes[subdomain] = route
	return nil
}

func (r *Responder) Unregister(subdomain string) {
	subdomain = strings.ToLower(strings.TrimSpace(subdomain))
	if subdomain == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.routes, subdomain)
}

func (r *Responder) Shutdown(ctx context.Context) error {
	if r.server == nil {
		return nil
	}
	return r.server.Shutdown(ctx)
}

func (r *Responder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("X-Portr-Ping-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	route, ok := r.routeForHost(req.Host)
	if !ok {
		http.NotFound(w, req)
		return
	}

	bodyValues, err := requestBodyValues(req)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	rendered := renderTemplate(route.ResponseTemplate, req.URL.Query(), bodyValues, isJSONResponse(route.ResponseFormat))
	w.Header().Set("Content-Type", route.ResponseFormat)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(rendered))
}

func (r *Responder) routeForHost(host string) (Route, bool) {
	hostname := strings.ToLower(strings.TrimSpace(stripPort(host)))

	r.mu.RLock()
	defer r.mu.RUnlock()

	if route, ok := r.routes[hostname]; ok {
		return route, true
	}
	for subdomain, route := range r.routes {
		if strings.HasPrefix(hostname, subdomain+".") {
			return route, true
		}
	}
	if len(r.routes) == 1 {
		for _, route := range r.routes {
			return route, true
		}
	}
	return Route{}, false
}

func stripPort(host string) string {
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		return hostname
	}
	if strings.Count(host, ":") == 1 {
		if idx := strings.LastIndex(host, ":"); idx > -1 {
			return host[:idx]
		}
	}
	return host
}

func requestBodyValues(req *http.Request) (map[string]string, error) {
	if req.Body == nil {
		return nil, nil
	}
	if req.ContentLength == 0 && len(req.TransferEncoding) == 0 {
		return nil, nil
	}

	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		return nil, nil
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("parse content type: %w", err)
	}

	switch mediaType {
	case "application/json":
		return jsonBodyValues(req.Body)
	case "application/x-www-form-urlencoded":
		if err := req.ParseForm(); err != nil {
			return nil, err
		}
		return formValues(req.PostForm), nil
	case "multipart/form-data":
		if err := req.ParseMultipartForm(32 << 20); err != nil {
			return nil, err
		}
		return formValues(req.PostForm), nil
	default:
		return nil, nil
	}
}

func formValues(form url.Values) map[string]string {
	values := make(map[string]string)
	for key, vals := range form {
		if len(vals) > 0 {
			values[key] = vals[0]
		}
	}
	return values
}

func jsonBodyValues(body io.Reader) (map[string]string, error) {
	var raw map[string]any
	decoder := json.NewDecoder(body)
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, fmt.Errorf("json body must contain one object")
	}

	values := make(map[string]string, len(raw))
	for key, value := range raw {
		values[key] = stringify(value)
	}
	return values, nil
}

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(encoded)
	}
}

func renderTemplate(template string, query url.Values, body map[string]string, jsonResponse bool) string {
	rendered := quotedCastPlaceholderPattern.ReplaceAllStringFunc(template, func(match string) string {
		parts := quotedCastPlaceholderPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return `""`
		}

		value := requestValue(parts[1], query, body)
		if casted, ok := castValue(value, parts[2]); ok {
			return casted
		}

		encoded, err := json.Marshal(value)
		if err != nil {
			return strconv.Quote(value)
		}
		return string(encoded)
	})

	if jsonResponse {
		rendered = quotedPlaceholderPattern.ReplaceAllStringFunc(rendered, func(match string) string {
			parts := quotedPlaceholderPattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return `""`
			}

			value := requestValue(parts[1], query, body)
			encoded, err := json.Marshal(value)
			if err != nil {
				return strconv.Quote(value)
			}
			return string(encoded)
		})
	}

	return placeholderPattern.ReplaceAllStringFunc(rendered, func(match string) string {
		parts := placeholderPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return ""
		}

		value := requestValue(parts[1], query, body)
		if parts[2] != "" {
			if casted, ok := castValue(value, parts[2]); ok {
				return casted
			}
			if jsonResponse {
				encoded, err := json.Marshal(value)
				if err != nil {
					return strconv.Quote(value)
				}
				return string(encoded)
			}
		}
		if jsonResponse {
			encoded, err := json.Marshal(value)
			if err != nil {
				return strconv.Quote(value)
			}
			return string(encoded)
		}
		return value
	})
}

func isJSONResponse(responseFormat string) bool {
	mediaType, _, err := mime.ParseMediaType(responseFormat)
	if err != nil {
		mediaType = responseFormat
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func requestValue(key string, query url.Values, body map[string]string) string {
	if values, ok := query[key]; ok && len(values) > 0 {
		return values[0]
	}
	if body != nil {
		if value, ok := body[key]; ok {
			return value
		}
	}
	return ""
}

func castValue(value string, format string) (string, bool) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(format) {
	case "int":
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return "", false
		}
		return strconv.FormatInt(parsed, 10), true
	case "bool":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return "", false
		}
		return strconv.FormatBool(parsed), true
	case "float":
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "", false
		}
		return strconv.FormatFloat(parsed, 'g', -1, 64), true
	default:
		return "", false
	}
}
