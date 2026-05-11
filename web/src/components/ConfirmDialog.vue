<template>
  <div v-if="visible" class="confirm-overlay" @click.self="cancel">
    <div class="confirm-modal" role="alertdialog" aria-modal="true">
      <p class="confirm-message">{{ message }}</p>
      <p v-if="warning" class="confirm-warn">{{ warning }}</p>
      <div class="confirm-actions">
        <button @click="cancel" class="cancel-btn">Cancel</button>
        <button @click="confirm" :disabled="busy" class="delete-btn">
          {{ busy ? 'Deleting…' : confirmLabel }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  message: { type: String, required: true },
  warning: { type: String, default: 'This cannot be undone.' },
  confirmLabel: { type: String, default: 'Delete' },
  busy: { type: Boolean, default: false },
})

const emit = defineEmits(['update:visible', 'confirm', 'cancel'])

function confirm() {
  emit('confirm')
}

function cancel() {
  emit('update:visible', false)
  emit('cancel')
}

function onKeydown(e) {
  if (e.key === 'Escape' && props.visible) {
    e.stopPropagation()
    cancel()
  }
}
onMounted(() => document.addEventListener('keydown', onKeydown, true))
onUnmounted(() => document.removeEventListener('keydown', onKeydown, true))
</script>

<style scoped>
.confirm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}
.confirm-modal {
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  padding: 24px;
  width: 340px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.18);
}
.confirm-message { font-size: 14px; margin: 0 0 8px; color: var(--color-text); }
.confirm-warn { font-size: 12px; color: #c0392b; margin: 0; }
.confirm-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; }
.cancel-btn {
  padding: 7px 16px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  font-size: 13px;
  cursor: pointer;
  color: var(--color-text);
}
.cancel-btn:hover { background: var(--color-bg); }
.delete-btn {
  padding: 7px 16px;
  background: #c0392b;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}
.delete-btn:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
