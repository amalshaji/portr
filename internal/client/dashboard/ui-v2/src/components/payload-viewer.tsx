import * as React from "react"
import { Light as SyntaxHighlighter } from "react-syntax-highlighter"
import jsonLanguage from "react-syntax-highlighter/dist/esm/languages/hljs/json"
import {
  atomOneDark,
  atomOneLight,
} from "react-syntax-highlighter/dist/esm/styles/hljs"
import { Download, FileCode2, ImageIcon, LoaderCircle } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useTheme } from "@/components/theme-provider"
import {
  contentLength,
  flattenHeaders,
  getHeaderValue,
  isFormUrlEncodedContentType,
  isHtmlContentType,
  isJsonContentType,
  isMediaContentType,
  isMultipartContentType,
  isTextContentType,
} from "@/lib/dashboard"
import type { RequestRecord } from "@/types"

SyntaxHighlighter.registerLanguage("json", jsonLanguage)

type MultipartEntry = {
  key: string
  type: "text" | "file"
  value?: string
  fileName?: string
  contentType?: string
  url?: string
}

type PayloadState =
  | { kind: "idle" | "loading" | "empty" }
  | { kind: "json" | "text"; value: string }
  | { kind: "form"; entries: [string, string][] }
  | { kind: "multipart"; entries: MultipartEntry[] }
  | { kind: "binary"; url: string }

type PayloadViewerProps = {
  request: RequestRecord
  type: "request" | "response"
}

function readOnlyLabel(type: "request" | "response") {
  return type === "request" ? "Request body" : "Response body"
}

export function PayloadViewer({ request, type }: PayloadViewerProps) {
  const { theme } = useTheme()
  const headers = type === "request" ? request.Headers : request.ResponseHeaders
  const rawBody = type === "request" ? request.Body : request.ResponseBody
  const length = contentLength(headers) || (rawBody ? 1 : 0)
  const contentType = getHeaderValue(headers, "Content-Type")
  const renderUrl = `/api/tunnels/render/${request.ID}?type=${type}`
  const [state, setState] = React.useState<PayloadState>({ kind: "idle" })

  React.useEffect(() => {
    if (length === 0) {
      setState({ kind: "empty" })
      return
    }

    if (
      isMediaContentType(contentType, "image/") ||
      isMediaContentType(contentType, "video/") ||
      isMediaContentType(contentType, "audio/") ||
      isHtmlContentType(contentType)
    ) {
      setState({ kind: "idle" })
      return
    }

    let active = true
    const objectUrls: string[] = []

    async function load() {
      setState({ kind: "loading" })
      try {
        if (isJsonContentType(contentType)) {
          const response = await fetch(renderUrl)
          const text = await response.text()
          const pretty = JSON.stringify(JSON.parse(text), null, 2)
          if (active) {
            setState({ kind: "json", value: pretty })
          }
          return
        }

        if (isFormUrlEncodedContentType(contentType)) {
          const response = await fetch(renderUrl)
          const text = await response.text()
          const entries = Array.from(new URLSearchParams(text).entries())
          if (active) {
            setState({ kind: "form", entries })
          }
          return
        }

        if (isMultipartContentType(contentType)) {
          const response = await fetch(renderUrl)
          const formData = await response.formData()
          const entries: MultipartEntry[] = []
          formData.forEach((value, key) => {
            if (value instanceof File) {
              const url = URL.createObjectURL(value)
              objectUrls.push(url)
              entries.push({
                key,
                type: "file",
                fileName: value.name,
                contentType: value.type,
                url,
              })
              return
            }
            entries.push({
              key,
              type: "text",
              value,
            })
          })
          if (active) {
            setState({ kind: "multipart", entries })
          }
          return
        }

        if (isTextContentType(contentType)) {
          const response = await fetch(renderUrl)
          const text = await response.text()
          if (active) {
            setState({ kind: "text", value: text })
          }
          return
        }

        const response = await fetch(renderUrl)
        const blob = await response.blob()
        const url = URL.createObjectURL(blob)
        objectUrls.push(url)
        if (active) {
          setState({ kind: "binary", url })
        }
      } catch {
        if (active) {
          setState({ kind: "text", value: "Unable to render payload." })
        }
      }
    }

    load()

    return () => {
      active = false
      objectUrls.forEach((url) => URL.revokeObjectURL(url))
    }
  }, [contentType, length, renderUrl])

  if (length === 0) {
    return (
      <div className="min-h-48 border border-dashed border-border/80 bg-muted/20 p-6 text-sm text-muted-foreground">
        No content
      </div>
    )
  }

  if (isMediaContentType(contentType, "image/")) {
    return (
      <div className="overflow-hidden border bg-background p-3">
        <img
          alt={readOnlyLabel(type)}
          className="max-h-[32rem] w-full rounded-sm object-contain"
          src={renderUrl}
        />
      </div>
    )
  }

  if (isMediaContentType(contentType, "video/")) {
    return (
      <div className="overflow-hidden border bg-background p-3">
        <video className="max-h-[32rem] w-full rounded-sm" controls>
          <source src={renderUrl} type={contentType} />
        </video>
      </div>
    )
  }

  if (isMediaContentType(contentType, "audio/")) {
    return (
      <div className="border bg-background p-4">
        <audio className="w-full" controls>
          <source src={renderUrl} type={contentType} />
        </audio>
      </div>
    )
  }

  if (isHtmlContentType(contentType)) {
    return (
      <div className="overflow-hidden border bg-background">
        <iframe
          className="h-[34rem] w-full bg-white"
          src={renderUrl}
          title={readOnlyLabel(type)}
        />
      </div>
    )
  }

  if (state.kind === "loading" || state.kind === "idle") {
    return (
      <div className="flex min-h-64 items-center justify-center border bg-background">
        <LoaderCircle className="size-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (state.kind === "json" || state.kind === "text") {
    return (
      <div className="border bg-background">
        <SyntaxHighlighter
          customStyle={{
            background: "transparent",
            fontSize: "0.8rem",
            margin: 0,
            minHeight: "20rem",
            overflowWrap: "anywhere",
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
          }}
          language={state.kind === "json" ? "json" : "text"}
          style={theme === "dark" ? atomOneDark : atomOneLight}
          wrapLongLines
        >
          {state.value}
        </SyntaxHighlighter>
      </div>
    )
  }

  if (state.kind === "form") {
    return (
      <div className="space-y-3 border bg-background p-4">
        {state.entries.map(([key, value]) => (
          <div className="grid gap-2" key={key}>
            <Label>{decodeURIComponent(key)}</Label>
            <Input readOnly value={decodeURIComponent(value)} />
          </div>
        ))}
      </div>
    )
  }

  if (state.kind === "multipart") {
    return (
      <div className="space-y-4 border bg-background p-4">
        {state.entries.map((entry) => (
          <div
            className="grid gap-2"
            key={`${entry.key}-${entry.fileName || entry.value}`}
          >
            <Label>{entry.key}</Label>
            {entry.type === "text" ? (
              <Input readOnly value={entry.value} />
            ) : entry.contentType?.startsWith("image/") ? (
              <img
                alt={entry.fileName || entry.key}
                className="max-h-64 rounded-sm border object-contain"
                src={entry.url}
              />
            ) : (
              <Button asChild className="w-fit" size="sm" variant="outline">
                <a download={entry.fileName} href={entry.url}>
                  <Download className="size-4" />
                  {entry.fileName || "Download file"}
                </a>
              </Button>
            )}
          </div>
        ))}
      </div>
    )
  }

  if (state.kind === "binary") {
    return (
      <div className="flex min-h-64 flex-col items-center justify-center gap-4 border bg-background p-6 text-center">
        <div className="rounded-md bg-muted p-4">
          <FileCode2 className="size-6 text-muted-foreground" />
        </div>
        <div>
          <p className="font-medium">Binary payload</p>
          <p className="text-sm text-muted-foreground">
            {flattenHeaders(headers)["Content-Type"] ||
              "application/octet-stream"}
          </p>
        </div>
        <Button asChild size="sm" variant="outline">
          <a download href={state.url}>
            <Download className="size-4" />
            Download payload
          </a>
        </Button>
      </div>
    )
  }

  return (
    <div className="flex min-h-64 items-center justify-center border bg-background">
      <ImageIcon className="size-5 text-muted-foreground" />
    </div>
  )
}
