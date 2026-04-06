<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { useSessionApi } from '~/composables/useSessionApi'

const props = defineProps<{
  sessionName: string
}>()

const emit = defineEmits<{
  error: [message: string]
}>()

const api = useSessionApi()

const isOpen = ref(false)
const showCustomInput = ref(false)
const customCommand = ref('')
const testCommandLabel = ref<string | null>(null)
const globalPresets = ref<{ name: string, label: string }[]>([])
const projectPresets = ref<{ name: string, label: string }[]>([])
const dropdownRef = ref<HTMLElement | null>(null)
const customInputRef = ref<HTMLInputElement | null>(null)

onClickOutside(dropdownRef, () => {
  close()
})

function close() {
  isOpen.value = false
  showCustomInput.value = false
  customCommand.value = ''
}

async function toggle() {
  if (isOpen.value) {
    close()
    return
  }
  isOpen.value = true
  showCustomInput.value = false
  customCommand.value = ''
  // Fetch presets and detect test command in parallel.
  const [testCmd, presets] = await Promise.all([
    api.detectTestCommand(props.sessionName),
    api.getPresets(props.sessionName),
  ])
  testCommandLabel.value = testCmd
  globalPresets.value = presets.global ?? []
  projectPresets.value = presets.project ?? []
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    close()
  }
}

async function safeCall(fn: () => Promise<void>) {
  try {
    await fn()
  }
  catch (e: any) {
    emit('error', e.message || 'Split failed')
  }
}

async function handleSplit(preset: string, command?: string) {
  close()
  await safeCall(() => api.splitWindow(props.sessionName, preset, command))
}

async function handleCustomSubmit() {
  const cmd = customCommand.value
  close()
  await safeCall(() => api.splitWindow(props.sessionName, 'custom', cmd))
}

async function showCustom() {
  showCustomInput.value = true
  await nextTick()
  customInputRef.value?.focus()
}
</script>

<template>
  <div ref="dropdownRef" class="split-dropdown" @keydown="handleKeydown">
    <button class="split-trigger" type="button" @click="toggle" title="Split pane">
      <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
        <path d="M2 2h12v12H2V2zm1 1v10h4.5V3H3zm5.5 0v10H13V3H8.5z" />
      </svg>
      <span>Split</span>
    </button>

    <div v-if="isOpen" class="split-menu">
      <button class="split-item" type="button" @click="handleSplit('lazygit')">
        lazygit
      </button>

      <button class="split-item" type="button" @click="handleSplit('test-runner')">
        Run Tests<span v-if="testCommandLabel" class="test-cmd"> — {{ testCommandLabel }}</span>
      </button>

      <template v-if="globalPresets.length > 0">
        <div class="split-separator" />
        <button
          v-for="preset in globalPresets"
          :key="'g-' + preset.name"
          class="split-item"
          type="button"
          @click="handleSplit(preset.name)"
        >
          {{ preset.label }}
        </button>
      </template>

      <template v-if="projectPresets.length > 0">
        <div class="split-separator" />
        <button
          v-for="preset in projectPresets"
          :key="'p-' + preset.name"
          class="split-item"
          type="button"
          @click="handleSplit(preset.name)"
        >
          {{ preset.label }}
        </button>
      </template>

      <div v-if="globalPresets.length > 0 || projectPresets.length > 0" class="split-separator" />

      <template v-if="!showCustomInput">
        <button class="split-item" type="button" @click="showCustom">
          Custom…
        </button>
      </template>
      <template v-else>
        <div class="custom-input-row">
          <input
            ref="customInputRef"
            v-model="customCommand"
            class="custom-input"
            type="text"
            placeholder="Command…"
            @keydown.enter="handleCustomSubmit"
            @keydown.escape.stop="close"
          />
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.split-dropdown {
  position: relative;
}

.split-trigger {
  height: 24px;
  padding: 0 8px;
  border: 1px solid #30363d;
  border-radius: 6px;
  background: #21262d;
  color: #e6edf3;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
}

.split-trigger:hover {
  background: #30363d;
}

.split-trigger:focus-visible {
  outline: 2px solid #1f6feb;
  outline-offset: 2px;
}

.split-menu {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  min-width: 200px;
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 4px 0;
  z-index: 100;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}

.split-item {
  display: block;
  width: 100%;
  padding: 6px 12px;
  border: none;
  background: none;
  color: #e6edf3;
  font-size: 12px;
  text-align: left;
  cursor: pointer;
  font-family: inherit;
}

.split-item:hover {
  background: #1f6feb;
  color: #fff;
}

.test-cmd {
  color: #8b949e;
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', monospace;
  font-size: 11px;
}

.split-item:hover .test-cmd {
  color: rgba(255, 255, 255, 0.8);
}

.split-separator {
  height: 1px;
  margin: 4px 0;
  background: #30363d;
}

.custom-input-row {
  padding: 4px 8px;
}

.custom-input {
  width: 100%;
  height: 26px;
  padding: 0 8px;
  border: 1px solid #30363d;
  border-radius: 4px;
  background: #0d1117;
  color: #e6edf3;
  font-size: 12px;
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', monospace;
  outline: none;
  box-sizing: border-box;
}

.custom-input:focus {
  border-color: #1f6feb;
}
</style>
