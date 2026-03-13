package logs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	clientdb "github.com/amalshaji/portr/internal/client/db"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const DefaultCount = 20
const JSONFlagUsage = "Emit full stored request records as JSON, including request and response payloads"

type QueryOptions struct {
	Count  int
	Since  *time.Time
	Filter string
}

type CommandOptions struct {
	Subdomain string
	Query     QueryOptions
	JSON      bool
}

type Store struct {
	conn *gorm.DB
}

func WantsHelp(args []string) bool {
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" {
			continue
		}
		if trimmed == "--" {
			return false
		}
		if trimmed == "--help" || trimmed == "-h" {
			return true
		}
	}

	return false
}

func DefaultDBPath() string {
	return filepath.Join(config.DefaultConfigDir, "db.sqlite")
}

func Open(path string) (*Store, error) {
	if path == "" {
		path = DefaultDBPath()
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("request log database not found at %s", path)
		}
		return nil, fmt.Errorf("failed to access request log database at %s: %w", path, err)
	}

	conn, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open request log database at %s: %w", path, err)
	}

	return &Store{conn: conn}, nil
}

func (s *Store) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}

	sqlDB, err := s.conn.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func (s *Store) List(subdomain string, opts QueryOptions) ([]clientdb.Request, error) {
	if s == nil || s.conn == nil {
		return nil, errors.New("request log store is not initialized")
	}

	subdomain = strings.TrimSpace(subdomain)
	if subdomain == "" {
		return nil, errors.New("subdomain is required")
	}

	normalized, err := normalizeQueryOptions(opts)
	if err != nil {
		return nil, err
	}

	query := s.conn.Where("subdomain = ?", subdomain)

	if normalized.Since != nil {
		query = query.Where("logged_at >= ?", normalized.Since.UTC())
	}

	if normalized.Filter != "" {
		query = query.Where("LOWER(url) LIKE ? ESCAPE '\\'", likePattern(normalized.Filter))
	}

	var requests []clientdb.Request
	if err := query.Order("logged_at desc").Order("id desc").Limit(normalized.Count).Find(&requests).Error; err != nil {
		return nil, fmt.Errorf("failed to query request logs: %w", err)
	}

	if requests == nil {
		requests = make([]clientdb.Request, 0)
	}

	return requests, nil
}

func ParseSince(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		parsed = parsed.UTC()
		return &parsed, nil
	}

	if parsed, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		parsed = parsed.UTC()
		return &parsed, nil
	}

	return nil, fmt.Errorf("invalid since value %q: use RFC3339 or YYYY-MM-DD", value)
}

func ParseCommandArgs(args []string) (CommandOptions, error) {
	opts := CommandOptions{
		Query: QueryOptions{
			Count: DefaultCount,
		},
	}

	var positional []string
	var sinceRaw string

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}

		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}

		switch {
		case arg == "--json":
			opts.JSON = true
		case arg == "--count" || arg == "-n":
			if i+1 >= len(args) {
				return CommandOptions{}, fmt.Errorf("missing value for %s", arg)
			}
			i++
			count, err := strconv.Atoi(strings.TrimSpace(args[i]))
			if err != nil {
				return CommandOptions{}, fmt.Errorf("invalid count value %q", args[i])
			}
			opts.Query.Count = count
		case strings.HasPrefix(arg, "--count="):
			count, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(arg, "--count=")))
			if err != nil {
				return CommandOptions{}, fmt.Errorf("invalid count value %q", strings.TrimPrefix(arg, "--count="))
			}
			opts.Query.Count = count
		case strings.HasPrefix(arg, "-n="):
			count, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(arg, "-n=")))
			if err != nil {
				return CommandOptions{}, fmt.Errorf("invalid count value %q", strings.TrimPrefix(arg, "-n="))
			}
			opts.Query.Count = count
		case arg == "--since":
			if i+1 >= len(args) {
				return CommandOptions{}, fmt.Errorf("missing value for %s", arg)
			}
			i++
			sinceRaw = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--since="):
			sinceRaw = strings.TrimSpace(strings.TrimPrefix(arg, "--since="))
		case strings.HasPrefix(arg, "-"):
			return CommandOptions{}, fmt.Errorf("unknown flag %s", arg)
		default:
			positional = append(positional, arg)
		}
	}

	if len(positional) == 0 || strings.TrimSpace(positional[0]) == "" {
		return CommandOptions{}, errors.New("please specify a subdomain")
	}

	opts.Subdomain = strings.TrimSpace(positional[0])
	if len(positional) > 1 {
		opts.Query.Filter = strings.TrimSpace(strings.Join(positional[1:], " "))
	}

	since, err := ParseSince(sinceRaw)
	if err != nil {
		return CommandOptions{}, err
	}
	opts.Query.Since = since

	normalized, err := normalizeQueryOptions(opts.Query)
	if err != nil {
		return CommandOptions{}, err
	}
	opts.Query = normalized

	return opts, nil
}

func RenderText(w io.Writer, requests []clientdb.Request) error {
	if len(requests) == 0 {
		_, err := fmt.Fprintln(w, "no logs found")
		return err
	}

	for _, request := range requests {
		if _, err := fmt.Fprintf(
			w,
			"%s %d %s %d %s",
			request.LoggedAt.UTC().Format(time.RFC3339),
			request.Localport,
			request.Method,
			request.ResponseStatusCode,
			request.Url,
		); err != nil {
			return err
		}

		if request.IsReplayed {
			if _, err := io.WriteString(w, " [replayed]"); err != nil {
				return err
			}
		}

		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}

	return nil
}

func RenderJSON(w io.Writer, requests []clientdb.Request) error {
	if requests == nil {
		requests = make([]clientdb.Request, 0)
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(requests)
}

func normalizeQueryOptions(opts QueryOptions) (QueryOptions, error) {
	if opts.Count == 0 {
		opts.Count = DefaultCount
	}

	if opts.Count < 0 {
		return QueryOptions{}, errors.New("count must be greater than 0")
	}

	opts.Filter = strings.TrimSpace(opts.Filter)
	return opts, nil
}

func likePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + strings.ToLower(replacer.Replace(value)) + "%"
}
