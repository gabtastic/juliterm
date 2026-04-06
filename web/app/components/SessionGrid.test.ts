import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { Session } from '~/composables/useSessionTree'
import SessionGrid from './SessionGrid.vue'

vi.mock('~/composables/useTimeAgo', () => ({
  useTimeAgo: () => ({
    tick: 0,
    formatTimeAgo: (timestamp: number) => `at ${timestamp}`,
  }),
}))

vi.mock('~/composables/useSessionApi', () => ({
  useSessionApi: () => ({
    createSession: vi.fn(),
    renameSession: vi.fn(),
    killSession: vi.fn(),
  }),
}))

vi.mock('~/composables/useLogout', () => ({
  useLogout: () => ({
    logout: vi.fn(),
  }),
}))

type SessionWithMonitoring = Session & {
  activityState?: string
  activitySummary?: string
}

const baseSession: Session = {
  id: 'session-1',
  name: 'agent-monitoring',
  windows: [{ id: 'window-1', name: 'editor', index: 0, active: true, panes: [] }],
  status: 'running',
  lastActivity: 1711645200,
  toolType: 'codex',
  gitBranch: 'feat/agent-monitoring',
  gitRepo: 'cihan/juli',
  workingDir: '/repo',
}

function mountGrid(sessions: SessionWithMonitoring[]) {
  return mount(SessionGrid, {
    props: {
      sessions: sessions as Session[],
    },
    global: {
      stubs: {
        ConfirmDialog: true,
      },
    },
  })
}

describe('SessionGrid monitoring', () => {
  it('renders activity state and summary', () => {
    const wrapper = mountGrid([{
      ...baseSession,
      activityState: 'executing',
      activitySummary: 'Running Go tests in internal/server',
    }])

    expect(wrapper.find('.activity-badge').text()).toBe('Running')
    expect(wrapper.text()).toContain('Running Go tests in internal/server')
  })

  it('highlights needs_attention sessions', () => {
    const wrapper = mountGrid([{
      ...baseSession,
      activityState: 'needs_attention',
      activitySummary: 'Build failed in web app',
    }])

    expect(wrapper.find('.session-card--attention').exists()).toBe(true)
  })

  it('omits monitoring UI when fields are absent', () => {
    const wrapper = mountGrid([baseSession])

    expect(wrapper.find('.activity-badge').exists()).toBe(false)
    expect(wrapper.find('.activity-summary').exists()).toBe(false)
  })
})
