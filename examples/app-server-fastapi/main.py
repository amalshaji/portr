# /// script
# dependencies = [
#   "fastapi>=0.115.0",
#   "uvicorn>=0.30.0",
# ]
# ///

from __future__ import annotations

import asyncio
import json
import os
import time
import urllib.error
import urllib.request
from contextlib import asynccontextmanager
from typing import Any

from fastapi import FastAPI


APP_SERVER = os.getenv("PORTR_APP_SERVER", "http://127.0.0.1:7778").rstrip("/")
APP_SERVER_TOKEN = os.getenv("PORTR_APP_SERVER_TOKEN")
FASTAPI_HOST = os.getenv("FASTAPI_HOST", "127.0.0.1")
FASTAPI_PORT = int(os.getenv("FASTAPI_PORT", "8000"))
TUNNEL_NAME = os.getenv("PORTR_TUNNEL_NAME", "fastapi-example")
TUNNEL_SUBDOMAIN = os.getenv("PORTR_TUNNEL_SUBDOMAIN")


def app_server_request(
    method: str,
    path: str,
    payload: dict[str, Any] | None = None,
) -> dict[str, Any]:
    data = json.dumps(payload).encode("utf-8") if payload is not None else None
    headers = {"Content-Type": "application/json"}
    if APP_SERVER_TOKEN:
        headers["Authorization"] = f"Bearer {APP_SERVER_TOKEN}"

    request = urllib.request.Request(
        f"{APP_SERVER}{path}",
        data=data,
        headers=headers,
        method=method,
    )

    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            body = response.read().decode("utf-8")
    except urllib.error.HTTPError as error:
        body = error.read().decode("utf-8")
        raise RuntimeError(
            f"app-server {method} {path} failed with {error.code}: {body}"
        ) from error
    except urllib.error.URLError as error:
        raise RuntimeError(
            f"could not reach portr app-server at {APP_SERVER}: {error.reason}"
        ) from error

    if not body:
        return {}
    return json.loads(body)


def create_tunnel() -> dict[str, Any]:
    payload: dict[str, Any] = {
        "name": TUNNEL_NAME,
        "type": "http",
        "host": FASTAPI_HOST,
        "port": FASTAPI_PORT,
    }
    if TUNNEL_SUBDOMAIN:
        payload["subdomain"] = TUNNEL_SUBDOMAIN

    tunnel = app_server_request("POST", "/api/v1/tunnels", payload)
    try:
        return wait_for_tunnel_url(tunnel["id"])
    except Exception:
        delete_tunnel(tunnel["id"])
        raise


def wait_for_tunnel_url(tunnel_id: str) -> dict[str, Any]:
    deadline = time.monotonic() + 20
    latest: dict[str, Any] = {}

    while time.monotonic() < deadline:
        latest = app_server_request("GET", f"/api/v1/tunnels/{tunnel_id}")
        if latest.get("status") == "failed":
            raise RuntimeError(
                f"tunnel failed: {latest.get('last_error', 'unknown error')}"
            )
        if latest.get("status") == "running" and latest.get("tunnel_url"):
            return latest
        time.sleep(0.5)

    raise RuntimeError(f"tunnel did not become ready in time: {latest}")


def delete_tunnel(tunnel_id: str) -> None:
    try:
        app_server_request("DELETE", f"/api/v1/tunnels/{tunnel_id}")
    except Exception as error:
        print(f"Failed to delete tunnel {tunnel_id}: {error}", flush=True)


@asynccontextmanager
async def lifespan(app: FastAPI):
    tunnel = await asyncio.to_thread(create_tunnel)
    app.state.tunnel = tunnel

    print(f"FastAPI local URL: http://{FASTAPI_HOST}:{FASTAPI_PORT}", flush=True)
    print(f"Portr tunnel URL: {tunnel['tunnel_url']}", flush=True)

    try:
        yield
    finally:
        tunnel_id = app.state.tunnel["id"]
        print(f"Deleting Portr tunnel: {tunnel_id}", flush=True)
        await asyncio.to_thread(delete_tunnel, tunnel_id)


app = FastAPI(lifespan=lifespan)


@app.get("/")
def read_root() -> dict[str, Any]:
    tunnel = getattr(app.state, "tunnel", {})
    return {
        "message": "Hello from FastAPI through Portr",
        "tunnel_url": tunnel.get("tunnel_url"),
    }


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn

    print("Make sure `portr app-server` is running before starting this example.")
    uvicorn.run(app, host=FASTAPI_HOST, port=FASTAPI_PORT)
