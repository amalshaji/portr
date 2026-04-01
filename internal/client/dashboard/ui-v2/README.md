# Portr Client Dashboard UI

React + TypeScript + Vite + shadcn app for the client inspector dashboard.

## Development

```bash
bun install
bun run dev
```

## Build

```bash
bun run build
bun run lint
```

## Adding shadcn components

Use the shadcn CLI instead of hand-rolling primitives:

```bash
bunx shadcn@latest add button
```

## Local test server

A standalone FastAPI harness is available in `test-server/`.

It is a single `uv run` script with inline dependencies, so you can exercise
the dashboard against JSON, forms, multipart uploads, binary payloads, HTML,
streaming responses, SSE, and WebSocket frame traffic without managing a
separate Python environment by hand.
