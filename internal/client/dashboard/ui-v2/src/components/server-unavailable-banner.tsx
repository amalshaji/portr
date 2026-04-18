import { TriangleAlertIcon } from "lucide-react"

import { cn } from "@/lib/utils"

export function ServerUnavailableBanner({
  className,
}: {
  className?: string
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-3 border px-4 py-3 font-mono text-sm",
        "border-amber-400/40 bg-amber-400/5 text-amber-700 dark:border-amber-400/30 dark:bg-amber-400/5 dark:text-amber-400",
        className
      )}
    >
      <TriangleAlertIcon className="mt-0.5 size-4 shrink-0" />
      <div>
        <p className="font-semibold leading-none tracking-tight">Local inspector unavailable</p>
        <p className="mt-1 opacity-75">Unable to reach dashboard server, is it running?</p>
      </div>
    </div>
  )
}
