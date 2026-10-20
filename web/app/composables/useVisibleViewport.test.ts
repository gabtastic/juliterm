import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { useVisibleViewport } from './useVisibleViewport'

class FakeVisualViewport extends EventTarget {
  height = 900
}

function mountWithViewport() {
  return mount(defineComponent({
    setup() { useVisibleViewport() },
    render: () => h('div'),
  }))
}

describe('useVisibleViewport', () => {
  let vv: FakeVisualViewport

  beforeEach(() => {
    vv = new FakeVisualViewport()
    vi.stubGlobal('visualViewport', vv)
    vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  const appHeight = () => document.documentElement.style.getPropertyValue('--app-height')

  it('sizes the app to the visible area and follows it', () => {
    const wrapper = mountWithViewport()
    expect(appHeight()).toBe('900px')

    vv.height = 612.4 // keyboard shown
    vv.dispatchEvent(new Event('resize'))
    expect(appHeight()).toBe('612px')

    wrapper.unmount()
    expect(appHeight()).toBe('')
  })

  it('undoes page scroll caused by the keyboard', () => {
    mountWithViewport()
    Object.defineProperty(window, 'scrollY', { value: 240, configurable: true })
    vv.dispatchEvent(new Event('scroll'))
    expect(window.scrollTo).toHaveBeenCalledWith(0, 0)
    Object.defineProperty(window, 'scrollY', { value: 0, configurable: true })
  })
})
