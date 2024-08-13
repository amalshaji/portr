FROM node:20-slim as frontend-builder

WORKDIR /app

COPY web/package.json web/pnpm-lock.yaml ./

RUN npm i -g pnpm && pnpm install --frozen-lockfile

COPY web .

RUN pnpm build

FROM python:3.12 as builder

ENV PATH="/app/.venv/bin:$PATH"

WORKDIR /app

COPY requirements.lock .

RUN python3 -m venv .venv

RUN sed '/-e/d' requirements.lock > requirements.txt && pip install --no-cache-dir -r requirements.txt

FROM python:3.12-slim as final

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
