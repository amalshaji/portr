import type {
  ReplayEditInput,
  RequestRecord,
  TunnelSummary,
  WebSocketEvent,
  WebSocketSession,
} from "@/types"

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

export function getTunnels() {
  return fetchJSON<{ tunnels: TunnelSummary[] }>("/api/tunnels")
}

export function getRequests(subdomain: string, localport: string) {
  return fetchJSON<{ requests: RequestRecord[] }>(
    `/api/tunnels/${encodeURIComponent(subdomain)}/${localport}`
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
