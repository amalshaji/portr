import Link from "next/link";
import { Logo } from "@/components/ui/logo";

const routePath = "M 214 86 C 214 150 332 132 332 222 C 332 302 218 286 218 412";

function ArrowRight({ className = "" }: { className?: string }) {
  return (
    <svg aria-hidden="true" className={className} fill="none" viewBox="0 0 20 20">
      <path
        d="M3.5 10h12m-4.5-4.5L15.5 10 11 14.5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.7"
      />
    </svg>
  );
}

function MenuIcon() {
  return (
    <svg aria-hidden="true" fill="none" viewBox="0 0 24 24">
      <path d="M4 7h16M4 12h16M4 17h16" stroke="currentColor" strokeLinecap="round" strokeWidth="1.8" />
    </svg>
  );
}

function GlobeIcon() {
  return (
    <svg aria-hidden="true" fill="none" viewBox="0 0 64 64">
      <circle cx="32" cy="32" r="23" stroke="currentColor" strokeWidth="2.6" />
      <path d="M9 32h46M32 9c8 7 12 14.7 12 23S40 48 32 55M32 9c-8 7-12 14.7-12 23s4 16 12 23M32 9v46" stroke="currentColor" strokeWidth="2" />
    </svg>
  );
}

function LaptopIcon() {
  return (
    <svg aria-hidden="true" fill="none" viewBox="0 0 72 58">
      <rect height="38" rx="2" stroke="currentColor" strokeWidth="2.4" width="54" x="9" y="4" />
      <path d="m24 24 6-6m-6 6 6 6m18-12 6 6-6 6m-10 2 7-16M4 49h64l-5 5H9l-5-5Z" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.2" />
    </svg>
  );
}

function TunnelMap() {
  return (
    <figure className="portr-map" aria-labelledby="portr-map-caption">
      <svg aria-hidden="true" className="portr-map__route" viewBox="0 0 440 510">
        <path className="portr-map__path" d={routePath} />
        <path className="portr-map__path portr-map__path--under" d={routePath} />
        <circle className="portr-map__packet" r="7">
          <animateMotion dur="4.4s" path={routePath} repeatCount="indefinite" />
        </circle>
        <circle className="portr-map__packet-static" cx="308" cy="171" r="7" />
      </svg>

      <div className="portr-map__node portr-map__node--public">
        <span className="portr-cut-label portr-cut-label--cyan">Public</span>
        <GlobeIcon />
        <span className="portr-map__annotation portr-map__annotation--own">own the edge</span>
      </div>

      <div className="portr-map__request-group">
        <div className="portr-map__request">
          <p>GET&nbsp;&nbsp; /api/user</p>
          <p>Host:&nbsp; quiet-otter.portr.dev</p>
          <p>200&nbsp;&nbsp; OK</p>
          <span className="portr-map__request-rule" />
          <span className="portr-map__request-rule" />
        </div>
        <span className="portr-map__annotation portr-map__annotation--inspect">inspect traffic</span>
      </div>

      <div className="portr-map__node portr-map__node--local">
        <LaptopIcon />
        <span className="portr-cut-label portr-cut-label--lime">Local</span>
        <span className="portr-map__annotation portr-map__annotation--replay">replay locally</span>
      </div>

      <figcaption className="sr-only" id="portr-map-caption">
        A request travels from a public Portr URL through the tunnel to a service running on localhost.
      </figcaption>
    </figure>
  );
}

const traffic = [
  ["10:21:14", "GET", "/api/user", "200"],
  ["10:21:15", "POST", "/api/login", "200"],
  ["10:21:16", "GET", "/api/projects", "200"],
  ["10:21:18", "GET", "/health", "200"],
];

function SessionLedger() {
  return (
    <section className="portr-ledger" aria-label="Example Portr tunnel session">
      <div className="portr-ledger__command">
        <code><span>›</span> portr http <b>9000</b></code>
        <span className="portr-ledger__connection"><i /> 1 connection</span>
      </div>
      <div className="portr-ledger__body">
        <div className="portr-ledger__summary">
          <p><span>INFO</span> Portr tunnel started</p>
          <p><span>URL</span> <b>https://quiet-otter-42.portr.dev</b> <i>→</i> http://localhost:9000</p>
          <p><span>Status</span> <em>online</em></p>
        </div>
        <div className="portr-ledger__traffic" role="table" aria-label="Example captured requests">
          <div className="portr-ledger__row portr-ledger__row--head" role="row">
            <span role="columnheader">Time</span>
            <span role="columnheader">Method</span>
            <span role="columnheader">Path</span>
            <span role="columnheader">Status</span>
          </div>
          {traffic.map(([time, method, path, status]) => (
            <div className="portr-ledger__row" role="row" key={`${time}-${path}`}>
              <span role="cell">{time}</span>
              <span role="cell">{method}</span>
              <span role="cell">{path}</span>
              <span role="cell">{status}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function LandingNav() {
  return (
    <header className="portr-nav">
      <Link className="portr-wordmark" href="/" aria-label="Portr home">
        <Logo width={32} height={32} />
        <span>Portr</span>
      </Link>
      <nav className="portr-nav__links" aria-label="Primary navigation">
        <Link href="/docs">Docs</Link>
        <Link href="/docs/server">Self-host</Link>
        <Link href="https://github.com/amalshaji/portr">GitHub</Link>
        <Link href="https://news.ycombinator.com/item?id=39913197">Community</Link>
      </nav>
      <Link className="portr-nav__cta" href="/docs/getting-started">Start with Portr</Link>
      <details className="portr-nav__menu">
        <summary aria-label="Open navigation"><MenuIcon /></summary>
        <nav aria-label="Mobile navigation">
          <Link href="/docs">Docs</Link>
          <Link href="/docs/server">Self-host</Link>
          <Link href="https://github.com/amalshaji/portr">GitHub</Link>
          <Link href="https://news.ycombinator.com/item?id=39913197">Community</Link>
          <Link className="portr-nav__menu-cta" href="/docs/getting-started">Start with Portr</Link>
        </nav>
      </details>
    </header>
  );
}

function InspectorStrip() {
  return (
    <div className="portr-inspector">
      <div className="portr-inspector__requests">
        <p>Captured requests</p>
        {[
          ["POST", "/webhooks/stripe", "200"],
          ["GET", "/api/orders/42", "200"],
          ["POST", "/api/events", "202"],
        ].map(([method, path, status], index) => (
          <div className={index === 0 ? "is-active" : ""} key={path}>
            <b>{method}</b><span>{path}</span><em>{status}</em>
          </div>
        ))}
      </div>
      <div className="portr-inspector__payload">
        <div className="portr-inspector__payload-head">
          <p><b>POST</b> /webhooks/stripe</p>
          <span>200 OK</span>
        </div>
        <pre><code>{`{
  "type": "invoice.paid",
  "customer": "cus_R4h2…",
  "amount_paid": 2400,
  "currency": "usd"
}`}</code></pre>
        <Link href="/docs/client/request-replay">Replay request <ArrowRight /></Link>
      </div>
    </div>
  );
}

export function LandingPage() {
  return (
    <main className="portr-landing">
      <section className="portr-stage">
        <div className="portr-frame">
          <LandingNav />
          <div className="portr-hero">
            <div className="portr-hero__copy">
              <h1>Ship localhost<br />into the wild<span aria-hidden="true">_</span></h1>
              <Link className="portr-torn-button" href="/docs/getting-started">
                Start with Portr <ArrowRight />
              </Link>
              <p>
                Portr is a self-hosted tunnel for teams.<br />
                Expose services. Inspect traffic.<br />
                Replay requests. Own your stack.
              </p>
            </div>
            <TunnelMap />
          </div>
          <div className="portr-torn-divider" aria-hidden="true" />
          <SessionLedger />
        </div>
      </section>

      <section className="portr-notes">
        <div className="portr-notes__intro">
          <h2>The public edge, traced back to your machine.</h2>
          <p>Open a tunnel with one command. Portr keeps the server, team access, request history, and replay path on infrastructure you control.</p>
          <Link href="/docs/client/http-tunnel">Follow an HTTP request <ArrowRight /></Link>
        </div>
        <div className="portr-notes__list">
          <article>
            <span className="portr-cut-label portr-cut-label--cyan">Inspect</span>
            <h3>Read the exact request.</h3>
            <p>Headers, payload, path, response, and timing stay visible in the local inspector.</p>
          </article>
          <article>
            <span className="portr-cut-label portr-cut-label--coral">Replay</span>
            <h3>Run the event again.</h3>
            <p>Retry a captured request while you fix the handler—without waiting for the original sender.</p>
          </article>
          <article>
            <span className="portr-cut-label portr-cut-label--lime">Self-host</span>
            <h3>Keep the route yours.</h3>
            <p>Operate the tunnel server, dashboards, reserved subdomains, and team access on your stack.</p>
          </article>
        </div>
      </section>

      <section className="portr-inspector-section">
        <div className="portr-inspector-section__copy">
          <h2>Stop guessing what crossed the tunnel.</h2>
          <p>Portr captures HTTP traffic beside your service. Open the body, compare headers, and replay the request from the same local workflow.</p>
        </div>
        <InspectorStrip />
      </section>

      <footer className="portr-footer">
        <div>
          <p>HTTP · TCP · WebSocket</p>
          <h2>Run the tunnel.<br />Own the stack.</h2>
        </div>
        <nav aria-label="Footer navigation">
          <Link href="/docs/server">Deploy Portr <ArrowRight /></Link>
          <Link href="/docs">Read docs <ArrowRight /></Link>
          <Link href="https://github.com/amalshaji/portr">View source <ArrowRight /></Link>
        </nav>
      </footer>
    </main>
  );
}
