---
title: Portr server setup
description: Guide to setting up portr server
---

### Prerequisites

- A virtual machine with docker installed (Hetzner 4GB 2 vCPU is cheap)
- [Cloudflare API token](/server/cloudflare-api-token/) - Required for wildcard subdomain SSL setup
- [Github oauth app credentials](/server/github-oauth-app/) - Required for admin dashboard login
- Port `2222` open on the server to accept incoming ssh connections
- Port range `30001-40001` open on the server to accept incoming tcp connections (only if you intend to use tcp tunnels)
