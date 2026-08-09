import type { ReactNode } from "react"
import { cn } from "@/lib/utils"

/**
 * The console's one surface: a bordered card, optionally with a titled header
 * and a trailing action. Every page is built from these, so the border, radius
 * and header rhythm are defined once here rather than re-typed per page.
 */
interface PanelProps {
  title?: ReactNode
  description?: ReactNode
  /** Rendered at the trailing edge of the header. */
  action?: ReactNode
  /** Shown before the title — used for the settings pages' section icons. */
  icon?: ReactNode
  /** Drop the body padding when the content manages its own (tables, lists). */
  flush?: boolean
  className?: string
  children: ReactNode
}

export default function Panel({
  title,
  description,
  action,
  icon,
  flush = false,
  className,
  children,
}: PanelProps) {
  return (
    <section
      className={cn(
        "overflow-hidden rounded-md border border-border bg-card",
        className,
      )}
    >
      {title && (
        <header className="flex items-start justify-between gap-3 border-b border-border px-4 py-3">
          <div className="flex min-w-0 items-start gap-3">
            {icon && (
              <span className="mt-0.5 shrink-0 text-muted-foreground">
                {icon}
              </span>
            )}
            <div className="min-w-0">
              <h2 className="text-sm font-semibold">{title}</h2>
              {description && (
                <p className="mt-0.5 text-xs text-muted-foreground">
                  {description}
                </p>
              )}
            </div>
          </div>
          {action}
        </header>
      )}
      <div className={cn(!flush && "space-y-4 p-4")}>{children}</div>
    </section>
  )
}
