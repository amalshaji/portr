import type {
  ReservedSubdomain,
  ReservedSubdomainsResponse,
  SubdomainClaimStatus,
} from "@/types"

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}

function isClaimStatus(value: unknown): value is SubdomainClaimStatus {
  return value === "idle" || value === "starting" || value === "active"
}

function parseReservation(value: unknown): ReservedSubdomain {
  if (
    !isRecord(value) ||
    typeof value.subdomain !== "string" ||
    typeof value.created_at !== "string" ||
    !isClaimStatus(value.claim_status)
  ) {
    throw new Error("The server returned an invalid reservation")
  }

  return {
    subdomain: value.subdomain,
    created_at: value.created_at,
    claim_status: value.claim_status,
  }
}

function parseReservationList(value: unknown): ReservedSubdomainsResponse {
  if (
    !isRecord(value) ||
    !Array.isArray(value.data) ||
    typeof value.count !== "number" ||
    typeof value.limit !== "number" ||
    typeof value.base_domain !== "string"
  ) {
    throw new Error("The server returned an invalid reservation list")
  }

  return {
    data: value.data.map(parseReservation),
    count: value.count,
    limit: value.limit,
    base_domain: value.base_domain,
  }
}

async function errorMessage(response: Response, fallback: string) {
  const payload: unknown = await response.json().catch(() => null)
  if (isRecord(payload) && typeof payload.message === "string") {
    return payload.message
  }
  return fallback
}

export async function listReservedSubdomains(
  team: string,
  signal?: AbortSignal,
) {
  const response = await fetch("/api/v1/reserved-subdomains/", {
    headers: { "x-team-slug": team },
    signal,
  })
  if (!response.ok) {
    throw new Error("Failed to load reserved subdomains")
  }
  return parseReservationList(await response.json())
}

export async function reserveSubdomain(team: string, subdomain: string) {
  const response = await fetch("/api/v1/reserved-subdomains/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "x-team-slug": team,
    },
    body: JSON.stringify({ subdomain }),
  })
  if (!response.ok) {
    throw new Error(await errorMessage(response, "Subdomain could not be reserved"))
  }
  return parseReservation(await response.json())
}

export async function releaseSubdomain(team: string, subdomain: string) {
  const response = await fetch(
    `/api/v1/reserved-subdomains/${encodeURIComponent(subdomain)}`,
    {
      method: "DELETE",
      headers: { "x-team-slug": team },
    },
  )
  if (!response.ok) {
    throw new Error(await errorMessage(response, "Reservation could not be released"))
  }
}
