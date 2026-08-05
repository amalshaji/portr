import * as React from "react"

/* Shared terminal-aesthetic primitives used across the dashboard pages. */

export function LogoMark() {
  return (
    <img
      alt=""
      aria-hidden="true"
      className="h-[18px] w-[18px] shrink-0 object-cover"
      src={`${import.meta.env.BASE_URL}portr-mark.svg`}
    />
  )
}

export function MethodTag({ method }: { method: string }) {
  const map: Record<string, React.CSSProperties> = {
    GET:     { color: "var(--tm-get-ink)",    background: "var(--tm-get-bg)",    borderColor: "var(--tm-get-border)" },
    POST:    { color: "var(--tm-post-ink)",   background: "var(--tm-post-bg)",   borderColor: "var(--tm-post-border)" },
    PUT:     { color: "var(--tm-put-ink)",    background: "var(--tm-put-bg)",    borderColor: "var(--tm-put-border)" },
    PATCH:   { color: "var(--tm-put-ink)",    background: "var(--tm-put-bg)",    borderColor: "var(--tm-put-border)" },
    DELETE:  { color: "var(--tm-delete-ink)", background: "var(--tm-delete-bg)", borderColor: "var(--tm-delete-border)" },
    WS:      { color: "var(--tm-ws-ink)",     background: "var(--tm-ws-bg)",     borderColor: "var(--tm-ws-border)" },
  }
  const style = map[method.toUpperCase()] || {
    color: "var(--muted-foreground)",
    background: "var(--muted)",
    borderColor: "var(--tm-line-2)",
  }
  return (
    <span
      className="inline-block min-w-[50px] rounded-[3px] border px-1 text-center font-mono text-[10px] font-semibold leading-5"
      style={style}
    >
      {method}
    </span>
  )
}

export function StatusPill({ code }: { code: number }) {
  let style: React.CSSProperties
  if (code >= 500)      style = { color: "var(--tm-5xx-ink)", background: "var(--tm-5xx-bg)" }
  else if (code >= 400) style = { color: "var(--tm-4xx-ink)", background: "var(--tm-4xx-bg)" }
  else if (code >= 300) style = { color: "var(--tm-3xx-ink)", background: "var(--tm-3xx-bg)" }
  else if (code >= 200) style = { color: "var(--tm-green-ink)", background: "var(--tm-green-bg)" }
  else                  style = { color: "var(--tm-1xx-ink)", background: "var(--tm-1xx-bg)" }
  return (
    <span className="inline-flex items-center rounded-[3px] px-1.5 font-mono text-[11px] leading-5" style={style}>
      {code}
    </span>
  )
}

export function Chip({
  active,
  onClick,
  children,
}: {
  active?: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className="inline-flex items-center gap-1 rounded-[3px] border px-1.5 font-mono text-[10px] leading-5 transition-colors"
      style={
        active
          ? { background: "var(--foreground)", color: "var(--background)", borderColor: "var(--foreground)" }
          : { background: "var(--background)", color: "var(--muted-foreground)", borderColor: "var(--tm-line-2)" }
      }
    >
      {children}
    </button>
  )
}

export function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="mb-2 flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.1em]"
      style={{ color: "var(--muted-foreground)" }}
    >
      {children}
      <div className="h-px flex-1" style={{ background: "var(--border)" }} />
    </div>
  )
}

export function KVTable({ rows }: { rows: [string, string][] }) {
  if (!rows.length) {
    return (
      <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
        none
      </p>
    )
  }
  return (
    <table className="w-full border-collapse">
      <tbody>
        {rows.map(([k, v], i) => (
          <tr key={i} style={{ borderBottom: "1px dashed var(--border)" }}>
            <td
              className="w-[180px] whitespace-nowrap py-1.5 pr-4 font-mono text-xs"
              style={{ color: "var(--muted-foreground)" }}
            >
              {k}
            </td>
            <td
              className="break-all py-1.5 font-mono text-xs"
              style={{ color: "var(--foreground)" }}
            >
              {v}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
