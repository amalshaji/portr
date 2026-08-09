package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/urfave/cli/v2"
)

type adminAddUserRequest struct {
	Email    string `json:"email"`
	TeamSlug string `json:"team_slug,omitempty"`
	Role     string `json:"role"`
}

type adminAddUserResult struct {
	TeamUser struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
		Role string `json:"role"`
	} `json:"team_user"`
	Team struct {
		Slug string `json:"slug"`
	} `json:"team"`
	Password string `json:"password,omitempty"`
}

type adminAddUserOptions struct {
	Email string
	Team  string
	Role  string
}

type adminErrorResponse struct {
	Error string `json:"error"`
}

func adminCmd() *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "Manage Portr users and teams",
		Subcommands: []*cli.Command{
			{
				Name:  "users",
				Usage: "Manage users",
				Subcommands: []*cli.Command{
					{
						Name:            "add",
						Usage:           "Add a user to a team",
						ArgsUsage:       "<email>",
						SkipFlagParsing: true,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "team", Usage: "Team slug"},
							&cli.StringFlag{Name: "role", Usage: "Team role", Value: "member"},
						},
						Action: addAdminUser,
					},
				},
			},
		},
	}
}

func addAdminUser(c *cli.Context) error {
	rawArgs := c.Args().Slice()
	if commandWantsHelp(rawArgs) {
		showCurrentCommandHelp(c)
		return nil
	}

	opts, err := parseAdminAddUserArgs(rawArgs)
	if err != nil {
		return err
	}

	cfg, err := config.Load(c.String("config"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if strings.TrimSpace(cfg.SecretKey) == "" {
		return fmt.Errorf("secret_key is required in the selected config")
	}

	payload, err := json.Marshal(adminAddUserRequest{
		Email:    opts.Email,
		TeamSlug: opts.Team,
		Role:     opts.Role,
	})
	if err != nil {
		return err
	}

	endpoint := strings.TrimRight(cfg.GetAdminAddress(), "/") + "/api/v1/admin/users"
	req, err := http.NewRequestWithContext(c.Context, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("add user: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var serverError adminErrorResponse
		if json.Unmarshal(body, &serverError) == nil && serverError.Error != "" {
			return fmt.Errorf("%s", serverError.Error)
		}
		return fmt.Errorf("server returned %s", resp.Status)
	}

	var result adminAddUserResult
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	fmt.Fprintf(c.App.Writer, "Added %s to %s as %s.\n", result.TeamUser.User.Email, result.Team.Slug, result.TeamUser.Role)
	if result.Password != "" {
		fmt.Fprintf(c.App.Writer, "Password: %s\n", result.Password)
	}
	return nil
}

func parseAdminAddUserArgs(args []string) (adminAddUserOptions, error) {
	opts := adminAddUserOptions{Role: "member"}
	positionalOnly := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if positionalOnly {
			if opts.Email != "" {
				return adminAddUserOptions{}, fmt.Errorf("unexpected argument %q", arg)
			}
			opts.Email = arg
			continue
		}

		switch {
		case arg == "--":
			positionalOnly = true
		case arg == "--team":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return adminAddUserOptions{}, fmt.Errorf("team requires a value")
			}
			i++
			opts.Team = strings.TrimSpace(args[i])
			if opts.Team == "" {
				return adminAddUserOptions{}, fmt.Errorf("team requires a value")
			}
		case strings.HasPrefix(arg, "--team="):
			opts.Team = strings.TrimSpace(strings.TrimPrefix(arg, "--team="))
			if opts.Team == "" {
				return adminAddUserOptions{}, fmt.Errorf("team requires a value")
			}
		case arg == "--role":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return adminAddUserOptions{}, fmt.Errorf("role requires a value")
			}
			i++
			opts.Role = args[i]
		case strings.HasPrefix(arg, "--role="):
			opts.Role = strings.TrimPrefix(arg, "--role=")
		case strings.HasPrefix(arg, "-"):
			return adminAddUserOptions{}, fmt.Errorf("unknown flag %q", arg)
		default:
			if opts.Email != "" {
				return adminAddUserOptions{}, fmt.Errorf("unexpected argument %q", arg)
			}
			opts.Email = arg
		}
	}

	opts.Email = strings.TrimSpace(opts.Email)
	opts.Team = strings.TrimSpace(opts.Team)
	opts.Role = strings.TrimSpace(opts.Role)
	if opts.Email == "" {
		return adminAddUserOptions{}, fmt.Errorf("email is required")
	}
	if opts.Role != "member" && opts.Role != "admin" {
		return adminAddUserOptions{}, fmt.Errorf("role must be either 'member' or 'admin'")
	}

	return opts, nil
}
