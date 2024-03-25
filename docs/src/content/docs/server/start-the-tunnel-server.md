---
title: Server setup
---

### Docker images

- [tunnel](https://hub.docker.com/r/amalshaji/portr-tunnel/tags)
- [admin](https://hub.docker.com/r/amalshaji/portr-admin/tags)

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
PORTR_ADMIN_GITHUB_CLIENT_ID=
PORTR_ADMIN_GITHUB_CLIENT_SECRET=

PORTR_DOMAIN=example.com
PORTR_DB_URL=postgres://postgres:postgres@localhost:5432/postgres

PORTR_SERVER_URL=example.com
PORTR_SSH_URL=example.com:2222

CLOUDFLARE_API_TOKEN=

POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=postgres

PORTR_ADMIN_ENCRYPTION_KEY=
```

Generate an encryption key using the following command

```shell
python -c "import base64, os; print(base64.urlsafe_b64encode(os.urandom(32)).decode())"
```

If you want to run postgres separately and not as a service, you can exclude the following environment values

- POSTGRES_USER
- POSTGRES_PASSWORD
- POSTGRES_DB

Run `docker compose up` to start the servers. Once the servers are up, go to example.com and login in to the admin.
First login will be treated as a superuser.


