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
  return getStateColor(result.state)
}

/**
 * Check if a given theme exists in the configuration.
 *
 * @param {string} theme - The name of the theme to check.
 * @returns {boolean} - True if the theme exists, false otherwise.
 */
export const themeExists = (theme) => {
  var themes = window.config?.themes || {}
  return Object.hasOwn(themes, theme)
}

/**
 * Get the color associated with a given state.
 * If the state is not found in the configuration, return the 'unknown' color.
 *
 * @param {string} state - The state for which to get the color.
 * @returns {string} - The color associated with the given state.
 */
export const getStateColor = (state) => {
  var theme = localStorage.getItem('gatus:theme') || 'default'
  themeExists(theme) || (theme = 'default')
  return window.config?.themes[theme]['stateColors'][state] ?? window.config?.localStateColors.invalid
}

/**
 * Get the list of available themes from the configuration.
 *
 * @returns {Array<string>} - An array of available theme names.
 */
export const getAvailableThemes = () => {
  var themes = window.config?.themes || {}
  return Object.keys(themes)
}