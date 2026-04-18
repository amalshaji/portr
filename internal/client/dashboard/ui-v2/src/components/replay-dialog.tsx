import * as React from "react"
import { LoaderCircle, RotateCcw } from "lucide-react"
import { toast } from "sonner"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { replayRequestWithEdits } from "@/lib/api"
import {
  decodeBase64ToText,
  getHeaderValue,
  headersToEditorValue,
  isFormUrlEncodedContentType,
  isJsonContentType,
  isTextContentType,
  parseHeaderEditorValue,
} from "@/lib/dashboard"
import type { ReplayEditInput, RequestRecord } from "@/types"

type ReplayDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  request: RequestRecord | null
  onReplayed: () => void
}

const replayMethods = ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "TRACE"]

function inferEditableBody(request: RequestRecord) {
  const contentType = getHeaderValue(request.Headers, "Content-Type")

  if (
    isJsonContentType(contentType) ||
    isFormUrlEncodedContentType(contentType) ||
    isTextContentType(contentType)
  ) {
    return { body: decodeBase64ToText(request.Body), bodyEncoding: "utf8" as const, note: "" }
  }

  const decoded = decodeBase64ToText(request.Body)
  if (decoded) {
    return { body: decoded, bodyEncoding: "utf8" as const, note: "" }
  }

  return {
    body: request.Body,
    bodyEncoding: "base64" as const,
    note: "Binary payload shown as base64.",
  }
}

function FieldLabel({ children }: { children: React.ReactNode }) {
  return (
    <span
      className="font-mono text-[10px] uppercase tracking-[0.08em]"
      style={{ color: "var(--muted-foreground)" }}
    >
      {children}
    </span>
  )
}

function MethodChip({
  method,
  active,
  onClick,
}: {
  method: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-[3px] border px-1.5 font-mono text-[10px] leading-5 transition-colors"
      style={
        active
          ? { background: "var(--foreground)", color: "var(--background)", borderColor: "var(--foreground)" }
          : { background: "var(--background)", color: "var(--muted-foreground)", borderColor: "var(--tm-line-2)" }
      }
    >
      {method}
    </button>
  )
}

export function ReplayDialog({ open, onOpenChange, request, onReplayed }: ReplayDialogProps) {
  const hydratedRequestIDRef = React.useRef<string | null>(null)
  const [form, setForm] = React.useState<ReplayEditInput>({
    method: "GET",
    path: "/",
    headers: {},
    body: "",
    bodyEncoding: "utf8",
  })
  const [headerEditor, setHeaderEditor] = React.useState("")
  const [bodyNote, setBodyNote] = React.useState("")
  const [submitting, setSubmitting] = React.useState(false)

  React.useEffect(() => {
    if (!open) {
      hydratedRequestIDRef.current = null
      return
    }
    if (!request || hydratedRequestIDRef.current === request.ID) return

    hydratedRequestIDRef.current = request.ID
    const inferredBody = inferEditableBody(request)
    setForm({
      method: request.Method,
      path: request.Url,
      headers: {},
      body: inferredBody.body,
      bodyEncoding: inferredBody.bodyEncoding,
    })
    setHeaderEditor(headersToEditorValue(request.Headers))
    setBodyNote(inferredBody.note)
  }, [open, request])

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!request) return

    setSubmitting(true)
    try {
      await replayRequestWithEdits(request.ID, {
        ...form,
        headers: parseHeaderEditorValue(headerEditor),
      })
      toast.success("Replay dispatched")
      onOpenChange(false)
      onReplayed()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to replay request")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 font-mono text-sm font-semibold tracking-[-0.01em]">
            <RotateCcw className="size-3.5" style={{ color: "var(--tm-green)" }} />
            edit &amp; replay
          </DialogTitle>
          <DialogDescription className="font-mono text-xs" style={{ color: "var(--muted-foreground)" }}>
            modify method, path, headers or body — replays through original tunnel host
          </DialogDescription>
        </DialogHeader>

        {request ? (
          <form className="mt-1 grid gap-5" onSubmit={handleSubmit}>
            {/* Method + Path */}
            <div className="grid gap-4 md:grid-cols-[auto_minmax(0,1fr)]">
              <div className="grid gap-2">
                <FieldLabel>Method</FieldLabel>
                <div className="flex flex-wrap gap-1">
                  {replayMethods.map((m) => (
                    <MethodChip
                      key={m}
                      method={m}
                      active={form.method === m}
                      onClick={() => setForm((c) => ({ ...c, method: m }))}
                    />
                  ))}
                </div>
              </div>
              <div className="grid gap-2">
                <FieldLabel>Path</FieldLabel>
                <input
                  value={form.path}
                  onChange={(e) => setForm((c) => ({ ...c, path: e.target.value }))}
                  className="h-7 w-full rounded-[4px] border border-border bg-background px-2 font-mono text-xs outline-none focus:border-foreground/40"
                  style={{ color: "var(--foreground)" }}
                  spellCheck={false}
                />
              </div>
            </div>

            {/* Headers */}
            <div className="grid gap-2">
              <FieldLabel>Headers</FieldLabel>
              <textarea
                value={headerEditor}
                onChange={(e) => setHeaderEditor(e.target.value)}
                rows={8}
                spellCheck={false}
                className="w-full rounded-[4px] border border-border bg-background px-3 py-2 font-mono text-xs outline-none focus:border-foreground/40"
                style={{ color: "var(--foreground)", resize: "vertical" }}
              />
            </div>

            {/* Body */}
            <div className="grid gap-2">
              <div className="flex items-center justify-between gap-3">
                <FieldLabel>Body</FieldLabel>
                <span className="font-mono text-[10px]" style={{ color: "var(--tm-muted-2)" }}>
                  encoding: {form.bodyEncoding}
                </span>
              </div>
              <textarea
                value={form.body}
                onChange={(e) => setForm((c) => ({ ...c, body: e.target.value }))}
                rows={12}
                spellCheck={false}
                className="w-full rounded-[4px] border border-border bg-background px-3 py-2 font-mono text-xs outline-none focus:border-foreground/40"
                style={{ color: "var(--foreground)", resize: "vertical" }}
              />
              {bodyNote ? (
                <p className="font-mono text-[11px]" style={{ color: "var(--tm-muted-2)" }}>
                  {bodyNote}
                </p>
              ) : null}
            </div>

            {/* Footer */}
            <div
              className="flex items-center justify-end gap-2 border-t pt-4"
              style={{ borderColor: "var(--border)" }}
            >
              <button
                type="button"
                disabled={submitting}
                onClick={() => onOpenChange(false)}
                className="rounded-[4px] border px-3 py-1.5 font-mono text-xs transition-colors hover:bg-muted disabled:pointer-events-none disabled:opacity-40"
                style={{ borderColor: "var(--border)", color: "var(--muted-foreground)" }}
              >
                cancel
              </button>
              <button
                type="submit"
                disabled={submitting}
                className="inline-flex items-center gap-1.5 rounded-[4px] border px-3 py-1.5 font-mono text-xs font-semibold transition-colors hover:opacity-90 disabled:pointer-events-none disabled:opacity-40"
                style={{
                  background: "var(--foreground)",
                  color: "var(--background)",
                  borderColor: "var(--foreground)",
                }}
              >
                {submitting ? (
                  <>
                    <LoaderCircle className="size-3 animate-spin" />
                    replaying…
                  </>
                ) : (
                  <>
                    <RotateCcw className="size-3" />
                    send replay
                  </>
                )}
              </button>
            </div>
          </form>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
