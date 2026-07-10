import { useState } from "react"
import { useParams } from "react-router-dom"
import { Check, Globe } from "lucide-react"
import { toast } from "sonner"
import { ReleaseReservationDialog } from "@/components/reserved-domains/ReleaseReservationDialog"
import { ReservationForm } from "@/components/reserved-domains/ReservationForm"
import { ReservationList } from "@/components/reserved-domains/ReservationList"
import { useReservedDomains } from "@/hooks/use-reserved-domains"
import type { ReservedSubdomain } from "@/types"

export default function ReservedDomains() {
  const { team } = useParams<{ team: string }>()
  const {
    reservations,
    limit,
    baseDomain,
    loading,
    loadError,
    submitting,
    releasing,
    loadReservations,
    reserve,
    release,
  } = useReservedDomains(team)
  const [releaseTarget, setReleaseTarget] =
    useState<ReservedSubdomain | null>(null)

  const reserveAndNotify = async (subdomain: string) => {
    const created = await reserve(subdomain)
    toast.success(`${created.subdomain}.${baseDomain} reserved`)
  }

  const releaseAndNotify = async () => {
    if (!releaseTarget) return
    try {
      await release(releaseTarget)
      toast.success(
        releaseTarget.claim_status === "idle"
          ? "Reservation released"
          : "Reservation released. The tunnel keeps this name until it stops.",
      )
      setReleaseTarget(null)
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Reservation could not be released",
      )
    }
  }

  return (
    <div className="mx-auto w-full max-w-5xl space-y-7">
      <header className="flex items-start gap-4">
        <div className="mt-0.5 flex size-10 shrink-0 items-center justify-center rounded-lg border bg-muted/45">
          <Globe className="size-5" aria-hidden="true" />
        </div>
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">
            Reserved domains
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Keep stable subdomains for this membership, even between tunnel runs.
          </p>
        </div>
      </header>

      <section className="overflow-hidden rounded-xl border bg-background shadow-xs">
        <ReservationForm
          baseDomain={baseDomain}
          count={reservations.length}
          limit={limit}
          loading={loading}
          submitting={submitting}
          onReserve={reserveAndNotify}
        />
        <ReservationList
          reservations={reservations}
          baseDomain={baseDomain}
          loading={loading}
          loadError={loadError}
          onRetry={() => void loadReservations()}
          onRelease={setReleaseTarget}
        />
      </section>

      <div className="flex items-start gap-2 text-xs text-muted-foreground">
        <Check className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
        <p>
          Reservations are unique across this Portr deployment. Other credentials
          cannot claim them.
        </p>
      </div>

      <ReleaseReservationDialog
        reservation={releaseTarget}
        baseDomain={baseDomain}
        releasing={releasing}
        onClose={() => setReleaseTarget(null)}
        onRelease={releaseAndNotify}
      />
    </div>
  )
}
