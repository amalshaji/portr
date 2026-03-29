import * as React from "react"
import { useNavigate } from "react-router-dom"
import {
  Activity,
  ArrowRight,
  RadioTower,
  RefreshCw,
  Search,
  Trash2,
} from "lucide-react"
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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ThemeToggle } from "@/components/theme-toggle"
import { deleteTunnelLogs, getTunnels } from "@/lib/api"
import {
  formatDateTime,
  methodTone,
  reasonPhrase,
  statusTone,
} from "@/lib/dashboard"
import type { TunnelSummary } from "@/types"

function tunnelKey(tunnel: TunnelSummary) {
  return `${tunnel.Subdomain}:${tunnel.Localport}`
}

function statsFromTunnels(tunnels: TunnelSummary[]) {
  return tunnels.reduce(
    (accumulator, tunnel) => {
      accumulator.http += tunnel.http_request_count
      accumulator.websocket += tunnel.websocket_session_count
      accumulator.active += tunnel.active_websocket_count
      return accumulator
    },
    { http: 0, websocket: 0, active: 0 }
  )
}

export function HomePage() {
  const navigate = useNavigate()
  const [tunnels, setTunnels] = React.useState<TunnelSummary[]>([])
  const [loading, setLoading] = React.useState(true)
  const [refreshing, setRefreshing] = React.useState(false)
  const [search, setSearch] = React.useState("")
  const [selectedTunnelKeys, setSelectedTunnelKeys] = React.useState<
    Set<string>
  >(new Set())
  const [confirmOpen, setConfirmOpen] = React.useState(false)
  const [deleting, setDeleting] = React.useState(false)

  async function loadTunnels(refresh = false) {
    if (refresh) {
      setRefreshing(true)
    } else {
      setLoading(true)
    }

    try {
      const data = await getTunnels()
      setTunnels(data.tunnels)
      setSelectedTunnelKeys((current) => {
        const validKeys = new Set(
          data.tunnels.map((tunnel) => tunnelKey(tunnel))
        )
        return new Set(Array.from(current).filter((key) => validKeys.has(key)))
      })
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Failed to load tunnels"
      )
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  React.useEffect(() => {
    loadTunnels()
  }, [])

  const filteredTunnels = tunnels.filter((tunnel) => {
    const query = search.trim().toLowerCase()
    if (!query) {
      return true
    }

    return (
      tunnel.Subdomain.toLowerCase().includes(query) ||
      String(tunnel.Localport).includes(query)
    )
  })

  const stats = statsFromTunnels(tunnels)

  async function handleDeleteSelected() {
    const selected = tunnels.filter((tunnel) =>
      selectedTunnelKeys.has(tunnelKey(tunnel))
    )
    if (selected.length === 0) {
      return
    }

    setDeleting(true)
    try {
      const results = await Promise.allSettled(
        selected.map((tunnel) =>
          deleteTunnelLogs(tunnel.Subdomain, tunnel.Localport)
        )
      )
      const succeeded = results.filter(
        (result) => result.status === "fulfilled"
      )
      const deletedCount = succeeded.reduce((count, result) => {
        if (result.status !== "fulfilled") {
          return count
        }
        return count + result.value.deleted_count
      }, 0)

      if (succeeded.length !== results.length) {
        toast.error(
          `Deleted ${deletedCount} records across ${succeeded.length} tunnel(s). Some deletions failed.`
        )
      } else {
        toast.success(`Deleted ${deletedCount} records.`)
      }

      setSelectedTunnelKeys(new Set())
      setConfirmOpen(false)
      await loadTunnels(true)
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Failed to delete tunnel logs"
      )
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="min-h-svh bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.10),_transparent_22%),linear-gradient(180deg,_rgba(255,255,255,0.96),_rgba(248,250,252,1))] dark:bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.18),_transparent_22%),linear-gradient(180deg,_rgba(2,6,23,1),_rgba(3,7,18,0.96))]">
      <div className="mx-auto flex w-full max-w-none flex-col gap-4 px-3 py-4 sm:px-4 lg:px-5">
        <header className="overflow-hidden border border-border/70 bg-background/85 shadow-sm backdrop-blur">
          <div className="grid gap-6 p-6 lg:grid-cols-[1.2fr_0.8fr] lg:p-8">
            <div className="space-y-4">
              <div className="inline-flex items-center gap-2 rounded-full border border-sky-500/20 bg-sky-500/10 px-3 py-1 text-xs font-medium text-sky-700 dark:text-sky-300">
                <Activity className="size-3.5" />
                Local inspector at `localhost:7777`
              </div>
              <div className="space-y-2">
                <h1 className="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
                  Portr inspector dashboard
                </h1>
                <p className="max-w-2xl text-sm leading-6 text-muted-foreground sm:text-base">
                  Watch recent tunnel traffic, jump straight into request
                  traces, and keep an eye on upgraded WebSocket sessions from
                  one place.
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-3">
                <div className="relative min-w-0 flex-1 sm:max-w-md">
                  <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    className="h-10 border-border/70 bg-background/80 pl-10"
                    onChange={(event) => setSearch(event.target.value)}
                    placeholder="Search by subdomain or port"
                    value={search}
                  />
                </div>
                <Button
                  onClick={() => loadTunnels(true)}
                  size="sm"
                  variant="outline"
                >
                  <RefreshCw
                    className={refreshing ? "size-4 animate-spin" : "size-4"}
                  />
                  Refresh
                </Button>
                <Button
                  disabled={selectedTunnelKeys.size === 0 || deleting}
                  onClick={() => setConfirmOpen(true)}
                  size="sm"
                  variant="destructive"
                >
                  <Trash2 className="size-4" />
                  Delete selected
                </Button>
                <ThemeToggle />
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-1">
              <Card className="border-border/70 bg-gradient-to-br from-slate-950 to-slate-900 text-white shadow-sm dark:from-slate-900 dark:to-slate-800">
                <CardContent className="space-y-3 p-5">
                  <p className="text-xs tracking-[0.2em] text-white/60 uppercase">
                    Tunnels
                  </p>
                  <div className="text-3xl font-semibold">{tunnels.length}</div>
                  <p className="text-sm text-white/70">
                    Distinct ports with recorded traffic
                  </p>
                </CardContent>
              </Card>
              <Card className="border-border/70 bg-background/90 shadow-sm">
                <CardContent className="space-y-3 p-5">
                  <p className="text-xs tracking-[0.2em] text-muted-foreground uppercase">
                    HTTP logs
                  </p>
                  <div className="text-3xl font-semibold">{stats.http}</div>
                  <p className="text-sm text-muted-foreground">
                    Stored requests across all tunnels
                  </p>
                </CardContent>
              </Card>
              <Card className="border-border/70 bg-background/90 shadow-sm">
                <CardContent className="space-y-3 p-5">
                  <p className="text-xs tracking-[0.2em] text-muted-foreground uppercase">
                    WebSockets
                  </p>
                  <div className="text-3xl font-semibold">
                    {stats.websocket}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {stats.active} session{stats.active === 1 ? "" : "s"} still
                    open
                  </p>
                </CardContent>
              </Card>
            </div>
          </div>
        </header>

        <Card className="overflow-hidden border-border/70 bg-background/85 shadow-sm backdrop-blur">
          <CardContent className="p-0">
            {loading ? (
              <div className="grid gap-3 p-6">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton className="h-16 rounded-md" key={index} />
                ))}
              </div>
            ) : filteredTunnels.length === 0 ? (
              <div className="flex min-h-72 flex-col items-center justify-center gap-3 px-6 py-12 text-center">
                <div className="rounded-md border border-dashed border-border/80 bg-muted/30 p-4">
                  <RadioTower className="size-6 text-muted-foreground" />
                </div>
                <div className="space-y-1">
                  <h2 className="font-medium">No tunnels match this filter</h2>
                  <p className="text-sm text-muted-foreground">
                    Try a different search term or refresh after new traffic
                    arrives.
                  </p>
                </div>
              </div>
            ) : (
              <div className="hidden lg:block">
                <Table>
                  <TableHeader>
                    <TableRow className="border-border/70">
                      <TableHead className="w-14">Select</TableHead>
                      <TableHead>Tunnel</TableHead>
                      <TableHead>Latest trace</TableHead>
                      <TableHead>Traffic</TableHead>
                      <TableHead className="text-right">Open</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredTunnels.map((tunnel) => (
                      <TableRow
                        className="cursor-pointer border-border/60 hover:bg-muted/40"
                        key={tunnelKey(tunnel)}
                        onClick={(event) => {
                          const target = event.target as HTMLElement
                          if (target.closest("[data-tunnel-selector]")) {
                            return
                          }
                          navigate(`/${tunnel.Subdomain}-${tunnel.Localport}`)
                        }}
                      >
                        <TableCell>
                          <div data-tunnel-selector>
                            <Checkbox
                              checked={selectedTunnelKeys.has(
                                tunnelKey(tunnel)
                              )}
                              onCheckedChange={(checked) =>
                                setSelectedTunnelKeys((current) => {
                                  const next = new Set(current)
                                  if (checked) {
                                    next.add(tunnelKey(tunnel))
                                  } else {
                                    next.delete(tunnelKey(tunnel))
                                  }
                                  return next
                                })
                              }
                            />
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="space-y-1">
                            <div className="font-medium">
                              {tunnel.Subdomain}
                              <span className="ml-2 text-sm text-muted-foreground">
                                :{tunnel.Localport}
                              </span>
                            </div>
                            <div className="text-xs text-muted-foreground">
                              Last activity{" "}
                              {formatDateTime(tunnel.last_activity_at)}
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-wrap items-center gap-2">
                            {tunnel.last_method ? (
                              <Badge
                                className={`ring-1 ${methodTone(tunnel.last_method)}`}
                                variant="outline"
                              >
                                {tunnel.last_method}
                              </Badge>
                            ) : null}
                            {typeof tunnel.last_status === "number" ? (
                              <Badge
                                className={`ring-1 ${statusTone(tunnel.last_status)}`}
                                variant="outline"
                              >
                                {tunnel.last_status}{" "}
                                {reasonPhrase(tunnel.last_status)}
                              </Badge>
                            ) : null}
                            <span className="truncate text-sm text-muted-foreground">
                              {tunnel.last_url || "Waiting for first request"}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                            <Badge variant="outline">
                              {tunnel.http_request_count} HTTP
                            </Badge>
                            <Badge variant="outline">
                              {tunnel.websocket_session_count} WS
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <Button size="sm" variant="ghost">
                            Inspect
                            <ArrowRight className="size-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            {!loading ? (
              <div className="grid gap-3 p-4 lg:hidden">
                {filteredTunnels.map((tunnel) => (
                  <button
                    className="border border-border/70 bg-background/90 p-4 text-left shadow-sm transition hover:border-sky-500/40 hover:bg-muted/30"
                    key={tunnelKey(tunnel)}
                    onClick={() =>
                      navigate(`/${tunnel.Subdomain}-${tunnel.Localport}`)
                    }
                    type="button"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="space-y-1">
                        <div className="font-medium">
                          {tunnel.Subdomain}
                          <span className="ml-2 text-sm text-muted-foreground">
                            :{tunnel.Localport}
                          </span>
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {formatDateTime(tunnel.last_activity_at)}
                        </p>
                      </div>
                      <Checkbox
                        checked={selectedTunnelKeys.has(tunnelKey(tunnel))}
                        onCheckedChange={(checked) =>
                          setSelectedTunnelKeys((current) => {
                            const next = new Set(current)
                            if (checked) {
                              next.add(tunnelKey(tunnel))
                            } else {
                              next.delete(tunnelKey(tunnel))
                            }
                            return next
                          })
                        }
                        onClick={(event) => event.stopPropagation()}
                      />
                    </div>
                    <div className="mt-3 flex flex-wrap gap-2">
                      {tunnel.last_method ? (
                        <Badge
                          className={`ring-1 ${methodTone(tunnel.last_method)}`}
                          variant="outline"
                        >
                          {tunnel.last_method}
                        </Badge>
                      ) : null}
                      {typeof tunnel.last_status === "number" ? (
                        <Badge
                          className={`ring-1 ${statusTone(tunnel.last_status)}`}
                          variant="outline"
                        >
                          {tunnel.last_status}
                        </Badge>
                      ) : null}
                      <Badge variant="outline">
                        {tunnel.http_request_count} HTTP
                      </Badge>
                      <Badge variant="outline">
                        {tunnel.websocket_session_count} WS
                      </Badge>
                    </div>
                    <p className="mt-3 truncate text-sm text-muted-foreground">
                      {tunnel.last_url || "Waiting for first request"}
                    </p>
                  </button>
                ))}
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>

      <AlertDialog onOpenChange={setConfirmOpen} open={confirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete selected tunnel logs</AlertDialogTitle>
            <AlertDialogDescription>
              Remove HTTP request logs and stored WebSocket session traces for{" "}
              {selectedTunnelKeys.size} selected tunnel
              {selectedTunnelKeys.size === 1 ? "" : "s"}. This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={deleting}
              onClick={handleDeleteSelected}
            >
              {deleting ? (
                <>
                  <RefreshCw className="size-4 animate-spin" />
                  Deleting...
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
