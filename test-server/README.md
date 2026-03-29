# Dashboard Test Server

This folder contains a standalone FastAPI app for exercising the client
dashboard inspector against different request bodies, response types, and
WebSocket behaviors.

## Run

```bash
cd test-server
uv run main.py
```

Optional:

```bash
PORT=8020 uv run main.py
RELOAD=1 uv run main.py
```

## Useful endpoints

- `GET /` HTML landing page with quick links.
- `ANY /requests/echo/{path}` Raw request echo for GET/POST/PUT/PATCH/DELETE.
- `POST /requests/json` JSON request inspector.
- `POST /requests/form` URL-encoded form inspector.
- `POST /requests/multipart` Multipart and file upload inspector.
- `POST /requests/binary` Binary body inspector.
- `POST /echo/json` Validate JSON and echo it back as `application/json`.
- `POST /echo/text` Echo a UTF-8 text body back as `text/plain`.
- `POST /echo/html` Echo markup back as `text/html`.
- `POST /echo/xml` Echo XML back as `application/xml`.
- `POST /echo/form` Echo URL-encoded fields back as `application/x-www-form-urlencoded`.
- `POST /echo/multipart` Echo the raw multipart payload back with its original boundary.
- `POST /echo/binary` Echo raw bytes back with the original content type.
- `GET /responses/json` JSON response.
- `GET /responses/text` Plain text response.
- `GET /responses/html` HTML response.
- `GET /responses/xml` XML response.
- `GET /responses/binary` Binary attachment response.
- `GET /responses/image.svg` SVG image response.
- `GET /responses/stream` Chunked text streaming response.
- `GET /responses/sse` Server-sent events response.
- `GET /responses/status/{status_code}` Arbitrary status code.
- `GET /responses/redirect` Redirect response.
- `GET /responses/cookies` Cookie-setting response.
- `GET /responses/delay/{milliseconds}` Delayed response.
- `WS /ws/echo` Echo text, JSON, and binary frames.
- `WS /ws/ticker` Periodic server-push frames, then clean close.
- `WS /ws/room/{room}` Multi-client room broadcast endpoint.

## Sample curl commands

```bash
curl -i http://127.0.0.1:8010/responses/json?items=4
curl -i http://127.0.0.1:8010/responses/html?title=Inspector+Preview
curl -i http://127.0.0.1:8010/responses/stream?chunks=5\&interval_ms=200
curl -i -X POST http://127.0.0.1:8010/requests/json \
  -H 'content-type: application/json' \
  -d '{"message":"hello","count":3}'
curl -i -X POST http://127.0.0.1:8010/requests/form \
  -H 'content-type: application/x-www-form-urlencoded' \
  -d 'name=portr&mode=form'
curl -i -X POST http://127.0.0.1:8010/requests/multipart \
  -F 'note=upload-test' \
  -F 'file=@README.md'
curl -i -X POST http://127.0.0.1:8010/echo/json \
  -H 'content-type: application/json' \
  -d '{"kind":"echo","nested":{"ok":true}}'
curl -i -X POST http://127.0.0.1:8010/echo/form \
  -H 'content-type: application/x-www-form-urlencoded' \
  -d 'name=portr&mode=echo&mode=repeat'
curl -i -X POST http://127.0.0.1:8010/echo/multipart \
  -F 'note=echo-test' \
  -F 'file=@README.md'
curl -i -X POST http://127.0.0.1:8010/echo/binary \
  -H 'content-type: application/octet-stream' \
  --data-binary @README.md
```

## WebSocket examples

Browser:

```js
const ws = new WebSocket("ws://127.0.0.1:8010/ws/echo?welcome=1")
ws.onmessage = (event) => console.log("message", event.data)
ws.onclose = (event) => console.log("closed", event.code, event.reason)
ws.send("hello")
ws.send(JSON.stringify({ kind: "json", ok: true }))
```

Ticker:

```js
const ws = new WebSocket("ws://127.0.0.1:8010/ws/ticker?count=5&interval_ms=500")
```

Room broadcast:

```js
const ws = new WebSocket("ws://127.0.0.1:8010/ws/room/demo?client=alice")
```
