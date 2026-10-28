/**
 * Get the color associated with a given result and its state.
 * If the result is null or undefined, return the 'nodata' color.
 * Otherwise, return the color corresponding to the state's value.
 *
 * @param {Object} result - The result object containing state information.
 * @returns {string} - The color associated with the result's state.
 */
export const getResultColor = (result) => {
  if (!result) return window.config?.localStateColors.unkown
  if (!result.state)
    result.state = result.success ? 'healthy' : 'unhealthy'
  return getStateColor(result.state)
}

/**
 * Get the color associated with a given state.
 * If the state is not found in the configuration, return the 'unknown' color.
 *
 * @param {string} state - The state for which to get the color.
 * @returns {string} - The color associated with the given state.
 */
export const getStateColor = (state) => {
  return window.config?.stateColors[state] ?? window.config?.localStateColors.invalid
}