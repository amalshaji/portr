import * as React from "react"
import { ArrowLeft, Copy, LoaderCircle, Pause, Play, RadioTower, Search } from "lucide-react"
import { toast } from "sonner"

import { PayloadViewer } from "@/components/payload-viewer"
import {
  Chip,
  KVTable,
  MethodTag,
  SectionLabel,
  StatusPill,
} from "@/components/terminal-primitives"
import { replayRequest } from "@/lib/api"
import {
  buildCurlCommand,
  decodeBase64ToText,
  flattenHeaders,
  formatTime,
  getHeaderValue,
  parseCookiesHeader,
  parseQueryParams,
  reasonPhrase,
} from "@/lib/dashboard"
import type { RequestRecord, RequestSummary } from "@/types"

/* ── Request sidebar ─────────────────────────────────────── */

const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "WS"]
const STATUS_FILTERS = ["all", "2xx", "3xx", "4xx", "5xx"] as const
type StatusFilter = (typeof STATUS_FILTERS)[number]

export function RequestSidebar({
  requests,
  total,
  hasMore,
  loadingMore,
  onLoadMore,
  selectedId,
  onSelect,
}: {
  requests: RequestSummary[]
  total: number
  hasMore: boolean
  loadingMore: boolean
  onLoadMore: () => void
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
      if (next.has(m)) next.delete(m)
      else next.add(m)
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
      <div
        className="portr-sidebar-list"
        onScroll={(e) => {
          const el = e.currentTarget
          if (
            hasMore &&
            !loadingMore &&
            el.scrollTop + el.clientHeight >= el.scrollHeight - 200
          ) {
            onLoadMore()
          }
        }}
      >
        {filtered.length === 0 ? (
          <div
            className="flex min-h-40 flex-col items-center justify-center gap-2 p-4 text-center font-mono text-[11px]"
            style={{ color: "var(--muted-foreground)" }}
          >
            <RadioTower className="size-4" style={{ color: "var(--tm-muted-2)" }} />
            {total === 0 ? "No request traces" : "no requests match filter"}
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
                ...(r.ID === selectedId ? { background: "var(--muted)" } : {}),
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
        {hasMore && (
          <button
            onClick={onLoadMore}
            disabled={loadingMore}
            className="flex w-full items-center justify-center gap-1.5 py-2.5 font-mono text-[10px] transition-colors hover:bg-muted/50 disabled:opacity-60"
            style={{ color: "var(--muted-foreground)" }}
          >
            {loadingMore ? (
              <>
                <LoaderCircle className="size-3 animate-spin" />
                loading older requests…
              </>
            ) : (
              `load older requests (${requests.length.toLocaleString()} of ${total.toLocaleString()})`
            )}
          </button>
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
          {requests.length.toLocaleString()} / {total.toLocaleString()} requests
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

  return <pre className="portr-raw-view">{raw}</pre>
}

export function RequestDetailPane({
  request,
  loading,
  onRefresh,
  onEditReplay,
}: {
  request: RequestRecord | null
  loading: boolean
  onRefresh: () => Promise<void>
  onEditReplay: () => void
}) {
  const [tab, setTab] = React.useState<DetailTab>("headers")
  const [replaying, setReplaying] = React.useState(false)

  React.useEffect(() => {
    setTab("headers")
  }, [request?.ID])

  if (!request && loading) {
    return (
      <div
        className="flex flex-col items-center justify-center gap-3 p-8 text-center"
        style={{ background: "var(--background)" }}
      >
        <LoaderCircle className="size-5 animate-spin" style={{ color: "var(--tm-muted-2)" }} />
        <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
          loading request…
        </p>
      </div>
    )
  }

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

        {tab === "query" &&
          (queryParams.length > 0 ? (
            <KVTable rows={queryParams} />
          ) : (
            <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
              no query parameters
            </p>
          ))}

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

        {tab === "cookies" &&
          (cookies.length > 0 ? (
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
          ))}

        {tab === "raw" && <RawView request={request} />}
      </div>
    </div>
  )
}
