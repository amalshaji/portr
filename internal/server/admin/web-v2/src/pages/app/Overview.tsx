import { useState } from "react"
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
import Panel from "@/components/Panel"
import SegmentedControl from "@/components/SegmentedControl"
import RouteLine, { connectionRouteState } from "@/components/RouteLine"
import { useSetupScript, useTeamOverview, type SetupState } from "@/hooks/use-team-overview"
import { relativeTime } from "@/lib/humanize"
import { cn, copyCodeToClipboard } from "@/lib/utils"

type InstallMethod = "script" | "homebrew"

const installCommands: Record<InstallMethod, string> = {
  script: "curl -sSf https://install.portr.dev | sh",
  homebrew: "brew install amalshaji/taps/portr",
}

const installOptions = [
  { value: "script" as const, label: "Install script" },
  { value: "homebrew" as const, label: "Homebrew" },
]

function CommandBlock({
  command,
  copyLabel,
}: {
  command: string
  copyLabel: string
}) {
  return (
    <div className="dark relative overflow-hidden rounded-md border border-border bg-[var(--portr-night-deep)]">
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
          <SegmentedControl<InstallMethod>
            ariaLabel="Install method"
            options={installOptions}
            value={installMethod}
            onChange={setInstallMethod}
          />
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
  const { script, state: setupState, retry } = useSetupScript(team)
  const {
    connections,
    totalConnections,
    activeConnections,
    teamMembers,
    loading,
  } = useTeamOverview(team)
  const [setupOpen, setSetupOpen] = useState(false)

  // Which page this is depends on whether the team has ever connected, so hold
  // a neutral frame rather than flashing the wrong one.
  if (loading) {
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
  if (totalConnections === 0) {
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

        <Panel>
          <ClientSetup
            team={team}
            setupState={setupState}
            setupScript={script}
            onRetry={retry}
          />
        </Panel>

        <div className="rounded-md border border-border bg-muted/40 p-4">
          <RouteLine name={`your-service.${team ?? "team"}`} state="unbound" />
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

      <Panel
        title="Recent routes"
        flush
        action={
          <Button asChild variant="ghost" size="sm">
            <Link to={`/${team}/connections`}>
              All connections
              <ArrowRight className="size-3.5" />
            </Link>
          </Button>
        }
      >
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
                  state={connectionRouteState(connection)}
                  className="min-w-0 max-w-2xl flex-1"
                />
                <span className="data w-20 shrink-0 text-right text-xs text-muted-foreground">
                  {relativeTime(connection.started_at ?? connection.created_at)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Panel>

      <Panel flush>
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
              setupScript={script}
              onRetry={retry}
            />
          </div>
        )}
      </Panel>
    </div>
  )
}
