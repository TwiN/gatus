export const DASHBOARD_FILTER_VALUES = Object.freeze(['none', 'failing', 'unstable'])
export const DASHBOARD_SORT_VALUES = Object.freeze(['name', 'group', 'health'])

export const getFirstQueryValue = (value) => {
  if (Array.isArray(value)) {
    return value.find(item => typeof item === 'string')
  }
  return typeof value === 'string' ? value : undefined
}

export const normalizeSearchQuery = (value) => (getFirstQueryValue(value) || '').trim()

export const normalizeDashboardOption = (value, allowedValues, fallback) => {
  const normalizedValue = getFirstQueryValue(value)
  return allowedValues.includes(normalizedValue) ? normalizedValue : fallback
}

export const withQueryValue = (currentQuery, key, value) => {
  const query = { ...currentQuery }
  if (value !== '' && value !== null && value !== undefined) {
    query[key] = value
  } else {
    delete query[key]
  }
  return query
}
