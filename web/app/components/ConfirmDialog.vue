<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'

const props = defineProps<{
  visible: boolean
  title: string
  message: string
  confirmLabel?: string
  danger?: boolean
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const cardRef = ref<HTMLDivElement | null>(null)

watch(() => props.visible, (v) => {
  if (v) {
    nextTick(() => cardRef.value?.focus())
  }
})

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('cancel')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="dialog-backdrop" @click.self="emit('cancel')" @keydown="onKeydown">
      <div ref="cardRef" class="dialog-card" role="alertdialog" :aria-label="title" tabindex="-1">
        <h3 class="dialog-title">{{ title }}</h3>
        <p class="dialog-message">{{ message }}</p>
        <div class="dialog-actions">
          <button class="btn btn-cancel" @click="emit('cancel')">Cancel</button>
          <button
            class="btn btn-confirm"
            :class="{ 'btn-danger': danger }"
            @click="emit('confirm')"
          >
            {{ confirmLabel ?? 'Confirm' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.dialog-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.dialog-card {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 20px;
  min-width: 320px;
  max-width: 420px;
}

.dialog-title {
  font-size: 14px;
  font-weight: 600;
  color: #e6edf3;
  margin: 0 0 8px;
}

.dialog-message {
  font-size: 13px;
  color: #8b949e;
  margin: 0 0 16px;
  line-height: 1.5;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btn {
  padding: 6px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid #30363d;
  transition: background 0.15s;
}

.btn-cancel {
  background: #21262d;
  color: #c9d1d9;
}

.btn-cancel:hover {
  background: #30363d;
}

.btn-confirm {
  background: #238636;
  color: #fff;
  border-color: #238636;
}

.btn-confirm:hover {
  background: #2ea043;
}

.btn-danger {
  background: #da3633;
  border-color: #da3633;
}

.btn-danger:hover {
  background: #f85149;
}
</style>
