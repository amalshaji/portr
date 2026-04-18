import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import {
  ArrowLeft,
  Copy,
  Download,
  LoaderCircle,
  Pause,
  Play,
  RadioTower,
  Search,
  Waves,
} from "lucide-react"
import { toast } from "sonner"

import { ServerUnavailableBanner } from "@/components/server-unavailable-banner"
import { ThemeToggle } from "@/components/theme-toggle"
import { PayloadViewer } from "@/components/payload-viewer"
import { ReplayDialog } from "@/components/replay-dialog"
import {
  getRequests,
  getWebSocketSession,
  getWebSocketSessions,
  replayRequest,
} from "@/lib/api"
import {
  buildCurlCommand,
  decodeBase64ToBytes,
  decodeBase64ToText,
  flattenHeaders,
  formatTime,
  getHeaderValue,
  parseCookiesHeader,
  parseQueryParams,
  parseTunnelId,
  payloadPreview,
  reasonPhrase,
  websocketPayloadLabel,
} from "@/lib/dashboard"
import type { RequestRecord, WebSocketEvent, WebSocketSession } from "@/types"

/* ── Shared visual primitives ─────────────────────────────── */

function LogoMark() {
  return (
    <div
      className="grid h-[18px] w-[18px] shrink-0 place-items-center rounded-[3px]"
      style={{ border: "1.5px solid var(--foreground)" }}
    >
      <div className="h-1.5 w-1.5 rounded-[1px]" style={{ background: "var(--tm-green)" }} />
    </div>
  )
}

function MethodTag({ method }: { method: string }) {
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

function StatusPill({ code }: { code: number }) {
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

function Chip({
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

function SectionLabel({ children }: { children: React.ReactNode }) {
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

function KVTable({ rows }: { rows: [string, string][] }) {
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

/* ── TopBar ──────────────────────────────────────────────── */

function TopBar({
  subdomain,
  localport,
  onBack,
}: {
  subdomain: string
  localport: string
  onBack: () => void
}) {
  return (
    <header
      className="sticky top-0 z-10 flex h-11 items-center gap-3 border-b border-border bg-background px-4"
      style={{ boxShadow: "0 1px 0 color-mix(in srgb, var(--foreground) 4%, transparent)" }}
    >
      <div className="flex items-center gap-2">
        <LogoMark />
        <span className="font-mono text-xs font-semibold tracking-[-0.01em]">portr</span>
      </div>
      <div className="flex items-center gap-1.5 font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
        <span style={{ color: "var(--tm-line-2)" }}>/</span>
        <button onClick={onBack} className="hover:underline" style={{ color: "var(--muted-foreground)" }}>
          connections
        </button>
        <span style={{ color: "var(--tm-line-2)" }}>/</span>
        <span style={{ color: "var(--foreground)" }}>
          {subdomain}:{localport}
        </span>
      </div>
      <div className="flex-1" />
      <ThemeToggle />
    </header>
  )
}

/* ── Request sidebar ─────────────────────────────────────── */

const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "WS"]
const STATUS_FILTERS = ["all", "2xx", "3xx", "4xx", "5xx"] as const
type StatusFilter = (typeof STATUS_FILTERS)[number]

function RequestSidebar({
  requests,
  selectedId,
  onSelect,
}: {
  requests: RequestRecord[]
  selectedId: string | null
  onSelect: (id: string) => void
}) {
  const [query, setQuery] = React.useState("")
  const [methodFilter, setMethodFilter] = React.useState<Set<string>>(new Set())
  const [statusFilter, setStatusFilter] = React.useState<StatusFilter>("all")
  const [paused, setPaused] = React.useState(false)
  const searchRef = React.useRef<HTMLInputElement>(null)

  React.useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault()
        searchRef.current?.focus()
        searchRef.current?.select()
      }
    }
    window.addEventListener("keydown", handler)
    return () => window.removeEventListener("keydown", handler)
  }, [])

  const toggleMethod = (m: string) =>
    setMethodFilter((cur) => {
      const next = new Set(cur)
      next.has(m) ? next.delete(m) : next.add(m)
      return next
    })

  const filtered = React.useMemo(() => {
    return requests.filter((r) => {
      if (query && !r.Url.toLowerCase().includes(query.toLowerCase())) return false
      if (methodFilter.size > 0 && !methodFilter.has(r.Method.toUpperCase())) return false
      if (statusFilter !== "all") {
        const bucket = `${Math.floor(r.ResponseStatusCode / 100)}xx` as StatusFilter
        if (bucket !== statusFilter) return false
      }
      return true
    })
  }, [requests, query, methodFilter, statusFilter])

  return (
    <div className="portr-sidebar">
      {/* Toolbar */}
      <div
        className="flex flex-col gap-2 p-2"
        style={{ borderBottom: "1px solid var(--border)", background: "var(--muted)" }}
      >
        <div className="relative flex items-center">
          <Search
            className="pointer-events-none absolute left-2 size-3"
            style={{ color: "var(--tm-muted-2)" }}
          />
          <input
            ref={searchRef}
            placeholder="filter path..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="h-7 w-full rounded-[4px] border border-border bg-background pl-6 font-mono text-xs outline-none focus:border-foreground/40"
            style={{ color: "var(--foreground)" }}
          />
        </div>
        <div className="flex flex-wrap gap-1">
          {HTTP_METHODS.map((m) => (
            <Chip key={m} active={methodFilter.has(m)} onClick={() => toggleMethod(m)}>
              {m}
            </Chip>
          ))}
        </div>
        <div className="flex flex-wrap items-center gap-1">
          {STATUS_FILTERS.map((s) => (
            <Chip key={s} active={statusFilter === s} onClick={() => setStatusFilter(s)}>
              {s}
            </Chip>
          ))}
          <div className="flex-1" />
          <Chip active={paused} onClick={() => setPaused((p) => !p)}>
            {paused ? <Play className="size-2.5" /> : <Pause className="size-2.5" />}
            {paused ? "paused" : "capturing"}
          </Chip>
        </div>
      </div>

      {/* List */}
      <div className="portr-sidebar-list">
        {filtered.length === 0 ? (
          <div
            className="flex min-h-40 flex-col items-center justify-center gap-2 p-4 text-center font-mono text-[11px]"
            style={{ color: "var(--muted-foreground)" }}
          >
            <RadioTower className="size-4" style={{ color: "var(--tm-muted-2)" }} />
            no requests match filter
          </div>
        ) : (
          filtered.map((r) => (
            <button
              key={r.ID}
              className="portr-req-row relative w-full text-left transition-colors hover:bg-muted/50"
              style={{
                display: "grid",
                gridTemplateColumns: "50px 1fr 40px",
                gap: "8px",
                alignItems: "center",
                padding: "8px 10px",
                borderBottom: "1px solid var(--border)",
                ...(r.ID === selectedId
                  ? { background: "var(--muted)" }
                  : {}),
              }}
              onClick={() => onSelect(r.ID)}
            >
              {r.ID === selectedId && (
                <div
                  className="absolute inset-y-0 left-0 w-0.5"
                  style={{ background: "var(--foreground)" }}
                />
              )}
              <MethodTag method={r.Method} />
              <div className="min-w-0">
                <p
                  className="truncate font-mono text-xs"
                  style={{ color: "var(--foreground)" }}
                >
                  {r.Url}
                </p>
                <p className="mt-0.5 font-mono text-[10px]" style={{ color: "var(--muted-foreground)" }}>
                  {formatTime(r.LoggedAt)}
                </p>
              </div>
              <div className="text-right">
                <StatusPill code={r.ResponseStatusCode} />
              </div>
            </button>
          ))
        )}
      </div>

      {/* Footer */}
      <div
        className="flex items-center justify-between px-3 py-1.5 font-mono text-[10px]"
        style={{
          borderTop: "1px solid var(--border)",
          background: "var(--muted)",
          color: "var(--muted-foreground)",
        }}
      >
        <span>
          {filtered.length} / {requests.length} requests
        </span>
        <span style={{ color: "var(--tm-muted-2)" }}>⌘K to search</span>
      </div>
    </div>
  )
}

/* ── Request detail pane ─────────────────────────────────── */

type DetailTab = "headers" | "query" | "reqbody" | "resbody" | "cookies" | "raw"

function RawView({ request }: { request: RequestRecord }) {
  const path = request.Url
  const reqHeaders = flattenHeaders(request.Headers)
  const resHeaders = flattenHeaders(request.ResponseHeaders)
  const reqBody = decodeBase64ToText(request.Body)
  const resBody = decodeBase64ToText(request.ResponseBody)

  const raw = [
    `${request.Method} ${path} HTTP/1.1`,
    ...Object.entries(reqHeaders).map(([k, v]) => `${k}: ${v}`),
    "",
    reqBody || "",
    "",
    "─".repeat(48),
    "",
    `HTTP/1.1 ${request.ResponseStatusCode} ${reasonPhrase(request.ResponseStatusCode)}`,
    ...Object.entries(resHeaders).map(([k, v]) => `${k}: ${v}`),
    "",
    resBody || "",
  ].join("\n")

  return (
    <pre className="portr-raw-view">{raw}</pre>
  )
}

function RequestDetailPane({
  request,
  onRefresh,
  onEditReplay,
}: {
  request: RequestRecord | null
  onRefresh: () => Promise<void>
  onEditReplay: () => void
}) {
  const [tab, setTab] = React.useState<DetailTab>("headers")
  const [replaying, setReplaying] = React.useState(false)

  React.useEffect(() => {
    setTab("headers")
  }, [request?.ID])

  if (!request) {
    return (
      <div
        className="flex flex-col items-center justify-center gap-3 p-8 text-center"
        style={{ background: "var(--background)" }}
      >
        <div
          className="grid h-12 w-12 place-items-center rounded-md"
          style={{ border: "1.5px dashed var(--tm-line-2)" }}
        >
          <ArrowLeft className="size-5" style={{ color: "var(--tm-muted-2)" }} />
        </div>
        <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
          select a request to inspect
        </p>
      </div>
    )
  }

  async function handleReplay() {
    if (!request) return
    setReplaying(true)
    try {
      await replayRequest(request.ID)
      toast.success("Replay dispatched")
      await onRefresh()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to replay")
    } finally {
      setReplaying(false)
    }
  }

  function handleCopyCurl() {
    navigator.clipboard
      ?.writeText(buildCurlCommand(request!))
      .then(() => toast.success("cURL copied to clipboard"))
      .catch(() => toast.error("Unable to copy to clipboard"))
  }

  const reqHeaders = Object.entries(flattenHeaders(request.Headers)) as [string, string][]
  const resHeaders = Object.entries(flattenHeaders(request.ResponseHeaders)) as [string, string][]
  const queryParams = parseQueryParams(request.Url)
  const cookieHeader = getHeaderValue(request.Headers, "Cookie")
  const cookies = parseCookiesHeader(cookieHeader)

  const tabDefs: { id: DetailTab; label: string; count?: number }[] = [
    { id: "headers",  label: "Headers",       count: reqHeaders.length + resHeaders.length },
    { id: "query",    label: "Query",          count: queryParams.length },
    { id: "reqbody",  label: "Request body" },
    { id: "resbody",  label: "Response body" },
    { id: "cookies",  label: "Cookies",        count: cookies.length },
    { id: "raw",      label: "Raw" },
  ]

  return (
    <div className="portr-detail-pane">
      {/* Pane header */}
      <div
        className="flex items-center justify-between gap-3 px-4 py-2.5"
        style={{ borderBottom: "1px solid var(--border)", background: "var(--background)" }}
      >
        <div className="flex min-w-0 items-center gap-2">
          <MethodTag method={request.Method} />
          <StatusPill code={request.ResponseStatusCode} />
          <span
            className="truncate font-mono text-[13px]"
            style={{ color: "var(--foreground)" }}
            title={request.Url}
          >
            {request.Url}
          </span>
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          <button
            onClick={handleCopyCurl}
            className="flex items-center gap-1 rounded-[6px] border border-border bg-background px-2 py-1 font-mono text-[11px] transition-colors hover:bg-muted"
            style={{ color: "var(--foreground)" }}
          >
            <Copy className="size-3" />
            copy as cURL
          </button>
          <button
            onClick={onEditReplay}
            className="flex items-center gap-1 rounded-[6px] border border-border bg-background px-2 py-1 font-mono text-[11px] transition-colors hover:bg-muted"
            style={{ color: "var(--foreground)" }}
          >
            edit &amp; replay
          </button>
          <button
            disabled={replaying}
            onClick={handleReplay}
            className="flex items-center gap-1 rounded-[6px] border px-2 py-1 font-mono text-[11px] font-medium transition-colors hover:opacity-80 disabled:opacity-50"
            style={{
              background: "var(--foreground)",
              color: "var(--background)",
              borderColor: "var(--foreground)",
            }}
          >
            {replaying ? <LoaderCircle className="size-3 animate-spin" /> : null}
            replay
          </button>
        </div>
      </div>

      {/* Meta strip */}
      <div
        className="flex flex-wrap gap-3 px-4 py-2"
        style={{
          borderBottom: "1px solid var(--border)",
          background: "var(--muted)",
          fontSize: "11px",
          fontFamily: "var(--font-mono)",
          color: "var(--muted-foreground)",
        }}
      >
        <span>
          <span style={{ color: "var(--tm-muted-2)" }}>time </span>
          <span style={{ color: "var(--tm-ink-2)" }}>{formatTime(request.LoggedAt)}</span>
        </span>
        <span>
          <span style={{ color: "var(--tm-muted-2)" }}>method </span>
          <span style={{ color: "var(--tm-ink-2)" }}>{request.Method}</span>
        </span>
        <span>
          <span style={{ color: "var(--tm-muted-2)" }}>status </span>
          <span style={{ color: "var(--tm-ink-2)" }}>
            {request.ResponseStatusCode} {reasonPhrase(request.ResponseStatusCode)}
          </span>
        </span>
        <span>
          <span style={{ color: "var(--tm-muted-2)" }}>host </span>
          <span style={{ color: "var(--tm-ink-2)" }}>{request.Host}</span>
        </span>
        {request.IsReplayed ? (
          <span
            className="rounded-[3px] px-1.5"
            style={{ background: "var(--tm-green-bg)", color: "var(--tm-green-ink)" }}
          >
            replayed
          </span>
        ) : null}
      </div>

      {/* Tabs */}
      <div
        className="flex overflow-x-auto"
        style={{ borderBottom: "1px solid var(--border)", background: "var(--background)" }}
      >
        {tabDefs.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className="flex shrink-0 items-center gap-1.5 border-b-2 px-3 py-2 font-mono text-xs transition-colors hover:text-foreground"
            style={
              tab === t.id
                ? { color: "var(--foreground)", borderColor: "var(--foreground)" }
                : { color: "var(--muted-foreground)", borderColor: "transparent" }
            }
          >
            {t.label}
            {t.count != null && (
              <span
                className="rounded-full border px-1.5 font-mono text-[10px] leading-[1.4]"
                style={{
                  borderColor: "var(--tm-line-2)",
                  color: tab === t.id ? "var(--foreground)" : "var(--tm-muted-2)",
                }}
              >
                {t.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab body */}
      <div className="portr-tab-body">
        {tab === "headers" && (
          <div>
            <SectionLabel>Request headers</SectionLabel>
            <KVTable rows={reqHeaders} />
            <div className="mt-5">
              <SectionLabel>Response headers</SectionLabel>
              <KVTable rows={resHeaders} />
            </div>
          </div>
        )}

        {tab === "query" && (
          queryParams.length > 0
            ? <KVTable rows={queryParams} />
            : (
              <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
                no query parameters
              </p>
            )
        )}

        {tab === "reqbody" && (
          <div className="h-full min-h-[200px]">
            <PayloadViewer request={request} type="request" />
          </div>
        )}

        {tab === "resbody" && (
          <div className="h-full min-h-[200px]">
            <PayloadViewer request={request} type="response" />
          </div>
        )}

        {tab === "cookies" && (
          cookies.length > 0 ? (
            <div className="space-y-2">
              {cookies.map((c, i) => (
                <div
                  key={i}
                  className="rounded-[4px] border p-3 font-mono text-xs"
                  style={{ borderColor: "var(--border)", background: "var(--background)" }}
                >
                  <div className="mb-1 font-semibold" style={{ color: "var(--foreground)" }}>
                    {c.name}
                  </div>
                  <div
                    className="mb-2 break-all text-[11px]"
                    style={{ color: "var(--tm-green-ink)" }}
                  >
                    {c.value || <span style={{ color: "var(--tm-muted-2)" }}>(empty)</span>}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
              no cookies in request
            </p>
          )
        )}

        {tab === "raw" && <RawView request={request} />}
      </div>
    </div>
  )
}

/* ── WebSocket detail ───────────────────────────────────── */

function WebSocketDetailPane({
  session,
  events,
}: {
  session: WebSocketSession | null
  events: WebSocketEvent[]
}) {
  const [selectedEventId, setSelectedEventId] = React.useState<string | null>(null)

  React.useEffect(() => {
    setSelectedEventId(events[0]?.id || null)
  }, [session?.ID])

  if (!session) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 p-8 text-center">
        <div
          className="grid h-12 w-12 place-items-center rounded-md"
          style={{ border: "1.5px dashed var(--tm-line-2)" }}
        >
          <Waves className="size-5" style={{ color: "var(--tm-muted-2)" }} />
        </div>
        <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
          select a WebSocket session
        </p>
      </div>
    )
  }

  const selectedEvent = events.find((e) => e.id === selectedEventId) || events[0] || null

  const reqHeaders = Object.entries(flattenHeaders(session.RequestHeaders)) as [string, string][]
  const resHeaders = Object.entries(flattenHeaders(session.ResponseHeaders)) as [string, string][]

  const [wsTab, setWsTab] = React.useState<"timeline" | "request" | "response">("timeline")

  return (
    <div className="portr-detail-pane">
      {/* Pane header */}
      <div
        className="px-4 py-2.5"
        style={{ borderBottom: "1px solid var(--border)" }}
      >
        <div className="flex items-center gap-2">
          <MethodTag method={session.Method} />
          <span
            className="truncate font-mono text-[13px]"
            style={{ color: "var(--foreground)" }}
          >
            {session.URL}
          </span>
        </div>
      </div>

      {/* Meta strip */}
      <div
        className="flex flex-wrap gap-3 px-4 py-2 font-mono text-[11px]"
        style={{
          borderBottom: "1px solid var(--border)",
          background: "var(--muted)",
          color: "var(--muted-foreground)",
        }}
      >
        <span>
          <span style={{ color: "var(--tm-muted-2)" }}>frames </span>
          <span style={{ color: "var(--tm-ink-2)" }}>{session.EventCount}</span>
        </span>
        <span>
          <span style={{ color: "var(--tm-muted-2)" }}>client </span>
          <span style={{ color: "var(--tm-ink-2)" }}>{session.ClientEventCount}</span>
        </span>
        <span>
          <span style={{ color: "var(--tm-muted-2)" }}>server </span>
          <span style={{ color: "var(--tm-ink-2)" }}>{session.ServerEventCount}</span>
        </span>
        <span>
          <span style={{ color: "var(--tm-muted-2)" }}>status </span>
          <span
            style={{
              color: session.ClosedAt ? "var(--tm-muted-2)" : "var(--tm-green-ink)",
            }}
          >
            {session.ClosedAt ? "closed" : "open"}
          </span>
        </span>
        {session.CloseCode ? (
          <span>
            <span style={{ color: "var(--tm-muted-2)" }}>close </span>
            <span style={{ color: "var(--tm-ink-2)" }}>
              {session.CloseCode}
              {session.CloseReason ? ` · ${session.CloseReason}` : ""}
            </span>
          </span>
        ) : null}
      </div>

      {/* Tabs */}
      <div
        className="flex"
        style={{ borderBottom: "1px solid var(--border)" }}
      >
        {(["timeline", "request", "response"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setWsTab(t)}
            className="border-b-2 px-3 py-2 font-mono text-xs capitalize transition-colors hover:text-foreground"
            style={
              wsTab === t
                ? { color: "var(--foreground)", borderColor: "var(--foreground)" }
                : { color: "var(--muted-foreground)", borderColor: "transparent" }
            }
          >
            {t === "timeline" ? "Timeline" : t === "request" ? "Request headers" : "Response headers"}
          </button>
        ))}
      </div>

      <div className="portr-tab-body">
        {wsTab === "timeline" && (
          <div>
            {events.length === 0 ? (
              <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
                no frames captured yet
              </p>
            ) : (
              <div
                className="overflow-hidden rounded-[4px]"
                style={{ border: "1px solid var(--border)" }}
              >
                <div
                  className="grid grid-cols-[auto_auto_auto_1fr_auto] gap-3 px-3 py-1.5 font-mono text-[10px] uppercase tracking-[0.12em]"
                  style={{
                    borderBottom: "1px solid var(--border)",
                    background: "var(--muted)",
                    color: "var(--muted-foreground)",
                  }}
                >
                  <span>Dir</span>
                  <span>Type</span>
                  <span>At</span>
                  <span>Preview</span>
                  <span className="text-right">Size</span>
                </div>
                <div style={{ borderTop: "none" }}>
                  {events.map((event) => (
                    <div
                      key={event.id}
                      className="grid cursor-pointer grid-cols-[auto_auto_auto_1fr_auto] items-center gap-3 px-3 py-2 transition-colors hover:bg-muted/40"
                      style={{
                        borderBottom: "1px solid var(--border)",
                        ...(event.id === selectedEvent?.id
                          ? { background: "var(--muted)", boxShadow: "inset 2px 0 0 var(--foreground)" }
                          : {}),
                      }}
                      onClick={() => setSelectedEventId(event.id)}
                    >
                      <span
                        className="rounded-[3px] border px-1.5 font-mono text-[10px] leading-5"
                        style={
                          event.direction === "client"
                            ? { color: "var(--tm-get-ink)", background: "var(--tm-get-bg)", borderColor: "var(--tm-get-border)" }
                            : { color: "var(--tm-post-ink)", background: "var(--tm-post-bg)", borderColor: "var(--tm-post-border)" }
                        }
                      >
                        {event.direction}
                      </span>
                      <span
                        className="rounded-[3px] border px-1.5 font-mono text-[10px] leading-5"
                        style={{ color: "var(--muted-foreground)", borderColor: "var(--tm-line-2)" }}
                      >
                        {event.opcode_name}
                      </span>
                      <span className="font-mono text-[10px]" style={{ color: "var(--muted-foreground)" }}>
                        {formatTime(event.logged_at)}
                      </span>
                      <div className="min-w-0">
                        <p className="truncate font-mono text-xs" style={{ color: "var(--foreground)" }}>
                          {payloadPreview(event)}
                        </p>
                        {event.payload && !event.payload_text ? (
                          <button
                            className="font-mono text-[10px] hover:underline"
                            style={{ color: "var(--muted-foreground)" }}
                            onClick={(e) => {
                              e.stopPropagation()
                              const url = URL.createObjectURL(
                                new Blob([decodeBase64ToBytes(event.payload)])
                              )
                              const a = document.createElement("a")
                              a.href = url
                              a.download = `${event.id}.bin`
                              a.click()
                              setTimeout(() => URL.revokeObjectURL(url), 1000)
                            }}
                          >
                            <Download className="mr-0.5 inline size-3" />
                            download
                          </button>
                        ) : null}
                      </div>
                      <span className="text-right font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
                        {event.payload_length} B
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {selectedEvent ? (
              <div className="mt-4">
                <SectionLabel>
                  {websocketPayloadLabel(selectedEvent)} · {selectedEvent.payload_length} B
                </SectionLabel>
                {selectedEvent.payload_text ? (
                  <pre
                    className="min-h-20 whitespace-pre-wrap break-all rounded-[4px] border p-3 font-mono text-xs leading-relaxed"
                    style={{
                      background: "var(--muted)",
                      borderColor: "var(--border)",
                      color: "var(--foreground)",
                    }}
                  >
                    {selectedEvent.payload_text}
                  </pre>
                ) : (
                  <p
                    className="font-mono text-xs"
                    style={{ color: "var(--muted-foreground)" }}
                  >
                    Binary payload. Download to inspect.
                  </p>
                )}
              </div>
            ) : null}
          </div>
        )}

        {wsTab === "request" && <KVTable rows={reqHeaders} />}
        {wsTab === "response" && <KVTable rows={resHeaders} />}
      </div>
    </div>
  )
}

/* ── WebSocket session sidebar ───────────────────────────── */

function WebSocketSidebar({
  sessions,
  selectedId,
  onSelect,
}: {
  sessions: WebSocketSession[]
  selectedId: string | null
  onSelect: (id: string) => void
}) {
  return (
    <div className="portr-sidebar">
      <div
        className="px-3 py-2.5"
        style={{
          borderBottom: "1px solid var(--border)",
          background: "var(--muted)",
          fontFamily: "var(--font-mono)",
          fontSize: "11px",
          color: "var(--muted-foreground)",
          textTransform: "uppercase",
          letterSpacing: "0.1em",
        }}
      >
        Sessions
      </div>
      <div className="portr-sidebar-list">
        {sessions.length === 0 ? (
          <div
            className="flex min-h-40 flex-col items-center justify-center gap-2 p-4 text-center font-mono text-[11px]"
            style={{ color: "var(--muted-foreground)" }}
          >
            <Waves className="size-4" style={{ color: "var(--tm-muted-2)" }} />
            no WebSocket sessions
          </div>
        ) : (
          sessions.map((s) => (
            <button
              key={s.ID}
              className="relative w-full text-left transition-colors hover:bg-muted/50"
              style={{
                padding: "10px",
                borderBottom: "1px solid var(--border)",
                ...(s.ID === selectedId ? { background: "var(--muted)" } : {}),
              }}
              onClick={() => onSelect(s.ID)}
            >
              {s.ID === selectedId && (
                <div
                  className="absolute inset-y-0 left-0 w-0.5"
                  style={{ background: "var(--foreground)" }}
                />
              )}
              <div className="flex items-center justify-between gap-2">
                <MethodTag method={s.Method} />
                <span className="font-mono text-[10px]" style={{ color: "var(--muted-foreground)" }}>
                  {s.ClosedAt ? formatTime(s.ClosedAt) : "open"}
                </span>
              </div>
              <p
                className="mt-1.5 truncate font-mono text-xs"
                style={{ color: "var(--foreground)" }}
              >
                {s.URL}
              </p>
              <div className="mt-1 flex gap-2 font-mono text-[10px]" style={{ color: "var(--muted-foreground)" }}>
                <span>{s.EventCount} frames</span>
                <span
                  style={{ color: s.ClosedAt ? "var(--tm-muted-2)" : "var(--tm-green-ink)" }}
                >
                  {s.ClosedAt ? "closed" : "open"}
                </span>
              </div>
            </button>
          ))
        )}
      </div>
    </div>
  )
}

/* ── Main page ───────────────────────────────────────────── */

export function TunnelPage() {
  const params = useParams()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const tunnel = React.useMemo(
    () => parseTunnelId(params.id || ""),
    [params.id]
  )

  const [requests, setRequests] = React.useState<RequestRecord[]>([])
  const [sessions, setSessions] = React.useState<WebSocketSession[]>([])
  const [selectedSession, setSelectedSession] = React.useState<WebSocketSession | null>(null)
  const [selectedSessionEvents, setSelectedSessionEvents] = React.useState<WebSocketEvent[]>([])
  const [loading, setLoading] = React.useState(true)
  const [pollingError, setPollingError] = React.useState<string | null>(null)
  const [replayOpen, setReplayOpen] = React.useState(false)

  const activeTab = searchParams.get("tab") === "websocket" ? "websocket" : "http"
  const selectedRequestId = searchParams.get("request")
  const selectedSessionId = searchParams.get("session")

  const updateSearch = React.useCallback(
    (values: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams)
      Object.entries(values).forEach(([k, v]) => {
        if (!v) next.delete(k)
        else next.set(k, v)
      })
      setSearchParams(next, { replace: true })
    },
    [searchParams, setSearchParams]
  )

  const loadSummary = React.useEffectEvent(async (refresh = false) => {
    if (!tunnel) return
    if (!refresh) setLoading(true)
    try {
      const [reqData, sessData] = await Promise.all([
        getRequests(tunnel.subdomain, tunnel.localport),
        getWebSocketSessions(tunnel.subdomain, tunnel.localport),
      ])
      setRequests(reqData.requests)
      setSessions(sessData.sessions)
      setPollingError(null)
    } catch (err) {
      setPollingError(err instanceof Error ? err.message : null)
    } finally {
      setLoading(false)
    }
  })

  const loadSession = React.useEffectEvent(async (sessionId: string) => {
    try {
      const data = await getWebSocketSession(sessionId)
      setSelectedSession(data.session)
      setSelectedSessionEvents(data.events)
    } catch {}
  })

  const pollTick = React.useEffectEvent(() => {
    loadSummary(true)
    if (selectedSessionId) loadSession(selectedSessionId)
  })

  React.useEffect(() => {
    if (!tunnel) return
    loadSummary()
    const interval = window.setInterval(pollTick, 2000)
    return () => window.clearInterval(interval)
  }, [tunnel])

  React.useEffect(() => {
    if (!requests.length) return
    const exists = selectedRequestId
      ? requests.some((r) => r.ID === selectedRequestId)
      : false
    if (!exists) updateSearch({ request: requests[0].ID })
  }, [requests, selectedRequestId])

  React.useEffect(() => {
    if (!sessions.length) {
      setSelectedSession(null)
      setSelectedSessionEvents([])
      return
    }
    const exists = selectedSessionId
      ? sessions.some((s) => s.ID === selectedSessionId)
      : false
    if (!selectedSessionId || !exists) {
      updateSearch({ session: sessions[0].ID })
      return
    }
    loadSession(selectedSessionId)
  }, [selectedSessionId, sessions])

  const selectedRequest =
    requests.find((r) => r.ID === selectedRequestId) || null

  if (!tunnel) {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <div
          className="max-w-sm space-y-3 rounded-md border p-8 text-center"
          style={{ borderColor: "var(--border)" }}
        >
          <h1 className="font-mono text-lg font-semibold">Tunnel not found</h1>
          <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
            Invalid route. Go back and select a tunnel.
          </p>
          <button
            onClick={() => navigate("/")}
            className="mt-2 rounded-[6px] border px-4 py-1.5 font-mono text-xs hover:bg-muted"
            style={{ borderColor: "var(--border)" }}
          >
            Back to dashboard
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-svh bg-background">
      <TopBar
        subdomain={tunnel.subdomain}
        localport={tunnel.localport}
        onBack={() => navigate("/")}
      />

      <div className="relative z-10 w-full px-6 pb-6 pt-5">
        {pollingError ? <ServerUnavailableBanner /> : null}

        {/* Detail header */}
        <div
          className="mb-4 pb-4"
          style={{ borderBottom: "1px solid var(--border)" }}
        >
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h1
                className="mb-1.5 flex flex-wrap items-center gap-2 font-mono text-lg font-semibold leading-none tracking-[-0.02em]"
                style={{ color: "var(--foreground)" }}
              >
                <span className="font-mono" style={{ color: "var(--tm-green-ink)" }}>
                  {tunnel.subdomain}
                </span>
                <span style={{ color: "var(--tm-muted-2)" }}>→</span>
                <span style={{ color: "var(--muted-foreground)" }}>
                  localhost:{tunnel.localport}
                </span>
              </h1>
              <div
                className="flex flex-wrap gap-3 font-mono text-[11px]"
                style={{ color: "var(--muted-foreground)" }}
              >
                <span>
                  <span style={{ color: "var(--tm-muted-2)" }}>http requests </span>
                  {requests.length}
                </span>
                <span>
                  <span style={{ color: "var(--tm-muted-2)" }}>ws sessions </span>
                  {sessions.length}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* HTTP / WebSocket tab switcher */}
        <div
          className="mb-4 flex gap-0"
          style={{ borderBottom: "1px solid var(--border)" }}
        >
          {(["http", "websocket"] as const).map((t) => (
            <button
              key={t}
              onClick={() =>
                updateSearch({
                  tab: t,
                  request: t === "http" ? selectedRequestId || requests[0]?.ID || null : null,
                  session: t === "websocket" ? selectedSessionId || sessions[0]?.ID || null : null,
                })
              }
              className="border-b-2 px-4 py-2 font-mono text-xs transition-colors hover:text-foreground"
              style={
                activeTab === t
                  ? { color: "var(--foreground)", borderColor: "var(--foreground)" }
                  : { color: "var(--muted-foreground)", borderColor: "transparent" }
              }
            >
              {t === "http" ? "HTTP inspector" : "WebSocket inspector"}
            </button>
          ))}
        </div>

        {/* Content */}
        {loading ? (
          <div className="portr-detail-layout animate-pulse">
            <div style={{ background: "var(--muted)" }} />
            <div style={{ background: "var(--background)" }} />
          </div>
        ) : activeTab === "http" ? (
          <div className="portr-detail-layout">
            <RequestSidebar
              requests={requests}
              selectedId={selectedRequestId}
              onSelect={(id) => updateSearch({ tab: "http", request: id })}
            />
            <RequestDetailPane
              request={selectedRequest}
              onRefresh={() => loadSummary(true)}
              onEditReplay={() => setReplayOpen(true)}
            />
          </div>
        ) : (
          <div className="portr-detail-layout">
            <WebSocketSidebar
              sessions={sessions}
              selectedId={selectedSessionId}
              onSelect={(id) => updateSearch({ tab: "websocket", session: id })}
            />
            <WebSocketDetailPane
              session={selectedSession}
              events={selectedSessionEvents}
            />
          </div>
        )}
      </div>

      <ReplayDialog
        open={replayOpen}
        onOpenChange={setReplayOpen}
        request={selectedRequest}
        onReplayed={() => loadSummary(true)}
      />
    </div>
  )
}
