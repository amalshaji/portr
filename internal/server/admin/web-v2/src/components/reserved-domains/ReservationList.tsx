import { AlertCircle, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import type { ReservedSubdomain, SubdomainClaimStatus } from "@/types"

interface ReservationListProps {
  reservations: ReservedSubdomain[]
  baseDomain: string
  loading: boolean
  loadError: string
  onRetry: () => void
  onRelease: (reservation: ReservedSubdomain) => void
}

const dateFormatter = new Intl.DateTimeFormat(undefined, {
  day: "numeric",
  month: "short",
  year: "numeric",
})

function statusLabel(status: SubdomainClaimStatus) {
  if (status === "active") return "Active"
  if (status === "starting") return "Starting"
  return "Reserved"
}

function statusClasses(status: SubdomainClaimStatus) {
  if (status === "active") return "bg-emerald-500"
  if (status === "starting") return "bg-amber-500"
  return "bg-muted-foreground/45"
}

export function ReservationList({
  reservations,
  baseDomain,
  loading,
  loadError,
  onRetry,
  onRelease,
}: ReservationListProps) {
  return (
    <div className="border-t">
      <div className="flex items-center justify-between px-5 py-3 sm:px-6">
        <h2 className="text-sm font-medium">Your reserved domains</h2>
        {!loading && !loadError && (
          <span className="text-xs text-muted-foreground">
            {reservations.length === 1
              ? "1 reservation"
              : `${reservations.length} reservations`}
          </span>
        )}
      </div>

      {loading ? (
        <div className="space-y-px border-t bg-border">
          {[0, 1].map((item) => (
            <div
              key={item}
              className="flex items-center gap-4 bg-background px-5 py-4 sm:px-6"
            >
              <div className="h-4 w-44 animate-pulse rounded bg-muted" />
              <div className="ml-auto h-4 w-20 animate-pulse rounded bg-muted" />
            </div>
          ))}
        </div>
      ) : loadError ? (
        <div className="flex flex-col items-center gap-3 border-t px-5 py-10 text-center">
          <AlertCircle className="size-5 text-destructive" />
          <p className="text-sm text-muted-foreground">{loadError}</p>
          <Button variant="outline" size="sm" onClick={onRetry}>
            Try again
          </Button>
        </div>
      ) : reservations.length === 0 ? (
        <div className="border-t px-5 py-10 text-center sm:px-6">
          <p className="text-sm font-medium">No reserved domains yet</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Reserve a name above to keep it between tunnel runs.
          </p>
        </div>
      ) : (
        <ul className="divide-y border-t">
          {reservations.map((reservation) => (
            <li
              key={reservation.subdomain}
              className="animate-in fade-in slide-in-from-top-1 flex flex-col gap-3 px-5 py-4 duration-200 motion-reduce:animate-none sm:flex-row sm:items-center sm:px-6"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate font-mono text-sm font-medium">
                  {reservation.subdomain}.{baseDomain}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Reserved {dateFormatter.format(new Date(reservation.created_at))}
                </p>
              </div>
              <div className="flex items-center justify-between gap-5 sm:justify-end">
                <span className="inline-flex items-center gap-2 text-xs text-muted-foreground">
                  <span
                    className={`size-1.5 rounded-full ${statusClasses(reservation.claim_status)}`}
                    aria-hidden="true"
                  />
                  {statusLabel(reservation.claim_status)}
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-muted-foreground hover:text-destructive"
                  onClick={() => onRelease(reservation)}
                >
                  <Trash2 />
                  Release
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
