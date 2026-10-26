// Railway Infrastructure as Code for a portr server (websocket transport).
//
// Edit DOMAIN, then from a directory linked to your Railway project:
//
//   bun install --cwd .railway
//   railway config plan
//   railway config apply
//
// The portr service listens on two ports:
//   8000 -> admin dashboard + API              (DOMAIN)
//   8001 -> public proxy and the client tunnel  (*.DOMAIN)
//           transport at /_portr/tunnel/connect
//
// The wildcard has to be a custom domain; Railway-provided *.up.railway.app
// domains cannot be wildcarded. PORTR_DOMAIN is both the tunnel base domain and
// the admin domain (admin CORS and the GitHub OAuth callback derive from it), so
// the admin dashboard is served at the apex rather than on its own subdomain.
//
// TCP tunnels do not work on Railway: the server allocates a random public port
// per tunnel, and Railway only exposes ports through manually created TCP proxies.
import { defineRailway, postgres, project, service } from "railway/iac";

// A named partial limits this file to the resources it declares, so applying it
// never deletes other services, volumes or buckets in the project.
export const partial = "portr";

const DOMAIN = "portr.example.com";

export default defineRailway((ctx) => {
  const db = postgres("Postgres");

  const portr = service("portr", {
    build: { builder: "DOCKERFILE", dockerfilePath: "Dockerfile" },
    deploy: {
      // No startCommand: the image ENTRYPOINT (docker-entrypoint.sh, with CMD
      // "start all") must stay in charge of startup, because it is what restores
      // and replicates SQLite databases through Litestream. A startCommand would
      // replace the ENTRYPOINT and silently disable those backups.
      //
      // No preDeployCommand either: PORTR_AUTO_MIGRATE=true migrates on boot
      // using whichever database driver PORTR_DB_URL selects. A PostgreSQL-only
      // deployment that prefers migrating before taking traffic can use
      // preDeployCommand: ["./portrd migrate --dialect postgres"] instead, but
      // that command rejects SQLite databases.
      //
      // No healthcheckPath: Railway probes a single port, and 8001 answers
      // unknown hosts with 404.
      //
      // Tunnel sessions and proxy routes are held in memory per replica, so a
      // second replica would receive requests for subdomains it has no session for.
      numReplicas: 1,
    },
    domains: [
      { domain: DOMAIN, port: 8000 },
      { domain: `*.${DOMAIN}`, port: 8001 },
    ],
    env: {
      PORTR_DB_URL: db.env.DATABASE_URL,
      PORTR_DOMAIN: DOMAIN,
      PORTR_SERVER_URL: `https://${DOMAIN}`,
      PORTR_TRANSPORT: "websocket",
      // Any subdomain under the wildcard reaches 8001.
      PORTR_WS_URL: `tunnel.${DOMAIN}`,
      PORTR_ADMIN_PORT: "8000",
      PORTR_PROXY_PORT: "8001",
      PORTR_AUTO_MIGRATE: "true",
      // Variables missing from this list are deleted on apply. To keep GitHub
      // login, set these on the service first, import preserve and uncomment:
      // PORTR_ADMIN_GITHUB_CLIENT_ID: preserve(),
      // PORTR_ADMIN_GITHUB_CLIENT_SECRET: preserve(),
    },
  });

  return project(ctx.projectName ?? "portr", {
    resources: [db, portr],
  });
});
