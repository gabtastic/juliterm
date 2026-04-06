import { mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import SplitDropdown from './SplitDropdown.vue'

const mockSplitWindow = vi.fn()
const mockDetectTestCommand = vi.fn()

const mockGetPresets = vi.fn().mockResolvedValue({ global: [], project: [] })

vi.mock('~/composables/useSessionApi', () => ({
  useSessionApi: () => ({
    splitWindow: mockSplitWindow,
    detectTestCommand: mockDetectTestCommand,
    getPresets: mockGetPresets,
    createSession: vi.fn(),
    renameSession: vi.fn(),
    killSession: vi.fn(),
    createWindow: vi.fn(),
    renameWindow: vi.fn(),
    killWindow: vi.fn(),
  }),
}))

// Stub onClickOutside since we're in happy-dom.
vi.mock('@vueuse/core', () => ({
  onClickOutside: vi.fn(),
}))

function mountDropdown() {
  return mount(SplitDropdown, {
    props: { sessionName: 'dev' },
  })
}

describe('SplitDropdown', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockDetectTestCommand.mockResolvedValue(null)
    mockGetPresets.mockResolvedValue({ global: [], project: [] })
  })

  it('renders the split trigger button', () => {
    const wrapper = mountDropdown()
    expect(wrapper.find('.split-trigger').exists()).toBe(true)
    expect(wrapper.find('.split-trigger').text()).toContain('Split')
  })

  it('opens dropdown on click', async () => {
    const wrapper = mountDropdown()
    expect(wrapper.find('.split-menu').exists()).toBe(false)

    await wrapper.find('.split-trigger').trigger('click')

    expect(wrapper.find('.split-menu').exists()).toBe(true)
  })

  it('closes dropdown on second click', async () => {
    const wrapper = mountDropdown()
    await wrapper.find('.split-trigger').trigger('click')
    expect(wrapper.find('.split-menu').exists()).toBe(true)

    await wrapper.find('.split-trigger').trigger('click')
    expect(wrapper.find('.split-menu').exists()).toBe(false)
  })

  it('closes dropdown on Escape key', async () => {
    const wrapper = mountDropdown()
    await wrapper.find('.split-trigger').trigger('click')
    expect(wrapper.find('.split-menu').exists()).toBe(true)

    await wrapper.find('.split-dropdown').trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('.split-menu').exists()).toBe(false)
  })

  it('calls splitWindow with lazygit preset', async () => {
    const wrapper = mountDropdown()
    await wrapper.find('.split-trigger').trigger('click')

    const items = wrapper.findAll('.split-item')
    await items[0].trigger('click')

    expect(mockSplitWindow).toHaveBeenCalledWith('dev', 'lazygit', undefined)
  })

  it('calls splitWindow with test-runner preset', async () => {
    const wrapper = mountDropdown()
    await wrapper.find('.split-trigger').trigger('click')

    const items = wrapper.findAll('.split-item')
    await items[1].trigger('click')

    expect(mockSplitWindow).toHaveBeenCalledWith('dev', 'test-runner', undefined)
  })

  it('shows detected test command in label', async () => {
    mockDetectTestCommand.mockResolvedValue('go test ./...')
    const wrapper = mountDropdown()

    await wrapper.find('.split-trigger').trigger('click')
    // Wait for the async detectTestCommand call.
    await vi.waitFor(() => {
      expect(wrapper.find('.test-cmd').exists()).toBe(true)
    })

    expect(wrapper.find('.test-cmd').text()).toContain('go test ./...')
  })

  it('shows no suffix when no test command detected', async () => {
    mockDetectTestCommand.mockResolvedValue(null)
    const wrapper = mountDropdown()

    await wrapper.find('.split-trigger').trigger('click')
    // Wait a tick to let the async call resolve.
    await vi.waitFor(() => {
      expect(wrapper.find('.split-menu').exists()).toBe(true)
    })

    expect(wrapper.find('.test-cmd').exists()).toBe(false)
  })

  it('shows custom input when Custom... is clicked', async () => {
    const wrapper = mountDropdown()
    await wrapper.find('.split-trigger').trigger('click')

    const items = wrapper.findAll('.split-item')
    // Custom... is the third item.
    await items[2].trigger('click')

    expect(wrapper.find('.custom-input').exists()).toBe(true)
  })

  it('submits custom command on Enter', async () => {
    const wrapper = mountDropdown()
    await wrapper.find('.split-trigger').trigger('click')

    const items = wrapper.findAll('.split-item')
    await items[2].trigger('click')

    const input = wrapper.find('.custom-input')
    await input.setValue('htop')
    await input.trigger('keydown.enter')

    expect(mockSplitWindow).toHaveBeenCalledWith('dev', 'custom', 'htop')
  })

  it('submits empty custom command on Enter', async () => {
    const wrapper = mountDropdown()
    await wrapper.find('.split-trigger').trigger('click')

    const items = wrapper.findAll('.split-item')
    await items[2].trigger('click')

    await wrapper.find('.custom-input').trigger('keydown.enter')

    expect(mockSplitWindow).toHaveBeenCalledWith('dev', 'custom', '')
  })

  it('emits error when API call fails', async () => {
    mockSplitWindow.mockRejectedValue(new Error('pane too small'))
    const wrapper = mountDropdown()

    await wrapper.find('.split-trigger').trigger('click')
    const items = wrapper.findAll('.split-item')
    await items[0].trigger('click')

    // Wait for async error propagation.
    await vi.waitFor(() => {
      expect(wrapper.emitted('error')).toBeTruthy()
    })

    expect(wrapper.emitted('error')![0]).toEqual(['pane too small'])
  })
})
