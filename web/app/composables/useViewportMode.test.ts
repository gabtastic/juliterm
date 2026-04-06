import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { useViewportMode } from './useViewportMode'

// Helper: create a test component that uses the composable
function createWrapper(breakpoint?: number) {
  const TestComponent = defineComponent({
    setup() {
      const { isMobile } = useViewportMode(breakpoint)
      return { isMobile }
    },
    render() {
      return h('div', { class: this.isMobile ? 'mobile' : 'desktop' })
    },
  })
  return mount(TestComponent)
}

describe('useViewportMode', () => {
  let listeners: Array<(e: MediaQueryListEvent) => void>
  let matchesValue: boolean

  beforeEach(() => {
    listeners = []
    matchesValue = false

    vi.stubGlobal('matchMedia', vi.fn((query: string) => ({
      matches: matchesValue,
      media: query,
      addEventListener: (_event: string, cb: (e: MediaQueryListEvent) => void) => {
        listeners.push(cb)
      },
      removeEventListener: (_event: string, cb: (e: MediaQueryListEvent) => void) => {
        listeners = listeners.filter(l => l !== cb)
      },
    })))
  })

  it('returns false (desktop) when viewport is wide', () => {
    matchesValue = false
    const wrapper = createWrapper()
    expect(wrapper.classes()).toContain('desktop')
  })

  it('returns true (mobile) when viewport is narrow', () => {
    matchesValue = true
    const wrapper = createWrapper()
    expect(wrapper.classes()).toContain('mobile')
  })

  it('reacts to viewport changes', async () => {
    matchesValue = false
    const wrapper = createWrapper()
    expect(wrapper.classes()).toContain('desktop')

    // Simulate viewport shrink
    listeners.forEach(cb => cb({ matches: true } as MediaQueryListEvent))
    await wrapper.vm.$nextTick()
    expect(wrapper.classes()).toContain('mobile')

    // Simulate viewport expand
    listeners.forEach(cb => cb({ matches: false } as MediaQueryListEvent))
    await wrapper.vm.$nextTick()
    expect(wrapper.classes()).toContain('desktop')
  })

  it('cleans up listener on unmount', () => {
    matchesValue = false
    const wrapper = createWrapper()
    expect(listeners.length).toBe(1)

    wrapper.unmount()
    expect(listeners.length).toBe(0)
  })

  it('uses custom breakpoint', () => {
    matchesValue = false
    createWrapper(480)
    expect(window.matchMedia).toHaveBeenCalledWith('(max-width: 480px)')
  })
})
