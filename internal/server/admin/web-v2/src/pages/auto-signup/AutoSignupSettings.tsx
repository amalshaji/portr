import { useState, useEffect } from 'react'
import { Plus, Save, Trash2, UserPlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Panel from '@/components/Panel'
import { Skeleton } from '@/components/ui/skeleton'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { toast } from 'sonner'
import type { AutoSignupDomain, AutoSignupSettings as AutoSignupSettingsType, Team } from '@/types'

interface UpdateAutoSignupSettingsPayload {
  auto_signup_enabled: boolean
  auto_signup_domains: Array<{
    domain: string
    team_id: number
  }>
}

export default function AutoSignupSettings() {
  const [settings, setSettings] = useState<AutoSignupSettingsType>({
    github_auth_enabled: false,
    auto_signup_enabled: false,
    auto_signup_domains: [],
  })
  const [teams, setTeams] = useState<Team[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetchSettings()
  }, [])

  const fetchSettings = async () => {
    setLoading(true)
    setLoadError(false)
    try {
      const [settingsResponse, teamsResponse] = await Promise.all([
        fetch('/api/v1/auto-signup/'),
        fetch('/api/v1/team/'),
      ])

      if (!settingsResponse.ok || !teamsResponse.ok) {
        throw new Error('Failed to load auto signup settings')
      }

      const [settingsData, teamsData] = await Promise.all([
        settingsResponse.json(),
        teamsResponse.json(),
      ])
      // A partial payload used to blank the page on the first render that read
      // auto_signup_domains. Treat a bad shape as a load failure so the retry
      // path below handles it.
      if (!settingsData || typeof settingsData !== 'object') {
        throw new Error('Malformed auto signup settings')
      }
      setSettings({
        github_auth_enabled: Boolean(settingsData.github_auth_enabled),
        auto_signup_enabled: Boolean(settingsData.auto_signup_enabled),
        auto_signup_domains: Array.isArray(settingsData.auto_signup_domains)
          ? settingsData.auto_signup_domains
          : [],
      })
      setTeams(Array.isArray(teamsData) ? teamsData : [])
    } catch (error) {
      console.error('Error fetching auto signup settings:', error)
      setLoadError(true)
      toast.error('Failed to load auto signup settings')
    } finally {
      setLoading(false)
    }
  }

  const handleAutoSignupEnabledChange = (enabled: boolean) => {
    setSettings((prev) => ({
      ...prev,
      auto_signup_enabled: enabled,
      // Auto signup needs at least one domain mapping, so start one for the admin.
      auto_signup_domains:
        enabled && prev.auto_signup_domains.length === 0
          ? [{ domain: '', team_id: null }]
          : prev.auto_signup_domains,
    }))
  }

  const handleDomainMappingChange = (index: number, patch: Partial<AutoSignupDomain>) => {
    setSettings((prev) => ({
      ...prev,
      auto_signup_domains: prev.auto_signup_domains.map((mapping, mappingIndex) =>
        mappingIndex === index ? { ...mapping, ...patch } : mapping
      ),
    }))
  }

  const addDomainMapping = () => {
    setSettings((prev) => ({
      ...prev,
      auto_signup_domains: [...prev.auto_signup_domains, { domain: '', team_id: null }],
    }))
  }

  const removeDomainMapping = (index: number) => {
    setSettings((prev) => ({
      ...prev,
      auto_signup_domains: prev.auto_signup_domains.filter((_, mappingIndex) => mappingIndex !== index),
    }))
  }

  const completeDomainMappings = settings.auto_signup_domains.filter(
    (mapping) => mapping.domain.trim() !== '' && Boolean(mapping.team_id)
  )
  const missingDomainMapping =
    settings.auto_signup_enabled && completeDomainMappings.length === 0

  const handleSave = async () => {
    setSaving(true)
    try {
      const payload: UpdateAutoSignupSettingsPayload = {
        auto_signup_enabled: settings.auto_signup_enabled,
        auto_signup_domains: settings.auto_signup_domains.map((mapping) => ({
          domain: mapping.domain,
          team_id: mapping.team_id ?? 0,
        })),
      }

      const response = await fetch('/api/v1/auto-signup/', {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })

      if (response.ok) {
        const data = await response.json()
        setSettings(data)
        toast.success('Auto signup settings updated')
      } else {
        const error = await response.json()
        toast.error(error.error || 'Failed to update auto signup settings')
      }
    } catch (error) {
      console.error('Error updating auto signup settings:', error)
      toast.error('Failed to update auto signup settings')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="mx-auto w-full max-w-3xl space-y-3">
        <Skeleton className="h-4 w-72" />
        <Skeleton className="h-56 w-full rounded-md" />
      </div>
    )
  }

  if (loadError) {
    return (
      <div className="mx-auto w-full max-w-3xl space-y-3 py-12 text-center">
        <p className="font-medium">Auto signup settings could not be loaded.</p>
        <p className="text-sm text-muted-foreground">
          Your saved configuration has not been changed.
        </p>
        <Button type="button" variant="outline" size="sm" onClick={fetchSettings}>
          Retry
        </Button>
      </div>
    )
  }

  return (
      <div className="mx-auto w-full max-w-3xl space-y-5">
        <p className="text-sm text-muted-foreground">
          Allow GitHub users from trusted domains to join selected teams.
        </p>

        <Panel
          icon={<UserPlus className="size-4" />}
          title="Trusted domain mappings"
          description="Each domain maps new GitHub signups to the team they should join."
        >
            <div className="flex items-center space-x-2">
              <Switch
                id="auto-signup-enabled"
                checked={settings.auto_signup_enabled}
                disabled={!settings.github_auth_enabled && !settings.auto_signup_enabled}
                onCheckedChange={handleAutoSignupEnabledChange}
              />
              <Label htmlFor="auto-signup-enabled">Enable auto signup</Label>
            </div>

            {!settings.github_auth_enabled && (
              <p className="text-sm text-muted-foreground">
                GitHub authentication must be configured on the server before auto signup can be enabled.
              </p>
            )}

            <div className="space-y-3">
              <div className="flex items-center justify-between gap-3">
                <Label>Domain mappings</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={!settings.auto_signup_enabled}
                  onClick={addDomainMapping}
                >
                  <Plus className="h-4 w-4 mr-2" />
                  Add domain
                </Button>
              </div>

              {settings.auto_signup_domains.length === 0 ? (
                <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">
                  No auto signup domains configured.
                </div>
              ) : (
                <div className="space-y-3">
                  {settings.auto_signup_domains.map((mapping, index) => (
                    <div
                      key={mapping.id ?? `new-${index}`}
                      className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_minmax(180px,240px)_auto] sm:items-end"
                    >
                      <div className="space-y-2">
                        <Label htmlFor={`auto-signup-domain-${index}`}>Domain</Label>
                        <Input
                          id={`auto-signup-domain-${index}`}
                          value={mapping.domain}
                          disabled={!settings.auto_signup_enabled}
                          onChange={(e) => handleDomainMappingChange(index, { domain: e.target.value })}
                          placeholder="example.com"
                        />
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor={`auto-signup-team-${index}`}>Team</Label>
                        <Select
                          value={mapping.team_id ? String(mapping.team_id) : ''}
                          disabled={!settings.auto_signup_enabled || teams.length === 0}
                          onValueChange={(value) => handleDomainMappingChange(index, { team_id: Number(value) })}
                        >
                          <SelectTrigger id={`auto-signup-team-${index}`}>
                            <SelectValue placeholder="Select a team" />
                          </SelectTrigger>
                          <SelectContent>
                            {teams.map((team) => (
                              <SelectItem key={team.id} value={String(team.id)}>
                                {team.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>

                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        disabled={!settings.auto_signup_enabled}
                        onClick={() => removeDomainMapping(index)}
                        aria-label="Remove domain mapping"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
        </Panel>

        <div className="flex items-center justify-end gap-3">
          {missingDomainMapping && (
            <p className="text-sm text-muted-foreground">
              Add at least one domain mapping to enable auto signup.
            </p>
          )}
          <Button onClick={handleSave} disabled={saving || missingDomainMapping} size="sm">
            <Save className="size-4" />
            {saving ? 'Saving...' : 'Save Settings'}
          </Button>
        </div>
      </div>
  )
}
