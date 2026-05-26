# Portr CLI Command Reference

This file mirrors the CLI surface for quick lookup. The skill body contains the operating workflow and harness guidance.

## Global

- `portr --config <path> <command>`: use a specific YAML config file.
- `portr --help`: list commands.
- `portr --version`: print version.

## Commands And Flags

| Command | Purpose | Key flags and args |
| --- | --- | --- |
| `portr help [command]` | Show general or command-specific help. | command name as optional positional arg |
| `portr auth set` | Configure client auth from a Portr server/admin UI token. | `--token`, `-t`; `--remote`, `-r` |
| `portr config edit` | Open the default config file in the OS editor. | none |
| `portr http <port>` | Expose a local HTTP/WebSocket port. | `--subdomain`, `-s` |
| `portr tcp <port>` | Expose a local TCP port. | none |
| `portr stub` | Serve a templated response through an HTTP tunnel. | `--subdomain`, `-s`; `--response-format`; `--response-tmpl`; `--response-tmpl-file` |
| `portr start [names...]` | Start config-defined tunnels. | tunnel names as positional args |
| `portr logs <subdomain> [filter]` | Read stored local HTTP request logs. | `--count`, `-n`; `--since`; `--json` |
| `portr replay <request-id>` | Replay a stored HTTP request. | `--latest`; `--subdomain`; `--filter`; `--since`; `--method`; `--path`; `--header`; `--drop-header`; `--body`; `--body-file`; `--stdin`; `--body-encoding`; `--json` |
| `portr app-server` | Run a local HTTP API for tunnel lifecycle control. | `--host`; `--port`; `--token`; `PORTR_APP_SERVER_TOKEN` |

## Config Keys

- Global: `server_url`, `ssh_url`, `tunnel_url`, `secret_key`, `use_localhost`, `debug`, `use_vite`, `dashboard_port`, `disable_dashboard`, `enable_request_logging`, `connection_log_retention_days`, `health_check_interval`, `health_check_max_retries`, `disable_tui`, `enable_http_reverse_proxy`, `disable_update_check`, `insecure_skip_host_key_verification`.
- Tunnel: `name`, `type`, `host`, `port`, `subdomain`, `pool_size`, `response_format`, `response_tmpl`, `response_tmpl_file`.

## App Server API

| Method | Path |
| --- | --- |
| `GET` | `/api/v1/health` |
| `POST` | `/api/v1/tunnels` |
| `GET` | `/api/v1/tunnels` |
| `GET` | `/api/v1/tunnels/{id}` |
| `DELETE` | `/api/v1/tunnels/{id}` |
| `POST` | `/api/v1/tunnels/{id}/shutdown` |
| `GET` | `/api/v1/events` |
| `GET` | `/api/v1/events?tunnel_id={id}` |

`POST /api/v1/tunnels` accepts `name`, `type`, `host`, `port`, `subdomain`, `pool_size`, `response_format`, `response_tmpl`, `response_tmpl_file`, `callback_url`, and `callback_urls`.
