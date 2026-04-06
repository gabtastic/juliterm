import { ref, readonly, onMounted, onUnmounted } from 'vue'

/**
 * Reactive viewport mode detection via matchMedia.
 * Returns `isMobile: true` when viewport width <= breakpoint.
 * Client-only — defaults to false (desktop) during SSR/SSG.
 */
export function useViewportMode(breakpoint = 640) {
  const isMobile = ref(false)

  if (typeof window !== 'undefined') {
    const mql = window.matchMedia(`(max-width: ${breakpoint}px)`)
    isMobile.value = mql.matches

    const handler = (e: MediaQueryListEvent) => {
      isMobile.value = e.matches
    }

    onMounted(() => mql.addEventListener('change', handler))
    onUnmounted(() => mql.removeEventListener('change', handler))
  }

  return { isMobile: readonly(isMobile) }
}
