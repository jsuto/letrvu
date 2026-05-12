<template>
  <div
    v-if="visible"
    class="fixed inset-0 bg-black/35 flex items-center justify-center z-[200]"
    @click.self="cancel"
  >
    <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-6 w-[340px] shadow-2xl">
      <p class="text-sm mb-2 text-[var(--color-text)]">{{ message }}</p>
      <p v-if="warning" class="text-xs text-red-600 m-0">{{ warning }}</p>
      <div class="flex justify-end gap-2 mt-5">
        <button
          @click="cancel"
          class="px-4 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm cursor-pointer text-[var(--color-text)] hover:bg-[var(--color-bg)]"
        >Cancel</button>
        <button
          @click="confirm"
          :disabled="busy"
          class="px-4 py-1.5 bg-red-600 text-white border-none rounded-md text-sm cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
        >{{ busy ? 'Deleting…' : confirmLabel }}</button>
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
