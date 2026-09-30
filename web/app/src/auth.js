export function getStoredAuth() {
    try {
        const stored = sessionStorage.getItem('gatus_auth')
        if (stored) {
            return JSON.parse(stored)
        }
    } catch (e) {
        sessionStorage.removeItem('gatus_auth')
    }
    return null
}

export function storeAuth(username, credentials) {
    sessionStorage.setItem('gatus_auth', JSON.stringify({
        username,
        credentials
    }))
}

export function clearAuth() {
    sessionStorage.removeItem('gatus_auth')
}

export async function authenticatedFetch(url, options = {}) {
    const auth = getStoredAuth()

    if (auth && auth.credentials) {
        options.headers = options.headers || {}
        options.headers['Authorization'] = `Basic ${auth.credentials}`
    }

    options.credentials = options.credentials || 'include'

    return fetch(url, options)
}
