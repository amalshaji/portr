import { useState, useEffect } from 'react'
import { Plus, Save, Server, Trash2, UserPlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
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
import type { AutoSignupDomain, InstanceSettings as InstanceSettingsType, Team } from '@/types'

export default function InstanceSettings() {
  const [settings, setSettings] = useState<InstanceSettingsType>({
    github_auth_enabled: false,
    auto_signup_enabled: false,
    auto_signup_domains: [],
  })
  const [teams, setTeams] = useState<Team[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetchSettings()
  }, [])

  const fetchSettings = async () => {
    setLoading(true)
    try {
      const [settingsResponse, teamsResponse] = await Promise.all([
        fetch('/api/v1/instance-settings/'),
        fetch('/api/v1/team/'),
      ])

      if (settingsResponse.ok) {
        const data = await settingsResponse.json()
        setSettings(data)
      }
      if (teamsResponse.ok) {
        const data = await teamsResponse.json()
        setTeams(data)
      }
    } catch (error) {
      console.error('Error fetching settings:', error)
      toast.error('Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  const handleAutoSignupEnabledChange = (enabled: boolean) => {
    setSettings((prev) => ({ ...prev, auto_signup_enabled: enabled }))
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

  const handleSave = async () => {
    setSaving(true)
    try {
      const response = await fetch('/api/v1/instance-settings/', {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(settings),
      })

      if (response.ok) {
        const data = await response.json()
        setSettings(data)
        toast.success('Settings updated successfully')
      } else {
        const error = await response.json()
        toast.error(error.error || 'Failed to update settings')
      }
    } catch (error) {
      console.error('Error updating settings:', error)
      toast.error('Failed to update settings')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen p-8">
        <div className="max-w-4xl mx-auto">
          <div className="text-center py-8">
            <p className="text-muted-foreground">Loading instance settings...</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen p-8">
      <div className="max-w-4xl mx-auto space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Instance Settings</h1>
          <p className="text-muted-foreground">
            Configure your Portr instance settings and preferences
          </p>
        </div>

        <div className="grid gap-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <UserPlus className="h-5 w-5" />
                GitHub Auto Signup
              </CardTitle>
              <CardDescription>
                Allow GitHub users from trusted domains to join a selected team
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
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
                            placeholder="amal.sh"
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
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                Server Information
              </CardTitle>
              <CardDescription>
                Information about your Portr instance
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Version</p>
                  <p className="text-sm">v1.0.0</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Build</p>
                  <p className="text-sm font-mono">abc123def</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Environment</p>
                  <p className="text-sm">Production</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">License</p>
                  <p className="text-sm">Open Source</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <div className="flex justify-end">
            <Button onClick={handleSave} disabled={saving}>
              <Save className="h-4 w-4 mr-2" />
              {saving ? 'Saving...' : 'Save Settings'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
