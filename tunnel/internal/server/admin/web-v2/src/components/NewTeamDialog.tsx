import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { LoaderCircle } from 'lucide-react'
import { toast } from 'sonner'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface NewTeamDialogProps {
  isOpen: boolean
  setIsOpen: (open: boolean) => void
}

export default function NewTeamDialog({ isOpen, setIsOpen }: NewTeamDialogProps) {
  const [teamName, setTeamName] = useState('')
  const [teamSlug, setTeamSlug] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const handleTeamNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const name = e.target.value
    setTeamName(name)
    // Generate the slug automatically
    setTeamSlug(
      name
        .toLowerCase()
        .replace(/\s+/g, '-')
        .replace(/[^a-z0-9-]/g, '')
    )
  }

  const createTeam = async () => {
    setError('')

    if (!teamName) {
      setError('Team name is required')
      return
    }

    if (!teamSlug) {
      setError('Team slug is required')
      return
    }

    setSubmitting(true)

    try {
      const response = await fetch('/api/v1/team', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: teamName,
          slug: teamSlug,
        }),
      })

      const data = await response.json()

      if (response.ok) {
        toast.success('Team created successfully!')
        setIsOpen(false)
        // Navigate to the new team
        navigate(`/${teamSlug}/overview`)
      } else {
        setError(data.error || data.message || 'Failed to create team')
      }
    } catch (err) {
      console.error(err)
      setError('Something went wrong')
    } finally {
      setSubmitting(false)
    }
  }

  const handleOpenChange = (open: boolean) => {
    setIsOpen(open)
    if (!open) {
      // Reset form when closing
      setTeamName('')
      setTeamSlug('')
      setError('')
      setSubmitting(false)
    }
  }

  return (
    <AlertDialog open={isOpen} onOpenChange={handleOpenChange}>
      <AlertDialogContent className="sm:max-w-md">
        <AlertDialogHeader>
          <AlertDialogTitle>Create New Team</AlertDialogTitle>
          <AlertDialogDescription>
            Create a new team to manage connections and users
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="team-name">Team Name</Label>
            <Input
              id="team-name"
              value={teamName}
              onChange={handleTeamNameChange}
              placeholder="My Awesome Team"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="team-slug">Team Slug</Label>
            <Input
              id="team-slug"
              value={teamSlug}
              placeholder="my-awesome-team"
              className="font-mono"
              readOnly
            />
            <p className="text-xs text-gray-600">
              The slug will be used in URLs and is automatically generated from the team name
            </p>
          </div>

          {error && (
            <div className="rounded-md border border-destructive bg-destructive/10 p-3">
              <p className="text-sm text-destructive">{error}</p>
            </div>
          )}
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <Button onClick={createTeam} disabled={submitting || !teamName || !teamSlug}>
            {submitting && <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />}
            Create Team
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
