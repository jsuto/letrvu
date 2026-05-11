<template>
  <div v-if="visible" class="overlay" @click.self="close">
    <div class="modal">
      <div class="modal-header">
        <span>Manage folders</span>
        <button @click="close" class="close">×</button>
      </div>
      <div class="modal-body">
        <p class="hint">Toggle folders to show or hide them in the sidebar.</p>
        <ul class="folder-list">
          <li v-for="folder in mail.folders" :key="folder.name" class="folder-row">
            <span class="folder-name">{{ folder.name }}</span>
            <button
              class="toggle-btn"
              :class="{ subscribed: folder.subscribed }"
              :disabled="busy === folder.name"
              @click="toggle(folder)"
            >
              {{ busy === folder.name ? '…' : folder.subscribed ? 'Subscribed' : 'Subscribe' }}
            </button>
          </li>
        </ul>
        <p v-if="error" class="error">{{ error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useMailStore } from '../stores/mail'

const mail = useMailStore()
const visible = ref(false)
const busy = ref(null)
const error = ref('')

async function open() {
  error.value = ''
  visible.value = true
  await mail.fetchFolders()
}

function close() {
  visible.value = false
}

async function toggle(folder) {
  busy.value = folder.name
  error.value = ''
  try {
    if (folder.subscribed) {
      await mail.unsubscribeFolder(folder.name)
    } else {
      await mail.subscribeFolder(folder.name)
    }
  } catch {
    error.value = `Could not update subscription for "${folder.name}".`
  } finally {
    busy.value = null
  }
}

defineExpose({ open, close })
</script>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal {
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  width: 380px;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.12);
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  font-weight: 500;
  flex-shrink: 0;
}
.close { background: none; border: none; font-size: 18px; cursor: pointer; color: var(--color-text-muted); }
.modal-body {
  overflow-y: auto;
  padding: 12px 16px;
  flex: 1;
}
.hint {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 12px;
}
.folder-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.folder-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 7px 0;
  border-bottom: 0.5px solid var(--color-border);
}
.folder-row:last-child { border-bottom: none; }
.folder-name { font-size: 13px; }
.toggle-btn {
  padding: 4px 12px;
  border-radius: 5px;
  font-size: 12px;
  cursor: pointer;
  border: 0.5px solid var(--color-border);
  background: transparent;
  color: var(--color-text-muted);
  min-width: 90px;
}
.toggle-btn.subscribed {
  background: var(--color-teal-light);
  border-color: var(--color-teal);
  color: var(--color-teal);
  font-weight: 500;
}
.toggle-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.error { font-size: 12px; color: #c0392b; margin-top: 10px; }
</style>
