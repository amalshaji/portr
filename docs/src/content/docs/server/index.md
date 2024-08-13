---
title: Portr server setup
description: Guide to setting up portr server
---

### Prerequisites

- A virtual machine with docker installed (Hetzner 4GB 2 vCPU is cheap)
- DNS records for example.com (or your domain)
    | Type | Name | Value |
    |---|---|---|
    | A | @ | your-server-ipv4 |
    | A | * | your-server-ipv4 |
- [Cloudflare API token](/server/cloudflare-api-token/) - Required for wildcard subdomain SSL setup
- [Github oauth app credentials](/server/github-oauth-app/) - For admin dashboard login (**optional**)
- Port `2222` open on the server to accept incoming ssh connections
- Port range `30001-40001` open on the server to accept incoming tcp connections (only if you intend to use tcp tunnels)
