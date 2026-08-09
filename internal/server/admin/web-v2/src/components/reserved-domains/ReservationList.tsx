import { AlertCircle, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import RouteLine from "@/components/RouteLine"
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
  if (status === "active") return "bg-signal-live"
  if (status === "starting") return "bg-signal-idle"
  return "bg-signal-unbound"
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
        <div className="divide-y border-t">
          {[0, 1].map((item) => (
            <div key={item} className="flex items-center gap-4 px-5 py-4 sm:px-6">
              <Skeleton className="h-4 w-44" />
              <Skeleton className="ml-auto h-4 w-20" />
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
        <div className="border-t px-5 py-10 sm:px-6">
          {/* A reserved name with nothing running is literally an unplugged
              route — the empty state shows the shape it will take. */}
          <div className="mx-auto max-w-sm">
            <RouteLine
              name={`your-name.${baseDomain || "example.com"}`}
              state="unbound"
            />
            <p className="mt-4 text-center text-sm font-medium">
              No reserved domains yet
            </p>
            <p className="mt-1 text-center text-sm text-muted-foreground">
              Reserve a name above to keep it between tunnel runs.
            </p>
          </div>
        </div>
      ) : (
        <ul className="divide-y border-t">
          {reservations.map((reservation) => (
            <li
              key={reservation.subdomain}
              className="animate-in fade-in slide-in-from-top-1 flex flex-col gap-3 px-5 py-4 duration-200 motion-reduce:animate-none sm:flex-row sm:items-center sm:px-6"
            >
              <div className="min-w-0 flex-1">
                <p className="data truncate text-sm font-medium">
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
