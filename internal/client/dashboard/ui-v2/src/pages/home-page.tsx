import * as React from "react"
import { useNavigate } from "react-router-dom"
import { RefreshCw, Search, Trash2 } from "lucide-react"
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
import { deleteTunnelLogs, getTunnels } from "@/lib/api"
import {
  deriveStatus,
  relativeTime,
} from "@/lib/dashboard"
import type { TunnelSummary } from "@/types"

function tunnelKey(tunnel: TunnelSummary) {
  return `${tunnel.Subdomain}:${tunnel.Localport}`
}

function statsFromTunnels(tunnels: TunnelSummary[]) {
  return tunnels.reduce(
    (acc, t) => {
      acc.http += t.http_request_count
      acc.websocket += t.websocket_session_count
      acc.active += t.active_websocket_count
      return acc
    },
    { http: 0, websocket: 0, active: 0 }
  )
}

/* ── Small shared terminal components ──────────────────── */

function LogoMark() {
  return (
    <div
      className="grid h-[18px] w-[18px] shrink-0 place-items-center rounded-[3px]"
      style={{ border: "1.5px solid var(--foreground)" }}
    >
      <div
        className="h-1.5 w-1.5 rounded-[1px]"
        style={{ background: "var(--tm-green)" }}
      />
    </div>
  )
}

type DotStatus = "live" | "idle" | "closed"

function Dot({ status }: { status: DotStatus }) {
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

function StatusBadge({ status }: { status: DotStatus }) {
  const styles: Record<DotStatus, React.CSSProperties> = {
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

/* ── Filter chip ────────────────────────────────────────── */

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
          ? {
              background: "var(--foreground)",
              color: "var(--background)",
              borderColor: "var(--foreground)",
            }
          : {
              background: "var(--background)",
              color: "var(--muted-foreground)",
              borderColor: "var(--tm-line-2)",
            }
      }
    >
      {children}
    </button>
  )
}

/* ── Main page ───────────────────────────────────────────── */

export function HomePage() {
  const navigate = useNavigate()
  const [tunnels, setTunnels] = React.useState<TunnelSummary[]>([])
  const [loading, setLoading] = React.useState(true)
  const [refreshing, setRefreshing] = React.useState(false)
  const [search, setSearch] = React.useState("")
  const [statusFilter, setStatusFilter] = React.useState<"all" | "live" | "idle" | "closed">("all")
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
      const data = await getTunnels()
      setTunnels(data.tunnels)
      setLastSyncAt(new Date())
      setPollingError(null)
      setSelectedKeys((cur) => {
        const valid = new Set(data.tunnels.map(tunnelKey))
        return new Set(Array.from(cur).filter((k) => valid.has(k)))
      })
    } catch (err) {
      setPollingError(err instanceof Error ? err.message : null)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  })

  React.useEffect(() => {
    loadTunnels()
    const interval = window.setInterval(() => loadTunnels(true), 2000)
    return () => window.clearInterval(interval)
  }, [])

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

  const filteredTunnels = tunnels.filter((t) => {
    const q = search.trim().toLowerCase()
    const matchesQuery =
      !q ||
      t.Subdomain.toLowerCase().includes(q) ||
      String(t.Localport).includes(q)
    const status = deriveStatus(t)
    const matchesStatus = statusFilter === "all" || status === statusFilter
    return matchesQuery && matchesStatus
  })

  const stats = statsFromTunnels(tunnels)
  const liveCount = tunnels.filter((t) => deriveStatus(t) === "live").length
  const lastActivity =
    tunnels.length > 0
      ? tunnels.reduce((a, b) =>
          new Date(a.last_activity_at) > new Date(b.last_activity_at) ? a : b
        ).last_activity_at
      : null

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
              Local inspector dashboard · {tunnels.length} tunnel
              {tunnels.length !== 1 ? "s" : ""} detected
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
            value={`${liveCount}`}
            sub={`of ${tunnels.length} total`}
            dot
          />
          <StatCell
            label="HTTP logs"
            value={stats.http.toLocaleString()}
            sub="across all tunnels"
          />
          <StatCell
            label="WebSockets"
            value={stats.websocket.toLocaleString()}
            sub={`${stats.active} session${stats.active !== 1 ? "s" : ""} open`}
          />
          <StatCell
            label="Last activity"
            value={relativeTime(lastActivity)}
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
                {(["all", "live", "idle", "closed"] as const).map((s) => (
                  <Chip
                    key={s}
                    active={statusFilter === s}
                    onClick={() => setStatusFilter(s)}
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
                onChange={(e) => setSearch(e.target.value)}
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
              ) : filteredTunnels.length === 0 ? (
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
                filteredTunnels.map((tunnel) => {
                  const key = tunnelKey(tunnel)
                  const status = deriveStatus(tunnel)
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
                        <StatusBadge status={status} />
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
