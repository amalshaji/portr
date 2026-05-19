import { useState, useEffect } from 'react'
import { Save, Server, UserPlus } from 'lucide-react'
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
import type { InstanceSettings as InstanceSettingsType, Team } from '@/types'

export default function InstanceSettings() {
  const [settings, setSettings] = useState<InstanceSettingsType>({
    github_auth_enabled: false,
    auto_signup_enabled: false,
    auto_signup_allowed_domains: '',
    auto_signup_team_id: null,
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

  const handleSettingChange = (field: keyof InstanceSettingsType, value: string | number | boolean | null) => {
    setSettings((prev) => ({ ...prev, [field]: value }))
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
                  onCheckedChange={(checked) => handleSettingChange('auto_signup_enabled', checked)}
                />
                <Label htmlFor="auto-signup-enabled">Enable auto signup</Label>
              </div>

              {!settings.github_auth_enabled && (
                <p className="text-sm text-muted-foreground">
                  GitHub authentication must be configured on the server before auto signup can be enabled.
                </p>
              )}

              <div className="space-y-2">
                <Label htmlFor="trusted-domains">Trusted domains</Label>
                <Input
                  id="trusted-domains"
                  value={settings.auto_signup_allowed_domains}
                  disabled={!settings.auto_signup_enabled}
                  onChange={(e) => handleSettingChange('auto_signup_allowed_domains', e.target.value)}
                  placeholder="example.com, acme.co"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="auto-signup-team">Team</Label>
                <Select
                  value={settings.auto_signup_team_id ? String(settings.auto_signup_team_id) : ''}
                  disabled={!settings.auto_signup_enabled || teams.length === 0}
                  onValueChange={(value) => handleSettingChange('auto_signup_team_id', Number(value))}
                >
                  <SelectTrigger id="auto-signup-team">
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
