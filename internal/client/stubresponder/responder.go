package stubresponder

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/amalshaji/portr/internal/client/localserver"
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

// Responder serves templated stub responses over the shared local server.
type Responder struct {
	*localserver.Server
}

func New() *Responder {
	return &Responder{Server: localserver.New("stub responder")}
}

func (r *Responder) Register(route Route) error {
	if strings.TrimSpace(route.ResponseFormat) == "" {
		return fmt.Errorf("response format is required")
	}
	if strings.TrimSpace(route.ResponseTemplate) == "" {
		return fmt.Errorf("response template is required")
	}

	return r.Server.Register(route.Subdomain, newStubHandler(route))
}

func newStubHandler(route Route) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		bodyValues, err := requestBodyValues(req)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		rendered := renderTemplate(route.ResponseTemplate, req.URL.Query(), bodyValues, isJSONResponse(route.ResponseFormat))
		w.Header().Set("Content-Type", route.ResponseFormat)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(rendered))
	})
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
