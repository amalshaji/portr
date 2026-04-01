export type HeaderMap = Record<string, string[]>

export type TunnelSummary = {
  Subdomain: string
  Localport: number
  last_request_id?: string
  last_method?: string
  last_url?: string
  last_status?: number
  last_activity_at: string
  last_activity_kind: "http" | "websocket" | ""
  http_request_count: number
  websocket_session_count: number
  active_websocket_count: number
}

export type RequestRecord = {
  ID: string
  Subdomain: string
  Host: string
  Localport: number
  Url: string
  Method: string
  Headers: HeaderMap
  Body: string
  ResponseStatusCode: number
  ResponseHeaders: HeaderMap
  ResponseBody: string
  IsReplayed: boolean
  ParentID: string
  LoggedAt: string
}

export type WebSocketSession = {
  ID: string
  HandshakeRequestID: string
  Subdomain: string
  Localport: number
  Host: string
  URL: string
  Method: string
  RequestHeaders: HeaderMap
  ResponseStatusCode: number
  ResponseHeaders: HeaderMap
  StartedAt: string
  LastEventAt: string | null
  ClosedAt: string | null
  CloseCode: number | null
  CloseReason: string
  EventCount: number
  ClientEventCount: number
  ServerEventCount: number
}

export type WebSocketEvent = {
  id: string
  direction: "client" | "server"
  opcode: number
  opcode_name: string
  is_final: boolean
  payload: string
  payload_text?: string
  payload_length: number
  logged_at: string
}

export type ReplayEditInput = {
  method: string
  path: string
  headers: Record<string, string>
  body: string
  bodyEncoding: "utf8" | "base64"
}
