/**
 * Formats a duration from nanoseconds to a human-readable string
 * @param {number} duration - Duration in nanoseconds
 * @returns {string} Formatted duration string (e.g., "123ms", "1.23s", "4m 5s", "1h 12m")
 */
export const formatDuration = (duration) => {
  if (!duration && duration !== 0) return 'N/A'

  // Convert nanoseconds to milliseconds
  const durationMs = duration / 1000000

  if (durationMs < 1000) {
    return `${Math.trunc(durationMs)}ms`
  }
  if (durationMs < 60000) {
    // Truncate (rather than round) to hundredths so e.g. 59.999s doesn't render as "60.00s"
    return `${(Math.trunc(duration / 10000000) / 100).toFixed(2)}s`
  }
  const totalSeconds = Math.trunc(durationMs / 1000)
  const hours = Math.trunc(totalSeconds / 3600)
  const minutes = Math.trunc((totalSeconds % 3600) / 60)
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m ${totalSeconds % 60}s`
}
