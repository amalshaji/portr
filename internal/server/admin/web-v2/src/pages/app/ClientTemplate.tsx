import { useCallback, useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { FileCode, LoaderCircle, Save } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Textarea } from "@/components/ui/textarea"
import { useUserStore } from "@/lib/store"
import type { ClientTemplate as ClientTemplateType } from "@/types"

const templatePlaceholder = `tunnels:
  - name: web
    subdomain: acme-web
    port: 3000
  - name: api
    subdomain: acme-api
    port: 8000
groups:
  frontend: [web, api]`

export default function ClientTemplate() {
  const { team } = useParams<{ team: string }>()
  const { currentUser } = useUserStore()
  const canEdit = currentUser?.role !== "member"

  const [template, setTemplate] = useState("")
  const [savedTemplate, setSavedTemplate] = useState("")
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState(false)
  const [saving, setSaving] = useState(false)
  const [validationError, setValidationError] = useState("")

  const loadTemplate = useCallback(async () => {
    if (!team) return

    setLoading(true)
    setLoadError(false)
    try {
      const response = await fetch("/api/v1/config/template", {
        headers: { "x-team-slug": team },
      })
      if (!response.ok) {
        throw new Error("Failed to load client template")
      }

      const data: ClientTemplateType = await response.json()
      setTemplate(data.template)
      setSavedTemplate(data.template)
    } catch (error) {
      console.error("Error fetching client template:", error)
      setLoadError(true)
      toast.error("Failed to load the client template")
    } finally {
      setLoading(false)
    }
  }, [team])

  useEffect(() => {
    loadTemplate()
  }, [loadTemplate])

  const handleSave = async () => {
    if (!team) return

    setSaving(true)
    setValidationError("")
    try {
      const response = await fetch("/api/v1/config/template", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
        body: JSON.stringify({ template }),
      })

      if (!response.ok) {
        const { error } = await response.json()
        setValidationError(error || "Failed to save the client template")
        return
      }

      const data: ClientTemplateType = await response.json()
      setTemplate(data.template)
      setSavedTemplate(data.template)
      toast.success("Client template saved")
    } catch (error) {
      console.error("Error saving client template:", error)
      toast.error("Failed to save the client template")
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen p-8">
        <div className="max-w-4xl mx-auto">
          <div className="text-center py-8">
            <p className="text-muted-foreground">Loading client template...</p>
          </div>
        </div>
      </div>
    )
  }

  if (loadError) {
    return (
      <div className="min-h-screen p-8">
        <div className="max-w-4xl mx-auto space-y-4 text-center py-8">
          <p className="font-medium">The client template could not be loaded.</p>
          <p className="text-sm text-muted-foreground">
            Your saved template has not been changed.
          </p>
          <Button type="button" variant="outline" onClick={loadTemplate}>
            Retry
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen p-8">
      <div className="max-w-4xl mx-auto space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Client template</h1>
          <p className="text-muted-foreground">
            Share one set of tunnels and groups with everyone on the team.
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileCode className="h-5 w-5" />
              Tunnels and groups
            </CardTitle>
            <CardDescription>
              Members get this template when they run{" "}
              <code>portr auth set</code>, and can refresh it any time with{" "}
              <code>portr config pull</code>. Only <code>tunnels</code> and{" "}
              <code>groups</code> can be set here — the server url, ssh url and
              auth token stay managed by portr. Leave it empty to remove the
              team template.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Textarea
              id="client-template"
              aria-label="Client template"
              className="font-mono text-sm"
              rows={18}
              spellCheck={false}
              readOnly={!canEdit}
              placeholder={templatePlaceholder}
              value={template}
              onChange={(e) => {
                setTemplate(e.target.value)
                setValidationError("")
              }}
            />

            {validationError && (
              <p className="text-sm text-destructive" role="alert">
                {validationError}
              </p>
            )}

            {!canEdit && (
              <p className="text-sm text-muted-foreground">
                Only team admins can change the client template.
              </p>
            )}
          </CardContent>
        </Card>

        <div className="flex justify-end">
          <Button
            onClick={handleSave}
            disabled={!canEdit || saving || template === savedTemplate}
          >
            {saving ? (
              <LoaderCircle className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <Save className="h-4 w-4 mr-2" />
            )}
            {saving ? "Saving..." : "Save template"}
          </Button>
        </div>
      </div>
    </div>
  )
}
