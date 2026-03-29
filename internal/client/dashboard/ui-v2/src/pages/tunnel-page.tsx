import * as React from "react"
import { Link, useParams, useSearchParams } from "react-router-dom"
import {
  ArrowLeft,
  ArrowUpRight,
  Copy,
  Download,
  LoaderCircle,
  Play,
  RadioTower,
  RefreshCw,
  Search,
  Sparkles,
  Waves,
} from "lucide-react"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { PayloadViewer } from "@/components/payload-viewer"
import { ReplayDialog } from "@/components/replay-dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ThemeToggle } from "@/components/theme-toggle"
import {
  getRequests,
  getWebSocketSession,
  getWebSocketSessions,
  replayRequest,
} from "@/lib/api"
import {
  buildCurlCommand,
  contentLength,
  decodeBase64ToBytes,
  flattenHeaders,
  formatDateTime,
  formatTime,
  getHeaderValue,
  methodTone,
  parseTunnelId,
  payloadPreview,
  reasonPhrase,
  statusTone,
  websocketPayloadLabel,
  websocketDirectionTone,
  websocketOpcodeTone,
} from "@/lib/dashboard"
import type { RequestRecord, WebSocketEvent, WebSocketSession } from "@/types"

function copyText(value: string, successMessage: string) {
  return navigator.clipboard
    .writeText(value)
    .then(() => toast.success(successMessage))
    .catch(() => toast.error("Unable to copy to clipboard"))
}

function formatBytes(value: number) {
  if (value <= 0) {
    return "0 B"
  }
  if (value < 1024) {
    return `${value} B`
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`
  }
  return `${(value / (1024 * 1024)).toFixed(1)} MB`
}

function bodyMetric(headers: Record<string, string[]>, body: string) {
  const bytes = contentLength(headers)
  if (bytes > 0) {
    return formatBytes(bytes)
  }
  if (body) {
    return "Captured"
  }
  return "No body"
}

function DetailMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-border/70 bg-muted/20 px-4 py-3">
      <p className="text-[11px] tracking-[0.18em] text-muted-foreground uppercase">
        {label}
      </p>
      <p className="mt-2 text-sm font-medium break-all">{value}</p>
    </div>
  )
}

function HeaderTable({ headers }: { headers: Record<string, string> }) {
  const entries = Object.entries(headers)

  if (entries.length === 0) {
    return (
      <div className="border border-dashed border-border/80 bg-muted/20 px-5 py-6 text-sm text-muted-foreground">
        No headers captured.
      </div>
    )
  }

  return (
    <div className="overflow-hidden border border-border/70 bg-background">
      <div className="grid grid-cols-[minmax(12rem,18rem)_minmax(0,1fr)] border-b border-border/70 bg-muted/30 px-4 py-2 text-[11px] font-medium tracking-[0.18em] text-muted-foreground uppercase">
        <span>Header</span>
        <span>Value</span>
      </div>
      <div className="max-h-[24rem] divide-y divide-border/60 overflow-auto">
        {entries.map(([key, value]) => (
          <div
            className="grid grid-cols-1 gap-2 px-4 py-3 md:grid-cols-[minmax(12rem,18rem)_minmax(0,1fr)] md:gap-4"
            key={key}
          >
            <div className="text-xs font-medium tracking-[0.08em] text-muted-foreground uppercase">
              {key}
            </div>
            <div className="text-sm leading-6 break-all whitespace-pre-wrap">
              {value}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function RequestDetail({
  request,
  onRefresh,
  onSelectParent,
}: {
  request: RequestRecord | null
  onRefresh: () => Promise<void>
  onSelectParent: () => void
}) {
  const [detailTab, setDetailTab] = React.useState("request")
  const [replaying, setReplaying] = React.useState(false)
  const [replayDialogOpen, setReplayDialogOpen] = React.useState(false)

  React.useEffect(() => {
    setDetailTab("request")
  }, [request?.ID])

  if (!request) {
    return (
      <Card className="border-border/70 bg-background/80 shadow-sm">
        <CardContent className="flex min-h-[32rem] flex-col items-center justify-center gap-3 p-8 text-center">
          <div className="rounded-md border border-dashed border-border/80 bg-muted/30 p-4">
            <ArrowLeft className="size-6 text-muted-foreground" />
          </div>
          <div className="space-y-1">
            <h2 className="font-medium">Select a request trace</h2>
            <p className="text-sm text-muted-foreground">
              Pick a request from the left rail to inspect its headers, body,
              and replay controls.
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  async function handleReplay() {
    if (!request) {
      return
    }

    setReplaying(true)
    try {
      await replayRequest(request.ID)
      toast.success("Replay dispatched")
      await onRefresh()
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Failed to replay request"
      )
    } finally {
      setReplaying(false)
    }
  }

  const requestHeaders = flattenHeaders(request.Headers)
  const responseHeaders = flattenHeaders(request.ResponseHeaders)
  const requestContentType =
    getHeaderValue(request.Headers, "Content-Type") || "No content type"
  const responseContentType =
    getHeaderValue(request.ResponseHeaders, "Content-Type") || "No content type"

  return (
    <>
      <Card className="border-border/70 bg-background/85 shadow-sm xl:h-full xl:min-h-0">
        <CardContent className="space-y-6 p-4 sm:p-5 xl:flex xl:h-full xl:min-h-0 xl:flex-col">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-2">
                <Badge
                  className={`ring-1 ${methodTone(request.Method)}`}
                  variant="outline"
                >
                  {request.Method}
                </Badge>
                <Badge
                  className={`ring-1 ${statusTone(request.ResponseStatusCode)}`}
                  variant="outline"
                >
                  {request.ResponseStatusCode}{" "}
                  {reasonPhrase(request.ResponseStatusCode)}
                </Badge>
                {request.IsReplayed ? (
                  <Badge variant="outline">Replayed</Badge>
                ) : null}
              </div>
              <div className="space-y-1">
                <h2 className="text-xl font-semibold tracking-tight break-all">
                  {request.Url}
                </h2>
                <p className="text-sm text-muted-foreground">
                  {formatDateTime(request.LoggedAt)}
                </p>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              {request.ParentID ? (
                <Button onClick={onSelectParent} size="sm" variant="outline">
                  <ArrowUpRight className="size-4" />
                  View parent
                </Button>
              ) : null}
              <Button
                disabled={replaying}
                onClick={handleReplay}
                size="sm"
                variant="outline"
              >
                {replaying ? (
                  <>
                    <LoaderCircle className="size-4 animate-spin" />
                    Replaying...
                  </>
                ) : (
                  <>
                    <Play className="size-4" />
                    Replay original
                  </>
                )}
              </Button>
              <Button onClick={() => setReplayDialogOpen(true)} size="sm">
                <Sparkles className="size-4" />
                Edit & send
              </Button>
              <Button
                onClick={() =>
                  copyText(buildCurlCommand(request), "cURL command copied")
                }
                size="sm"
                variant="outline"
              >
                <Copy className="size-4" />
                Copy cURL
              </Button>
            </div>
          </div>

          <Tabs
            className="xl:flex xl:min-h-0 xl:flex-1 xl:flex-col"
            onValueChange={setDetailTab}
            value={detailTab}
          >
            <TabsList variant="line">
              <TabsTrigger value="request">Request</TabsTrigger>
              <TabsTrigger value="response">Response</TabsTrigger>
            </TabsList>
            <TabsContent
              className="mt-5 space-y-5 xl:mt-4 xl:min-h-0 xl:overflow-y-auto xl:pr-1"
              value="request"
            >
              <div className="grid gap-3 lg:grid-cols-3">
                <DetailMetric label="Content type" value={requestContentType} />
                <DetailMetric
                  label="Body size"
                  value={bodyMetric(request.Headers, request.Body)}
                />
                <DetailMetric
                  label="Headers"
                  value={`${Object.keys(requestHeaders).length} captured`}
                />
              </div>
              <section className="space-y-3">
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-medium">Body</h3>
                  <span className="text-xs text-muted-foreground">
                    Primary payload view
                  </span>
                </div>
                <PayloadViewer request={request} type="request" />
              </section>
              <section className="space-y-3">
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-medium">Headers</h3>
                  <span className="text-xs text-muted-foreground">
                    Structured key-value view
                  </span>
                </div>
                <HeaderTable headers={requestHeaders} />
              </section>
            </TabsContent>
            <TabsContent
              className="mt-5 space-y-5 xl:mt-4 xl:min-h-0 xl:overflow-y-auto xl:pr-1"
              value="response"
            >
              <div className="grid gap-3 lg:grid-cols-3">
                <DetailMetric
                  label="Content type"
                  value={responseContentType}
                />
                <DetailMetric
                  label="Body size"
                  value={bodyMetric(
                    request.ResponseHeaders,
                    request.ResponseBody
                  )}
                />
                <DetailMetric
                  label="Status"
                  value={`${request.ResponseStatusCode} ${reasonPhrase(
                    request.ResponseStatusCode
                  )}`}
                />
              </div>
              <section className="space-y-3">
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-medium">Body</h3>
                  <span className="text-xs text-muted-foreground">
                    Full-width response preview
                  </span>
                </div>
                <PayloadViewer request={request} type="response" />
              </section>
              <section className="space-y-3">
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-medium">Headers</h3>
                  <span className="text-xs text-muted-foreground">
                    Structured key-value view
                  </span>
                </div>
                <HeaderTable headers={responseHeaders} />
              </section>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      <ReplayDialog
        onOpenChange={setReplayDialogOpen}
        onReplayed={onRefresh}
        open={replayDialogOpen}
        request={request}
      />
    </>
  )
}

function WebSocketDetail({
  session,
  events,
}: {
  session: WebSocketSession | null
  events: WebSocketEvent[]
}) {
  const [selectedEventID, setSelectedEventID] = React.useState<string | null>(null)

  React.useEffect(() => {
    setSelectedEventID(events[0]?.id || null)
  }, [events, session?.ID])

  if (!session) {
    return (
      <Card className="border-border/70 bg-background/80 shadow-sm">
        <CardContent className="flex min-h-[32rem] flex-col items-center justify-center gap-3 p-8 text-center">
          <div className="rounded-md border border-dashed border-border/80 bg-muted/30 p-4">
            <Waves className="size-6 text-muted-foreground" />
          </div>
          <div className="space-y-1">
            <h2 className="font-medium">Select a WebSocket session</h2>
            <p className="text-sm text-muted-foreground">
              Inspect the handshake metadata and message timeline for upgraded
              connections.
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  const selectedEvent =
    events.find((event) => event.id === selectedEventID) || events[0] || null

  return (
    <Card className="border-border/70 bg-background/85 shadow-sm xl:h-full xl:min-h-0">
      <CardContent className="space-y-6 p-4 sm:p-5 xl:flex xl:h-full xl:min-h-0 xl:flex-col">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <Badge
                className={`ring-1 ${methodTone(session.Method)}`}
                variant="outline"
              >
                {session.Method}
              </Badge>
              <Badge variant="outline">{session.EventCount} frames</Badge>
              <Badge variant="outline">
                {session.ClosedAt
                  ? `Closed ${formatTime(session.ClosedAt)}`
                  : "Open"}
              </Badge>
            </div>
            <div className="space-y-1">
              <h2 className="text-xl font-semibold tracking-tight break-all">
                {session.URL}
              </h2>
              <p className="text-sm text-muted-foreground">
                Started {formatDateTime(session.StartedAt)}
              </p>
            </div>
          </div>
          <div className="grid gap-2 text-right text-sm text-muted-foreground">
            <div>{session.ClientEventCount} client frames</div>
            <div>{session.ServerEventCount} server frames</div>
            {session.CloseCode ? (
              <div>
                Close code {session.CloseCode}
                {session.CloseReason ? ` · ${session.CloseReason}` : ""}
              </div>
            ) : null}
          </div>
        </div>

        <Tabs
          className="xl:flex xl:min-h-0 xl:flex-1 xl:flex-col"
          defaultValue="timeline"
        >
          <TabsList variant="line">
            <TabsTrigger value="timeline">Timeline</TabsTrigger>
            <TabsTrigger value="request">Request headers</TabsTrigger>
            <TabsTrigger value="response">Response headers</TabsTrigger>
          </TabsList>
          <TabsContent
            className="xl:min-h-0 xl:overflow-y-auto xl:pr-1"
            value="timeline"
          >
            {events.length === 0 ? (
              <div className="border border-dashed border-border/80 bg-muted/30 p-6 text-sm text-muted-foreground">
                No frames captured yet.
              </div>
            ) : (
              <div className="overflow-hidden border border-border/70">
                <div className="grid grid-cols-[auto_auto_auto_1fr_auto] gap-3 border-b bg-muted/40 px-4 py-3 text-xs font-medium tracking-[0.18em] text-muted-foreground uppercase">
                  <span>Dir</span>
                  <span>Type</span>
                  <span>At</span>
                  <span>Preview</span>
                  <span className="text-right">Size</span>
                </div>
                <div className="divide-y">
                  {events.map((event) => (
                    <div
                      className={`grid cursor-pointer grid-cols-[auto_auto_auto_1fr_auto] items-center gap-3 px-4 py-3 text-sm transition hover:bg-muted/20 ${
                        event.id === selectedEvent?.id ? "bg-muted/30" : ""
                      }`}
                      key={event.id}
                      onClick={() => setSelectedEventID(event.id)}
                    >
                      <Badge
                        className={`ring-1 ${websocketDirectionTone(event.direction)}`}
                        variant="outline"
                      >
                        {event.direction}
                      </Badge>
                      <Badge
                        className={`ring-1 ${websocketOpcodeTone(event.opcode)}`}
                        variant="outline"
                      >
                        {event.opcode_name}
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        {formatTime(event.logged_at)}
                      </span>
                      <div className="min-w-0">
                        <p className="truncate font-medium">
                          {payloadPreview(event)}
                        </p>
                        {event.payload && !event.payload_text ? (
                          <Button
                            className="h-auto px-0 py-0 text-xs text-muted-foreground"
                            onClick={() => {
                              const url = URL.createObjectURL(
                                new Blob([decodeBase64ToBytes(event.payload)])
                              )
                              const anchor = document.createElement("a")
                              anchor.href = url
                              anchor.download = `${event.id}.bin`
                              anchor.click()
                              setTimeout(() => URL.revokeObjectURL(url), 1000)
                            }}
                            size="sm"
                            variant="link"
                          >
                            <Download className="size-3.5" />
                            Download payload
                          </Button>
                        ) : null}
                      </div>
                      <span className="text-right text-xs text-muted-foreground">
                        {event.payload_length} B
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {selectedEvent ? (
              <div className="mt-4 space-y-3">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <h3 className="text-sm font-medium">Selected frame</h3>
                    <p className="text-xs text-muted-foreground">
                      {websocketPayloadLabel(selectedEvent)} ·{" "}
                      {selectedEvent.payload_length} B
                    </p>
                  </div>
                  {!selectedEvent.payload_text && selectedEvent.payload ? (
                    <Button
                      onClick={() => {
                        const url = URL.createObjectURL(
                          new Blob([decodeBase64ToBytes(selectedEvent.payload)])
                        )
                        const anchor = document.createElement("a")
                        anchor.href = url
                        anchor.download = `${selectedEvent.id}.bin`
                        anchor.click()
                        setTimeout(() => URL.revokeObjectURL(url), 1000)
                      }}
                      size="sm"
                      variant="outline"
                    >
                      <Download className="size-4" />
                      Download payload
                    </Button>
                  ) : null}
                </div>

                {selectedEvent.payload_text ? (
                  <div className="border bg-background">
                    <pre className="min-h-32 whitespace-pre-wrap break-all p-4 font-mono text-xs leading-6">
                      {selectedEvent.payload_text}
                    </pre>
                  </div>
                ) : (
                  <div className="border border-dashed border-border/80 bg-muted/20 p-4 text-sm text-muted-foreground">
                    This frame is stored as raw bytes. Download the payload to inspect it outside the dashboard.
                  </div>
                )}
              </div>
            ) : null}
          </TabsContent>
          <TabsContent
            className="xl:min-h-0 xl:overflow-y-auto xl:pr-1"
            value="request"
          >
            <HeaderTable headers={flattenHeaders(session.RequestHeaders)} />
          </TabsContent>
          <TabsContent
            className="xl:min-h-0 xl:overflow-y-auto xl:pr-1"
            value="response"
          >
            <HeaderTable headers={flattenHeaders(session.ResponseHeaders)} />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

export function TunnelPage() {
  const params = useParams()
  const [searchParams, setSearchParams] = useSearchParams()
  const tunnel = React.useMemo(
    () => parseTunnelId(params.id || ""),
    [params.id]
  )
  const [requests, setRequests] = React.useState<RequestRecord[]>([])
  const [sessions, setSessions] = React.useState<WebSocketSession[]>([])
  const [selectedSession, setSelectedSession] =
    React.useState<WebSocketSession | null>(null)
  const [selectedSessionEvents, setSelectedSessionEvents] = React.useState<
    WebSocketEvent[]
  >([])
  const [loading, setLoading] = React.useState(true)
  const [refreshing, setRefreshing] = React.useState(false)
  const [search, setSearch] = React.useState("")
  const activeTab =
    searchParams.get("tab") === "websocket" ? "websocket" : "http"
  const selectedRequestId = searchParams.get("request")
  const selectedSessionId = searchParams.get("session")

  const updateSearchParams = React.useCallback(
    (values: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams)
      Object.entries(values).forEach(([key, value]) => {
        if (!value) {
          next.delete(key)
          return
        }
        next.set(key, value)
      })
      setSearchParams(next, { replace: true })
    },
    [searchParams, setSearchParams]
  )

  const loadSummary = React.useEffectEvent(async (refresh = false) => {
    if (!tunnel) {
      return
    }

    if (refresh) {
      setRefreshing(true)
    } else {
      setLoading(true)
    }

    try {
      const [requestsData, sessionsData] = await Promise.all([
        getRequests(tunnel.subdomain, tunnel.localport),
        getWebSocketSessions(tunnel.subdomain, tunnel.localport),
      ])
      setRequests(requestsData.requests)
      setSessions(sessionsData.sessions)
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Failed to load tunnel data"
      )
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  })

  const loadSelectedSession = React.useEffectEvent(
    async (sessionId: string) => {
      try {
        const data = await getWebSocketSession(sessionId)
        setSelectedSession(data.session)
        setSelectedSessionEvents(data.events)
      } catch (error) {
        toast.error(
          error instanceof Error
            ? error.message
            : "Failed to load WebSocket session"
        )
      }
    }
  )

  React.useEffect(() => {
    if (!tunnel) {
      return
    }

    loadSummary()
    const interval = window.setInterval(() => {
      loadSummary(true)
      if (selectedSessionId) {
        loadSelectedSession(selectedSessionId)
      }
    }, 2000)

    return () => {
      window.clearInterval(interval)
    }
  }, [selectedSessionId, tunnel])

  React.useEffect(() => {
    if (!requests.length) {
      return
    }

    const exists = selectedRequestId
      ? requests.some((request) => request.ID === selectedRequestId)
      : false
    if (exists) {
      return
    }

    updateSearchParams({ request: requests[0].ID })
  }, [requests, selectedRequestId, updateSearchParams])

  React.useEffect(() => {
    if (!sessions.length) {
      setSelectedSession(null)
      setSelectedSessionEvents([])
      return
    }

    const exists = selectedSessionId
      ? sessions.some((session) => session.ID === selectedSessionId)
      : false
    if (!selectedSessionId || !exists) {
      updateSearchParams({ session: sessions[0].ID })
      return
    }

    loadSelectedSession(selectedSessionId)
  }, [selectedSessionId, sessions, updateSearchParams])

  const selectedRequest =
    requests.find((request) => request.ID === selectedRequestId) || null

  const filteredRequests = requests.filter((request) =>
    request.Url.toLowerCase().includes(search.trim().toLowerCase())
  )
  const filteredSessions = sessions.filter((session) => {
    const query = search.trim().toLowerCase()
    if (!query) {
      return true
    }

    return (
      session.URL.toLowerCase().includes(query) ||
      session.Host.toLowerCase().includes(query) ||
      session.CloseReason.toLowerCase().includes(query)
    )
  })

  if (!tunnel) {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <Card className="max-w-lg">
          <CardContent className="space-y-3 p-8 text-center">
            <h1 className="text-xl font-semibold">Tunnel not found</h1>
            <p className="text-sm text-muted-foreground">
              The inspector route is invalid. Go back to the dashboard and
              select a tunnel again.
            </p>
            <Button asChild>
              <Link to="/">Back to dashboard</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-svh bg-[radial-gradient(circle_at_top_left,_rgba(6,182,212,0.12),_transparent_24%),linear-gradient(180deg,_rgba(255,255,255,0.96),_rgba(248,250,252,1))] xl:h-svh xl:overflow-hidden dark:bg-[radial-gradient(circle_at_top_left,_rgba(6,182,212,0.22),_transparent_24%),linear-gradient(180deg,_rgba(2,6,23,1),_rgba(3,7,18,0.96))]">
      <div className="mx-auto flex w-full max-w-none flex-col gap-4 px-3 py-4 sm:px-4 lg:px-5 xl:h-full xl:min-h-0">
        <header className="overflow-hidden border border-border/70 bg-background/85 p-4 shadow-sm backdrop-blur sm:p-5">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-3">
              <Link
                className="inline-flex items-center gap-2 text-sm text-muted-foreground transition hover:text-foreground"
                to="/"
              >
                <ArrowLeft className="size-4" />
                Back to dashboard
              </Link>
              <div className="space-y-1">
                <h1 className="text-3xl font-semibold tracking-tight">
                  {tunnel.subdomain}
                  <span className="ml-3 text-base font-normal text-muted-foreground">
                    :{tunnel.localport}
                  </span>
                </h1>
                <p className="text-sm text-muted-foreground">
                  Live HTTP traces, replay tools, and captured WebSocket frames
                  for this tunnel.
                </p>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <div className="relative min-w-0 flex-1 sm:min-w-72">
                <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  className="h-10 border-border/70 bg-background/80 pl-10"
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder={
                    activeTab === "http"
                      ? "Filter requests by URL"
                      : "Filter sessions by URL or host"
                  }
                  value={search}
                />
              </div>
              <Button
                onClick={() => loadSummary(true)}
                size="sm"
                variant="outline"
              >
                <RefreshCw
                  className={refreshing ? "size-4 animate-spin" : "size-4"}
                />
                Refresh
              </Button>
              <ThemeToggle />
            </div>
          </div>
        </header>

        <div className="grid gap-4 sm:grid-cols-3">
          <Card className="border-border/70 bg-background/85 shadow-sm">
            <CardContent className="p-5">
              <p className="text-xs tracking-[0.2em] text-muted-foreground uppercase">
                HTTP requests
              </p>
              <p className="mt-2 text-3xl font-semibold">{requests.length}</p>
            </CardContent>
          </Card>
          <Card className="border-border/70 bg-background/85 shadow-sm">
            <CardContent className="p-5">
              <p className="text-xs tracking-[0.2em] text-muted-foreground uppercase">
                WS sessions
              </p>
              <p className="mt-2 text-3xl font-semibold">{sessions.length}</p>
            </CardContent>
          </Card>
          <Card className="border-border/70 bg-background/85 shadow-sm">
            <CardContent className="p-5">
              <p className="text-xs tracking-[0.2em] text-muted-foreground uppercase">
                Active upgrades
              </p>
              <p className="mt-2 text-3xl font-semibold">
                {sessions.filter((session) => !session.ClosedAt).length}
              </p>
            </CardContent>
          </Card>
        </div>

        <Tabs
          className="xl:flex xl:min-h-0 xl:flex-1 xl:flex-col"
          onValueChange={(value) =>
            updateSearchParams({
              tab: value,
              request:
                value === "http"
                  ? selectedRequestId || requests[0]?.ID || null
                  : null,
              session:
                value === "websocket"
                  ? selectedSessionId || sessions[0]?.ID || null
                  : null,
            })
          }
          value={activeTab}
        >
          <TabsList variant="line">
            <TabsTrigger value="http">HTTP inspector</TabsTrigger>
            <TabsTrigger value="websocket">WebSocket inspector</TabsTrigger>
          </TabsList>

          <TabsContent className="xl:min-h-0" value="http">
            <div className="grid gap-4 xl:h-full xl:min-h-0 xl:grid-cols-[18rem_minmax(0,1fr)] 2xl:grid-cols-[19rem_minmax(0,1fr)]">
              <Card className="border-border/70 bg-background/80 shadow-sm xl:min-h-0">
                <CardContent className="flex h-full min-h-0 flex-col p-0">
                  {loading ? (
                    <div className="grid flex-1 gap-3 p-4">
                      {Array.from({ length: 8 }).map((_, index) => (
                        <div
                          className="h-20 animate-pulse rounded-md bg-muted"
                          key={index}
                        />
                      ))}
                    </div>
                  ) : filteredRequests.length === 0 ? (
                    <div className="flex min-h-80 flex-1 flex-col items-center justify-center gap-3 px-6 py-10 text-center">
                      <div className="rounded-md border border-dashed border-border/80 bg-muted/30 p-4">
                        <RadioTower className="size-6 text-muted-foreground" />
                      </div>
                      <div className="space-y-1">
                        <h2 className="font-medium">No request traces</h2>
                        <p className="text-sm text-muted-foreground">
                          {search
                            ? "Try a different filter."
                            : "Waiting for traffic to arrive on this tunnel."}
                        </p>
                      </div>
                    </div>
                  ) : (
                    <div className="min-h-0 flex-1 divide-y overflow-y-auto">
                      {filteredRequests.map((request) => (
                        <button
                          className={`w-full px-4 py-4 text-left transition hover:bg-muted/30 ${
                            request.ID === selectedRequest?.ID
                              ? "bg-muted/40"
                              : ""
                          }`}
                          key={request.ID}
                          onClick={() =>
                            updateSearchParams({
                              tab: "http",
                              request: request.ID,
                            })
                          }
                          type="button"
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div className="min-w-0 space-y-2">
                              <div className="flex flex-wrap items-center gap-2">
                                <Badge
                                  className={`ring-1 ${methodTone(request.Method)}`}
                                  variant="outline"
                                >
                                  {request.Method}
                                </Badge>
                                <Badge
                                  className={`ring-1 ${statusTone(
                                    request.ResponseStatusCode
                                  )}`}
                                  variant="outline"
                                >
                                  {request.ResponseStatusCode}
                                </Badge>
                                {request.IsReplayed ? (
                                  <Badge variant="outline">Replay</Badge>
                                ) : null}
                              </div>
                              <p className="truncate font-medium">
                                {request.Url}
                              </p>
                            </div>
                            <p className="shrink-0 text-xs text-muted-foreground">
                              {formatTime(request.LoggedAt)}
                            </p>
                          </div>
                        </button>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>

              <div className="min-w-0 xl:min-h-0">
                <RequestDetail
                  onRefresh={() => loadSummary(true)}
                  onSelectParent={() =>
                    selectedRequest?.ParentID
                      ? updateSearchParams({
                          tab: "http",
                          request: selectedRequest.ParentID,
                        })
                      : undefined
                  }
                  request={selectedRequest}
                />
              </div>
            </div>
          </TabsContent>

          <TabsContent className="xl:min-h-0" value="websocket">
            <div className="grid gap-4 xl:h-full xl:min-h-0 xl:grid-cols-[18rem_minmax(0,1fr)] 2xl:grid-cols-[19rem_minmax(0,1fr)]">
              <Card className="border-border/70 bg-background/80 shadow-sm xl:min-h-0">
                <CardContent className="flex h-full min-h-0 flex-col p-0">
                  {loading ? (
                    <div className="grid flex-1 gap-3 p-4">
                      {Array.from({ length: 6 }).map((_, index) => (
                        <div
                          className="h-24 animate-pulse rounded-md bg-muted"
                          key={index}
                        />
                      ))}
                    </div>
                  ) : filteredSessions.length === 0 ? (
                    <div className="flex min-h-80 flex-1 flex-col items-center justify-center gap-3 px-6 py-10 text-center">
                      <div className="rounded-md border border-dashed border-border/80 bg-muted/30 p-4">
                        <Waves className="size-6 text-muted-foreground" />
                      </div>
                      <div className="space-y-1">
                        <h2 className="font-medium">No WebSocket sessions</h2>
                        <p className="text-sm text-muted-foreground">
                          {search
                            ? "Try a different filter."
                            : "Upgraded connections will appear here as soon as they open."}
                        </p>
                      </div>
                    </div>
                  ) : (
                    <div className="min-h-0 flex-1 divide-y overflow-y-auto">
                      {filteredSessions.map((session) => (
                        <button
                          className={`w-full px-4 py-4 text-left transition hover:bg-muted/30 ${
                            session.ID === selectedSessionId
                              ? "bg-muted/40"
                              : ""
                          }`}
                          key={session.ID}
                          onClick={() =>
                            updateSearchParams({
                              tab: "websocket",
                              session: session.ID,
                            })
                          }
                          type="button"
                        >
                          <div className="space-y-2">
                            <div className="flex items-center justify-between gap-3">
                              <Badge
                                className={`ring-1 ${methodTone(session.Method)}`}
                                variant="outline"
                              >
                                {session.Method}
                              </Badge>
                              <span className="text-xs text-muted-foreground">
                                {session.ClosedAt
                                  ? formatTime(session.ClosedAt)
                                  : "Open"}
                              </span>
                            </div>
                            <p className="truncate font-medium">
                              {session.URL}
                            </p>
                            <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                              <Badge variant="outline">
                                {session.EventCount} frames
                              </Badge>
                              <Badge variant="outline">
                                {session.ClosedAt ? "Closed" : "Open"}
                              </Badge>
                            </div>
                          </div>
                        </button>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>

              <div className="min-w-0 xl:min-h-0">
                <WebSocketDetail
                  events={selectedSessionEvents}
                  session={selectedSession}
                />
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
