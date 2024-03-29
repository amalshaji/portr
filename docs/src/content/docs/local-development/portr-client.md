---
title: Setup portr client in local
---

The portr server is built using go for the backend and svelte for the portr inspector.

## Requirements

- go (1.22+)
- pnpm (16+)
- [admin server](/local-development/admin/)
- [tunnel server](/local-development/tunnel-server/)

## Frontend setup

### Installation

```shell
make installclient
```

### Start the client

```shell
make runclient
```

## Cli setup

Build the binary

```shell
make buildcli
```

Login to the admin, copy your secret key and add it to client.dev.yaml.

Start the tunnel connection

```shell
./portr -c client.dev.yaml http 9999
```
