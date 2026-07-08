import * as React from "react"
import { useNavigate } from "react-router-dom"
import { ChevronLeft, ChevronRight, RefreshCw, Search, Trash2 } from "lucide-react"
import { toast } from "sonner"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { ServerUnavailableBanner } from "@/components/server-unavailable-banner"
import { ThemeToggle } from "@/components/theme-toggle"
import { Chip, LogoMark } from "@/components/terminal-primitives"
import { deleteTunnelLogs, getTunnels } from "@/lib/api"
import { relativeTime } from "@/lib/dashboard"
import type { TunnelStats, TunnelStatus, TunnelSummary } from "@/types"

const STATUS_FILTERS = ["all", "live", "idle", "closed"] as const
type StatusFilter = (typeof STATUS_FILTERS)[number]

const TUNNEL_PAGE_SIZE = 25

const EMPTY_STATS: TunnelStats = {
  live_tunnel_count: 0,
  http_request_count: 0,
  websocket_session_count: 0,
  active_websocket_count: 0,
  last_activity_at: null,
}

function tunnelKey(tunnel: TunnelSummary) {
  return `${tunnel.Subdomain}:${tunnel.Localport}`
}

/* ── Small home-specific status components ─────────────── */

function Dot({ status }: { status: TunnelStatus }) {
  const base = "h-1.5 w-1.5 shrink-0 rounded-full"
  if (status === "live")
    return (
      <span
        className={`${base} portr-dot-live`}
        style={{ background: "var(--tm-green)" }}
      />
    )
  if (status === "idle")
    return (
      <span className={base} style={{ background: "var(--tm-amber)" }} />
    )
  return (
    <span className={base} style={{ background: "var(--tm-muted-2)" }} />
  )
}

function StatusBadge({ status }: { status: TunnelStatus }) {
  const styles: Record<TunnelStatus, React.CSSProperties> = {
    live: {
      color: "var(--tm-green-ink)",
      background: "var(--tm-green-bg)",
      borderColor: "var(--tm-green-border)",
    },
    idle: {
      color: "var(--muted-foreground)",
      background: "var(--muted)",
      borderColor: "var(--tm-line-2)",
    },
    closed: {
      color: "var(--tm-muted-2)",
      background: "var(--background)",
      borderColor: "var(--border)",
      textDecoration: "line-through",
      textDecorationColor: "var(--tm-line-2)",
    },
  }
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-[3px] border px-1.5 font-mono text-[11px] leading-5"
      style={styles[status]}
    >
      <Dot status={status} />
      {status}
    </span>
  )
}

function ProtoTag({ proto }: { proto: string }) {
  return (
    <span
      className="rounded-[3px] border px-1.5 font-mono text-[10px] font-semibold leading-5 tracking-[0.02em]"
      style={{ borderColor: "var(--tm-line-2)", color: "var(--tm-ink-2)" }}
    >
      {proto.toUpperCase()}
    </span>
  )
}

/* ── TopBar ─────────────────────────────────────────────── */

function TopBar({
  synced,
  refreshing,
  onRefresh,
}: {
  synced: string
  refreshing: boolean
  onRefresh: () => void
}) {
  return (
    <header
      className="sticky top-0 z-10 flex h-11 items-center gap-3 border-b border-border bg-background px-4"
      style={{ boxShadow: "0 1px 0 color-mix(in srgb, var(--foreground) 4%, transparent)" }}
    >
      <div className="flex items-center gap-2">
        <LogoMark />
        <span className="font-mono text-xs font-semibold tracking-[-0.01em]">
          portr
        </span>
      </div>
      <div className="flex items-center gap-2 font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
        <span style={{ color: "var(--tm-line-2)" }}>/</span>
        <span style={{ color: "var(--foreground)" }}>connections</span>
      </div>
      <div className="flex-1" />
      <button
        aria-label="Refresh tunnels"
        onClick={onRefresh}
        className="flex items-center gap-1.5 rounded px-2 py-1 font-mono text-[11px] transition-colors hover:bg-muted"
        style={{ color: "var(--muted-foreground)" }}
      >
        <RefreshCw className={`size-3 ${refreshing ? "animate-spin" : ""}`} />
        {synced}
      </button>
      <ThemeToggle />
    </header>
  )
}

/* ── Stats row ──────────────────────────────────────────── */

function StatCell({
  label,
  value,
  sub,
  dot,
  isLast,
}: {
  label: string
  value: string
  sub?: string
  dot?: boolean
  isLast?: boolean
}) {
  return (
    <div
      className="px-4 py-3"
      style={!isLast ? { borderRight: "1px solid var(--border)" } : undefined}
    >
      <div
        className="mb-1.5 flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.08em]"
        style={{ color: "var(--muted-foreground)" }}
      >
        {dot && <Dot status="live" />}
        {label}
      </div>
      <div
        className="font-mono text-[22px] font-semibold leading-none tracking-[-0.02em]"
        style={{ color: "var(--foreground)" }}
      >
        {value}
      </div>
      {sub && (
        <div
          className="mt-1 font-mono text-[11px]"
          style={{ color: "var(--muted-foreground)" }}
        >
          {sub}
        </div>
      )}
    </div>
  )
}

/* ── Main page ───────────────────────────────────────────── */

export function HomePage() {
  const navigate = useNavigate()
  const [tunnels, setTunnels] = React.useState<TunnelSummary[]>([])
  const [total, setTotal] = React.useState(0)
  const [stats, setStats] = React.useState<TunnelStats>(EMPTY_STATS)
  const [page, setPage] = React.useState(0)
  const [loading, setLoading] = React.useState(true)
  const [refreshing, setRefreshing] = React.useState(false)
  const [search, setSearch] = React.useState("")
  const [statusFilter, setStatusFilter] = React.useState<StatusFilter>("all")
  const [selectedKeys, setSelectedKeys] = React.useState<Set<string>>(new Set())
  const [confirmOpen, setConfirmOpen] = React.useState(false)
  const [deleting, setDeleting] = React.useState(false)
  const [pollingError, setPollingError] = React.useState<string | null>(null)
  const [lastSyncAt, setLastSyncAt] = React.useState<Date | null>(null)
  const [syncedAgo, setSyncedAgo] = React.useState("—")

  const loadTunnels = React.useEffectEvent(async (refresh = false) => {
    if (refresh) setRefreshing(true)
    else setLoading(true)
    try {
      const data = await getTunnels({
        limit: TUNNEL_PAGE_SIZE,
        offset: page * TUNNEL_PAGE_SIZE,
        search: search.trim(),
        status: statusFilter,
      })
      setTunnels(data.tunnels)
      setTotal(data.total)
      setStats(data.stats)
      setLastSyncAt(new Date())
      setPollingError(null)
      setSelectedKeys((cur) => {
        const valid = new Set(data.tunnels.map(tunnelKey))
        return new Set(Array.from(cur).filter((k) => valid.has(k)))
      })
      // page can fall off the end after deletes or a narrower search
      const lastPage = Math.max(0, Math.ceil(data.total / TUNNEL_PAGE_SIZE) - 1)
      if (page > lastPage) setPage(lastPage)
    } catch (err) {
      setPollingError(err instanceof Error ? err.message : null)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  })

  const initialLoadDone = React.useRef(false)
  React.useEffect(() => {
    loadTunnels(initialLoadDone.current)
    initialLoadDone.current = true
    const interval = window.setInterval(() => loadTunnels(true), 2000)
    return () => window.clearInterval(interval)
  }, [page, search, statusFilter])

  React.useEffect(() => {
    if (!lastSyncAt) return
    const tick = () => {
      const diff = Math.floor((Date.now() - lastSyncAt.getTime()) / 1000)
      setSyncedAgo(diff < 5 ? "just now" : `${diff}s ago`)
    }
    tick()
    const id = window.setInterval(tick, 1000)
    return () => window.clearInterval(id)
  }, [lastSyncAt])

  // search and status are both applied server-side; render the page as-is
  const pageCount = Math.max(1, Math.ceil(total / TUNNEL_PAGE_SIZE))
  const rangeStart = total === 0 ? 0 : page * TUNNEL_PAGE_SIZE + 1
  const rangeEnd = Math.min(total, page * TUNNEL_PAGE_SIZE + tunnels.length)

  async function handleDeleteSelected() {
    const selected = tunnels.filter((t) => selectedKeys.has(tunnelKey(t)))
    if (!selected.length) return
    setDeleting(true)
    try {
      const results = await Promise.allSettled(
        selected.map((t) => deleteTunnelLogs(t.Subdomain, t.Localport))
      )
      const ok = results.filter((r) => r.status === "fulfilled")
      const count = ok.reduce(
        (n, r) => n + (r.status === "fulfilled" ? r.value.deleted_count : 0),
        0
      )
      if (ok.length < results.length) {
        toast.error(`Deleted ${count} records. Some deletions failed.`)
      } else {
        toast.success(`Deleted ${count} records.`)
      }
      setSelectedKeys(new Set())
      setConfirmOpen(false)
      await loadTunnels(true)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete logs")
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="flex h-svh flex-col overflow-hidden bg-background">
      <TopBar synced={syncedAgo} refreshing={refreshing} onRefresh={() => loadTunnels(true)} />

      <div className="relative z-10 mx-auto flex w-full max-w-[1320px] flex-1 flex-col overflow-hidden px-6 pt-6">
        {pollingError ? <ServerUnavailableBanner className="mb-4" /> : null}

        {/* Page header */}
        <div
          className="mb-4 flex items-end justify-between gap-6 pb-[18px]"
          style={{ borderBottom: "1px solid var(--border)" }}
        >
          <div>
            <h1
              className="mb-1 flex items-center gap-2 font-mono text-xl font-semibold leading-none tracking-[-0.02em]"
              style={{ color: "var(--foreground)" }}
            >
              <span style={{ color: "var(--tm-green-ink)", fontWeight: 500 }}>~/</span>
              connections
            </h1>
            <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
              Local inspector dashboard · {total} tunnel
              {total !== 1 ? "s" : ""} detected
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              disabled={selectedKeys.size === 0 || deleting}
              onClick={() => setConfirmOpen(true)}
              className="inline-flex items-center gap-1.5 rounded-[6px] border border-destructive/60 bg-destructive/10 px-2.5 py-1.5 font-mono text-xs font-medium transition-colors hover:bg-destructive/20 disabled:pointer-events-none disabled:opacity-40"
              style={{ color: "var(--destructive)" }}
            >
              <Trash2 className="size-3" />
              Delete selected
            </button>
          </div>
        </div>

        {/* Stats row */}
        <div
          className="mb-5 grid grid-cols-4 overflow-hidden rounded-md bg-background"
          style={{ border: "1px solid var(--border)" }}
        >
          <StatCell
            label="Live tunnels"
            value={`${stats.live_tunnel_count}`}
            sub={`of ${total} total`}
            dot
          />
          <StatCell
            label="HTTP logs"
            value={stats.http_request_count.toLocaleString()}
            sub="across all tunnels"
          />
          <StatCell
            label="WebSockets"
            value={stats.websocket_session_count.toLocaleString()}
            sub={`${stats.active_websocket_count} session${
              stats.active_websocket_count !== 1 ? "s" : ""
            } open`}
          />
          <StatCell
            label="Last activity"
            value={relativeTime(stats.last_activity_at)}
            sub="most recent tunnel"
            isLast
          />
        </div>

        {/* Connections panel */}
        <div
          className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md bg-background"
          style={{ border: "1px solid var(--border)" }}
        >
          {/* Panel head */}
          <div
            className="flex items-center justify-between gap-3 px-3 py-2.5"
            style={{
              borderBottom: "1px solid var(--border)",
              background: "var(--muted)",
            }}
          >
            <div className="flex items-center gap-3">
              <span
                className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.1em]"
                style={{ color: "var(--muted-foreground)" }}
              >
                Tunnels
              </span>
              <div className="flex items-center gap-1.5">
                {STATUS_FILTERS.map((s) => (
                  <Chip
                    key={s}
                    active={statusFilter === s}
                    onClick={() => {
                      setStatusFilter(s)
                      setPage(0)
                    }}
                  >
                    {s}
                  </Chip>
                ))}
              </div>
            </div>
            <div className="relative flex items-center">
              <Search
                className="pointer-events-none absolute left-2 size-3.5"
                style={{ color: "var(--tm-muted-2)" }}
              />
              <input
                placeholder="filter by subdomain or port..."
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value)
                  setPage(0)
                }}
                className="h-7 w-[260px] rounded-[4px] border border-border bg-background pl-7 font-mono text-xs outline-none focus:border-foreground/40"
                style={{ color: "var(--foreground)" }}
              />
            </div>
          </div>

          {/* Table */}
          <div className="flex-1 overflow-y-auto">
          <table
            className="w-full border-collapse font-mono text-xs"
            style={{ fontSize: "12px" }}
          >
            <thead className="sticky top-0 z-10">
              <tr>
                {["SUBDOMAIN", "LOCAL", "PROTO", "STATUS", "REQUESTS", "WS SESSIONS", "LAST ACTIVITY", ""].map(
                  (h, i) => (
                    <th
                      key={i}
                      className="bg-background px-3 py-2 text-left font-medium uppercase tracking-[0.08em]"
                      style={{
                        fontSize: "10px",
                        color: "var(--muted-foreground)",
                        borderBottom: "1px solid var(--border)",
                      }}
                    >
                      {h}
                    </th>
                  )
                )}
              </tr>
            </thead>
            <tbody>
              {loading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <tr key={i}>
                    {Array.from({ length: 8 }).map((_, j) => (
                      <td key={j} className="px-3 py-3">
                        <div
                          className="h-3 animate-pulse rounded"
                          style={{ background: "var(--muted)", width: j === 0 ? "120px" : "60px" }}
                        />
                      </td>
                    ))}
                  </tr>
                ))
              ) : tunnels.length === 0 ? (
                <tr>
                  <td
                    colSpan={8}
                    className="py-10 text-center font-mono text-xs"
                    style={{ color: "var(--muted-foreground)" }}
                  >
                    {search || statusFilter !== "all"
                      ? "no tunnels match filter"
                      : "no tunnels recorded yet — start a tunnel to see it here"}
                  </td>
                </tr>
              ) : (
                tunnels.map((tunnel) => {
                  const key = tunnelKey(tunnel)
                  const proto =
                    tunnel.last_activity_kind === "websocket"
                      ? "WS"
                      : "HTTP"
                  return (
                    <tr
                      key={key}
                      className="cursor-pointer transition-colors hover:bg-muted/60"
                      style={{ borderBottom: "1px solid var(--border)" }}
                      onClick={(e) => {
                        if ((e.target as HTMLElement).closest("[data-selector]")) return
                        navigate(`/${tunnel.Subdomain}-${tunnel.Localport}`)
                      }}
                    >
                      {/* SUBDOMAIN */}
                      <td className="px-3 py-2.5">
                        <div className="flex items-center gap-1.5">
                          <input
                            data-selector
                            type="checkbox"
                            className="accent-foreground"
                            checked={selectedKeys.has(key)}
                            onChange={(e) =>
                              setSelectedKeys((cur) => {
                                const next = new Set(cur)
                                if (e.target.checked) next.add(key)
                                else next.delete(key)
                                return next
                              })
                            }
                            onClick={(e) => e.stopPropagation()}
                          />
                          <span className="font-semibold" style={{ color: "var(--foreground)" }}>
                            {tunnel.Subdomain}
                          </span>
                        </div>
                      </td>
                      {/* LOCAL */}
                      <td className="px-3 py-2.5">
                        <span style={{ color: "var(--muted-foreground)" }}>localhost</span>
                        <span style={{ color: "var(--tm-muted-2)" }}>:</span>
                        <span style={{ color: "var(--foreground)" }}>{tunnel.Localport}</span>
                      </td>
                      {/* PROTO */}
                      <td className="px-3 py-2.5">
                        <ProtoTag proto={proto} />
                      </td>
                      {/* STATUS */}
                      <td className="px-3 py-2.5">
                        <StatusBadge status={tunnel.status} />
                      </td>
                      {/* REQUESTS */}
                      <td className="px-3 py-2.5" style={{ color: "var(--foreground)" }}>
                        {tunnel.http_request_count.toLocaleString()}
                      </td>
                      {/* WS SESSIONS */}
                      <td className="px-3 py-2.5" style={{ color: "var(--foreground)" }}>
                        {tunnel.websocket_session_count.toLocaleString()}
                      </td>
                      {/* LAST ACTIVITY */}
                      <td className="px-3 py-2.5" style={{ color: "var(--muted-foreground)" }}>
                        {relativeTime(tunnel.last_activity_at)}
                      </td>
                      {/* ARROW */}
                      <td className="px-3 py-2.5 text-right" style={{ color: "var(--tm-muted-2)" }}>
                        ›
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
          </div>

          {/* Pagination */}
          <div
            className="flex items-center justify-between px-3 py-2"
            style={{
              borderTop: "1px solid var(--border)",
              background: "var(--muted)",
            }}
          >
            <span
              className="font-mono text-[11px]"
              style={{ color: "var(--muted-foreground)" }}
            >
              {total === 0
                ? "0 tunnels"
                : `${rangeStart}–${rangeEnd} of ${total.toLocaleString()}`}
            </span>
            <div className="flex items-center gap-1.5">
              <button
                disabled={page === 0}
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                className="flex items-center gap-1 rounded-[4px] border border-border bg-background px-2 py-1 font-mono text-[11px] transition-colors hover:bg-muted disabled:pointer-events-none disabled:opacity-40"
                style={{ color: "var(--foreground)" }}
              >
                <ChevronLeft className="size-3" />
                prev
              </button>
              <span
                className="px-1 font-mono text-[11px]"
                style={{ color: "var(--muted-foreground)" }}
              >
                {page + 1} / {pageCount}
              </span>
              <button
                disabled={page + 1 >= pageCount}
                onClick={() => setPage((p) => p + 1)}
                className="flex items-center gap-1 rounded-[4px] border border-border bg-background px-2 py-1 font-mono text-[11px] transition-colors hover:bg-muted disabled:pointer-events-none disabled:opacity-40"
                style={{ color: "var(--foreground)" }}
              >
                next
                <ChevronRight className="size-3" />
              </button>
            </div>
          </div>
        </div>

        {/* Keyboard hint */}
        <p
          className="py-3 text-center font-mono text-[11px]"
          style={{ color: "var(--muted-foreground)" }}
        >
          click row to inspect ·{" "}
          <kbd
            className="rounded-[3px] border border-border px-1 font-mono text-[10px]"
            style={{ color: "var(--muted-foreground)" }}
          >
            Del
          </kbd>{" "}
          to remove selected logs
        </p>
      </div>

      <AlertDialog onOpenChange={setConfirmOpen} open={confirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete selected tunnel logs</AlertDialogTitle>
            <AlertDialogDescription>
              Remove HTTP request logs and WebSocket session traces for{" "}
              {selectedKeys.size} selected tunnel
              {selectedKeys.size !== 1 ? "s" : ""}. This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction disabled={deleting} onClick={handleDeleteSelected}>
              {deleting ? (
                <>
                  <RefreshCw className="size-4 animate-spin" />
                  Deleting…
                </>
              ) : (
                "Delete logs"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
