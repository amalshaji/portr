/**
 * The tunnel diagram from the portr.dev landing page: a request leaves a public
 * Portr URL, travels the tunnel, and arrives at a service on localhost.
 *
 * Ported from docs-v2 components/landing/landing-page.tsx (TunnelMap). docs-v2
 * is a separate Next app, so the markup and the route path are duplicated here
 * rather than imported; styles live in index.css under .tunnel-map.
 */

const routePath = "M 214 86 C 214 150 332 132 332 222 C 332 302 218 286 218 412"

function GlobeIcon() {
  return (
    <svg aria-hidden="true" fill="none" viewBox="0 0 64 64">
      <circle cx="32" cy="32" r="23" stroke="currentColor" strokeWidth="2.6" />
      <path
        d="M9 32h46M32 9c8 7 12 14.7 12 23S40 48 32 55M32 9c-8 7-12 14.7-12 23s4 16 12 23M32 9v46"
        stroke="currentColor"
        strokeWidth="2"
      />
    </svg>
  )
}

function LaptopIcon() {
  return (
    <svg aria-hidden="true" fill="none" viewBox="0 0 72 58">
      <rect
        height="38"
        rx="2"
        stroke="currentColor"
        strokeWidth="2.4"
        width="54"
        x="9"
        y="4"
      />
      <path
        d="m24 24 6-6m-6 6 6 6m18-12 6 6-6 6m-10 2 7-16M4 49h64l-5 5H9l-5-5Z"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2.2"
      />
    </svg>
  )
}

export default function TunnelMap({ className }: { className?: string }) {
  return (
    <figure className={`tunnel-map ${className ?? ""}`} aria-labelledby="tunnel-map-caption">
      <svg aria-hidden="true" className="tunnel-map__route" viewBox="0 0 440 510">
        <path className="tunnel-map__path tunnel-map__path--under" d={routePath} />
        <path className="tunnel-map__path" d={routePath} />
        {/* Driven by CSS offset-path in index.css rather than SMIL: an
            <animateMotion> inserted by React after parse does not begin on its
            own, and CSS also lets prefers-reduced-motion park the packet. */}
        <circle className="tunnel-map__packet" r="7" />
      </svg>

      <div className="tunnel-map__node tunnel-map__node--public">
        <span className="tunnel-map__label tunnel-map__label--cyan">Public</span>
        <GlobeIcon />
      </div>

      <div className="tunnel-map__request">
        <p>GET&nbsp;&nbsp; /api/user</p>
        <p>Host:&nbsp; quiet-otter.portr.dev</p>
        <p>200&nbsp;&nbsp; OK</p>
        <span className="tunnel-map__request-rule" />
        <span className="tunnel-map__request-rule" />
      </div>

      <div className="tunnel-map__node tunnel-map__node--local">
        <LaptopIcon />
        <span className="tunnel-map__label tunnel-map__label--lime">Local</span>
      </div>

      <figcaption className="sr-only" id="tunnel-map-caption">
        A request travels from a public Portr URL through the tunnel to a service
        running on localhost.
      </figcaption>
    </figure>
  )
}
