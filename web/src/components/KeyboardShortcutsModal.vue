<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-sm"
        @click.self="close"
      >
        <div class="w-[560px] max-h-[85vh] flex flex-col rounded-xl bg-[var(--color-surface)] border border-[var(--color-border)] shadow-2xl overflow-hidden">

          <!-- Header -->
          <div class="flex items-center justify-between px-5 py-3.5 border-b border-[var(--color-border)]">
            <div class="flex items-center gap-2.5">
              <span class="text-sm font-semibold text-[var(--color-text)]">Keyboard shortcuts</span>
              <span class="text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-teal/10 text-teal">?</span>
            </div>
            <button
              @click="close"
              class="text-[var(--color-text-muted)] hover:text-[var(--color-text)] hover:bg-[var(--color-bg)] w-7 h-7 flex items-center justify-center rounded-md transition-colors text-lg leading-none"
            >×</button>
          </div>

          <!-- Body -->
          <div class="overflow-y-auto px-5 py-4 flex flex-col gap-6">

            <section v-for="group in groups" :key="group.title">
              <h3 class="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-text-muted)] mb-2.5">
                {{ group.title }}
              </h3>
              <div class="flex flex-col gap-0.5">
                <div
                  v-for="shortcut in group.shortcuts"
                  :key="shortcut.label"
                  class="flex items-center justify-between px-3 py-2 rounded-lg hover:bg-[var(--color-bg)] group transition-colors"
                >
                  <span class="text-sm text-[var(--color-text)]">{{ shortcut.label }}</span>
                  <div class="flex items-center gap-1">
                    <kbd
                      v-for="key in shortcut.keys"
                      :key="key"
                      class="inline-flex items-center justify-center min-w-[26px] h-[22px] px-1.5 rounded-md border border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-text-muted)] font-mono text-[11px] font-medium shadow-sm group-hover:border-teal/40 group-hover:text-teal transition-colors"
                    >{{ key }}</kbd>
                    <span v-if="shortcut.keys.length > 1 && shortcut.combo !== true" class="text-[var(--color-text-muted)] text-xs mx-0.5">then</span>
                  </div>
                </div>
              </div>
            </section>

          </div>

          <!-- Footer -->
          <div class="px-5 py-3 border-t border-[var(--color-border)] flex items-center justify-between">
            <span class="text-xs text-[var(--color-text-muted)]">Press <kbd class="inline-flex items-center justify-center px-1.5 h-5 rounded border border-[var(--color-border)] bg-[var(--color-bg)] font-mono text-[10px]">?</kbd> to toggle this panel</span>
            <button
              @click="close"
              class="text-xs px-3 py-1.5 rounded-md bg-teal text-white font-medium hover:bg-teal/90 transition-colors"
            >Done</button>
          </div>

        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const visible = ref(false)

const groups = [
  {
    title: 'Mail',
    shortcuts: [
      { label: 'Compose new message',    keys: ['C'] },
      { label: 'Reply to message',       keys: ['R'] },
      { label: 'Delete message',         keys: ['D'] },
      { label: 'Next message',           keys: ['N'] },
      { label: 'Previous message',       keys: ['P'] },
    ],
  },
  {
    title: 'Navigation',
    shortcuts: [
      { label: 'Open keyboard shortcuts', keys: ['?'] },
      { label: 'Close modal / panel',     keys: ['Esc'] },
    ],
  },
  {
    title: 'Selection',
    shortcuts: [
      { label: 'Select message',              keys: ['Click'] },
      { label: 'Select range',                keys: ['⇧', 'Click'], combo: true },
      { label: 'Toggle individual selection', keys: ['⌘', 'Click'], combo: true },
    ],
  },
]

function open()  { visible.value = true }
function close() { visible.value = false }

function onKeydown(e) {
  if (e.key === 'Escape' && visible.value) { close(); return }
  const tag = document.activeElement?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || document.activeElement?.isContentEditable) return
  if (e.key === '?') { e.preventDefault(); visible.value = !visible.value }
}

onMounted(()   => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

defineExpose({ open, close })
</script>
