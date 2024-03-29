---
title: Setup tunnel server in local
---

The tunnel server is built using go. It uses ssh remote port forwarding to tunnel http/tcp connections.

## Requirements

- go (1.22+)
- postgres (16+)

## Setup

Create a new `.env` using the `.env.template` file.

### Start the server

```shell
make runserver
```

You should see the following message

```shell
time=2024-03-29T19:16:35.023+05:30 level=INFO msg="starting SSH server" port=:2222
time=2024-03-29T19:16:35.023+05:30 level=INFO msg="starting proxy server" port=:8001
time=2024-03-29T19:16:35.023+05:30 level=INFO msg="Starting 1 cron jobs"
```

This starts the ssh server on port `:2222` and proxy server on port `:8001`

For all configuration variables, check out the [tunnel server config file](https://github.com/amalshaji/portr/blob/main/tunnel/internal/server/config/config.go).
