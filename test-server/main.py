# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "fastapi>=0.115,<1.0",
#   "uvicorn[standard]>=0.34,<1.0",
#   "python-multipart>=0.0.9,<1.0",
# ]
# ///

from __future__ import annotations

import asyncio
import base64
import hashlib
import json
import os
from collections import defaultdict
from contextlib import suppress
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.parse import urlencode

import uvicorn
from fastapi import FastAPI, Request, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import (
    HTMLResponse,
    JSONResponse,
    PlainTextResponse,
    RedirectResponse,
    Response,
    StreamingResponse,
)
from starlette.datastructures import UploadFile

app = FastAPI(
    title="Portr Dashboard Test Server",
    summary="FastAPI playground for HTTP and WebSocket inspector testing.",
    version="1.0.0",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

room_connections: dict[str, set[WebSocket]] = defaultdict(set)
room_lock = asyncio.Lock()


def now_iso() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def collapse_multi_items(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    grouped: dict[str, list[Any]] = defaultdict(list)
    for key, value in pairs:
        grouped[key].append(value)

    result: dict[str, Any] = {}
    for key, values in grouped.items():
        result[key] = values[0] if len(values) == 1 else values
    return result


def decode_text_preview(payload: bytes, limit: int = 1000) -> str | None:
    if not payload:
        return ""

    preview = payload[:limit]
    try:
        text = preview.decode("utf-8")
    except UnicodeDecodeError:
        return None

    if len(payload) > limit:
        return f"{text}... [truncated]"
    return text


def make_payload_summary(payload: bytes) -> dict[str, Any]:
    return {
        "size_bytes": len(payload),
        "sha256": hashlib.sha256(payload).hexdigest(),
        "text_preview": decode_text_preview(payload),
        "base64_preview": base64.b64encode(payload[:256]).decode("ascii"),
    }


def request_snapshot(request: Request, body: bytes) -> dict[str, Any]:
    return {
        "method": request.method,
        "url": str(request.url),
        "path": request.url.path,
        "query": collapse_multi_items(list(request.query_params.multi_items())),
        "headers": dict(request.headers),
        "cookies": request.cookies,
        "client": {
            "host": request.client.host if request.client else None,
            "port": request.client.port if request.client else None,
        },
        "content_type": request.headers.get("content-type", ""),
        "body": make_payload_summary(body),
        "timestamp": now_iso(),
    }


def request_content_type(request: Request, fallback: str) -> str:
    content_type = request.headers.get("content-type", "").strip()
    return content_type or fallback


def html_index() -> str:
    return """<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Portr Dashboard Test Server</title>
    <style>
      :root {
        color-scheme: light dark;
        font-family: ui-sans-serif, system-ui, sans-serif;
      }
      body {
        margin: 0;
        padding: 2rem;
        background: #0f172a;
        color: #e2e8f0;
      }
      main {
        max-width: 72rem;
        margin: 0 auto;
      }
      h1, h2 {
        margin-bottom: 0.5rem;
      }
      p, li {
        line-height: 1.6;
      }
      .panel {
        border: 1px solid rgba(148, 163, 184, 0.3);
        background: rgba(15, 23, 42, 0.55);
        padding: 1rem 1.25rem;
        margin: 1rem 0;
      }
      code {
        font-family: ui-monospace, SFMono-Regular, monospace;
      }
      a {
        color: #7dd3fc;
      }
      ul {
        padding-left: 1.25rem;
      }
    </style>
  </head>
  <body>
    <main>
      <h1>Portr Dashboard Test Server</h1>
      <p>
        Use this app to generate HTTP traffic, multipart uploads, streams,
        redirects, cookies, and WebSocket frames for the inspector UI.
      </p>

      <div class="panel">
        <h2>HTTP endpoints</h2>
        <ul>
          <li><a href="/responses/json">/responses/json</a></li>
          <li><a href="/responses/text">/responses/text</a></li>
          <li><a href="/responses/html">/responses/html</a></li>
          <li><a href="/responses/xml">/responses/xml</a></li>
          <li><a href="/responses/image.svg">/responses/image.svg</a></li>
          <li><a href="/responses/binary">/responses/binary</a></li>
          <li><a href="/responses/stream">/responses/stream</a></li>
          <li><a href="/responses/sse">/responses/sse</a></li>
          <li><a href="/responses/cookies">/responses/cookies</a></li>
          <li><a href="/responses/redirect">/responses/redirect</a></li>
        </ul>
      </div>

      <div class="panel">
        <h2>Request inspectors</h2>
        <p><code>POST /requests/json</code></p>
        <p><code>POST /requests/form</code></p>
        <p><code>POST /requests/multipart</code></p>
        <p><code>ANY /requests/echo/demo/path?source=index</code></p>
        <p><code>POST /requests/binary</code></p>
      </div>

      <div class="panel">
        <h2>Payload echo endpoints</h2>
        <p><code>POST /echo/json</code></p>
        <p><code>POST /echo/text</code></p>
        <p><code>POST /echo/html</code></p>
        <p><code>POST /echo/xml</code></p>
        <p><code>POST /echo/form</code></p>
        <p><code>POST /echo/multipart</code></p>
        <p><code>POST /echo/binary</code></p>
      </div>

      <div class="panel">
        <h2>WebSocket endpoints</h2>
        <p><code>ws://host/ws/echo?welcome=1</code></p>
        <p><code>ws://host/ws/ticker?count=5&interval_ms=500</code></p>
        <p><code>ws://host/ws/room/demo?client=alice</code></p>
      </div>
    </main>
  </body>
</html>"""


async def broadcast_room(room: str, message: dict[str, Any]) -> None:
    payload = json.dumps(message)
    async with room_lock:
        sockets = list(room_connections[room])

    stale: list[WebSocket] = []
    for socket in sockets:
        try:
            await socket.send_text(payload)
        except RuntimeError:
            stale.append(socket)

    if stale:
        async with room_lock:
            for socket in stale:
                room_connections[room].discard(socket)


@app.get("/", response_class=HTMLResponse)
async def index() -> str:
    return html_index()


@app.get("/healthz")
async def healthz() -> dict[str, Any]:
    return {"ok": True, "timestamp": now_iso()}


@app.api_route(
    "/requests/echo/{tail:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"],
)
async def echo_request(request: Request, tail: str) -> JSONResponse:
    body = await request.body()
    payload = request_snapshot(request, body)
    payload["tail"] = tail
    return JSONResponse(payload)


@app.post("/requests/json")
async def inspect_json(request: Request) -> JSONResponse:
    body = await request.body()
    payload = request_snapshot(request, body)
    try:
        payload["json"] = json.loads(body.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        payload["json_error"] = str(error)
    return JSONResponse(payload)


@app.post("/requests/form")
async def inspect_form(request: Request) -> JSONResponse:
    body = await request.body()
    form = await request.form()
    payload = request_snapshot(request, body)
    payload["form"] = collapse_multi_items(list(form.multi_items()))
    return JSONResponse(payload)


@app.post("/requests/multipart")
async def inspect_multipart(request: Request) -> JSONResponse:
    body = await request.body()
    form = await request.form()
    files: list[dict[str, Any]] = []
    fields: list[tuple[str, Any]] = []

    for key, value in form.multi_items():
        if isinstance(value, UploadFile):
            file_bytes = await value.read()
            files.append(
                {
                    "field": key,
                    "filename": value.filename,
                    "content_type": value.content_type,
                    "size_bytes": len(file_bytes),
                    "sha256": hashlib.sha256(file_bytes).hexdigest(),
                    "text_preview": decode_text_preview(file_bytes, limit=300),
                }
            )
            continue
        fields.append((key, value))

    payload = request_snapshot(request, body)
    payload["fields"] = collapse_multi_items(fields)
    payload["files"] = files
    return JSONResponse(payload)


@app.post("/requests/binary")
async def inspect_binary(request: Request) -> JSONResponse:
    body = await request.body()
    payload = request_snapshot(request, body)
    return JSONResponse(payload)


@app.post("/echo/json")
async def echo_json(request: Request) -> Response:
    body = await request.body()
    try:
        json.loads(body.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        return JSONResponse(
            {"message": "invalid JSON payload", "detail": str(error)},
            status_code=400,
        )

    return Response(
        body,
        media_type="application/json",
        headers={"X-Portr-Test": "echo-json"},
    )


@app.post("/echo/text")
async def echo_text(request: Request) -> Response:
    body = await request.body()
    try:
        body.decode("utf-8")
    except UnicodeDecodeError as error:
        return JSONResponse(
            {"message": "invalid UTF-8 text payload", "detail": str(error)},
            status_code=400,
        )

    return Response(
        body,
        media_type=request_content_type(request, "text/plain; charset=utf-8"),
        headers={"X-Portr-Test": "echo-text"},
    )


@app.post("/echo/html")
async def echo_html(request: Request) -> Response:
    body = await request.body()
    try:
        body.decode("utf-8")
    except UnicodeDecodeError as error:
        return JSONResponse(
            {"message": "invalid UTF-8 HTML payload", "detail": str(error)},
            status_code=400,
        )

    return Response(
        body,
        media_type=request_content_type(request, "text/html; charset=utf-8"),
        headers={"X-Portr-Test": "echo-html"},
    )


@app.post("/echo/xml")
async def echo_xml(request: Request) -> Response:
    body = await request.body()
    try:
        body.decode("utf-8")
    except UnicodeDecodeError as error:
        return JSONResponse(
            {"message": "invalid UTF-8 XML payload", "detail": str(error)},
            status_code=400,
        )

    return Response(
        body,
        media_type=request_content_type(request, "application/xml"),
        headers={"X-Portr-Test": "echo-xml"},
    )


@app.post("/echo/form")
async def echo_form(request: Request) -> Response:
    form = await request.form()
    fields: list[tuple[str, str]] = []

    for key, value in form.multi_items():
        if isinstance(value, UploadFile):
            return JSONResponse(
                {
                    "message": "file uploads are not supported on /echo/form",
                    "hint": "use /echo/multipart for file payloads",
                },
                status_code=400,
            )
        fields.append((key, value))

    return Response(
        urlencode(fields, doseq=True),
        media_type="application/x-www-form-urlencoded",
        headers={"X-Portr-Test": "echo-form"},
    )


@app.post("/echo/multipart")
async def echo_multipart(request: Request) -> Response:
    body = await request.body()
    form = await request.form()
    for _, value in form.multi_items():
        if isinstance(value, UploadFile):
            await value.close()

    return Response(
        body,
        media_type=request_content_type(request, "multipart/form-data"),
        headers={"X-Portr-Test": "echo-multipart"},
    )


@app.post("/echo/binary")
async def echo_binary(request: Request) -> Response:
    body = await request.body()
    return Response(
        body,
        media_type=request_content_type(request, "application/octet-stream"),
        headers={"X-Portr-Test": "echo-binary"},
    )


@app.get("/responses/json")
async def json_response(items: int = 5) -> JSONResponse:
    items = min(max(items, 1), 50)
    payload = {
        "message": "Structured JSON response",
        "timestamp": now_iso(),
        "items": [
            {
                "id": index + 1,
                "label": f"item-{index + 1}",
                "active": index % 2 == 0,
            }
            for index in range(items)
        ],
    }
    return JSONResponse(payload, headers={"X-Portr-Test": "json"})


@app.get("/responses/text")
async def text_response(lines: int = 12) -> PlainTextResponse:
    lines = min(max(lines, 1), 200)
    body = "\n".join(
        f"line {index + 1}: generated at {now_iso()}" for index in range(lines)
    )
    return PlainTextResponse(body, headers={"X-Portr-Test": "text"})


@app.get("/responses/html")
async def html_response(title: str = "Portr HTML Preview") -> HTMLResponse:
    markup = f"""<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>{title}</title>
  </head>
  <body>
    <main>
      <h1>{title}</h1>
      <p>This HTML response is useful for iframe rendering checks.</p>
      <table border="1" cellpadding="8">
        <tr><th>Type</th><th>Value</th></tr>
        <tr><td>timestamp</td><td>{now_iso()}</td></tr>
        <tr><td>status</td><td>ok</td></tr>
      </table>
    </main>
  </body>
</html>"""
    return HTMLResponse(markup, headers={"X-Portr-Test": "html"})


@app.get("/responses/xml")
async def xml_response(items: int = 3) -> Response:
    items = min(max(items, 1), 20)
    body = [
        '<?xml version="1.0" encoding="UTF-8"?>',
        "<response>",
        f"  <timestamp>{now_iso()}</timestamp>",
        "  <items>",
    ]
    body.extend(
        f'    <item id="{index + 1}">item-{index + 1}</item>'
        for index in range(items)
    )
    body.extend(["  </items>", "</response>"])
    return Response(
        "\n".join(body),
        media_type="application/xml",
        headers={"X-Portr-Test": "xml"},
    )


@app.get("/responses/binary")
async def binary_response(size: int = 512, filename: str = "payload.bin") -> Response:
    size = min(max(size, 1), 1024 * 1024)
    payload = bytes(index % 256 for index in range(size))
    headers = {
        "Content-Disposition": f'attachment; filename="{filename}"',
        "X-Portr-Test": "binary",
    }
    return Response(payload, media_type="application/octet-stream", headers=headers)


@app.get("/responses/image.svg")
async def svg_response(label: str = "Portr", width: int = 480, height: int = 200) -> Response:
    width = min(max(width, 120), 1600)
    height = min(max(height, 80), 1000)
    safe_label = label.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
    svg = f"""<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">
  <defs>
    <linearGradient id="bg" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#0f172a" />
      <stop offset="100%" stop-color="#0369a1" />
    </linearGradient>
  </defs>
  <rect width="{width}" height="{height}" fill="url(#bg)" />
  <circle cx="{height // 2}" cy="{height // 2}" r="{height // 4}" fill="#38bdf8" opacity="0.45" />
  <text x="40" y="{height // 2}" fill="#e0f2fe" font-family="ui-sans-serif, system-ui, sans-serif" font-size="32" dominant-baseline="middle">{safe_label}</text>
  <text x="40" y="{height // 2 + 42}" fill="#bae6fd" font-family="ui-monospace, monospace" font-size="16">generated {now_iso()}</text>
</svg>"""
    return Response(svg, media_type="image/svg+xml", headers={"X-Portr-Test": "svg"})


@app.get("/responses/status/{status_code}")
async def arbitrary_status(status_code: int, body_type: str = "json") -> Response:
    if body_type == "text":
        return PlainTextResponse(
            f"status {status_code} generated at {now_iso()}",
            status_code=status_code,
            headers={"X-Portr-Test": "status"},
        )
    if body_type == "empty":
        return Response(status_code=status_code, headers={"X-Portr-Test": "status"})
    return JSONResponse(
        {"status_code": status_code, "timestamp": now_iso()},
        status_code=status_code,
        headers={"X-Portr-Test": "status"},
    )


@app.get("/responses/redirect")
async def redirect_response(target: str = "/responses/json") -> RedirectResponse:
    return RedirectResponse(url=target, status_code=307)


@app.get("/responses/cookies")
async def cookie_response() -> Response:
    response = JSONResponse(
        {"message": "cookies set", "timestamp": now_iso()},
        headers={"X-Portr-Test": "cookies"},
    )
    response.set_cookie("portr-session", "demo-session", httponly=True, samesite="lax")
    response.set_cookie("portr-mode", "inspector")
    return response


@app.get("/responses/delay/{milliseconds}")
async def delayed_response(milliseconds: int) -> JSONResponse:
    milliseconds = min(max(milliseconds, 0), 30000)
    await asyncio.sleep(milliseconds / 1000)
    return JSONResponse(
        {
            "delayed_ms": milliseconds,
            "timestamp": now_iso(),
        },
        headers={"X-Portr-Test": "delay"},
    )


@app.get("/responses/stream")
async def stream_response(chunks: int = 5, interval_ms: int = 300) -> StreamingResponse:
    chunks = min(max(chunks, 1), 100)
    interval_ms = min(max(interval_ms, 0), 5000)

    async def iterator():
        for index in range(chunks):
            yield f"chunk {index + 1}/{chunks} at {now_iso()}\n"
            if interval_ms:
                await asyncio.sleep(interval_ms / 1000)

    return StreamingResponse(
        iterator(),
        media_type="text/plain",
        headers={"X-Portr-Test": "stream"},
    )


@app.get("/responses/sse")
async def sse_response(events: int = 5, interval_ms: int = 1000) -> StreamingResponse:
    events = min(max(events, 1), 100)
    interval_ms = min(max(interval_ms, 100), 5000)

    async def iterator():
        for index in range(events):
            payload = json.dumps({"index": index + 1, "timestamp": now_iso()})
            yield f"event: tick\ndata: {payload}\n\n"
            await asyncio.sleep(interval_ms / 1000)

    return StreamingResponse(
        iterator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Portr-Test": "sse",
        },
    )


@app.websocket("/ws/echo")
async def websocket_echo(websocket: WebSocket) -> None:
    await websocket.accept()

    if websocket.query_params.get("welcome", "1") != "0":
        await websocket.send_json(
            {
                "kind": "welcome",
                "message": "connected to /ws/echo",
                "timestamp": now_iso(),
            }
        )

    try:
        while True:
            message = await websocket.receive()
            text = message.get("text")
            binary = message.get("bytes")

            if text is not None:
                if text == "close":
                    await websocket.close(code=1000, reason="client requested close")
                    return

                with suppress(json.JSONDecodeError):
                    payload = json.loads(text)
                    await websocket.send_json(
                        {
                            "kind": "json-echo",
                            "received": payload,
                            "timestamp": now_iso(),
                        }
                    )
                    continue

                await websocket.send_text(f"echo:{text}")
                continue

            if binary is not None:
                await websocket.send_bytes(b"echo:" + binary)
    except WebSocketDisconnect:
        return


@app.websocket("/ws/ticker")
async def websocket_ticker(websocket: WebSocket) -> None:
    await websocket.accept()

    count = min(max(int(websocket.query_params.get("count", "5")), 1), 100)
    interval_ms = min(
        max(int(websocket.query_params.get("interval_ms", "500")), 50),
        10000,
    )

    await websocket.send_json(
        {
            "kind": "welcome",
            "message": "ticker stream starting",
            "count": count,
            "interval_ms": interval_ms,
            "timestamp": now_iso(),
        }
    )

    for index in range(count):
        await websocket.send_json(
            {
                "kind": "tick",
                "index": index + 1,
                "count": count,
                "timestamp": now_iso(),
            }
        )
        await asyncio.sleep(interval_ms / 1000)

    await websocket.close(code=1000, reason="ticker complete")


@app.websocket("/ws/room/{room}")
async def websocket_room(websocket: WebSocket, room: str) -> None:
    client_name = websocket.query_params.get("client", "anonymous")
    await websocket.accept()

    async with room_lock:
        room_connections[room].add(websocket)
        room_size = len(room_connections[room])

    await broadcast_room(
        room,
        {
            "kind": "join",
            "room": room,
            "client": client_name,
            "connections": room_size,
            "timestamp": now_iso(),
        },
    )

    try:
        while True:
            text = await websocket.receive_text()
            await broadcast_room(
                room,
                {
                    "kind": "message",
                    "room": room,
                    "client": client_name,
                    "text": text,
                    "timestamp": now_iso(),
                },
            )
    except WebSocketDisconnect:
        async with room_lock:
            room_connections[room].discard(websocket)
            remaining = len(room_connections[room])
            if remaining == 0:
                room_connections.pop(room, None)

        await broadcast_room(
            room,
            {
                "kind": "leave",
                "room": room,
                "client": client_name,
                "connections": remaining,
                "timestamp": now_iso(),
            },
        )


if __name__ == "__main__":
    host = os.environ.get("HOST", "127.0.0.1")
    port = int(os.environ.get("PORT", "8010"))
    reload_enabled = os.environ.get("RELOAD") == "1"

    if reload_enabled:
        os.chdir(Path(__file__).resolve().parent)
        uvicorn.run("main:app", host=host, port=port, reload=True)
    else:
        uvicorn.run(app, host=host, port=port)
