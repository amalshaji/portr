package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	clientdb "github.com/amalshaji/portr/internal/client/db"
	requestlogs "github.com/amalshaji/portr/internal/client/logs"
	clientreplay "github.com/amalshaji/portr/internal/client/replay"
	"github.com/urfave/cli/v2"
)

const replayJSONFlagUsage = "Emit replay results as JSON, including effective request details and response payloads"

const (
	replayBodyInline = "inline"
	replayBodyFile   = "file"
	replayBodyStdin  = "stdin"
)

type replayBodySource struct {
	Kind     string
	Value    string
	Encoding string
}

type replayCommandOptions struct {
	RequestID string
	UseLatest bool
	Subdomain string
	Query     requestlogs.QueryOptions
	Edit      clientreplay.EditOptions
	Body      replayBodySource
	JSON      bool
}

type replayJSONOutput struct {
	OK                 bool                   `json:"ok"`
	RequestID          string                 `json:"request_id,omitempty"`
	SelectedBy         string                 `json:"selected_by,omitempty"`
	Subdomain          string                 `json:"subdomain,omitempty"`
	Localport          int                    `json:"localport,omitempty"`
	Host               string                 `json:"host,omitempty"`
	OriginalMethod     string                 `json:"original_method,omitempty"`
	OriginalPath       string                 `json:"original_path,omitempty"`
	EffectiveMethod    string                 `json:"effective_method,omitempty"`
	EffectivePath      string                 `json:"effective_path,omitempty"`
	EffectiveURL       string                 `json:"effective_url,omitempty"`
	EffectiveHeaders   map[string]string      `json:"effective_headers,omitempty"`
	EffectiveBody      string                 `json:"effective_body,omitempty"`
	EffectiveBodyText  *string                `json:"effective_body_text,omitempty"`
	ResponseStatusCode int                    `json:"response_status_code,omitempty"`
	ResponseHeaders    map[string][]string    `json:"response_headers,omitempty"`
	ResponseBody       string                 `json:"response_body,omitempty"`
	ResponseBodyText   *string                `json:"response_body_text,omitempty"`
	Error              *replayJSONOutputError `json:"error,omitempty"`
}

type replayJSONOutputError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

func replayCmd() *cli.Command {
	return &cli.Command{
		Name:            "replay",
		Usage:           "Replay a stored HTTP request",
		ArgsUsage:       "<request-id>",
		SkipFlagParsing: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "latest",
				Usage: "Replay the latest matching request instead of providing an explicit request ID",
			},
			&cli.StringFlag{
				Name:  "subdomain",
				Usage: "Tunnel subdomain to search when using --latest",
			},
			&cli.StringFlag{
				Name:  "filter",
				Usage: "Case-insensitive URL substring filter when using --latest",
			},
			&cli.StringFlag{
				Name:  "since",
				Usage: "Only consider requests on or after the given RFC3339 timestamp or YYYY-MM-DD date when using --latest",
			},
			&cli.StringFlag{
				Name:  "method",
				Usage: "Override the HTTP method before replaying",
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "Override the request path and query before replaying",
			},
			&cli.StringSliceFlag{
				Name:  "header",
				Usage: "Add or override a header using 'Key: Value'; can be repeated",
			},
			&cli.StringSliceFlag{
				Name:  "drop-header",
				Usage: "Remove an inherited header; can be repeated",
			},
			&cli.StringFlag{
				Name:  "body",
				Usage: "Override the request body with an inline value",
			},
			&cli.StringFlag{
				Name:  "body-file",
				Usage: "Override the request body with the contents of a file",
			},
			&cli.BoolFlag{
				Name:  "stdin",
				Usage: "Override the request body with bytes read from stdin",
			},
			&cli.StringFlag{
				Name:  "body-encoding",
				Usage: "Encoding for the supplied body override: utf8 or base64",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: replayJSONFlagUsage,
			},
		},
		Action: runReplayCommand,
	}
}

func runReplayCommand(c *cli.Context) error {
	rawArgs := c.Args().Slice()
	if replayWantsHelp(rawArgs) {
		showCurrentCommandHelp(c)
		return nil
	}

	opts, err := parseReplayCommandArgs(rawArgs)
	if err != nil {
		return err
	}

	store, err := requestlogs.Open("")
	if err != nil {
		return err
	}
	defer store.Close()

	request, err := resolveReplayRequest(store, opts)
	if err != nil {
		if opts.JSON {
			if renderErr := renderReplayJSON(c.App.Writer, opts, nil, nil, err); renderErr != nil {
				return renderErr
			}
			return cli.Exit("", 1)
		}
		return err
	}

	edit := opts.Edit
	edit.Body, err = resolveReplayBody(opts.Body, os.Stdin)
	if err != nil {
		if opts.JSON {
			if renderErr := renderReplayJSON(c.App.Writer, opts, request, nil, err); renderErr != nil {
				return renderErr
			}
			return cli.Exit("", 1)
		}
		return err
	}

	result, err := clientreplay.Execute(request, edit)
	if opts.JSON {
		if renderErr := renderReplayJSON(c.App.Writer, opts, request, result, err); renderErr != nil {
			return renderErr
		}
		if err != nil {
			return cli.Exit("", 1)
		}
		return nil
	}

	if err != nil {
		return err
	}

	return renderReplayText(c.App.Writer, result)
}

func replayWantsJSON(args []string) bool {
	for _, arg := range args {
		if strings.TrimSpace(arg) == "--" {
			return false
		}
		if strings.TrimSpace(arg) == "--json" {
			return true
		}
	}

	return false
}

func replayWantsHelp(args []string) bool {
	for _, arg := range args {
		if strings.TrimSpace(arg) == "--" {
			return false
		}
		trimmed := strings.TrimSpace(arg)
		if trimmed == "--help" || trimmed == "-h" {
			return true
		}
	}

	return false
}

func parseReplayCommandArgs(args []string) (replayCommandOptions, error) {
	opts := replayCommandOptions{
		Query: requestlogs.QueryOptions{Count: 1},
		Edit: clientreplay.EditOptions{
			Headers: make(map[string]string),
		},
		Body: replayBodySource{
			Encoding: "utf8",
		},
	}

	var positional []string
	var sinceRaw string
	var sinceSet bool
	var filterSet bool
	var subdomainSet bool
	bodySources := 0

	for i := 0; i < len(args); i++ {
		arg := args[i]
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" {
			continue
		}

		if trimmed == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}

		switch {
		case trimmed == "--json":
			opts.JSON = true
		case trimmed == "--latest":
			opts.UseLatest = true
		case trimmed == "--subdomain":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			opts.Subdomain = strings.TrimSpace(args[i])
			subdomainSet = true
		case strings.HasPrefix(trimmed, "--subdomain="):
			opts.Subdomain = strings.TrimSpace(strings.TrimPrefix(trimmed, "--subdomain="))
			subdomainSet = true
		case trimmed == "--filter":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			opts.Query.Filter = strings.TrimSpace(args[i])
			filterSet = true
		case strings.HasPrefix(trimmed, "--filter="):
			opts.Query.Filter = strings.TrimSpace(strings.TrimPrefix(trimmed, "--filter="))
			filterSet = true
		case trimmed == "--since":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			sinceRaw = strings.TrimSpace(args[i])
			sinceSet = true
		case strings.HasPrefix(trimmed, "--since="):
			sinceRaw = strings.TrimSpace(strings.TrimPrefix(trimmed, "--since="))
			sinceSet = true
		case trimmed == "--method":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			opts.Edit.Method = strings.TrimSpace(args[i])
		case strings.HasPrefix(trimmed, "--method="):
			opts.Edit.Method = strings.TrimSpace(strings.TrimPrefix(trimmed, "--method="))
		case trimmed == "--path":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			opts.Edit.Path = args[i]
		case strings.HasPrefix(trimmed, "--path="):
			opts.Edit.Path = strings.TrimPrefix(arg, "--path=")
		case trimmed == "--header":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			key, value, err := parseReplayHeader(args[i])
			if err != nil {
				return replayCommandOptions{}, err
			}
			setReplayHeaderOverride(opts.Edit.Headers, key, value)
		case strings.HasPrefix(trimmed, "--header="):
			key, value, err := parseReplayHeader(strings.TrimPrefix(arg, "--header="))
			if err != nil {
				return replayCommandOptions{}, err
			}
			setReplayHeaderOverride(opts.Edit.Headers, key, value)
		case trimmed == "--drop-header":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			key := strings.TrimSpace(args[i])
			if key == "" {
				return replayCommandOptions{}, errors.New("drop-header name is required")
			}
			opts.Edit.DropHeaders = append(opts.Edit.DropHeaders, key)
		case strings.HasPrefix(trimmed, "--drop-header="):
			key := strings.TrimSpace(strings.TrimPrefix(trimmed, "--drop-header="))
			if key == "" {
				return replayCommandOptions{}, errors.New("drop-header name is required")
			}
			opts.Edit.DropHeaders = append(opts.Edit.DropHeaders, key)
		case trimmed == "--body":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			opts.Body.Kind = replayBodyInline
			opts.Body.Value = args[i]
			bodySources++
		case strings.HasPrefix(trimmed, "--body="):
			opts.Body.Kind = replayBodyInline
			opts.Body.Value = strings.TrimPrefix(arg, "--body=")
			bodySources++
		case trimmed == "--body-file":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			opts.Body.Kind = replayBodyFile
			opts.Body.Value = strings.TrimSpace(args[i])
			bodySources++
		case strings.HasPrefix(trimmed, "--body-file="):
			opts.Body.Kind = replayBodyFile
			opts.Body.Value = strings.TrimSpace(strings.TrimPrefix(trimmed, "--body-file="))
			bodySources++
		case trimmed == "--stdin":
			opts.Body.Kind = replayBodyStdin
			bodySources++
		case trimmed == "--body-encoding":
			if i+1 >= len(args) {
				return replayCommandOptions{}, fmt.Errorf("missing value for %s", trimmed)
			}
			i++
			opts.Body.Encoding = strings.TrimSpace(args[i])
		case strings.HasPrefix(trimmed, "--body-encoding="):
			opts.Body.Encoding = strings.TrimSpace(strings.TrimPrefix(trimmed, "--body-encoding="))
		case strings.HasPrefix(trimmed, "-"):
			return replayCommandOptions{}, fmt.Errorf("unknown flag %s", trimmed)
		default:
			positional = append(positional, arg)
		}
	}

	if bodySources > 1 {
		return replayCommandOptions{}, errors.New("only one of --body, --body-file, or --stdin may be used")
	}

	opts.Body.Encoding = normalizeReplayBodyEncoding(opts.Body.Encoding)
	if opts.Body.Encoding == "" {
		return replayCommandOptions{}, errors.New("body-encoding must be utf8 or base64")
	}
	if opts.Body.Kind == "" && opts.Body.Encoding != "utf8" {
		return replayCommandOptions{}, errors.New("body-encoding requires --body, --body-file, or --stdin")
	}
	if opts.Body.Kind == replayBodyFile && opts.Body.Value == "" {
		return replayCommandOptions{}, errors.New("body-file path is required")
	}

	if sinceSet {
		since, err := requestlogs.ParseSince(sinceRaw)
		if err != nil {
			return replayCommandOptions{}, err
		}
		opts.Query.Since = since
	}

	if len(positional) > 1 {
		return replayCommandOptions{}, errors.New("expected a single request id")
	}
	if len(positional) == 1 {
		opts.RequestID = strings.TrimSpace(positional[0])
	}

	if opts.UseLatest {
		if opts.RequestID != "" {
			return replayCommandOptions{}, errors.New("request id and --latest cannot be used together")
		}
		if strings.TrimSpace(opts.Subdomain) == "" {
			return replayCommandOptions{}, errors.New("--subdomain is required with --latest")
		}
	} else {
		if opts.RequestID == "" {
			return replayCommandOptions{}, errors.New("please specify a request id or use --latest")
		}
		if subdomainSet || filterSet || sinceSet {
			return replayCommandOptions{}, errors.New("--subdomain, --filter, and --since can only be used with --latest")
		}
	}

	return opts, nil
}

func parseReplayHeader(value string) (string, string, error) {
	key, rawValue, ok := strings.Cut(value, ":")
	if !ok {
		return "", "", fmt.Errorf("invalid header %q: use 'Key: Value'", value)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", fmt.Errorf("invalid header %q: header name is required", value)
	}

	return key, strings.TrimSpace(rawValue), nil
}

func normalizeReplayBodyEncoding(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "utf8"
	}
	if trimmed == "utf8" || trimmed == "base64" {
		return trimmed
	}
	return ""
}

func setReplayHeaderOverride(headers map[string]string, key string, value string) {
	for existing := range headers {
		if strings.EqualFold(existing, key) {
			delete(headers, existing)
		}
	}
	headers[key] = value
}

func resolveReplayRequest(store *requestlogs.Store, opts replayCommandOptions) (*clientdb.Request, error) {
	if opts.UseLatest {
		request, err := store.Latest(opts.Subdomain, opts.Query)
		if err != nil {
			if errors.Is(err, requestlogs.ErrRequestNotFound) {
				return nil, fmt.Errorf("no matching requests found for subdomain %q", opts.Subdomain)
			}
			return nil, err
		}
		return request, nil
	}

	request, err := store.GetByID(opts.RequestID)
	if err != nil {
		if errors.Is(err, requestlogs.ErrRequestNotFound) {
			return nil, fmt.Errorf("request %q not found", opts.RequestID)
		}
		return nil, err
	}

	return request, nil
}

func resolveReplayBody(source replayBodySource, reader io.Reader) (clientreplay.BodyOverride, error) {
	if source.Kind == "" {
		return clientreplay.BodyOverride{}, nil
	}

	switch source.Kind {
	case replayBodyInline:
		return clientreplay.BodyOverride{
			Set:      true,
			Value:    source.Value,
			Encoding: source.Encoding,
		}, nil
	case replayBodyFile:
		data, err := os.ReadFile(source.Value)
		if err != nil {
			return clientreplay.BodyOverride{}, fmt.Errorf("failed to read body file %s: %w", source.Value, err)
		}
		return clientreplay.BodyOverride{
			Set:      true,
			Value:    string(data),
			Encoding: source.Encoding,
		}, nil
	case replayBodyStdin:
		data, err := io.ReadAll(reader)
		if err != nil {
			return clientreplay.BodyOverride{}, fmt.Errorf("failed to read stdin: %w", err)
		}
		return clientreplay.BodyOverride{
			Set:      true,
			Value:    string(data),
			Encoding: source.Encoding,
		}, nil
	default:
		return clientreplay.BodyOverride{}, errors.New("unsupported body source")
	}
}

func renderReplayText(w io.Writer, result *clientreplay.Result) error {
	if result == nil || result.OriginalRequest == nil {
		return errors.New("missing replay result")
	}

	_, err := fmt.Fprintf(
		w,
		"replayed %s %s %s -> %d\n",
		result.OriginalRequest.ID,
		result.EffectiveMethod,
		result.EffectiveURL,
		result.ResponseStatus,
	)
	return err
}

func renderReplayJSON(w io.Writer, opts replayCommandOptions, request *clientdb.Request, result *clientreplay.Result, replayErr error) error {
	output := replayJSONOutput{
		OK:         replayErr == nil,
		RequestID:  opts.RequestID,
		SelectedBy: replayJSONSelection(opts),
		Subdomain:  opts.Subdomain,
	}

	if request != nil {
		output.RequestID = request.ID
		output.Subdomain = request.Subdomain
		output.Localport = request.Localport
		output.Host = request.Host
		output.OriginalMethod = request.Method
		output.OriginalPath = request.Url
	}

	if result != nil {
		output.EffectiveMethod = result.EffectiveMethod
		output.EffectivePath = result.EffectivePath
		output.EffectiveURL = result.EffectiveURL
		output.EffectiveHeaders = result.EffectiveHeaders
		output.EffectiveBody = base64.StdEncoding.EncodeToString(result.EffectiveBody)
		output.EffectiveBodyText = utf8Text(result.EffectiveBody)
		output.ResponseStatusCode = result.ResponseStatus
		output.ResponseHeaders = result.ResponseHeaders
		output.ResponseBody = base64.StdEncoding.EncodeToString(result.ResponseBody)
		output.ResponseBodyText = utf8Text(result.ResponseBody)
	}

	if replayErr != nil {
		replayJSONErr := &replayJSONOutputError{
			Message: replayErr.Error(),
		}

		var failure *clientreplay.Failure
		if errors.As(replayErr, &failure) {
			replayJSONErr.StatusCode = failure.StatusCode
			replayJSONErr.Reason = failure.Reason
		}

		output.Error = replayJSONErr
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(output)
}

func replayJSONSelection(opts replayCommandOptions) string {
	if opts.UseLatest {
		return "latest"
	}
	return "id"
}

func utf8Text(value []byte) *string {
	if value == nil || !utf8.Valid(value) {
		return nil
	}

	text := string(value)
	return &text
}
