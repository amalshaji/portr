import { useEffect, useState } from "react"
import { Link, useParams } from "react-router-dom"
import {
  ArrowRight,
  Check,
  Copy,
  ExternalLink,
  Globe2,
  LoaderCircle,
  RefreshCw,
  Terminal,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn, copyCodeToClipboard } from "@/lib/utils"

type SetupState = "loading" | "ready" | "error"
type InstallMethod = "script" | "homebrew"

const installCommands: Record<InstallMethod, string> = {
  script: "curl -sSf https://install.portr.dev | sh",
  homebrew: "brew install amalshaji/taps/portr",
}

function CommandBlock({
  command,
  copyLabel,
}: {
  command: string
  copyLabel: string
}) {
  return (
    <div className="group relative overflow-hidden rounded-xl border border-white/10 bg-[#101715] shadow-[inset_0_1px_0_rgba(255,255,255,0.05)]">
      <div className="flex min-h-14 items-center gap-3 px-4 py-3 pr-14">
        <span className="select-none font-mono text-xs text-[#39cbd1]">$</span>
        <code className="min-w-0 overflow-x-auto whitespace-nowrap text-[0.78rem] leading-6 text-[#edf5f1] sm:text-[0.84rem]">
          {command}
        </code>
      </div>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={copyLabel}
        className="absolute top-2.5 right-2.5 size-9 rounded-lg text-white/55 hover:bg-white/10 hover:text-white"
        onClick={() => copyCodeToClipboard(command)}
      >
        <Copy className="size-4" />
      </Button>
    </div>
  )
}

export default function Overview() {
  const { team } = useParams<{ team: string }>()
  const [installMethod, setInstallMethod] =
    useState<InstallMethod>("script")
  const [setupScript, setSetupScript] = useState("")
  const [setupState, setSetupState] = useState<SetupState>("loading")
  const [retryCount, setRetryCount] = useState(0)

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

  const setupStatus = {
    loading: "Preparing command",
    ready: "Ready to copy",
    error: "Needs attention",
  }[setupState]

  const steps = [
    {
      title: "Install the client",
      description: "Use the install script or Homebrew on the machine you develop on.",
    },
    {
      title: "Connect this workspace",
      description: "Run the generated auth command once to connect the client to your team.",
    },
    {
      title: "Start a tunnel",
      description: "Choose a local service and use the CLI help to expose its port.",
    },
  ]

  return (
    <div className="space-y-6 lg:space-y-8">
      <section className="relative overflow-hidden rounded-[1.6rem] border border-[#25332f] bg-[#17211e] text-white shadow-[0_18px_45px_rgba(23,33,30,0.16)]">
        <div className="grid gap-8 p-6 sm:p-8 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-start lg:p-10">
          <div className="max-w-3xl">
            <div className="mb-5 flex items-center gap-2 text-[0.68rem] font-bold uppercase tracking-[0.18em] text-white/55">
              <span className="size-1.5 rounded-full bg-[#39cbd1] shadow-[0_0_0_4px_rgba(57,203,209,0.12)]" />
              {team || "Workspace"} setup
            </div>
            <h1 className="max-w-2xl text-[clamp(2rem,4vw,3.65rem)] font-semibold leading-[1.02] tracking-[-0.045em]">
              Bring your first tunnel online
            </h1>
            <p className="mt-4 max-w-xl text-sm leading-6 text-white/62 sm:text-base sm:leading-7">
              Install the Portr client, connect it to this workspace, then turn
              any local service into a shareable endpoint.
            </p>
          </div>

          <div className="flex flex-wrap gap-2 lg:justify-end">
            <Button
              asChild
              className="rounded-full bg-[#f8f1e4] text-[#17211e] shadow-none hover:bg-white"
            >
              <Link to={`/${team}/connections`}>
                View connections
                <ArrowRight className="size-4" />
              </Link>
            </Button>
            <Button
              asChild
              variant="ghost"
              className="rounded-full border border-white/12 bg-white/5 text-white hover:bg-white/10 hover:text-white"
            >
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

        <div className="border-t border-white/10 bg-white/[0.035] px-6 py-5 sm:px-8 lg:px-10">
          <div
            aria-label="Tunnel route from local service through Portr to a public URL"
            className="overflow-x-auto"
          >
            <div className="grid min-w-[34rem] grid-cols-[1fr_auto_1fr_auto_1fr] items-center gap-3">
              <div className="flex items-center gap-3 rounded-xl border border-white/9 bg-white/[0.045] p-3.5">
                <span className="flex size-9 items-center justify-center rounded-lg bg-white/8 text-white/75">
                  <Terminal className="size-4" />
                </span>
                <span>
                  <span className="block text-xs font-semibold">Local service</span>
                  <span className="block font-mono text-[0.65rem] text-white/55">
                    localhost
                  </span>
                </span>
              </div>
              <ArrowRight className="size-4 text-[#39cbd1]/60" />
              <div className="flex items-center gap-3 rounded-xl border border-[#39cbd1]/20 bg-[#39cbd1]/8 p-3.5">
                <span className="flex size-9 items-center justify-center rounded-lg bg-[#101715]">
                  <img
                    src={`${import.meta.env.BASE_URL}portr-mark.svg`}
                    alt=""
                    className="size-7"
                  />
                </span>
                <span>
                  <span className="block text-xs font-semibold">Portr relay</span>
                  <span className="block text-[0.65rem] text-white/55">
                    Encrypted route
                  </span>
                </span>
              </div>
              <ArrowRight className="size-4 text-[#39cbd1]/60" />
              <div className="flex items-center gap-3 rounded-xl border border-white/9 bg-white/[0.045] p-3.5">
                <span className="flex size-9 items-center justify-center rounded-lg bg-white/8 text-white/75">
                  <Globe2 className="size-4" />
                </span>
                <span>
                  <span className="block text-xs font-semibold">Public URL</span>
                  <span className="block text-[0.65rem] text-white/55">
                    Shareable endpoint
                  </span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section
        aria-labelledby="client-setup-title"
        className="overflow-hidden rounded-[1.5rem] border border-border/90 bg-card shadow-[0_12px_35px_rgba(23,33,30,0.055)]"
      >
        <header className="flex flex-col gap-3 border-b border-border/80 px-5 py-5 sm:flex-row sm:items-center sm:justify-between sm:px-7 sm:py-6">
          <div>
            <p className="text-[0.68rem] font-bold uppercase tracking-[0.16em] text-muted-foreground">
              Guided setup
            </p>
            <h2
              id="client-setup-title"
              className="mt-1 text-xl font-semibold tracking-[-0.025em] sm:text-2xl"
            >
              Connect the Portr client
            </h2>
          </div>
          <div
            className={cn(
              "flex w-fit items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-semibold",
              setupState === "error"
                ? "border-destructive/20 bg-destructive/8 text-destructive"
                : "border-[#2f7d67]/15 bg-[#2f7d67]/8 text-[#285f50]",
            )}
          >
            {setupState === "loading" ? (
              <LoaderCircle className="size-3.5 animate-spin" />
            ) : setupState === "ready" ? (
              <Check className="size-3.5" />
            ) : (
              <span className="size-1.5 rounded-full bg-current" />
            )}
            {setupStatus}
          </div>
        </header>

        <div className="grid lg:grid-cols-[0.78fr_1.4fr]">
          <aside className="hidden border-r border-border/80 bg-muted/35 p-7 lg:block">
            <ol className="space-y-1">
              {steps.map((step, index) => (
                <li
                  key={step.title}
                  className="relative grid grid-cols-[2.25rem_1fr] gap-3 pb-6 last:pb-0"
                >
                  {index < steps.length - 1 && (
                    <span className="absolute top-9 bottom-0 left-[1.08rem] w-px bg-border" />
                  )}
                  <span
                    className={cn(
                      "relative z-10 flex size-9 items-center justify-center rounded-xl border text-xs font-bold shadow-sm",
                      index === 0
                        ? "border-[#17211e] bg-[#17211e] text-white"
                        : "border-border bg-white text-muted-foreground",
                    )}
                  >
                    {index + 1}
                  </span>
                  <div className="pt-1">
                    <h3 className="text-sm font-semibold">{step.title}</h3>
                    <p className="mt-1 text-xs leading-5 text-muted-foreground">
                      {step.description}
                    </p>
                  </div>
                </li>
              ))}
            </ol>
          </aside>

          <div className="min-w-0 p-5 sm:p-7">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <p className="text-sm font-semibold">1. Install Portr</p>
                <p className="mt-1 text-xs leading-5 text-muted-foreground">
                  Pick the method that matches this machine.
                </p>
              </div>
              <div
                role="group"
                aria-label="Install method"
                className="flex w-fit rounded-xl bg-muted p-1"
              >
                <button
                  type="button"
                  aria-pressed={installMethod === "script"}
                  onClick={() => setInstallMethod("script")}
                  className={cn(
                    "rounded-lg px-3 py-1.5 text-xs font-semibold outline-none transition-[background-color,color,box-shadow,transform] focus-visible:ring-2 focus-visible:ring-ring",
                    installMethod === "script"
                      ? "bg-white text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground",
                  )}
                >
                  Install script
                </button>
                <button
                  type="button"
                  aria-pressed={installMethod === "homebrew"}
                  onClick={() => setInstallMethod("homebrew")}
                  className={cn(
                    "rounded-lg px-3 py-1.5 text-xs font-semibold outline-none transition-[background-color,color,box-shadow,transform] focus-visible:ring-2 focus-visible:ring-ring",
                    installMethod === "homebrew"
                      ? "bg-white text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground",
                  )}
                >
                  Homebrew
                </button>
              </div>
            </div>

            <div className="mt-4">
              <CommandBlock
                command={installCommands[installMethod]}
                copyLabel="Copy install command"
              />
              <a
                href="https://github.com/amalshaji/portr/releases"
                target="_blank"
                rel="noopener noreferrer"
                className="mt-2.5 inline-flex items-center gap-1 text-xs font-semibold text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
              >
                Prefer a binary? Open GitHub releases
                <ExternalLink className="size-3" />
              </a>
            </div>

            <div className="my-6 h-px bg-border/80" />

            <div>
              <p className="text-sm font-semibold">2. Connect this workspace</p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                This command is generated for the {team || "current"} team.
              </p>

              <div className="mt-4" aria-live="polite">
                {setupState === "loading" && (
                  <div className="flex min-h-14 items-center gap-3 rounded-xl border border-border bg-muted/45 px-4 text-xs text-muted-foreground">
                    <LoaderCircle className="size-4 animate-spin" />
                    Preparing your setup command…
                  </div>
                )}

                {setupState === "ready" && (
                  <CommandBlock
                    command={setupScript}
                    copyLabel="Copy setup command"
                  />
                )}

                {setupState === "error" && (
                  <div className="flex flex-col gap-4 rounded-xl border border-destructive/20 bg-destructive/5 p-4 sm:flex-row sm:items-center sm:justify-between">
                    <div>
                      <p className="text-sm font-semibold text-foreground">
                        Setup command unavailable
                      </p>
                      <p className="mt-1 text-xs leading-5 text-muted-foreground">
                        Check the server connection, then try generating it again.
                      </p>
                    </div>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      aria-label="Retry setup command"
                      className="shrink-0 rounded-lg bg-white"
                      onClick={() => setRetryCount((count) => count + 1)}
                    >
                      <RefreshCw className="size-3.5" />
                      Retry
                    </Button>
                  </div>
                )}
              </div>
            </div>

            <div className="mt-6 flex flex-col gap-3 rounded-xl border border-border/80 bg-muted/35 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <p className="text-xs font-semibold text-foreground">
                  3. Start a tunnel when you’re ready
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Run <code className="font-semibold text-foreground">portr -h</code> to see the available tunnel commands.
                </p>
              </div>
              <Button asChild variant="ghost" size="sm" className="shrink-0 rounded-lg">
                <Link to={`/${team}/connections`}>
                  Open connections
                  <ArrowRight className="size-3.5" />
                </Link>
              </Button>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
