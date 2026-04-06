// Route middleware: check authentication before navigating to protected pages.
// Makes a HEAD-like GET request to /api/sessions; if 401, redirect to /login.
// The auth check result is cached for the lifetime of the SPA session.

let authChecked = false
let authValid = false

/** Reset cached auth state. Call on logout so subsequent navigations re-check. */
export function resetAuth() {
  authChecked = false
  authValid = false
}

export default defineNuxtRouteMiddleware(async (to) => {
  if (to.path === '/login') return

  // Use cached result if already checked
  if (authChecked && authValid) return

  try {
    const res = await fetch('/api/sessions', {
      method: 'GET',
      credentials: 'same-origin',
    })

    if (res.status === 401) {
      authChecked = true
      authValid = false
      return navigateTo('/login')
    }

    // Any non-401 response means we're authenticated (even 500, the server is
    // there and accepted our cookie).
    authChecked = true
    authValid = true
  } catch {
    // Network error — server might be down. Let the page load; the WebSocket
    // composable will show the connection error state.
    authChecked = false
    authValid = false
  }
})
