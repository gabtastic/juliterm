import { resetAuth } from '~/middleware/auth'

/** Perform a full logout: call server, disconnect WS, clear auth, redirect. */
export function useLogout() {
  const socket = useTerminalSocket()

  async function logout() {
    try {
      await fetch('/api/logout', {
        method: 'POST',
        credentials: 'same-origin',
      })
    } catch {
      // Server may be unreachable — proceed with local cleanup anyway.
    }

    socket.disconnect()
    resetAuth()
    window.location.href = '/login'
  }

  return { logout }
}
