// Authentication utilities for Gatus

/**
 * Get stored authentication credentials
 * @returns {Object|null} The stored credentials or null if not found
 */
export function getStoredAuth() {
    try {
        const stored = localStorage.getItem('gatus_auth')
        if (stored) {
            return JSON.parse(stored)
        }
    } catch (e) {
        localStorage.removeItem('gatus_auth')
    }
    return null
}

/**
 * Store authentication credentials
 * @param {string} username
 * @param {string} credentials - Base64 encoded credentials
 */
export function storeAuth(username, credentials) {
    localStorage.setItem('gatus_auth', JSON.stringify({
        username,
        credentials
    }))
}

/**
 * Clear stored authentication
 */
export function clearAuth() {
    localStorage.removeItem('gatus_auth')
}

/**
 * Create a fetch wrapper that automatically adds auth headers
 * @param {string} url
 * @param {Object} options
 * @returns {Promise<Response>}
 */
export async function authenticatedFetch(url, options = {}) {
    const auth = getStoredAuth()

    if (auth && auth.credentials) {
        options.headers = options.headers || {}
        options.headers['Authorization'] = `Basic ${auth.credentials}`
    }

    // Always include credentials for cookie-based auth (OIDC)
    options.credentials = options.credentials || 'include'

    return fetch(url, options)
}
