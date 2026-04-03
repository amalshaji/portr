import { getReasonPhrase } from "http-status-codes"

import type { HeaderMap, RequestRecord, WebSocketEvent } from "@/types"

const textDecoder = new TextDecoder()
const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: "medium",
  timeStyle: "short",
})
const timeFormatter = new Intl.DateTimeFormat(undefined, {
  timeStyle: "medium",
})

export function currentDashboardHost(
  locationValue:
    | Pick<Location, "host" | "hostname">
    | null = typeof window === "undefined" ? null : window.location
) {
  if (!locationValue) {
    return "localhost"
  }

  return locationValue.host || locationValue.hostname || "localhost"
}

export function parseTunnelId(id: string) {
  const separatorIndex = id.lastIndexOf("-")
  if (separatorIndex < 0) {
    return null
  }

  return {
    subdomain: id.slice(0, separatorIndex),
    localport: id.slice(separatorIndex + 1),
  }
}

export function formatDateTime(value: string | null | undefined) {
  if (!value) {
    return "Waiting"
  }

  return dateTimeFormatter.format(new Date(value))
}

export function formatTime(value: string | null | undefined) {
  if (!value) {
    return "Waiting"
  }

  return timeFormatter.format(new Date(value))
}

export function methodTone(method: string) {
  switch (method.toUpperCase()) {
    case "GET":
      return "bg-emerald-500/10 text-emerald-600 ring-emerald-500/25 dark:bg-emerald-500/15 dark:text-emerald-400 dark:ring-emerald-500/30"
    case "POST":
      return "bg-blue-500/10 text-blue-600 ring-blue-500/25 dark:bg-blue-500/15 dark:text-blue-400 dark:ring-blue-500/30"
    case "PUT":
      return "bg-amber-500/10 text-amber-600 ring-amber-500/25 dark:bg-amber-500/15 dark:text-amber-400 dark:ring-amber-500/30"
    case "PATCH":
      return "bg-violet-500/10 text-violet-600 ring-violet-500/25 dark:bg-violet-500/15 dark:text-violet-400 dark:ring-violet-500/30"
    case "DELETE":
      return "bg-red-500/10 text-red-600 ring-red-500/25 dark:bg-red-500/15 dark:text-red-400 dark:ring-red-500/30"
    default:
      return "bg-zinc-500/10 text-zinc-600 ring-zinc-500/25 dark:bg-zinc-500/15 dark:text-zinc-400 dark:ring-zinc-500/30"
  }
}

export function statusTone(status: number) {
  if (status >= 500) {
    return "bg-red-500/10 text-red-600 ring-red-500/25 dark:bg-red-500/15 dark:text-red-400 dark:ring-red-500/30"
  }
  if (status >= 400) {
    return "bg-orange-500/10 text-orange-600 ring-orange-500/25 dark:bg-orange-500/15 dark:text-orange-400 dark:ring-orange-500/30"
  }
  if (status >= 300) {
    return "bg-amber-500/10 text-amber-600 ring-amber-500/25 dark:bg-amber-500/15 dark:text-amber-400 dark:ring-amber-500/30"
  }
  if (status >= 200) {
    return "bg-emerald-500/10 text-emerald-600 ring-emerald-500/25 dark:bg-emerald-500/15 dark:text-emerald-400 dark:ring-emerald-500/30"
  }
  return "bg-zinc-500/10 text-zinc-600 ring-zinc-500/25 dark:bg-zinc-500/15 dark:text-zinc-400 dark:ring-zinc-500/30"
}

export function websocketDirectionTone(direction: string) {
  return direction === "client"
    ? "bg-blue-500/10 text-blue-600 ring-blue-500/25 dark:bg-blue-500/15 dark:text-blue-400 dark:ring-blue-500/30"
    : "bg-teal-500/10 text-teal-600 ring-teal-500/25 dark:bg-teal-500/15 dark:text-teal-400 dark:ring-teal-500/30"
}

export function websocketOpcodeTone(opcode: number) {
  if (opcode === 1) {
    return "bg-emerald-500/10 text-emerald-600 ring-emerald-500/25 dark:bg-emerald-500/15 dark:text-emerald-400 dark:ring-emerald-500/30"
  }
  if (opcode === 2) {
    return "bg-violet-500/10 text-violet-600 ring-violet-500/25 dark:bg-violet-500/15 dark:text-violet-400 dark:ring-violet-500/30"
  }
  if (opcode === 8) {
    return "bg-red-500/10 text-red-600 ring-red-500/25 dark:bg-red-500/15 dark:text-red-400 dark:ring-red-500/30"
  }
  if (opcode === 9 || opcode === 10) {
    return "bg-amber-500/10 text-amber-600 ring-amber-500/25 dark:bg-amber-500/15 dark:text-amber-400 dark:ring-amber-500/30"
  }
  return "bg-zinc-500/10 text-zinc-600 ring-zinc-500/25 dark:bg-zinc-500/15 dark:text-zinc-400 dark:ring-zinc-500/30"
}

export function getHeaderValue(headers: HeaderMap | undefined, name: string) {
  if (!headers) {
    return ""
  }

  return headers[name]?.[0] || headers[name.toLowerCase()]?.[0] || ""
}

export function flattenHeaders(headers: HeaderMap | undefined) {
  const flat: Record<string, string> = {}
  if (!headers) {
    return flat
  }

  Object.entries(headers).forEach(([key, values]) => {
    if (Array.isArray(values) && values.length > 0) {
      flat[key] = values[0]
    }
  })

  return flat
}

export function headersToEditorValue(headers: HeaderMap | undefined) {
  return Object.entries(flattenHeaders(headers))
    .map(([key, value]) => `${key}: ${value}`)
    .join("\n")
}

export function parseHeaderEditorValue(value: string) {
  const headers: Record<string, string> = {}
  value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .forEach((line) => {
      const separatorIndex = line.indexOf(":")
      if (separatorIndex <= 0) {
        return
      }

      const key = line.slice(0, separatorIndex).trim()
      const headerValue = line.slice(separatorIndex + 1).trim()
      if (!key) {
        return
      }
      headers[key] = headerValue
    })

  return headers
}

export function decodeBase64ToBytes(value: string) {
  const binary = atob(value || "")
  return Uint8Array.from(binary, (character) => character.charCodeAt(0))
}

export function decodeBase64ToText(value: string) {
  try {
    return textDecoder.decode(decodeBase64ToBytes(value))
  } catch {
    return ""
  }
}

export function isLikelyText(value: string) {
  return decodeBase64ToText(value) !== ""
}

export function contentLength(headers: HeaderMap | undefined) {
  return Number(getHeaderValue(headers, "Content-Length") || "0")
}

export function isJsonContentType(contentType: string) {
  return contentType.startsWith("application/json")
}

export function isMultipartContentType(contentType: string) {
  return contentType.startsWith("multipart/form-data")
}

export function isFormUrlEncodedContentType(contentType: string) {
  return contentType.startsWith("application/x-www-form-urlencoded")
}

export function isHtmlContentType(contentType: string) {
  return contentType.startsWith("text/html")
}

export function isTextContentType(contentType: string) {
  return (
    contentType.startsWith("text/") ||
    contentType.includes("xml") ||
    contentType.includes("javascript")
  )
}

export function isMediaContentType(contentType: string, prefix: string) {
  return contentType.startsWith(prefix)
}

export function reasonPhrase(status: number) {
  try {
    return getReasonPhrase(status)
  } catch {
    return "Unknown"
  }
}

export function buildCurlCommand(request: RequestRecord) {
  const tunnelUrl = `https://${request.Host}${request.Url}`
  let curl = `curl -X ${request.Method} '${tunnelUrl}'`
  const headers = flattenHeaders(request.Headers)
  const contentType = headers["Content-Type"] || ""
  const isMultipart = contentType.startsWith("multipart/form-data")

  Object.entries(headers).forEach(([key, value]) => {
    if (key === "Content-Type" || key === "Content-Length" || !value) {
      return
    }

    curl += ` \\\n  -H '${key}: ${value}'`
  })

  if (!request.Body) {
    return curl
  }

  const decodedBody = decodeBase64ToText(request.Body)
  if (!decodedBody) {
    return curl
  }

  if (isMultipart) {
    decodedBody
      .split("\r\n")
      .filter((line) => line.startsWith("Content-Disposition:"))
      .forEach((line) => {
        const nameMatch = line.match(/name="([^"]+)"/)
        const fileMatch = line.match(/filename="([^"]+)"/)
        if (!nameMatch) {
          return
        }
        if (fileMatch) {
          curl += ` \\\n  -F '${nameMatch[1]}=@path/to/${fileMatch[1]}'`
          return
        }
        curl += ` \\\n  -F '${nameMatch[1]}=value'`
      })

    return curl
  }

  curl += ` \\\n  -d '${decodedBody.replace(/'/g, "'\\''")}'`
  return curl
}

export function payloadPreview(event: WebSocketEvent) {
  if (event.payload_text) {
    return event.payload_text
  }

  if (event.opcode === 8) {
    return "Close frame"
  }
  if (event.opcode === 9) {
    return "Ping"
  }
  if (event.opcode === 10) {
    return "Pong"
  }
  return "Binary payload"
}

export function websocketPayloadLabel(event: WebSocketEvent) {
  if (event.payload_text) {
    return "Decoded payload"
  }
  if (event.opcode === 8) {
    return "Close frame payload"
  }
  if (event.opcode === 9) {
    return "Ping payload"
  }
  if (event.opcode === 10) {
    return "Pong payload"
  }
  return "Binary payload"
}

export function makeBlobUrl(
  content: BlobPart,
  type = "application/octet-stream"
) {
  return URL.createObjectURL(new Blob([content], { type }))
}
