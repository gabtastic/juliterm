import { ref } from 'vue'

// This machine's name (e.g. "macstudio36"), so several juli instances can
// be told apart: in the tab title, the Home Screen shortcut's name and the
// sidebar. The server also serves a matching manifest and icon.
const name = ref<string | null>(null)
let loading: Promise<void> | null = null

function apply(n: string) {
  name.value = n
  document.title = `${n} · Juli`
  // iPadOS names a Home Screen shortcut from this when adding it.
  let meta = document.querySelector<HTMLMetaElement>('meta[name="apple-mobile-web-app-title"]')
  if (!meta) {
    meta = document.createElement('meta')
    meta.name = 'apple-mobile-web-app-title'
    document.head.appendChild(meta)
  }
  meta.content = n
}

export function useIdentity() {
  if (!loading && typeof window !== 'undefined') {
    loading = fetch('/api/identity')
      .then(r => (r.ok ? r.json() : null))
      .then((id: { name?: string } | null) => { if (id?.name) apply(id.name) })
      .catch(() => {})
  }
  return { name }
}
