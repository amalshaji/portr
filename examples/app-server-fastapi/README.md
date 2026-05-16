# FastAPI app-server example

This example starts a basic FastAPI app, creates an HTTP tunnel through a
running `portr app-server`, prints the public tunnel URL, and deletes the tunnel
when the app shuts down.

Start the app server first:

```bash
portr app-server
```

Then run the example:

```bash
cd examples/app-server-fastapi
uv run main.py
```

If your app server requires a local API token, pass the same token to the
example:

```bash
PORTR_APP_SERVER_TOKEN="change-me" uv run main.py
```

Optional environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `PORTR_APP_SERVER` | `http://127.0.0.1:7778` | App-server API base URL. |
| `PORTR_APP_SERVER_TOKEN` | unset | Bearer token for the app-server API. |
| `FASTAPI_HOST` | `127.0.0.1` | Local host for the FastAPI server. |
| `FASTAPI_PORT` | `8000` | Local port for the FastAPI server and tunnel. |
| `PORTR_TUNNEL_NAME` | `fastapi-example` | Name sent to the app server. |
| `PORTR_TUNNEL_SUBDOMAIN` | unset | Optional fixed HTTP tunnel subdomain. |

Press `Ctrl+C` to stop the FastAPI server. The lifespan shutdown hook deletes
the tunnel from the app server before the process exits.
