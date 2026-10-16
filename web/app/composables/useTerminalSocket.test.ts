import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'

vi.mock('~/middleware/auth', () => ({ resetAuth: vi.fn() }))

// Minimal stand-in for the browser WebSocket: records sent frames and lets
// tests drive open/message/close.
class FakeWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3
  static instances: FakeWebSocket[] = []

  readyState = FakeWebSocket.CONNECTING
  binaryType = 'blob'
  sent: Uint8Array[] = []
  onopen: (() => void) | null = null
  onclose: ((e: { code: number }) => void) | null = null
  onerror: (() => void) | null = null
  onmessage: ((e: { data: ArrayBuffer }) => void) | null = null

  constructor(public url: string) {
    FakeWebSocket.instances.push(this)
  }

  send(data: Uint8Array) { this.sent.push(data) }
  close() {
    this.readyState = FakeWebSocket.CLOSED
    this.onclose?.({ code: 1000 })
  }

  open() {
    this.readyState = FakeWebSocket.OPEN
    this.onopen?.()
  }

  drop(code = 1006) {
    this.readyState = FakeWebSocket.CLOSED
    this.onclose?.({ code })
  }

  receive(type: number, payload: Uint8Array | string = new Uint8Array()) {
    const body = typeof payload === 'string' ? new TextEncoder().encode(payload) : payload
    const frame = new Uint8Array(1 + body.length)
    frame[0] = type
    frame.set(body, 1)
    this.onmessage?.({ data: frame.buffer })
  }

  /** Sent frames decoded as [type, JSON payload or bytes]. */
  frames(): Array<[number, unknown]> {
    return this.sent.map((f) => {
      const body = f.slice(1)
      try {
        return [f[0], JSON.parse(new TextDecoder().decode(body))]
      }
      catch {
        return [f[0], body]
      }
    })
  }
}

const SELECT = 0x04
const RESIZE = 0x03
const SELECT_WINDOW = 0x05
const ERROR = 0x07
const HEARTBEAT = 0x08

// The composable is a module-scoped singleton: load a fresh copy per test.
async function loadSocket() {
  vi.resetModules()
  const mod = await import('./useTerminalSocket')
  return mod.useTerminalSocket()
}

function latest(): FakeWebSocket {
  const ws = FakeWebSocket.instances.at(-1)
  if (!ws) throw new Error('no WebSocket created')
  return ws
}

describe('useTerminalSocket', () => {
  beforeEach(() => {
    FakeWebSocket.instances = []
    vi.useFakeTimers()
    vi.stubGlobal('WebSocket', FakeWebSocket)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('sends the select with its size when open', async () => {
    const socket = await loadSocket()
    socket.connect()
    latest().open()

    socket.selectSession('dev', { cols: 120, rows: 40 })
    expect(latest().frames()).toEqual([[SELECT, { sessionName: 'dev', cols: 120, rows: 40 }]])
  })

  it('sends a select made while connecting once the socket opens', async () => {
    const socket = await loadSocket()
    socket.connect()

    socket.selectSession('dev', { cols: 100, rows: 30 })
    expect(latest().sent).toHaveLength(0)

    latest().open()
    expect(latest().frames()).toEqual([
      [SELECT, { sessionName: 'dev', cols: 100, rows: 30 }],
      [RESIZE, { cols: 100, rows: 30 }],
    ])
  })

  it('re-attaches the session after a reconnect and notifies subscribers', async () => {
    const socket = await loadSocket()
    const reconnects = vi.fn()
    socket.onReconnect(reconnects)

    socket.connect()
    latest().open()
    socket.selectSession('dev')
    socket.sendResize(90, 25)

    latest().drop()
    expect(socket.connected.value).toBe(false)
    vi.advanceTimersByTime(1000)
    const second = latest()
    expect(FakeWebSocket.instances).toHaveLength(2)

    second.open()
    expect(reconnects).toHaveBeenCalledTimes(1)
    expect(second.frames()).toEqual([
      [SELECT, { sessionName: 'dev', cols: 90, rows: 25 }],
      [RESIZE, { cols: 90, rows: 25 }],
    ])
  })

  it('does not re-attach after the session was cleared', async () => {
    const socket = await loadSocket()
    socket.connect()
    latest().open()
    socket.selectSession('dev')
    socket.clearSession()

    latest().drop()
    vi.advanceTimersByTime(1000)
    latest().open()
    expect(latest().sent).toHaveLength(0)
  })

  it('queues window selection until the socket opens', async () => {
    const socket = await loadSocket()
    socket.connect()
    socket.selectWindow('dev', 2)
    latest().open()
    expect(latest().frames()).toEqual([[SELECT_WINDOW, { sessionName: 'dev', windowIndex: 2 }]])
  })

  it('re-selects the session when the server reports not_attached', async () => {
    const socket = await loadSocket()
    const reconnects = vi.fn()
    socket.onReconnect(reconnects)
    socket.connect()
    latest().open()
    socket.selectSession('dev')
    latest().sent = []

    latest().receive(ERROR, JSON.stringify({ code: 'not_attached', message: 'no session attached' }))
    expect(reconnects).toHaveBeenCalledTimes(1)
    expect(latest().frames()).toEqual([[SELECT, { sessionName: 'dev' }]])
  })

  it('treats a heartbeat as a no-op frame', async () => {
    const socket = await loadSocket()
    const output = vi.fn()
    socket.onOutput(output)
    socket.connect()
    latest().open()
    latest().receive(HEARTBEAT)
    expect(output).not.toHaveBeenCalled()
  })
})
