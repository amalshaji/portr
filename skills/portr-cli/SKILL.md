---
name: portr-cli
description: "Use when an AI agent or harness needs to operate the Portr CLI: authenticate a Portr client, create HTTP/WebSocket/TCP/stub tunnels, run config-defined tunnels, inspect request logs, replay captured requests, or control tunnels through the local app-server API. This skill is compatible with Claude Code and Codex through vercel-labs/skills."
---

# Portr CLI

Use this skill to operate an installed `portr` binary from an AI agent, test harness, or automation script. Portr exposes local services through public HTTP/WebSocket or TCP tunnels, can serve stubbed templated responses, stores local HTTP request logs, can replay stored requests, and can run a local API for programmatic tunnel lifecycle control.

## Agent Rules

- Prefer `portr --config <temp-config.yaml> ...` for automation so user config, auth tokens, and local request logs are not accidentally changed.
- Treat `portr http`, `portr tcp`, `portr stub`, `portr start`, and `portr app-server` as long-running processes. Keep their process/session IDs so the harness can stop them.
- Prefer `--json` for `portr logs` and `portr replay` when a harness needs to parse output.
- Use `portr app-server` for programmatic lifecycle management instead of scraping TUI output.
- Do not overwrite `~/.portr/config.yaml`, `~/.portr/db.sqlite`, or auth tokens unless the user explicitly asks.
- Do not create public tunnels for production services or sensitive local ports unless the user explicitly asks.
- If a command's flags are uncertain, run `portr <command> --help` against the installed binary.

## Command Map

```bash
portr [--config <path>] [--help] [--version] <command>
```

- `--config`, `-c`: YAML config path. Defaults to `~/.portr/config.yaml`.
- `--help`, `-h`: show help.
- `--version`, `-v`: print version.

Commands:

- `portr help [command]`: show general help or command-specific help.
- `portr auth set`: configure client auth.
- `portr config edit`: open the default config in the OS editor.
- `portr http`: expose a local HTTP/WebSocket port.
- `portr tcp`: expose a local TCP port.
- `portr stub`: serve a templated response through a public HTTP tunnel without a local server.
- `portr start`: start one or more tunnels from config.
- `portr logs`: read local stored HTTP request logs.
- `portr replay`: replay a stored HTTP request.
- `portr app-server`: start a local HTTP API for harness-controlled tunnels.

## Auth And Config

```bash
portr auth set --token <token> --remote <domain-or-url>
portr auth set -t <token> -r <domain-or-url>
portr config edit
```

- `--token`, `-t`: Portr client secret token from the server/admin UI. Required.
- `--remote`, `-r`: Portr server domain or URL. Required. Bare domains become HTTPS; `localhost:*` becomes HTTP unless a scheme is already provided.
- `config edit` only edits the default config path. For harnesses, write a temp config file and pass `--config`.

Minimal automation config:

```yaml
server_url: example.com
ssh_url: example.com:2222
tunnel_url: example.com
secret_key: your-secret-key
disable_tui: true
disable_dashboard: true
disable_update_check: true
enable_request_logging: true
```

Global config keys:

- `server_url`: admin/server URL.
- `ssh_url`: SSH tunnel server address.
- `tunnel_url`: public tunnel host suffix.
- `secret_key`: client auth secret.
- `use_localhost`: use HTTP rather than HTTPS for server/tunnel URLs.
- `debug`: enable debug behavior.
- `use_vite`: use the Vite-backed local UI path when supported by the running binary.
- `dashboard_port`: local inspector port. Default `7777`.
- `disable_dashboard`: skip the local inspector dashboard.
- `enable_request_logging`: store HTTP request logs locally. Default `true`.
- `connection_log_retention_days`: auto-delete old connection logs; `0` disables cleanup.
- `health_check_interval`: health check interval in seconds. Default `3`.
- `health_check_max_retries`: max health check retry count. Default `10`.
- `disable_tui`: run without the interactive terminal UI.
- `enable_http_reverse_proxy`: enable the HTTP reverse-proxy path when supported by the running binary.
- `disable_update_check`: suppress release update checks.
- `insecure_skip_host_key_verification`: skip SSH host key verification. Default `true`.

## One-Off Tunnels

HTTP/WebSocket:

```bash
portr http <local-port>
portr http <local-port> --subdomain <subdomain>
portr http <local-port> -s <subdomain>
```

- Exposes `localhost:<local-port>` as a public HTTP/WebSocket tunnel.
- If `--subdomain` is omitted, Portr generates one.
- Starts local request capture when request logging is enabled.

TCP:

```bash
portr tcp <local-port>
```

- Exposes `localhost:<local-port>` as a public TCP tunnel.
- Use for databases and raw TCP protocols.
- TCP traffic is not available through `portr logs`.

Stub:

```bash
portr stub --subdomain <subdomain> \
  --response-format application/json \
  --response-tmpl '{"ok": true}'

portr stub --subdomain <subdomain> \
  --response-format application/json \
  --response-tmpl-file ./response.json
```

- `--subdomain`, `-s`: public stub subdomain. Required.
- `--response-format`: response `Content-Type`, such as `application/json`, `application/yml`, or `text/plain`. Required.
- `--response-tmpl`: inline response template.
- `--response-tmpl-file`: path to response template file.
- Use exactly one of `--response-tmpl` or `--response-tmpl-file`.
- Stub templates can read request values using Portr's template placeholders.

## Config-Defined Tunnels

Config tunnel fields:

```yaml
tunnels:
  - name: app
    type: http
    host: localhost
    port: 3000
    subdomain: app-dev
    pool_size: 2
  - name: pg
    type: tcp
    host: localhost
    port: 5432
    subdomain: pg-dev
  - name: mock
    type: stub
    subdomain: mock-dev
    response_format: application/json
    response_tmpl_file: ./response.json
```

- `name`: identifier used by `portr start`.
- `type`: `http`, `tcp`, or `stub`. Defaults to `http`.
- `host`: local host. Defaults to `localhost` except stubs.
- `port`: local port for `http` and `tcp`.
- `subdomain`: public subdomain. Required for `stub`; generated for one-off HTTP when omitted.
- `pool_size`: worker count for non-stub tunnels. Defaults to `2`; stubs use `1`.
- `response_format`, `response_tmpl`, `response_tmpl_file`: stub response settings.

Start configured tunnels:

```bash
portr start
portr start app
portr start app pg mock
```

- No names starts all configured tunnels.
- Passing names starts only those tunnel entries.

## Request Logs

```bash
portr logs <subdomain>
portr logs <subdomain> <url-substring-filter>
portr logs <subdomain> --count 100
portr logs <subdomain> -n 100
portr logs <subdomain> --since 2026-04-04
portr logs <subdomain> --since 2026-04-04T10:30:00Z
portr logs <subdomain> --json
```

- Reads local HTTP request logs from `~/.portr/db.sqlite`.
- Logs are only available for traffic captured on the current machine with request logging enabled.
- Results are newest first.
- Positional filter is a case-insensitive URL substring.
- `--count`, `-n`: max records to return. Default `20`.
- `--since`: RFC3339 timestamp or `YYYY-MM-DD`.
- `--json`: emit full records. Binary body fields stay base64-encoded; UTF-8 payloads also include text fields.
- Use log IDs from this output with `portr replay`.

## Request Replay

```bash
portr replay <request-id>
portr replay --latest --subdomain <subdomain>
portr replay --latest --subdomain <subdomain> --filter /api/orders --since 2026-04-04
portr replay <request-id> --method POST --header 'Content-Type: application/json' --body '{"message":"hello"}'
cat payload.json | portr replay <request-id> --method POST --header 'Content-Type: application/json' --stdin
portr replay <request-id> --json
```

- Replays a stored HTTP request through the original tunnel host.
- `--latest`: choose the newest matching stored request instead of passing an ID.
- `--subdomain`: required with `--latest`.
- `--filter`: case-insensitive URL substring filter for `--latest`.
- `--since`: RFC3339 timestamp or `YYYY-MM-DD` filter for `--latest`.
- `--method`: override HTTP method.
- `--path`: override path and query. Must start with `/`.
- `--header 'Key: Value'`: add or override a header. Repeatable.
- `--drop-header <name>`: remove inherited header. Repeatable.
- `--body <value>`: inline body override.
- `--body-file <path>`: read body override from file.
- `--stdin`: read body override from stdin.
- `--body-encoding`: `utf8` or `base64`.
- `--json`: emit replay details, effective request, response, and structured errors.
- Body sources are mutually exclusive: use only one of `--body`, `--body-file`, or `--stdin`.
- Changing the body does not change the method; set `--method POST`, `PUT`, or `PATCH` when the body must matter.

## App Server For Harnesses

```bash
PORTR_APP_SERVER_TOKEN=change-me portr app-server
portr app-server --host 127.0.0.1 --port 7778 --token change-me
```

- Starts a local HTTP API around Portr tunnel lifecycle management.
- Default bind address is `127.0.0.1:7778`.
- `--host`: bind host.
- `--port`: bind port.
- `--token`: bearer token required by the API.
- `PORTR_APP_SERVER_TOKEN`: env var alternative for `--token`.
- If no token is configured, the local API is unauthenticated. Prefer a token in harnesses.

API endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/health` | Health check. |
| `POST` | `/api/v1/tunnels` | Start a managed tunnel. |
| `GET` | `/api/v1/tunnels` | List managed tunnels. |
| `GET` | `/api/v1/tunnels/{id}` | Get one tunnel status. |
| `DELETE` | `/api/v1/tunnels/{id}` | Stop one tunnel. |
| `POST` | `/api/v1/tunnels/{id}/shutdown` | Stop one tunnel. |
| `GET` | `/api/v1/events` | List lifecycle events. |
| `GET` | `/api/v1/events?tunnel_id={id}` | List lifecycle events for one tunnel. |

Create tunnel JSON:

```json
{
  "name": "app",
  "type": "http",
  "host": "localhost",
  "port": 3000,
  "subdomain": "app-dev",
  "pool_size": 2,
  "callback_url": "http://127.0.0.1:9000/portr-events",
  "callback_urls": ["https://automation.example.com/webhooks/portr"]
}
```

Stub tunnel JSON:

```json
{
  "name": "mock",
  "type": "stub",
  "subdomain": "mock-dev",
  "response_format": "application/json",
  "response_tmpl": "{\"ok\": true}"
}
```

Useful response fields:

- `id`: app-server tunnel ID for status and shutdown.
- `status`: lifecycle state such as `starting`, `running`, `stopped`, or `error`.
- `type`, `host`, `port`, `subdomain`, `remote_port`, `tunnel_url`: tunnel addressing.
- `callback_urls`: configured lifecycle callbacks.
- `last_error`: error text when startup or shutdown fails.

Harness pattern:

```bash
APP_SERVER=http://127.0.0.1:7778
TOKEN=change-me

curl -sS -H "Authorization: Bearer $TOKEN" "$APP_SERVER/api/v1/health"

TUNNEL_ID="$(curl -sS -X POST "$APP_SERVER/api/v1/tunnels" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"app","type":"http","host":"localhost","port":3000,"subdomain":"app-dev"}' \
  | jq -r '.id')"

curl -sS -H "Authorization: Bearer $TOKEN" "$APP_SERVER/api/v1/tunnels/$TUNNEL_ID"
curl -sS -X DELETE -H "Authorization: Bearer $TOKEN" "$APP_SERVER/api/v1/tunnels/$TUNNEL_ID"
```

## Choosing The Right Interface

- Use `portr http`, `portr tcp`, or `portr stub` for quick one-off terminal workflows.
- Use `portr start` when a harness already has a prepared config file and wants multiple named tunnels.
- Use `portr app-server` when a harness must create, inspect, and stop tunnels programmatically.
- Use `portr logs --json` to inspect captured traffic.
- Use `portr replay --json` to replay captured traffic and parse the result.
