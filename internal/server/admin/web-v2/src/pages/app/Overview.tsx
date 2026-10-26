import { Fragment } from "react"
import { Link, useParams } from "react-router-dom"
import { ArrowRight, ExternalLink, Globe2, Terminal } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import GuidedSetup from "@/components/GuidedSetup"
import Panel from "@/components/Panel"
import RouteLine, {
  connectionRouteName,
  connectionRouteState,
} from "@/components/RouteLine"
import { useSetupScript, useTeamOverview } from "@/hooks/use-team-overview"
import { relativeTime } from "@/lib/humanize"
import { cn } from "@/lib/utils"

/** The page's anchor in both states: eyebrow, display headline, actions, and a
 *  strip along the bottom — the tunnel diagram on first run, the workspace
 *  readouts once the team has connected. */
function Hero({
  team,
  eyebrow,
  title,
  description,
  children,
}: {
  team?: string
  eyebrow: string
  title: string
  description: string
  children: React.ReactNode
}) {
  return (
    <section className="dark overflow-hidden rounded-lg border border-border bg-[var(--portr-night-deep)] text-foreground">
      <div className="grid gap-8 p-6 sm:p-8 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-start">
        <div className="max-w-2xl">
          <p className="eyebrow flex items-center gap-2">
            <span className="size-1.5 rounded-full bg-signal-live" />
            {eyebrow}
          </p>
          <h2 className="mt-4 max-w-xl text-[clamp(2rem,4vw,3.25rem)] leading-[1.04] font-semibold">
            {title}
          </h2>
          <p className="mt-4 max-w-lg text-sm leading-6 text-muted-foreground">
            {description}
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
        {children}
      </div>
    </section>
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
        <Hero
          team={team}
          eyebrow={`${team || "Workspace"} setup`}
          title="Bring your first tunnel online"
          description="Install the Portr client, connect it to this workspace, then turn any local service into a shareable endpoint."
        >
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
        </Hero>

        <GuidedSetup
          team={team}
          setupState={setupState}
          setupScript={script}
          onRetry={retry}
        />
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
      <Hero
        team={team}
        eyebrow={`${team || "Workspace"} workspace`}
        title={
          activeConnections === 0
            ? "No tunnels running right now"
            : activeConnections === 1
              ? "1 tunnel running"
              : `${activeConnections} tunnels running`
        }
        description={
          activeConnections === 0
            ? "Start a tunnel from the Portr client and it appears here within seconds."
            : "Live routes are serving traffic through this workspace right now."
        }
      >
        <dl className="grid gap-3 sm:grid-cols-3">
          {readouts.map(({ label, value }) => (
            <div
              key={label}
              className="rounded-md border border-border bg-background/40 p-3.5"
            >
              <dt className="eyebrow">{label}</dt>
              <dd className="tabular mt-1 font-display text-2xl font-semibold">
                {value}
              </dd>
            </div>
          ))}
        </dl>
      </Hero>

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
                  name={connectionRouteName(connection)}
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
        <Link
          to={`/${team}/connect`}
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
          <ArrowRight className="size-4 shrink-0 text-muted-foreground" />
        </Link>
      </Panel>
    </div>
  )
}
