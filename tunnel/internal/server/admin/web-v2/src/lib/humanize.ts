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
