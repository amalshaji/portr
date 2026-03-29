import * as React from "react"
import { LoaderCircle, Sparkles } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
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

const replayMethods = [
  "GET",
  "POST",
  "PUT",
  "PATCH",
  "DELETE",
  "OPTIONS",
  "HEAD",
  "TRACE",
]

function inferEditableBody(request: RequestRecord) {
  const contentType = getHeaderValue(request.Headers, "Content-Type")

  if (
    isJsonContentType(contentType) ||
    isFormUrlEncodedContentType(contentType) ||
    isTextContentType(contentType)
  ) {
    return {
      body: decodeBase64ToText(request.Body),
      bodyEncoding: "utf8" as const,
      note: "",
    }
  }

  const decoded = decodeBase64ToText(request.Body)
  if (decoded) {
    return {
      body: decoded,
      bodyEncoding: "utf8" as const,
      note: "",
    }
  }

  return {
    body: request.Body,
    bodyEncoding: "base64" as const,
    note: "This payload is binary or non-textual. It is shown as base64 for editing.",
  }
}

export function ReplayDialog({
  open,
  onOpenChange,
  request,
  onReplayed,
}: ReplayDialogProps) {
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
    if (!request) {
      return
    }
    if (hydratedRequestIDRef.current === request.ID) {
      return
    }

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
    if (!request) {
      return
    }

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
      toast.error(
        error instanceof Error ? error.message : "Failed to replay request"
      )
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="size-4 text-primary" />
            Edit and replay
          </DialogTitle>
          <DialogDescription>
            Adjust the method, path, headers, or body before sending the replay
            through the original tunnel host.
          </DialogDescription>
        </DialogHeader>

        {request ? (
          <form className="grid gap-5" onSubmit={handleSubmit}>
            <div className="grid gap-4 md:grid-cols-[180px_minmax(0,1fr)]">
              <div className="grid gap-2">
                <Label>Method</Label>
                <Select
                  onValueChange={(value) =>
                    setForm((current) => ({ ...current, method: value }))
                  }
                  value={form.method}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Method" />
                  </SelectTrigger>
                  <SelectContent>
                    {replayMethods.map((method) => (
                      <SelectItem key={method} value={method}>
                        {method}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>Path</Label>
                <Input
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      path: event.target.value,
                    }))
                  }
                  value={form.path}
                />
              </div>
            </div>

            <div className="grid gap-2">
              <Label>Headers</Label>
              <Textarea
                className="min-h-44 font-mono text-xs"
                onChange={(event) => setHeaderEditor(event.target.value)}
                spellCheck={false}
                value={headerEditor}
              />
            </div>

            <div className="grid gap-2">
              <div className="flex items-center justify-between gap-3">
                <Label>Body</Label>
                <p className="text-xs text-muted-foreground">
                  Encoding: {form.bodyEncoding}
                </p>
              </div>
              <Textarea
                className="min-h-56 font-mono text-xs"
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    body: event.target.value,
                  }))
                }
                spellCheck={false}
                value={form.body}
              />
              {bodyNote ? (
                <p className="text-xs text-muted-foreground">{bodyNote}</p>
              ) : null}
            </div>

            <DialogFooter>
              <Button
                disabled={submitting}
                onClick={() => onOpenChange(false)}
                type="button"
                variant="outline"
              >
                Cancel
              </Button>
              <Button disabled={submitting} type="submit">
                {submitting ? (
                  <>
                    <LoaderCircle className="size-4 animate-spin" />
                    Replaying...
                  </>
                ) : (
                  "Send replay"
                )}
              </Button>
            </DialogFooter>
          </form>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
