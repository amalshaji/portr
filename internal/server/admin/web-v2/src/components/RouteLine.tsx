import { cn } from "@/lib/utils"

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

const stateLabel: Record<RouteState, string> = {
  live: "live",
  idle: "idle",
  closed: "closed",
  unbound: "not running",
}

const ruleColor: Record<RouteState, string> = {
  live: "border-signal-live",
  idle: "border-signal-idle",
  closed: "border-signal-closed",
  unbound: "border-signal-unbound",
}

const nodeColor: Record<RouteState, string> = {
  live: "bg-signal-live",
  idle: "bg-signal-idle",
  closed: "bg-signal-closed",
  unbound: "bg-transparent",
}

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
  const dashed = state === "idle" || state === "unbound"

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
            nodeColor[state],
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
          {protocol && large && (
            <span className="eyebrow block">{protocol}</span>
          )}
        </span>
      </div>

      <span
        aria-hidden="true"
        className={cn(
          "min-w-6 flex-1 border-t",
          large ? "border-t-2" : "border-t",
          dashed && "border-dashed",
          ruleColor[state],
          animate && "animate-route-draw origin-left",
        )}
      />

      <div className={cn("flex shrink-0 items-center gap-2", animate && "animate-route-node")}>
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
          {large && (
            <span className="eyebrow block">{stateLabel[state]}</span>
          )}
        </span>
        <span
          aria-hidden="true"
          className={cn(
            "shrink-0 rounded-full",
            large ? "size-2.5" : "size-1.5",
            nodeColor[state],
            // An unbound name gets an open jack rather than a filled node.
            state === "unbound" && "border-2 border-signal-unbound",
          )}
        />
      </div>

      <span className="sr-only">
        {port
          ? `${name} bound to port ${port}, ${stateLabel[state]}`
          : `${name} reserved, ${stateLabel[state]}`}
      </span>
    </div>
  )
}
