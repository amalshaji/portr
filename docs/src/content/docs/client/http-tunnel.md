---
title: HTTP tunnel
---

Use the following command to tunnel an http connection.

```bash
portr http 9000
```

Or start it with a custom subdomain

```bash
portr http 9000 --subdomain amal-test
```

This also starts the portr inspector on [http://localhost:7777](http://localhost:7777), where you can inspect and replay http requests.
