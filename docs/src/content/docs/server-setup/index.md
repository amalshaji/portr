---
title: Portr server setup
description: Guide to setting up portr server
---

### Docker images

- [tunnel](https://hub.docker.com/r/amalshaji/portr-tunnel/tags)
- [admin](https://hub.docker.com/r/amalshaji/portr-admin/tags)

### Prerequisites

- [Cloudflare API token](/server-setup/cloudflare-api-token/) - Required for wildcard subdomain SSL setup
- [Github oauth app credentials](/server-setup/github-oauth-app/) - Required for admin dashboard login
- Port `2222` open on the server to accept incoming ssh connections
- Port range `30001-40001` open on the server to accept incoming tcp connections (only if you intend to use tcp tunnels)

### Quick setup

For quick setup, use the `docker-compose.yml`.

[https://github.com/amalshaji/portr/blob/main/docker-compose.yaml](https://github.com/amalshaji/portr/blob/main/docker-compose.yaml)

The compose file has 4 services

- caddy - the reverse proxy
- admin - the admin server
- tunnel - the tunnel server
- postgres - the postgres database

### Setup environment variables

Once you copy the compose file, create a `.env` with the following keys.

```text
GITHUB_APP_CLIENT_ID=
GITHUB_APP_CLIENT_SECRET=

DOMAIN=example.com
DB_URL=postgres://postgres:postgres@localhost:5432/postgres

SERVER_URL=example.com
SSH_URL=example.com:2222
CLOUDFLARE_API_TOKEN=

POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=postgres
```

If you want to run postgres separately and not as a service, you can exclude the following environment values

- POSTGRES_USER
- POSTGRES_PASSWORD
- POSTGRES_DB

Run `docker compose up` to start the servers. Once the servers are up, go to example.com and login in to the admin.
First login will be treated as a superuser.
