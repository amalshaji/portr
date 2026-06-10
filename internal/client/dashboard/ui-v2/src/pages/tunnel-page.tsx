import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { ServerUnavailableBanner } from "@/components/server-unavailable-banner"
import { ThemeToggle } from "@/components/theme-toggle"
import { ReplayDialog } from "@/components/replay-dialog"
import { LogoMark } from "@/components/terminal-primitives"
import { RequestDetailPane, RequestSidebar } from "@/components/http-inspector"
import {
  WebSocketDetailPane,
  WebSocketSidebar,
} from "@/components/websocket-inspector"
import {
  getRequest,
  getRequests,
  getWebSocketSession,
  getWebSocketSessions,
} from "@/lib/api"
import { parseTunnelId } from "@/lib/dashboard"
import type {
  RequestRecord,
  RequestSummary,
  WebSocketEvent,
  WebSocketSession,
} from "@/types"

const REQUEST_PAGE_SIZE = 100

// Concat pages, dropping duplicates by ID (polling and scroll pages can overlap).
function dedupeRequests(first: RequestSummary[], rest: RequestSummary[]) {
  const seen = new Set(first.map((r) => r.ID))
  return [...first, ...rest.filter((r) => !seen.has(r.ID))]
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

/* ── Main page ───────────────────────────────────────────── */

export function TunnelPage() {
  const params = useParams()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const tunnel = React.useMemo(() => parseTunnelId(params.id || ""), [params.id])

  const [requests, setRequests] = React.useState<RequestSummary[]>([])
  const [totalRequests, setTotalRequests] = React.useState(0)
  const [loadingMore, setLoadingMore] = React.useState(false)
  const [selectedRequest, setSelectedRequest] = React.useState<RequestRecord | null>(null)
  const [requestLoading, setRequestLoading] = React.useState(false)
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
        getRequests(tunnel.subdomain, tunnel.localport, {
          limit: REQUEST_PAGE_SIZE,
          offset: 0,
        }),
        getWebSocketSessions(tunnel.subdomain, tunnel.localport),
      ])
      // When the first page already covers the whole dataset it is
      // authoritative — replace outright so deleted rows don't linger.
      // Otherwise keep older scrolled-in rows and merge the fresh head.
      setRequests((prev) =>
        reqData.total <= reqData.requests.length
          ? reqData.requests
          : dedupeRequests(reqData.requests, prev)
      )
      setTotalRequests(reqData.total)
      setSessions(sessData.sessions)
      setPollingError(null)
    } catch (err) {
      setPollingError(err instanceof Error ? err.message : null)
    } finally {
      setLoading(false)
    }
  })

  const loadMoreRequests = React.useEffectEvent(async () => {
    if (!tunnel || loadingMore || requests.length >= totalRequests) return
    setLoadingMore(true)
    try {
      const data = await getRequests(tunnel.subdomain, tunnel.localport, {
        limit: REQUEST_PAGE_SIZE,
        offset: requests.length,
      })
      setRequests((prev) => dedupeRequests(prev, data.requests))
      setTotalRequests(data.total)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to load more requests")
    } finally {
      setLoadingMore(false)
    }
  })

  const loadSession = React.useEffectEvent(async (sessionId: string) => {
    try {
      const data = await getWebSocketSession(sessionId)
      setSelectedSession(data.session)
      setSelectedSessionEvents(data.events)
    } catch {
      /* transient poll failure — surfaced via loadSummary's banner */
    }
  })

  const pollTick = React.useEffectEvent(() => {
    loadSummary(true)
    if (selectedSessionId) loadSession(selectedSessionId)
  })

  React.useEffect(() => {
    if (!tunnel) return
    setRequests([])
    setTotalRequests(0)
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

  React.useEffect(() => {
    if (!selectedRequestId) {
      setSelectedRequest(null)
      return
    }
    let cancelled = false
    setRequestLoading(true)
    getRequest(selectedRequestId)
      .then((data) => {
        if (!cancelled) setSelectedRequest(data.request)
      })
      .catch(() => {
        if (!cancelled) setSelectedRequest(null)
      })
      .finally(() => {
        if (!cancelled) setRequestLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [selectedRequestId])

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
        {pollingError ? <ServerUnavailableBanner className="mb-4" /> : null}

        {/* Detail header */}
        <div className="mb-4 pb-4" style={{ borderBottom: "1px solid var(--border)" }}>
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
                  {totalRequests.toLocaleString()}
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
        <div className="mb-4 flex gap-0" style={{ borderBottom: "1px solid var(--border)" }}>
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
              total={totalRequests}
              hasMore={requests.length < totalRequests}
              loadingMore={loadingMore}
              onLoadMore={loadMoreRequests}
              selectedId={selectedRequestId}
              onSelect={(id) => updateSearch({ tab: "http", request: id })}
            />
            <RequestDetailPane
              request={selectedRequest}
              loading={requestLoading}
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
