import * as React from "react"
import { Download, Waves } from "lucide-react"

import { KVTable, MethodTag, SectionLabel } from "@/components/terminal-primitives"
import {
  decodeBase64ToBytes,
  flattenHeaders,
  formatTime,
  payloadPreview,
  websocketPayloadLabel,
} from "@/lib/dashboard"
import type { WebSocketEvent, WebSocketSession } from "@/types"

/* ── WebSocket session sidebar ───────────────────────────── */

export function WebSocketSidebar({
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
                <span style={{ color: s.ClosedAt ? "var(--tm-muted-2)" : "var(--tm-green-ink)" }}>
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

/* ── WebSocket detail pane ───────────────────────────────── */

export function WebSocketDetailPane({
  session,
  events,
}: {
  session: WebSocketSession | null
  events: WebSocketEvent[]
}) {
  const [selectedEventId, setSelectedEventId] = React.useState<string | null>(null)
  const [wsTab, setWsTab] = React.useState<"timeline" | "request" | "response">("timeline")

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

  return (
    <div className="portr-detail-pane">
      {/* Pane header */}
      <div className="px-4 py-2.5" style={{ borderBottom: "1px solid var(--border)" }}>
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
          <span style={{ color: session.ClosedAt ? "var(--tm-muted-2)" : "var(--tm-green-ink)" }}>
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
      <div className="flex" style={{ borderBottom: "1px solid var(--border)" }}>
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
                  <p className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
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
