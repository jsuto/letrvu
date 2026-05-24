<template>
  <Transition name="toast">
    <div
      v-if="undoSend.pending.value"
      class="fixed bottom-6 left-1/2 -translate-x-1/2 z-[500] flex flex-col gap-0 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-2xl overflow-hidden"
      style="min-width: 280px"
    >
      <!-- Progress bar — depletes over the delay period -->
      <div class="h-0.5 bg-[var(--color-border)]">
        <div
          class="h-full bg-teal transition-none"
          :style="{ width: progressPct + '%' }"
        />
      </div>
      <div class="flex items-center gap-3 px-4 py-3">
        <span class="flex-1 text-sm text-[var(--color-text)]">
          {{ $t('undoSend.sending', { n: secondsLeft }) }}
        </span>
        <button
          @click="undoSend.undo()"
          class="px-3 py-1 rounded-md border border-teal text-teal text-sm font-medium cursor-pointer bg-transparent hover:bg-[var(--color-teal-light)]"
        >{{ $t('undoSend.undo') }}</button>
        <button
          @click="undoSend.flush()"
          class="px-3 py-1 rounded-md border border-[var(--color-border)] text-[var(--color-text-muted)] text-sm cursor-pointer bg-transparent hover:bg-[var(--color-bg)]"
          :title="$t('undoSend.sendNow')"
        >{{ $t('undoSend.sendNow') }}</button>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useUndoSend } from '../composables/useUndoSend'

const undoSend = useUndoSend()
const progressPct = ref(100)
const secondsLeft = ref(0)
let rafId = null

function tick() {
  const p = undoSend.pending.value
  if (p) {
    const elapsed = (Date.now() - p.startedAt) / 1000
    const remaining = Math.max(0, p.delay - elapsed)
    progressPct.value = (remaining / p.delay) * 100
    secondsLeft.value = Math.ceil(remaining)
  }
  rafId = requestAnimationFrame(tick)
}

onMounted(() => { rafId = requestAnimationFrame(tick) })
onUnmounted(() => { if (rafId) cancelAnimationFrame(rafId) })
</script>

<style scoped>
.toast-enter-active { transition: opacity 0.15s ease, transform 0.15s ease; }
.toast-leave-active { transition: opacity 0.2s ease, transform 0.2s ease; }
.toast-enter-from  { opacity: 0; transform: translate(-50%, 12px); }
.toast-leave-to    { opacity: 0; transform: translate(-50%, 12px); }
</style>
