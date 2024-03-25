---
title: Portr server setup
description: Guide to setting up portr server
---

### Prerequisites

- [Cloudflare API token](/server-setup/cloudflare-api-token/) - Required for wildcard subdomain SSL setup
- [Github oauth app credentials](/server-setup/github-oauth-app/) - Required for admin dashboard login
- Port `2222` open on the server to accept incoming ssh connections
- Port range `30001-40001` open on the server to accept incoming tcp connections (only if you intend to use tcp tunnels)
