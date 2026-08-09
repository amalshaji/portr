import { cn } from "@/lib/utils"
import type { Connection } from "@/types"

/**
 * A Portr route: a public name bound to a local port.
 *
 *   api-dev.example.com  ●━━━━━━━━━━━●  :8000
 *           http                       live
 *
 * The rule between the two nodes carries the state, so the binding reads as one
 * object instead of three separate table columns. "unbound" is a reserved name
 * with nothing running behind it — the right node is drawn as an open ring.
 */

export type RouteState = "live" | "idle" | "closed" | "unbound"

const ROUTE_STATES: Record<
  RouteState,
  { label: string; rule: string; node: string; dashed: boolean }
> = {
  live: {
    label: "live",
    rule: "border-signal-live",
    node: "bg-signal-live",
    dashed: false,
  },
  idle: {
    label: "idle",
    rule: "border-signal-idle",
    node: "bg-signal-idle",
    dashed: true,
  },
  closed: {
    label: "closed",
    rule: "border-signal-closed",
    node: "bg-signal-closed",
    dashed: false,
  },
  unbound: {
    label: "not running",
    rule: "border-signal-unbound",
    // An unbound name gets an open jack rather than a filled node.
    node: "bg-transparent border-2 border-signal-unbound",
    dashed: true,
  },
}

/** Maps a connection onto the state its route line should show. */
export const connectionRouteState = (connection: Connection): RouteState => {
  if (connection.status === "active") return "live"
  if (connection.status === "reserved") return "unbound"
  return "closed"
}

/**
 * The public endpoint the server actually exposes: a subdomain for HTTP
 * tunnels, a port for TCP ones. HTTP connections carry no port — the local
 * port stays on the client and is never reported.
 */
export const connectionRouteName = (connection: Connection) =>
  connection.type === "tcp"
    ? connection.port
      ? `:${connection.port}`
      : "tcp tunnel"
    : connection.subdomain || "http tunnel"

interface RouteLineProps {
  /** Public name, e.g. api-dev.example.com */
  name: string
  /** Local port. Omit for a reserved name with nothing bound to it. */
  port?: number | string | null
  /** http, tcp — shown under the public name. */
  protocol?: string | null
  state: RouteState
  size?: "lg" | "inline"
  /** Draw the line in once on mount. Reserved for the login hero. */
  animate?: boolean
  className?: string
}

export default function RouteLine({
  name,
  port,
  protocol,
  state,
  size = "inline",
  animate = false,
  className,
}: RouteLineProps) {
  const large = size === "lg"
  const { label, rule, node, dashed } = ROUTE_STATES[state]

  // The server only records a port for TCP tunnels — for HTTP the local port
  // stays on the client and is never reported. Rendering the binding without it
  // would put a dash where the local end should be, so show just the public
  // endpoint. "unbound" is different: there the missing end is the whole point.
  if (!port && state !== "unbound") {
    return (
      <div
        className={cn(
          "flex min-w-0 items-center",
          large ? "gap-3" : "gap-2",
          className,
        )}
      >
        <span
          aria-hidden="true"
          className={cn(
            "shrink-0 rounded-full",
            large ? "size-2.5" : "size-1.5",
            node,
            state === "live" && "animate-route-pulse",
          )}
        />
        <span className="min-w-0">
          <span
            className={cn(
              "data block truncate",
              large ? "text-base font-medium" : "text-xs",
            )}
          >
            {name}
          </span>
          {large && (
            <span className="eyebrow block">
              {protocol ? `${protocol} · ${label}` : label}
            </span>
          )}
        </span>
        <span className="sr-only">{`${name}, ${label}`}</span>
      </div>
    )
  }

  return (
    <div
      className={cn(
        "flex min-w-0 items-center",
        large ? "gap-3" : "gap-2",
        className,
      )}
    >
      <div className="flex min-w-0 items-center gap-2">
        <span
          aria-hidden="true"
          className={cn(
            "shrink-0 rounded-full",
            large ? "size-2.5" : "size-1.5",
            node,
            state === "live" && "animate-route-pulse",
          )}
        />
        <span className="min-w-0">
          <span
            className={cn(
              "data block truncate",
              large ? "text-base font-medium" : "text-xs",
            )}
          >
            {name}
          </span>
          {protocol && large && <span className="eyebrow block">{protocol}</span>}
        </span>
      </div>

      <span
        aria-hidden="true"
        className={cn(
          "min-w-6 flex-1 border-t",
          large && "border-t-2",
          dashed && "border-dashed",
          rule,
          animate && "animate-route-draw origin-left",
        )}
      />

      <div
        className={cn(
          "flex shrink-0 items-center gap-2",
          animate && "animate-route-node",
        )}
      >
        <span className="text-right">
          <span
            className={cn(
              "data block",
              large ? "text-base font-medium" : "text-xs",
              state === "unbound" && "text-muted-foreground",
            )}
          >
            {port ? `:${port}` : "—"}
          </span>
          {large && <span className="eyebrow block">{label}</span>}
        </span>
        <span
          aria-hidden="true"
          className={cn("shrink-0 rounded-full", large ? "size-2.5" : "size-1.5", node)}
        />
      </div>

      <span className="sr-only">
        {port
          ? `${name} bound to port ${port}, ${label}`
          : `${name} reserved, ${label}`}
      </span>
    </div>
  )
}
