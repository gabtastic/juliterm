import type { ActivityState, ToolType } from '~/composables/useSessionTree'

interface SessionSnapshot {
  activityState: ActivityState
  toolType: ToolType
}

type NavigateCallback = (sessionName: string) => void

const AGENT_TOOLS: ToolType[] = ['claude', 'codex', 'opencode', 'aider', 'cursor']

const previousStates = new Map<string, SessionSnapshot>()
let navigateCb: NavigateCallback | null = null

function isAgent(toolType?: ToolType): boolean {
  return !!toolType && AGENT_TOOLS.includes(toolType)
}

function notificationTitle(sessionName: string, kind: 'completed' | 'failed' | 'input'): string {
  switch (kind) {
    case 'completed': return `Juli — ${sessionName} finished`
    case 'failed': return `Juli — ${sessionName} failed`
    case 'input': return `Juli — ${sessionName} needs input`
  }
}

function detectTransition(
  prev: ActivityState | undefined,
  curr: ActivityState | undefined,
): 'completed' | 'failed' | 'input' | null {
  if (!curr) return null
  if (curr === 'needs_attention' && prev !== 'needs_attention') return 'failed'
  if (curr === 'waiting' && prev !== 'waiting') return 'input'
  if (curr === 'idle' && (prev === 'coding' || prev === 'executing')) return 'completed'
  return null
}

function showNotification(sessionName: string, sessionId: string, kind: 'completed' | 'failed' | 'input', body?: string) {
  if (typeof Notification === 'undefined') return
  if (Notification.permission !== 'granted') return

  const n = new Notification(notificationTitle(sessionName, kind), {
    body: body || undefined,
    tag: `juli-${sessionId}`,
  })

  n.onclick = () => {
    window.focus()
    navigateCb?.(sessionName)
    n.close()
  }
}

export function useNotifications() {
  function requestPermission() {
    if (typeof Notification === 'undefined') return
    if (Notification.permission === 'default') {
      Notification.requestPermission()
    }
  }

  function processSessionUpdate(
    data: unknown[],
    selectedSessionName: string | null,
    onNavigate: NavigateCallback,
  ) {
    if (!Array.isArray(data)) return
    navigateCb = onNavigate

    const liveIds = new Set<string>()

    for (const raw of data as any[]) {
      const id: string = raw.id ?? ''
      const name: string = raw.name ?? ''
      const toolType: ToolType | undefined = raw.toolType
      const activityState: ActivityState | undefined = raw.activityState
      const activitySummary: string | undefined = raw.activitySummary

      liveIds.add(id)

      if (!isAgent(toolType)) continue
      if (name === selectedSessionName && !document.hidden) continue

      const prev = previousStates.get(id)

      if (!prev) {
        previousStates.set(id, { activityState: activityState!, toolType: toolType! })
        continue
      }

      const kind = detectTransition(prev.activityState, activityState)
      if (kind) {
        showNotification(name, id, kind, activitySummary)
      }

      previousStates.set(id, { activityState: activityState!, toolType: toolType! })
    }

    // Prune stale entries for sessions no longer in the tree.
    for (const id of previousStates.keys()) {
      if (!liveIds.has(id)) previousStates.delete(id)
    }
  }

  return {
    requestPermission,
    processSessionUpdate,
  }
}
