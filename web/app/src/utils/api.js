const normalizePath = (path) => {
  if (!path) {
    return ''
  }
  return path.startsWith('/') ? path : `/${path}`
}

// Remote keys contain characters such as @ : that must be encoded in URL paths.
export const endpointApiUrl = (key, subPath) =>
  `/api/v1/endpoints/${encodeURIComponent(key)}${normalizePath(subPath)}`
