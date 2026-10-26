import { useState } from "react"
import {
  Check,
  Copy,
  ExternalLink,
  LoaderCircle,
  RefreshCw,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import SegmentedControl from "@/components/SegmentedControl"
import type { SetupState } from "@/hooks/use-team-overview"
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

interface GuidedSetupProps {
  team?: string
  setupState: SetupState
  setupScript: string
  onRetry: () => void
  /** The step overview beside the commands; off where the page is only setup. */
  showSteps?: boolean
}

/** Install + connect, shared by the first-run overview and the Connect page. */
export default function GuidedSetup({
  team,
  setupState,
  setupScript,
  onRetry,
  showSteps = true,
}: GuidedSetupProps) {
  return (
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

      <div
        className={cn("grid", showSteps && "lg:grid-cols-[0.8fr_1.4fr]")}
      >
        {showSteps && (
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
        )}

        <div className="min-w-0 p-5 sm:p-6">
          <ClientSetup
            team={team}
            setupState={setupState}
            setupScript={setupScript}
            onRetry={onRetry}
          />
        </div>
      </div>
    </section>
  )
}
