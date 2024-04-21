---
title: Server setup
---


For quick setup, use the `docker-compose.yml`.

[https://github.com/amalshaji/portr/blob/main/docker-compose.yaml](https://github.com/amalshaji/portr/blob/main/docker-compose.yaml)

### Services

The compose file has 4 services

- caddy - the reverse proxy
- admin - the admin server
- tunnel - the tunnel server
- postgres - the postgres database

### Setup environment variables

Once you copy the compose file, create a `.env` with the following keys.

```shell
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
PORTR_ADMIN_REMOTE_USER_HEADER=
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

### Reverse Proxy

#### Configure Auth Proxy Authentication

You can configure portr's admin interface to trust an HTTP reverse proxy
to handle authentication.  Web servers and reverse proxies have many
authentication integrations, and any of those can then be used with portr.

Even with auth proxy authentication, users are not automatically provisioned.
Except for the superuser - which is provisioned automatically - all users
must be invited to a team to use portr.

> [!WARNING]
> If you use this feature the portr admin interface **MUST**
> only be accessible via the appropriate auth proxy.
>
> A failure here will allow actors to spoof their identity.

To activate this feature, configure your reverse proxy to authenticate
and pass the authenticate user's email as a header.  This varies from
solution to solution, so consult your reverse proxy's documentation on
how to set it up.

Once you've confirmed that to be working you may set the environment
variable `PORTR_ADMIN_REMOTE_USER_HEADER` in your `.env` with the
value of the header that will contain the user's email.
