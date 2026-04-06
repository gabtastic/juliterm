import { ref, onUnmounted } from 'vue'

/**
 * Formats a unix epoch (seconds) as a relative time string.
 * Returns empty string for falsy/zero input.
 */
export function formatTimeAgo(epochSeconds: number | undefined): string {
  if (!epochSeconds) return ''

  const now = Math.floor(Date.now() / 1000)
  const diff = now - epochSeconds

  if (diff < 0) return 'now'
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}

/**
 * Reactive composable that triggers re-renders every `intervalMs` to keep
 * relative timestamps up to date.
 */
export function useTimeAgo(intervalMs = 30000) {
  const tick = ref(0)

  const timer = import.meta.client
    ? setInterval(() => {
        tick.value++
      }, intervalMs)
    : null

  onUnmounted(() => {
    if (timer !== null) clearInterval(timer)
  })

  return { tick, formatTimeAgo }
}
