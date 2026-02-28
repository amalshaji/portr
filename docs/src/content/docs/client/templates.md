---
title: Tunnel templates
description: Learn how to setup tunnel templates to reuse tunnel settings
---

### Why templates

- Run multiple tunnels at the same time.
- If you use certain subdomains/port regularly, it is easier to create them as services and reuse using simple commands.

Open the portr client config file by running the following command

```bash
portr config edit
```

This should open a file with the following contents

```yaml
server_url: example.com
ssh_url: example.com:2222
secret_key: { your-secret-key }
connection_log_retention_days: 0
tunnels:
  - name: portr
    subdomain: portr
    port: 4321
```

You can create tunnel templates under the tunnels key to quickly start them.
For example, you can start the portr tunnel using `portr start portr`. You can also add a tcp connection by specifying the type of the connection.

```yaml
tunnels:
  - name: portr
    subdomain: portr
    port: 4321
  - name: pg
    subdomain: portr
    port: 5432
    type: tcp
```

And start multiple services by using the command `portr start portr pg`.

To start all the services, use the command `portr start`.

Set `connection_log_retention_days` to a value greater than `0` to auto-clear old connection logs. Keep it `0` to disable auto-cleanup.

For more details, run `portr --help`.
