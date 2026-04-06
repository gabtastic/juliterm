<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

import { useTerminalSocket } from '~/composables/useTerminalSocket'

const props = defineProps<{
  sessionName: string | null
}>()

const terminalRef = ref<HTMLDivElement | null>(null)
const socket = useTerminalSocket()

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null
let unsubOutput: (() => void) | null = null

onMounted(() => {
  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: "'SF Mono', 'Cascadia Code', 'Fira Code', 'Menlo', monospace",
    theme: {
      background: '#0d1117',
      foreground: '#c9d1d9',
      cursor: '#58a6ff',
      selectionBackground: '#1f6feb66',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39d353',
      white: '#c9d1d9',
      brightBlack: '#6e7681',
      brightRed: '#ffa198',
      brightGreen: '#56d364',
      brightYellow: '#e3b341',
      brightBlue: '#79c0ff',
      brightMagenta: '#d2a8ff',
      brightCyan: '#56d364',
      brightWhite: '#f0f6fc',
    },
    allowProposedApi: true,
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  if (terminalRef.value) {
    terminal.open(terminalRef.value)
    requestAnimationFrame(() => {
      fitAddon?.fit()
    })
  }

  // Handle user input — send raw bytes directly to PTY
  terminal.onData((data: string) => {
    const encoded = new TextEncoder().encode(data)
    socket.sendInput(encoded)
  })

  // Handle binary input (for paste, etc.)
  terminal.onBinary((data: string) => {
    const bytes = new Uint8Array(data.length)
    for (let i = 0; i < data.length; i++) {
      bytes[i] = data.charCodeAt(i)
    }
    socket.sendInput(bytes)
  })

  // Receive PTY output — write raw bytes to xterm.js
  unsubOutput = socket.onOutput((data: Uint8Array) => {
    if (terminal) {
      terminal.write(data)
    }
  })

  // Resize observer — fit terminal to container, then tell server
  resizeObserver = new ResizeObserver(() => {
    requestAnimationFrame(() => {
      if (fitAddon && terminal) {
        fitAddon.fit()
        socket.sendResize(terminal.cols, terminal.rows)
      }
    })
  })
  if (terminalRef.value) {
    resizeObserver.observe(terminalRef.value)
  }
})

// Watch sessionName changes to select/switch session
watch(
  () => props.sessionName,
  (newSession, oldSession) => {
    if (terminal && oldSession) {
      terminal.clear()
      terminal.reset()
    }

    if (newSession) {
      socket.selectSession(newSession)
      // Send resize after selecting so the PTY gets correct dimensions
      nextTick(() => {
        requestAnimationFrame(() => {
          if (fitAddon && terminal) {
            fitAddon.fit()
            socket.sendResize(terminal.cols, terminal.rows)
          }
        })
      })
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  unsubOutput?.()
  unsubOutput = null
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  fitAddon = null
})
</script>

<template>
  <div class="terminal-container">
    <!-- Connection status banner -->
    <div v-if="!socket.connected.value" class="status-banner">
      <template v-if="socket.reconnecting.value">
        <span class="status-dot reconnecting" />
        Reconnecting...
      </template>
      <template v-else>
        <span class="status-dot disconnected" />
        Disconnected
      </template>
    </div>

    <!-- Error banner -->
    <div v-if="socket.error.value" class="error-banner">
      {{ socket.error.value }}
    </div>

    <!-- Terminal -->
    <div ref="terminalRef" class="terminal-element" @contextmenu.prevent />
  </div>
</template>

<style scoped>
.terminal-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #0d1117;
  position: relative;
  overflow: hidden;
}

.terminal-element {
  flex: 1;
  padding: 4px;
}

/* Ensure xterm fills the container */
.terminal-element :deep(.xterm) {
  height: 100%;
}
.terminal-element :deep(.xterm-viewport) {
  overflow-y: auto;
}

.status-banner {
  padding: 6px 16px;
  background: #1c2129;
  border-bottom: 1px solid #21262d;
  font-size: 12px;
  color: #d29922;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.error-banner {
  padding: 6px 16px;
  background: #3d1117;
  border-bottom: 1px solid #ff7b7233;
  font-size: 12px;
  color: #ff7b72;
  flex-shrink: 0;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.reconnecting {
  background: #d29922;
  animation: pulse 1.5s ease-in-out infinite;
}

.status-dot.disconnected {
  background: #ff7b72;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
