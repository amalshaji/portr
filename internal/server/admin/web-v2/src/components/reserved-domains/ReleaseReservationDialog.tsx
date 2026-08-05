import { LoaderCircle } from "lucide-react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import type { ReservedSubdomain } from "@/types"

interface ReleaseReservationDialogProps {
  reservation: ReservedSubdomain | null
  baseDomain: string
  releasing: boolean
  onClose: () => void
  onRelease: () => Promise<void>
}

export function ReleaseReservationDialog({
  reservation,
  baseDomain,
  releasing,
  onClose,
  onRelease,
}: ReleaseReservationDialogProps) {
  const fullDomain = reservation
    ? `${reservation.subdomain}.${baseDomain}`
    : "This subdomain"

  return (
    <AlertDialog
      open={Boolean(reservation)}
      onOpenChange={(open) => !open && !releasing && onClose()}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Release this subdomain?</AlertDialogTitle>
          <AlertDialogDescription>
            {reservation?.claim_status === "idle"
              ? `${fullDomain} will become available to other users.`
              : `${fullDomain} will stay unavailable until its current tunnel stops, then other users can claim it.`}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={releasing}>Keep reserved</AlertDialogCancel>
          <AlertDialogAction
            disabled={releasing}
            className="bg-destructive text-white hover:bg-destructive/90"
            onClick={(event) => {
              event.preventDefault()
              void onRelease()
            }}
          >
            {releasing && <LoaderCircle className="animate-spin" />}
            Release
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
