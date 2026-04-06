// Binary WebSocket composable for the Juli terminal protocol (PTY-based).
// Singleton — module-scoped state shared across all components (safe with ssr: false).
//
// Message type bytes must match Go protocol/message.go exactly.

import { ref, readonly } from 'vue'
import { resetAuth } from '~/middleware/auth'

const MSG = {
  OUTPUT: 0x01,       // server→client: raw PTY output
  INPUT: 0x02,        // client→server: raw keyboard input
  RESIZE: 0x03,       // client→server: JSON {cols, rows}
  SESSION_TREE: 0x04,   // bidirectional: JSON session tree / session select
  SELECT_WINDOW: 0x05,  // client→server: JSON {sessionName, windowIndex}
  ERROR: 0x07,          // server→client: JSON error
} as const

type Callback<T> = (data: T) => void

interface ErrorPayload {
  code: string
  message: string
}

// ---- Module-scoped singleton state ----

const connected = ref(false)
const reconnecting = ref(false)
const error = ref<string | null>(null)

let ws: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectDelay = 500
let intentionallyClosed = false

// Use Set for O(1) add/delete, prevents duplicates
const onOutputCbs = new Set<Callback<Uint8Array>>()
const onSessionTreeCbs = new Set<Callback<unknown[]>>()
const onErrorCbs = new Set<Callback<ErrorPayload>>()

// Buffer last session tree so late subscribers get it immediately
let lastSessionTree: unknown[] | null = null

// ---- Internal helpers ----

function getWsUrl(): string {
  const loc = window.location
  const proto = loc.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${loc.host}/ws`
}

function connect() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return
  }

  intentionallyClosed = false
  error.value = null

  try {
    ws = new WebSocket(getWsUrl())
    ws.binaryType = 'arraybuffer'
  } catch (e) {
    error.value = `Failed to create WebSocket: ${e}`
    scheduleReconnect()
    return
  }

  ws.onopen = () => {
    connected.value = true
    reconnecting.value = false
    reconnectDelay = 500
    error.value = null
  }

  ws.onclose = (event: CloseEvent) => {
    connected.value = false
    ws = null

    // Close code 4001 = session revoked by server (logout or expiry).
    // Stop reconnecting and redirect to login.
    if (event.code === 4001) {
      intentionallyClosed = true
      resetAuth()
      window.location.href = '/login'
      return
    }

    if (!intentionallyClosed) {
      scheduleReconnect()
    }
  }

  ws.onerror = () => {
    error.value = 'WebSocket error'
  }

  ws.onmessage = (event: MessageEvent) => {
    if (!(event.data instanceof ArrayBuffer)) return
    const buf = new Uint8Array(event.data)
    if (buf.length === 0) return

    const msgType = buf[0]
    const payload = buf.slice(1)

    switch (msgType) {
      case MSG.OUTPUT:
        for (const cb of onOutputCbs) cb(payload)
        break
      case MSG.SESSION_TREE:
        try {
          const parsed = JSON.parse(new TextDecoder().decode(payload)) as unknown[]
          lastSessionTree = parsed
          for (const cb of onSessionTreeCbs) cb(parsed)
        } catch { /* ignore malformed JSON */ }
        break
      case MSG.ERROR:
        try {
          const parsed = JSON.parse(new TextDecoder().decode(payload)) as ErrorPayload
          for (const cb of onErrorCbs) cb(parsed)
        } catch { /* ignore malformed JSON */ }
        break
    }
  }
}

function scheduleReconnect() {
  if (intentionallyClosed) return
  if (reconnectTimer) return

  reconnecting.value = true
  const jitter = Math.random() * reconnectDelay * 0.25
  const delay = reconnectDelay + jitter

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    reconnectDelay = Math.min(reconnectDelay * 2, 30000)
    connect()
  }, delay)
}

function send(data: Uint8Array) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(data)
  }
}

function buildJsonFrame(msgType: number, payload: object): Uint8Array {
  const json = new TextEncoder().encode(JSON.stringify(payload))
  const frame = new Uint8Array(1 + json.length)
  frame[0] = msgType
  frame.set(json, 1)
  return frame
}

function disconnect() {
  intentionallyClosed = true
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (ws) {
    ws.close()
    ws = null
  }
  connected.value = false
  reconnecting.value = false
}

// ---- Public composable ----

export function useTerminalSocket() {
  // Don't auto-connect here — let the consumer call connect() after
  // auth is confirmed. The auth middleware validates the cookie first,
  // and only then should the WebSocket be opened.

  function selectSession(sessionName: string) {
    send(buildJsonFrame(MSG.SESSION_TREE, { sessionName }))
  }

  function selectWindow(sessionName: string, windowIndex: number) {
    send(buildJsonFrame(MSG.SELECT_WINDOW, { sessionName, windowIndex }))
  }

  function sendInput(data: Uint8Array) {
    const frame = new Uint8Array(1 + data.length)
    frame[0] = MSG.INPUT
    frame.set(data, 1)
    send(frame)
  }

  function sendResize(cols: number, rows: number) {
    send(buildJsonFrame(MSG.RESIZE, { cols, rows }))
  }

  /** Subscribe to PTY output. Returns unsubscribe function. */
  function onOutput(cb: Callback<Uint8Array>): () => void {
    onOutputCbs.add(cb)
    return () => { onOutputCbs.delete(cb) }
  }

  /** Subscribe to session tree updates. Replays last value immediately. Returns unsubscribe function. */
  function onSessionTree(cb: Callback<unknown[]>): () => void {
    onSessionTreeCbs.add(cb)
    if (lastSessionTree !== null) {
      cb(lastSessionTree)
    }
    return () => { onSessionTreeCbs.delete(cb) }
  }

  /** Subscribe to server errors. Returns unsubscribe function. */
  function onError(cb: Callback<ErrorPayload>): () => void {
    onErrorCbs.add(cb)
    return () => { onErrorCbs.delete(cb) }
  }

  return {
    connected: readonly(connected),
    reconnecting: readonly(reconnecting),
    error: readonly(error),
    selectSession,
    selectWindow,
    sendInput,
    sendResize,
    disconnect,
    connect,
    onOutput,
    onSessionTree,
    onError,
  }
}

export type { ErrorPayload }
