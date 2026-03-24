/**
 * Formats a duration from nanoseconds to a human-readable string
 * @param {number} duration - Duration in nanoseconds
 * @returns {string} Formatted duration string (e.g., "123ms", "1.23s")
 */
export const formatDuration = (duration) => {
  if (!duration && duration !== 0) return 'N/A'
  
  // Convert nanoseconds to milliseconds
  const durationMs = duration / 1000000
  
  if (durationMs < 1000) {
    return `${Math.trunc(durationMs)}ms`
  } else {
    return `${(durationMs / 1000).toFixed(2)}s`
  }
}