import { useCallback, useEffect, useState } from "react"
import {
  listReservedSubdomains,
  releaseSubdomain,
  reserveSubdomain,
} from "@/lib/reserved-subdomains-api"
import type { ReservedSubdomain } from "@/types"

interface ReservationsState {
  reservations: ReservedSubdomain[]
  limit: number
  baseDomain: string
  loading: boolean
  loadError: string
}

const initialState: ReservationsState = {
  reservations: [],
  limit: 0,
  baseDomain: "",
  loading: true,
  loadError: "",
}

export function useReservedDomains(team?: string) {
  const [state, setState] = useState(initialState)
  const [submitting, setSubmitting] = useState(false)
  const [releasing, setReleasing] = useState(false)

  const loadReservations = useCallback(
    async (signal?: AbortSignal) => {
      if (!team) return
      setState((current) => ({ ...current, loading: true, loadError: "" }))
      try {
        const response = await listReservedSubdomains(team, signal)
        if (signal?.aborted) return
        setState({
          reservations: response.data,
          limit: response.limit,
          baseDomain: response.base_domain,
          loading: false,
          loadError: "",
        })
      } catch (error) {
        if (signal?.aborted) return
        console.error(error)
        setState((current) => ({
          ...current,
          loading: false,
          loadError: "Reserved subdomains could not be loaded",
        }))
      }
    },
    [team],
  )

  useEffect(() => {
    const controller = new AbortController()
    void loadReservations(controller.signal)
    return () => controller.abort()
  }, [loadReservations])

  const reserve = useCallback(
    async (subdomain: string) => {
      if (!team) throw new Error("Team context required")
      setSubmitting(true)
      try {
        const created = await reserveSubdomain(team, subdomain)
        setState((current) => ({
          ...current,
          reservations: [created, ...current.reservations],
        }))
        return created
      } finally {
        setSubmitting(false)
      }
    },
    [team],
  )

  const release = useCallback(
    async (reservation: ReservedSubdomain) => {
      if (!team) throw new Error("Team context required")
      setReleasing(true)
      try {
        await releaseSubdomain(team, reservation.subdomain)
        setState((current) => ({
          ...current,
          reservations: current.reservations.filter(
            (item) => item.subdomain !== reservation.subdomain,
          ),
        }))
      } finally {
        setReleasing(false)
      }
    },
    [team],
  )

  return {
    ...state,
    submitting,
    releasing,
    loadReservations,
    reserve,
    release,
  }
}
