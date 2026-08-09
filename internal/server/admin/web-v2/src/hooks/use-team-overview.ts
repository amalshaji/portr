import { useCallback, useEffect, useState } from "react"
import type { Connection } from "@/types"

export type SetupState = "loading" | "ready" | "error"

/** The one-off command that authenticates a machine against this team. */
export function useSetupScript(team?: string) {
  const [script, setScript] = useState("")
  const [state, setState] = useState<SetupState>("loading")
  const [attempt, setAttempt] = useState(0)

  const retry = useCallback(() => setAttempt((count) => count + 1), [])

  useEffect(() => {
    if (!team) {
      setState("error")
      return
    }

    const controller = new AbortController()
    setState("loading")

    const load = async () => {
      try {
        const res = await fetch("/api/v1/config/setup-script", {
          headers: { "x-team-slug": team },
          signal: controller.signal,
        })
        if (!res.ok) throw new Error("setup command request failed")

        const data: unknown = await res.json()
        const command =
          typeof data === "object" &&
          data !== null &&
          "message" in data &&
          typeof data.message === "string"
            ? data.message.trim()
            : ""
        if (!command) throw new Error("setup command was empty")

        setScript(command)
        setState("ready")
      } catch {
        if (controller.signal.aborted) return
        setScript("")
        setState("error")
      }
    }

    void load()
    return () => controller.abort()
  }, [team, attempt])

  return { script, state, retry }
}

interface TeamOverview {
  connections: Connection[]
  totalConnections: number
  activeConnections: number
  teamMembers: number
  loading: boolean
}

const EMPTY: TeamOverview = {
  connections: [],
  totalConnections: 0,
  activeConnections: 0,
  teamMembers: 0,
  loading: true,
}

/** Recent routes plus the headline counts, in one pass. */
export function useTeamOverview(team?: string, recentLimit = 5) {
  const [overview, setOverview] = useState<TeamOverview>(EMPTY)

  useEffect(() => {
    if (!team) return

    const controller = new AbortController()
    const headers = { "x-team-slug": team }
    setOverview((current) => ({ ...current, loading: true }))

    const load = async () => {
      try {
        const [connectionsRes, statsRes] = await Promise.all([
          fetch(
            `/api/v1/connections/?type=recent&page=1&page_size=${recentLimit}`,
            { headers, signal: controller.signal },
          ),
          fetch("/api/v1/config/stats", {
            headers,
            signal: controller.signal,
          }),
        ])

        const connections = connectionsRes.ok
          ? await connectionsRes.json()
          : null
        const stats = statsRes.ok ? await statsRes.json() : null

        setOverview({
          connections: Array.isArray(connections?.data) ? connections.data : [],
          totalConnections:
            typeof connections?.count === "number" ? connections.count : 0,
          activeConnections: stats?.team_stats?.active_connections ?? 0,
          teamMembers: stats?.team_stats?.team_members ?? 0,
          loading: false,
        })
      } catch (error) {
        if (controller.signal.aborted) return
        console.error("Failed to load overview:", error)
        setOverview({ ...EMPTY, loading: false })
      }
    }

    void load()
    return () => controller.abort()
  }, [team, recentLimit])

  return overview
}
