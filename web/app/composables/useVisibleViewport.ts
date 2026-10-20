import { onMounted, onBeforeUnmount } from 'vue'

// On iPadOS, the on-screen keyboard (or the shortcut bar shown with a
// hardware keyboard) covers the bottom of the page without resizing it, and
// Safari scrolls the whole page up to keep the focused field in view. That
// dragged the key bar up over the terminal. Instead, size the app to the
// visible area above the keyboard (the visual viewport) through
// --app-height, and undo any page scroll so the layout stays put.

function sync() {
  const vv = window.visualViewport
  if (vv) {
    document.documentElement.style.setProperty('--app-height', `${Math.round(vv.height)}px`)
  }
  if (window.scrollX !== 0 || window.scrollY !== 0) window.scrollTo(0, 0)
}

export function useVisibleViewport() {
  onMounted(() => {
    sync()
    window.visualViewport?.addEventListener('resize', sync)
    window.visualViewport?.addEventListener('scroll', sync)
    window.addEventListener('scroll', sync, { passive: true })
  })
  onBeforeUnmount(() => {
    window.visualViewport?.removeEventListener('resize', sync)
    window.visualViewport?.removeEventListener('scroll', sync)
    window.removeEventListener('scroll', sync)
    document.documentElement.style.removeProperty('--app-height')
  })
}
