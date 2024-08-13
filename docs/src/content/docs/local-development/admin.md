---
title: Setup admin in local
description: Learn how to setup portr admin for local development
---

The admin is built using python for the backend and svelte for the frontend.

## Requirements

- [rye](https://github.com/astral-sh/rye) (0.32.0+)
- pnpm (8.7.5+)
- postgres (16+)

## Frontend setup

### Installation

```shell
make installclient
```

### Start the client

```shell
make runclient
```

## Backend setup

Inside the admin folder, run the following

```shell
rye sync
```

This sets up the relevant python version and install packages in a virtual environment.

Create a new `.env` using the `.env.template` file. Make sure the following environment variables are setup,

- PORTR_ADMIN_ENCRYPTION_KEY
- PORTR_ADMIN_GITHUB_CLIENT_ID
- PORTR_ADMIN_GITHUB_CLIENT_SECRET

### Start the server

```shell
make runserver
```

This should run the migrations and start the server. You can access the server at [http://localhost:8000](http://localhost:8000)

For more commands, check out the [admin makefile](https://github.com/amalshaji/portr/blob/main/admin/Makefile).

For settings, check out the [admin config file](https://github.com/amalshaji/portr/blob/main/admin/config/settings.py).
