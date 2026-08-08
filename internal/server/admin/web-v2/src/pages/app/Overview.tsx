import { useEffect, useState } from "react"
import { Link, useParams } from "react-router-dom"
import {
  ArrowRight,
  Check,
  ChevronDown,
  Copy,
  ExternalLink,
  LoaderCircle,
  RefreshCw,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import RouteLine, { type RouteState } from "@/components/RouteLine"
import { cn, copyCodeToClipboard } from "@/lib/utils"
import type { Connection } from "@/types"

type SetupState = "loading" | "ready" | "error"
type InstallMethod = "script" | "homebrew"

const installCommands: Record<InstallMethod, string> = {
  script: "curl -sSf https://install.portr.dev | sh",
  homebrew: "brew install amalshaji/taps/portr",
}

const routeState = (connection: Connection): RouteState => {
  if (connection.status === "active") return "live"
  if (connection.status === "reserved") return "unbound"
  return "closed"
}

const relativeTime = (value: string | null) => {
  if (!value) return "—"
  const elapsed = Date.now() - new Date(value).getTime()
  const minutes = Math.floor(elapsed / 60000)
  if (minutes < 1) return "just now"
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

function CommandBlock({
  command,
  copyLabel,
}: {
  command: string
  copyLabel: string
}) {
  return (
    <div className="dark group relative overflow-hidden rounded-md border border-border bg-[var(--portr-night-deep)]">
      <div className="flex min-h-12 items-center gap-3 px-3.5 py-3 pr-12">
        <span className="data shrink-0 select-none text-xs text-signal-live">
          $
        </span>
        <code className="min-w-0 overflow-x-auto whitespace-nowrap text-[0.8rem] leading-6 text-foreground">
          {command}
        </code>
      </div>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={copyLabel}
        className="absolute top-1.5 right-1.5 size-8 text-muted-foreground hover:text-foreground"
        onClick={() => copyCodeToClipboard(command)}
      >
        <Copy className="size-4" />
      </Button>
    </div>
  )
}

/** Install + connect. Shown in full on first run, collapsed once a team has
 *  connected at least once. */
function ClientSetup({
  team,
  setupState,
  setupScript,
  onRetry,
}: {
  team?: string
  setupState: SetupState
  setupScript: string
  onRetry: () => void
}) {
  const [installMethod, setInstallMethod] = useState<InstallMethod>("script")

  return (
    <div className="space-y-6">
      <div>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-sm font-medium">1. Install the client</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Pick the method that matches the machine you develop on.
            </p>
          </div>
          <div
            role="group"
            aria-label="Install method"
            className="flex w-fit rounded-md border border-border bg-muted/60 p-0.5"
          >
            {(["script", "homebrew"] as const).map((method) => (
              <button
                key={method}
                type="button"
                aria-pressed={installMethod === method}
                onClick={() => setInstallMethod(method)}
                className={cn(
                  "rounded-sm px-2.5 py-1 text-xs font-medium outline-none transition-colors duration-(--portr-duration-micro) ease-portr focus-visible:ring-2 focus-visible:ring-ring",
                  installMethod === method
                    ? "bg-background text-foreground shadow-xs"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                {method === "script" ? "Install script" : "Homebrew"}
              </button>
            ))}
          </div>
        </div>

        <div className="mt-3">
          <CommandBlock
            command={installCommands[installMethod]}
            copyLabel="Copy install command"
          />
          <a
            href="https://github.com/amalshaji/portr/releases"
            target="_blank"
            rel="noopener noreferrer"
            className="mt-2 inline-flex items-center gap-1 text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            Prefer a binary? Open GitHub releases
            <ExternalLink className="size-3" />
          </a>
        </div>
      </div>

      <div>
        <p className="text-sm font-medium">2. Connect this workspace</p>
        <p className="mt-1 text-xs text-muted-foreground">
          Generated for the <span className="data">{team || "current"}</span>{" "}
          team. Run it once per machine.
        </p>

        <div className="mt-3" aria-live="polite">
          {setupState === "loading" && (
            <Skeleton className="h-12 w-full rounded-md" />
          )}

          {setupState === "ready" && (
            <CommandBlock command={setupScript} copyLabel="Copy setup command" />
          )}

          {setupState === "error" && (
            <div className="flex flex-col gap-3 rounded-md border border-destructive/30 bg-destructive/10 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <p className="text-sm font-medium">Setup command unavailable</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Check the server connection, then generate it again.
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                aria-label="Retry setup command"
                className="shrink-0"
                onClick={onRetry}
              >
                <RefreshCw className="size-3.5" />
                Retry
              </Button>
            </div>
          )}
        </div>
      </div>

      <div>
        <p className="text-sm font-medium">3. Start a tunnel</p>
        <p className="mt-1 text-xs text-muted-foreground">
          Run <code className="text-foreground">portr -h</code> to see the
          available tunnel commands.
        </p>
      </div>
    </div>
  )
}

export default function Overview() {
  const { team } = useParams<{ team: string }>()
  const [setupScript, setSetupScript] = useState("")
  const [setupState, setSetupState] = useState<SetupState>("loading")
  const [retryCount, setRetryCount] = useState(0)

  const [connections, setConnections] = useState<Connection[]>([])
  const [totalConnections, setTotalConnections] = useState(0)
  const [activeConnections, setActiveConnections] = useState(0)
  const [teamMembers, setTeamMembers] = useState(0)
  const [routesLoading, setRoutesLoading] = useState(true)
  const [setupOpen, setSetupOpen] = useState(false)

  useEffect(() => {
    let cancelled = false

    const getSetupScript = async () => {
      if (!team) {
        setSetupState("error")
        return
      }

      setSetupState("loading")
      try {
        const res = await fetch("/api/v1/config/setup-script", {
          headers: { "x-team-slug": team },
        })
        if (!res.ok) throw new Error("setup command request failed")

        const data: unknown = await res.json()
        const command =
          typeof data === "object" &&
          data !== null &&
          "message" in data &&
          typeof data.message === "string"
            ? data.message.trim()
            : ""
        if (!command) throw new Error("setup command was empty")

        if (!cancelled) {
          setSetupScript(command)
          setSetupState("ready")
        }
      } catch {
        if (!cancelled) {
          setSetupScript("")
          setSetupState("error")
        }
      }
    }

    getSetupScript()
    return () => {
      cancelled = true
    }
  }, [retryCount, team])

  useEffect(() => {
    if (!team) return
    let cancelled = false

    const load = async () => {
      setRoutesLoading(true)
      try {
        const [connectionsRes, statsRes] = await Promise.all([
          fetch("/api/v1/connections/?type=recent&page=1&page_size=5", {
            headers: { "x-team-slug": team },
          }),
          fetch("/api/v1/config/stats", { headers: { "x-team-slug": team } }),
        ])

        if (cancelled) return

        if (connectionsRes.ok) {
          const data = await connectionsRes.json()
          setConnections(Array.isArray(data?.data) ? data.data : [])
          setTotalConnections(typeof data?.count === "number" ? data.count : 0)
        }

        if (statsRes.ok) {
          const stats = await statsRes.json()
          setActiveConnections(stats?.team_stats?.active_connections ?? 0)
          setTeamMembers(stats?.team_stats?.team_members ?? 0)
        }
      } catch (error) {
        console.error("Failed to load overview:", error)
      } finally {
        if (!cancelled) setRoutesLoading(false)
      }
    }

    load()
    return () => {
      cancelled = true
    }
  }, [team])

  const hasConnected = totalConnections > 0

  // Which page this is depends on whether the team has ever connected, so hold
  // a neutral frame rather than flashing the wrong one.
  if (routesLoading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-3 sm:grid-cols-3">
          {[0, 1, 2].map((item) => (
            <Skeleton key={item} className="h-20 rounded-md" />
          ))}
        </div>
        <Skeleton className="h-56 rounded-md" />
      </div>
    )
  }

  // First run: the whole page is the setup path, because nothing else exists yet.
  if (!hasConnected) {
    return (
      <div className="mx-auto w-full max-w-3xl space-y-8">
        <header>
          <p className="eyebrow">First tunnel</p>
          <h2 className="mt-1.5 text-2xl font-semibold">
            Bring this workspace online
          </h2>
          <p className="mt-2 text-sm text-muted-foreground">
            Install the Portr client, connect it to{" "}
            <span className="data">{team}</span>, then expose a local service.
          </p>
        </header>

        <section className="rounded-md border border-border bg-card p-5 sm:p-6">
          <ClientSetup
            team={team}
            setupState={setupState}
            setupScript={setupScript}
            onRetry={() => setRetryCount((count) => count + 1)}
          />
        </section>

        <div className="rounded-md border border-border bg-muted/40 p-4">
          <div className="flex items-center gap-3">
            <RouteLine
              name={`your-service.${team ?? "team"}`}
              state="unbound"
              className="flex-1"
            />
          </div>
          <p className="mt-3 text-xs text-muted-foreground">
            Once a tunnel is running, its public name and local port appear here
            as a live route.
          </p>
        </div>
      </div>
    )
  }

  const readouts = [
    { label: "Active tunnels", value: activeConnections },
    { label: "Team members", value: teamMembers },
    { label: "Connections all time", value: totalConnections },
  ]

  return (
    <div className="space-y-6">
      <section className="grid gap-3 sm:grid-cols-3">
        {readouts.map(({ label, value }) => (
          <div
            key={label}
            className="rounded-md border border-border bg-card p-4"
          >
            <p className="eyebrow">{label}</p>
            <p className="tabular mt-1 font-display text-3xl font-semibold">
              {value}
            </p>
          </div>
        ))}
      </section>

      <section className="overflow-hidden rounded-md border border-border bg-card">
        <header className="flex items-center justify-between gap-3 border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold">Recent routes</h2>
          <Button asChild variant="ghost" size="sm">
            <Link to={`/${team}/connections`}>
              All connections
              <ArrowRight className="size-3.5" />
            </Link>
          </Button>
        </header>

        {connections.length === 0 ? (
          <p className="px-4 py-8 text-center text-sm text-muted-foreground">
            No connections yet. Start a tunnel to see it here.
          </p>
        ) : (
          <ul className="divide-y divide-border">
            {connections.map((connection) => (
              <li
                key={connection.id}
                className="flex items-center gap-4 px-4 py-3.5"
              >
                <RouteLine
                  name={connection.subdomain || `${connection.type} tunnel`}
                  port={connection.port}
                  state={routeState(connection)}
                  className="min-w-0 max-w-2xl flex-1"
                />
                <span className="data w-20 shrink-0 text-right text-xs text-muted-foreground">
                  {relativeTime(connection.started_at ?? connection.created_at)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="overflow-hidden rounded-md border border-border bg-card">
        <button
          type="button"
          aria-expanded={setupOpen}
          onClick={() => setSetupOpen((open) => !open)}
          className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left outline-none transition-colors hover:bg-accent/50 focus-visible:ring-2 focus-visible:ring-ring"
        >
          <span>
            <span className="block text-sm font-semibold">
              Connect another machine
            </span>
            <span className="block text-xs text-muted-foreground">
              Install the client and authenticate it against this team.
            </span>
          </span>
          <span className="flex items-center gap-2">
            {setupState === "ready" && (
              <Check className="size-3.5 text-signal-live" />
            )}
            {setupState === "loading" && (
              <LoaderCircle className="size-3.5 animate-spin text-muted-foreground" />
            )}
            <ChevronDown
              className={cn(
                "size-4 text-muted-foreground transition-transform duration-(--portr-duration-short) ease-portr",
                setupOpen && "rotate-180",
              )}
            />
          </span>
        </button>

        {setupOpen && (
          <div className="border-t border-border p-4 sm:p-5">
            <ClientSetup
              team={team}
              setupState={setupState}
              setupScript={setupScript}
              onRetry={() => setRetryCount((count) => count + 1)}
            />
          </div>
        )}
      </section>
    </div>
  )
}
