/** "3 hours" — a single coarse unit, for prose. */
export const humanizeTimeMs = (ms: number): string => {
  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  const months = Math.floor(days / 30)
  const years = Math.floor(months / 12)

  if (years > 0) {
    return `${years} years`
  }
  if (months > 0) {
    return `${months} months`
  }
  if (days > 0) {
    return `${days} days`
  }
  if (hours > 0) {
    return `${hours} hours`
  }
  if (minutes > 0) {
    return `${minutes} minutes`
  }
  if (seconds > 0) {
    return `${seconds} seconds`
  }
  return "0 seconds"
}

/**
 * "2d 18h 51m 40s" — every unit down to seconds, largest non-zero first.
 * Used for connection durations and server uptime, which are read as precise
 * measurements rather than prose.
 */
export const formatDuration = (ms: number): string => {
  const clamped = Math.max(0, ms)
  const seconds = Math.floor(clamped / 1000) % 60
  const minutes = Math.floor(clamped / (1000 * 60)) % 60
  const hours = Math.floor(clamped / (1000 * 60 * 60)) % 24
  const days = Math.floor(clamped / (1000 * 60 * 60 * 24))

  if (days > 0) return `${days}d ${hours}h ${minutes}m ${seconds}s`
  if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

/** "just now", "12m ago", "3h ago", "2d ago". */
export const relativeTime = (value: string | null, now: number = Date.now()) => {
  if (!value) return "—"
  const minutes = Math.floor((now - new Date(value).getTime()) / 60000)
  if (minutes < 1) return "just now"
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}
