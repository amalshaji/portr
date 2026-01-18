import { useState, useEffect } from 'react'
import { Save, Mail, Server } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { toast } from 'sonner'
import type { InstanceSettings as InstanceSettingsType } from '@/types'

export default function InstanceSettings() {
  const [settings, setSettings] = useState<InstanceSettingsType>({
    smtp_enabled: false,
    smtp_host: '',
    smtp_port: 587,
    smtp_username: '',
    smtp_password: '',
    from_address: '',
    add_user_email_subject: 'Welcome to {team_name}',
    add_user_email_body: 'You have been invited to join {team_name}. Click the link below to get started:\n\n{invite_link}',
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetchSettings()
  }, [])

  const fetchSettings = async () => {
    setLoading(true)
    try {
      const response = await fetch('/api/v1/instance/settings')
      if (response.ok) {
        const data = await response.json()
        setSettings(data)
      }
    } catch (error) {
      console.error('Error fetching settings:', error)
      toast.error('Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  const handleSettingChange = (field: keyof InstanceSettingsType, value: string | number | boolean) => {
    setSettings((prev) => ({ ...prev, [field]: value }))
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const response = await fetch('/api/v1/instance/settings', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(settings),
      })

      if (response.ok) {
        toast.success('Settings updated successfully')
      } else {
        const error = await response.json()
        toast.error(error.message || 'Failed to update settings')
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
                <Mail className="h-5 w-5" />
                Email Configuration
              </CardTitle>
              <CardDescription>
                Configure SMTP settings for sending emails to team members
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center space-x-2">
                <Switch
                  id="smtp-enabled"
                  checked={settings.smtp_enabled}
                  onCheckedChange={(checked) => handleSettingChange('smtp_enabled', checked)}
                />
                <Label htmlFor="smtp-enabled">Enable SMTP</Label>
              </div>

              {settings.smtp_enabled && (
                <>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="smtp-host">SMTP Host</Label>
                      <Input
                        id="smtp-host"
                        value={settings.smtp_host}
                        onChange={(e) => handleSettingChange('smtp_host', e.target.value)}
                        placeholder="smtp.gmail.com"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="smtp-port">SMTP Port</Label>
                      <Input
                        id="smtp-port"
                        type="number"
                        value={settings.smtp_port}
                        onChange={(e) => handleSettingChange('smtp_port', parseInt(e.target.value))}
                        placeholder="587"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="smtp-username">SMTP Username</Label>
                      <Input
                        id="smtp-username"
                        value={settings.smtp_username}
                        onChange={(e) => handleSettingChange('smtp_username', e.target.value)}
                        placeholder="your@email.com"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="smtp-password">SMTP Password</Label>
                      <Input
                        id="smtp-password"
                        type="password"
                        value={settings.smtp_password}
                        onChange={(e) => handleSettingChange('smtp_password', e.target.value)}
                        placeholder="••••••••"
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="from-address">From Address</Label>
                    <Input
                      id="from-address"
                      value={settings.from_address}
                      onChange={(e) => handleSettingChange('from_address', e.target.value)}
                      placeholder="noreply@yourcompany.com"
                    />
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {settings.smtp_enabled && (
            <Card>
              <CardHeader>
                <CardTitle>Email Templates</CardTitle>
                <CardDescription>
                  customize email templates sent to users
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="email-subject">User Invitation Subject</Label>
                  <Input
                    id="email-subject"
                    value={settings.add_user_email_subject}
                    onChange={(e) => handleSettingChange('add_user_email_subject', e.target.value)}
                    placeholder="Welcome to {team_name}"
                  />
                  <p className="text-xs text-muted-foreground">
                    Available variables: {'{team_name}'}
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="email-body">User Invitation Body</Label>
                  <Textarea
                    id="email-body"
                    value={settings.add_user_email_body}
                    onChange={(e) => handleSettingChange('add_user_email_body', e.target.value)}
                    placeholder="You have been invited to join {team_name}..."
                    rows={6}
                  />
                  <p className="text-xs text-muted-foreground">
                    Available variables: {'{team_name}, {invite_link}'}
                  </p>
                </div>
              </CardContent>
            </Card>
          )}

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
