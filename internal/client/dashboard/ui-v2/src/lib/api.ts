import type {
  ReplayEditInput,
  RequestRecord,
  RequestSummary,
  TunnelStats,
  TunnelSummary,
  WebSocketEvent,
  WebSocketSession,
} from "@/types"

export type PageParams = {
  limit?: number
  offset?: number
}

function pageQuery(params?: PageParams & { search?: string; status?: string }) {
  const query = new URLSearchParams()
  if (params?.limit != null) query.set("limit", String(params.limit))
  if (params?.offset != null) query.set("offset", String(params.offset))
  if (params?.search) query.set("search", params.search)
  if (params?.status && params.status !== "all") query.set("status", params.status)
  const qs = query.toString()
  return qs ? `?${qs}` : ""
}

async function fetchJSON<T>(
  input: RequestInfo,
  init?: RequestInit
): Promise<T> {
  const response = await fetch(input, init)
  if (!response.ok) {
    const data = await response.json().catch(() => null)
    throw new Error(data?.message || "Request failed")
  }

  return response.json() as Promise<T>
}

export function getTunnels(
  params?: PageParams & { search?: string; status?: string }
) {
  return fetchJSON<{ tunnels: TunnelSummary[]; total: number; stats: TunnelStats }>(
    `/api/tunnels${pageQuery(params)}`
  )
}

export function getRequests(
  subdomain: string,
  localport: string,
  params?: PageParams
) {
  return fetchJSON<{ requests: RequestSummary[]; total: number }>(
    `/api/tunnels/${encodeURIComponent(subdomain)}/${localport}${pageQuery(params)}`
  )
}

export function getRequest(requestId: string) {
  return fetchJSON<{ request: RequestRecord }>(
    `/api/tunnels/requests/${encodeURIComponent(requestId)}`
  )
}

export function getWebSocketSessions(subdomain: string, localport: string) {
  return fetchJSON<{ sessions: WebSocketSession[] }>(
    `/api/tunnels/${encodeURIComponent(subdomain)}/${localport}/websocket-sessions`
  )
}

export function getWebSocketSession(sessionId: string) {
  return fetchJSON<{ session: WebSocketSession; events: WebSocketEvent[] }>(
    `/api/tunnels/websocket-sessions/${encodeURIComponent(sessionId)}`
  )
}

export function deleteTunnelLogs(subdomain: string, localport: number) {
  return fetchJSON<{ deleted_count: number }>(
    `/api/tunnels/${encodeURIComponent(subdomain)}/${localport}`,
    {
      method: "DELETE",
    }
  )
}

export function replayRequest(requestId: string) {
  return fetchJSON<{ message: string }>(
    `/api/tunnels/replay/${encodeURIComponent(requestId)}`
  )
}

export function replayRequestWithEdits(
  requestId: string,
  input: ReplayEditInput
) {
  return fetchJSON<{ message: string }>(
    `/api/tunnels/replay/${encodeURIComponent(requestId)}`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        method: input.method,
        path: input.path,
        headers: input.headers,
        body: input.body,
        body_encoding: input.bodyEncoding,
      }),
    }
  )
}
