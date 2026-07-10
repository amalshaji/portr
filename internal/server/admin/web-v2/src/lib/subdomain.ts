const subdomainPattern = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/

export function normalizeSubdomain(value: string): string {
  return value.trim().toLowerCase()
}

export function validateSubdomain(value: string): string | null {
  const normalized = normalizeSubdomain(value)
  if (!normalized) return "Enter a subdomain"
  if (!subdomainPattern.test(normalized)) {
    return "Use 1–63 letters, numbers, or internal hyphens"
  }
  return null
}
