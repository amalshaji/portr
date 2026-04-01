package replay

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/amalshaji/portr/internal/client/db"
	"github.com/go-resty/resty/v2"
	"gorm.io/datatypes"
)

var (
	ErrUnsupportedMethod    = errors.New("unsupported replay method")
	ErrInvalidPath          = errors.New("path must start with /")
	ErrInvalidBodyEncoding  = errors.New("invalid request body encoding")
	ErrRequestHeadersDecode = errors.New("failed to parse request headers")
)

var allowedMethods = map[string]struct{}{
	"DELETE":  {},
	"GET":     {},
	"HEAD":    {},
	"OPTIONS": {},
	"PATCH":   {},
	"POST":    {},
	"PUT":     {},
	"TRACE":   {},
}

var newRestyClient = resty.New

type BodyOverride struct {
	Set      bool
	Value    string
	Encoding string
}

type EditOptions struct {
	Method         string
	Path           string
	Headers        map[string]string
	ReplaceHeaders bool
	DropHeaders    []string
	Body           BodyOverride
}

type Result struct {
	OriginalRequest  *db.Request
	EffectiveMethod  string
	EffectivePath    string
	EffectiveURL     string
	EffectiveHeaders map[string]string
	EffectiveBody    []byte
	ResponseStatus   int
	ResponseHeaders  map[string][]string
	ResponseBody     []byte
}

type Failure struct {
	StatusCode int
	Reason     string
	Message    string
}

func (f *Failure) Error() string {
	if f == nil {
		return ""
	}
	return f.Message
}

func Execute(request *db.Request, opts EditOptions) (*Result, error) {
	if request == nil {
		return nil, errors.New("request is required")
	}

	method := normalizeMethod(opts.Method)
	if method == "" {
		method = normalizeMethod(request.Method)
	}
	if _, ok := allowedMethods[method]; !ok {
		return nil, ErrUnsupportedMethod
	}

	path := strings.TrimSpace(opts.Path)
	if path == "" {
		path = request.Url
	}
	if !strings.HasPrefix(path, "/") {
		return nil, ErrInvalidPath
	}

	headers, err := buildHeaders(request.ID, request.Headers, opts)
	if err != nil {
		return nil, err
	}

	body := request.Body
	if opts.Body.Set {
		body, err = decodeBody(opts.Body.Value, opts.Body.Encoding)
		if err != nil {
			return nil, err
		}
	}

	targetURL := fmt.Sprintf("https://%s%s", request.Host, path)
	response, err := execute(method, targetURL, headers, body)
	if err != nil {
		return nil, err
	}

	result := &Result{
		OriginalRequest:  request,
		EffectiveMethod:  method,
		EffectivePath:    path,
		EffectiveURL:     targetURL,
		EffectiveHeaders: cloneFlatHeaders(headers),
		EffectiveBody:    append([]byte(nil), body...),
		ResponseStatus:   response.StatusCode(),
		ResponseHeaders:  cloneHeaderValues(response.Header()),
		ResponseBody:     append([]byte(nil), response.Body()...),
	}

	if failure := replayFailure(response); failure != nil {
		return result, failure
	}

	return result, nil
}

func normalizeMethod(method string) string {
	method = strings.TrimSpace(strings.ToUpper(method))
	if method == "" {
		return ""
	}
	return method
}

func buildHeaders(requestID string, raw datatypes.JSON, opts EditOptions) (map[string]string, error) {
	headersMap, err := decodeHeaderValues(raw)
	if err != nil {
		return nil, err
	}

	headers := flattenHeaders(headersMap)

	if opts.ReplaceHeaders {
		headers = make(map[string]string, len(opts.Headers))
		for key, value := range opts.Headers {
			headers[key] = value
		}
	} else {
		for key, value := range opts.Headers {
			setHeader(headers, key, value)
		}
	}

	for _, key := range opts.DropHeaders {
		deleteHeader(headers, key)
	}

	setHeader(headers, "X-Portr-Replayed-Request-Id", requestID)
	return headers, nil
}

func decodeHeaderValues(raw datatypes.JSON) (map[string][]string, error) {
	headers := make(map[string][]string)
	if len(raw) == 0 {
		return headers, nil
	}

	if err := json.Unmarshal(raw, &headers); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestHeadersDecode, err)
	}

	return headers, nil
}

func flattenHeaders(headers map[string][]string) map[string]string {
	flat := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		flat[key] = values[0]
	}
	return flat
}

func decodeBody(body string, encoding string) ([]byte, error) {
	if strings.EqualFold(strings.TrimSpace(encoding), "base64") {
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return nil, ErrInvalidBodyEncoding
		}
		return decoded, nil
	}

	if encoding != "" && !strings.EqualFold(strings.TrimSpace(encoding), "utf8") {
		return nil, ErrInvalidBodyEncoding
	}

	return []byte(body), nil
}

func execute(method string, targetURL string, headers map[string]string, body []byte) (*resty.Response, error) {
	client := newRestyClient().R()
	for key, value := range headers {
		if value == "" {
			continue
		}
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		client.SetHeader(key, value)
	}

	if body != nil {
		client.SetBody(body)
	}
	return client.Execute(method, targetURL)
}

func replayFailure(response *resty.Response) *Failure {
	reason := response.Header().Get("X-Portr-Error-Reason")
	if reason == "" {
		return nil
	}

	switch reason {
	case "unregistered-subdomain":
		return &Failure{
			StatusCode: 500,
			Reason:     reason,
			Message:    "The tunnel is not active. Please start the tunnel and try again",
		}
	case "local-server-not-online":
		return &Failure{
			StatusCode: 500,
			Reason:     reason,
			Message:    "The local server is not online. Please start the local server and try again",
		}
	case "connection-lost":
		return &Failure{
			StatusCode: 503,
			Reason:     reason,
			Message:    "The tunnel connection was lost. Please try again in a bit.",
		}
	default:
		return &Failure{
			StatusCode: 502,
			Reason:     reason,
			Message:    "Failed to replay request",
		}
	}
}

func setHeader(headers map[string]string, key string, value string) {
	if headers == nil {
		return
	}
	deleteHeader(headers, key)
	headers[key] = value
}

func deleteHeader(headers map[string]string, key string) {
	for existing := range headers {
		if strings.EqualFold(existing, key) {
			delete(headers, existing)
		}
	}
}

func cloneFlatHeaders(headers map[string]string) map[string]string {
	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}

func cloneHeaderValues(headers map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(headers))
	for key, values := range headers {
		copied := make([]string, len(values))
		copy(copied, values)
		cloned[key] = copied
	}
	return cloned
}
