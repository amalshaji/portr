import { Fragment, useState } from "react"
import { Link, useParams } from "react-router-dom"
import {
  ArrowRight,
  Check,
  ChevronDown,
  Copy,
  ExternalLink,
  Globe2,
  LoaderCircle,
  RefreshCw,
  Terminal,
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

const setupStatus: Record<SetupState, string> = {
  loading: "Preparing command",
  ready: "Ready to copy",
  error: "Needs attention",
}

const steps = [
  {
    title: "Install the client",
    description:
      "Use the install script or Homebrew on the machine you develop on.",
  },
  {
    title: "Connect this workspace",
    description:
      "Run the generated auth command once to connect the client to your team.",
  },
  {
    title: "Start a tunnel",
    description: "Choose a local service and use the CLI help to expose its port.",
  },
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
      <div className="space-y-6">
        <section className="dark overflow-hidden rounded-lg border border-border bg-[var(--portr-night-deep)] text-foreground">
          <div className="grid gap-8 p-6 sm:p-8 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-start">
            <div className="max-w-2xl">
              <p className="eyebrow flex items-center gap-2">
                <span className="size-1.5 rounded-full bg-signal-live" />
                {team || "Workspace"} setup
              </p>
              <h2 className="mt-4 max-w-xl text-[clamp(2rem,4vw,3.25rem)] leading-[1.04] font-semibold">
                Bring your first tunnel online
              </h2>
              <p className="mt-4 max-w-lg text-sm leading-6 text-muted-foreground">
                Install the Portr client, connect it to this workspace, then
                turn any local service into a shareable endpoint.
              </p>
            </div>

            <div className="flex flex-wrap gap-2 lg:justify-end">
              <Button asChild>
                <Link to={`/${team}/connections`}>
                  View connections
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
              <Button asChild variant="outline">
                <a
                  href="https://portr.dev/docs/client"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Client docs
                  <ExternalLink className="size-3.5" />
                </a>
              </Button>
            </div>
          </div>

          <div className="border-t border-border bg-card/40 px-6 py-5 sm:px-8">
            <div
              aria-label="Tunnel route from local service through Portr to a public URL"
              className="overflow-x-auto"
            >
              <div className="grid min-w-[34rem] grid-cols-[1fr_auto_1fr_auto_1fr] items-center gap-3">
                {[
                  {
                    Icon: Terminal,
                    title: "Local service",
                    detail: "localhost",
                    mono: true,
                  },
                  {
                    Icon: null,
                    title: "Portr relay",
                    detail: "Encrypted route",
                    mono: false,
                  },
                  {
                    Icon: Globe2,
                    title: "Public URL",
                    detail: "Shareable endpoint",
                    mono: false,
                  },
                ].map(({ Icon, title, detail, mono }, index) => (
                  <Fragment key={title}>
                    {index > 0 && (
                      <ArrowRight className="size-4 text-signal-live/60" />
                    )}
                    <div
                      className={cn(
                        "flex items-center gap-3 rounded-md border p-3.5",
                        Icon
                          ? "border-border bg-background/40"
                          : "border-signal-live/25 bg-signal-live/10",
                      )}
                    >
                      <span className="flex size-9 shrink-0 items-center justify-center rounded-sm bg-[var(--portr-night-deep)] text-muted-foreground">
                        {Icon ? (
                          <Icon className="size-4" />
                        ) : (
                          <img
                            src={`${import.meta.env.BASE_URL}portr-mark.svg`}
                            alt=""
                            className="size-6"
                          />
                        )}
                      </span>
                      <span className="min-w-0">
                        <span className="block text-xs font-semibold">
                          {title}
                        </span>
                        <span
                          className={cn(
                            "block text-[0.65rem] text-muted-foreground",
                            mono && "data",
                          )}
                        >
                          {detail}
                        </span>
                      </span>
                    </div>
                  </Fragment>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section className="overflow-hidden rounded-md border border-border bg-card">
          <header className="flex flex-col gap-3 border-b border-border px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="eyebrow">Guided setup</p>
              <h2 className="mt-1 text-lg font-semibold">
                Connect the Portr client
              </h2>
            </div>
            <span
              className={cn(
                "flex w-fit items-center gap-2 rounded-full border px-2.5 py-1 text-xs font-medium",
                setupState === "error"
                  ? "border-destructive/30 bg-destructive/10 text-destructive"
                  : "border-signal-live/30 bg-signal-live/10 text-foreground",
              )}
            >
              {setupState === "loading" ? (
                <LoaderCircle className="size-3.5 animate-spin" />
              ) : setupState === "ready" ? (
                <Check className="size-3.5" />
              ) : (
                <span className="size-1.5 rounded-full bg-current" />
              )}
              {setupStatus[setupState]}
            </span>
          </header>

          <div className="grid lg:grid-cols-[0.8fr_1.4fr]">
            <aside className="hidden border-r border-border bg-muted/40 p-6 lg:block">
              <ol className="space-y-1">
                {steps.map((step, index) => (
                  <li
                    key={step.title}
                    className="relative grid grid-cols-[2rem_1fr] gap-3 pb-6 last:pb-0"
                  >
                    {index < steps.length - 1 && (
                      <span className="absolute top-8 bottom-0 left-[0.97rem] w-px bg-border" />
                    )}
                    <span
                      className={cn(
                        "data relative z-10 flex size-8 items-center justify-center rounded-sm border text-xs font-semibold",
                        index === 0
                          ? "border-primary bg-primary text-primary-foreground"
                          : "border-border bg-background text-muted-foreground",
                      )}
                    >
                      {index + 1}
                    </span>
                    <div className="pt-1">
                      <h3 className="text-sm font-medium">{step.title}</h3>
                      <p className="mt-1 text-xs leading-5 text-muted-foreground">
                        {step.description}
                      </p>
                    </div>
                  </li>
                ))}
              </ol>
            </aside>

            <div className="min-w-0 p-5 sm:p-6">
              <ClientSetup
                team={team}
                setupState={setupState}
                setupScript={script}
                onRetry={retry}
              />
            </div>
          </div>
        </section>
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
