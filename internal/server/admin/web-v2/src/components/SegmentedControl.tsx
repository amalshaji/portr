import type { ReactNode } from "react"
import { cn } from "@/lib/utils"

/**
 * A small set of mutually exclusive choices shown inline — filters, install
 * methods, the theme switch. One implementation so the padding and selected
 * treatment cannot drift between them.
 */
export interface SegmentedOption<T extends string> {
  value: T
  /** Visible content. Omit when the option is icon-only, and set `label`. */
  content?: ReactNode
  /** Accessible name. Required when `content` is an icon. */
  label?: string
}

interface SegmentedControlProps<T extends string> {
  /** Names the group for screen readers. */
  ariaLabel: string
  options: SegmentedOption<T>[]
  value: T
  onChange: (value: T) => void
  /** `icon` renders square buttons for glyph-only options. */
  variant?: "text" | "icon"
  className?: string
}

export default function SegmentedControl<T extends string>({
  ariaLabel,
  options,
  value,
  onChange,
  variant = "text",
  className,
}: SegmentedControlProps<T>) {
  return (
    <div
      role="radiogroup"
      aria-label={ariaLabel}
      className={cn(
        "flex w-fit items-center gap-0.5 rounded-md border border-border bg-muted/60 p-0.5",
        className,
      )}
    >
      {options.map((option) => {
        const selected = option.value === value
        return (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={selected}
            aria-label={option.label}
            title={option.label}
            onClick={() => onChange(option.value)}
            className={cn(
              "flex items-center justify-center rounded-sm text-xs font-medium outline-none transition-colors duration-(--portr-duration-micro) ease-portr focus-visible:ring-2 focus-visible:ring-ring",
              variant === "icon" ? "size-7" : "px-2.5 py-1",
              selected
                ? "bg-background text-foreground shadow-xs"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            {option.content ?? option.label}
          </button>
        )
      })}
    </div>
  )
}
