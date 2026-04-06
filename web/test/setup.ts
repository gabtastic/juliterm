import { afterEach, vi } from 'vitest'

vi.stubGlobal('ResizeObserver', class {
  observe() {}
  unobserve() {}
  disconnect() {}
})

afterEach(() => {
  vi.restoreAllMocks()
})
