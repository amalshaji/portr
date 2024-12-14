FROM node:20-slim AS frontend-builder

WORKDIR /app

COPY web/package.json web/pnpm-lock.yaml ./

RUN npm i -g pnpm@9.7.0 && pnpm install --frozen-lockfile

COPY web .

RUN pnpm build

FROM python:3.12 AS builder

ENV PATH="/app/.venv/bin:$PATH"

WORKDIR /app

COPY pyproject.toml requirements.lock README.md ./

RUN pip install --no-cache-dir uv==0.5.2

RUN uv venv && \
    uv pip install --no-cache -r requirements.lock

FROM python:3.12-slim AS final

LABEL maintainer="Amal Shaji" \
    org.opencontainers.image.title="Portr Admin" \
    org.opencontainers.image.description="Admin server for Portr" \
    org.opencontainers.image.source="https://github.com/amalshaji/portr"

ENV PATH="/app/.venv/bin:$PATH" \
    PYTHONPATH="/app/:$PYTHONPATH"

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/.venv/ /app/.venv/
COPY --from=frontend-builder /app/dist /app/web/dist
COPY . .

ENTRYPOINT ["sh", "scripts/start.sh"]
