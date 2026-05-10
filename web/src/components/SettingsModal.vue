<template>
  <div v-if="visible" class="overlay" @click.self="close">
    <div class="modal">
      <div class="modal-header">
        <span>Settings</span>
        <button @click="close" class="close">×</button>
      </div>
      <div class="modal-body">
        <label>
          Display name
          <input v-model="form.display_name" type="text" placeholder="Your Name" />
        </label>
        <label>
          Signature
          <textarea v-model="form.signature" placeholder="Your name&#10;your@email.com" />
        </label>
      </div>
      <div class="modal-footer">
        <button @click="save" :disabled="saving" class="save-btn">
          {{ saving ? 'Saving…' : saved ? 'Saved ✓' : 'Save' }}
        </button>
        <p v-if="error" class="error">{{ error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useSettingsStore } from '../stores/settings'

const settings = useSettingsStore()
const visible = ref(false)
const saving = ref(false)
const saved = ref(false)
const error = ref('')

const form = reactive({ display_name: '', signature: '' })

async function open() {
  if (!settings.loaded) await settings.fetchSettings()
  form.display_name = settings.settings.display_name ?? ''
  form.signature = settings.settings.signature ?? ''
  saved.value = false
  error.value = ''
  visible.value = true
}

function close() {
  visible.value = false
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    await settings.saveSettings({ display_name: form.display_name, signature: form.signature })
    saved.value = true
    setTimeout(() => { saved.value = false }, 2000)
  } catch {
    error.value = 'Could not save settings.'
  } finally {
    saving.value = false
  }
}

defineExpose({ open, close })
</script>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.3);
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
}
.modal {
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  width: 440px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.15);
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  font-weight: 500;
}
.close { background: none; border: none; font-size: 18px; cursor: pointer; color: var(--color-text-muted); }
.modal-body {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
label {
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 12px;
  color: var(--color-text-muted);
}
input, textarea {
  padding: 8px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 13px;
  font-family: inherit;
  background: var(--color-bg);
  color: var(--color-text);
  outline: none;
}
input:focus, textarea:focus { border-color: var(--color-teal); }
textarea {
  resize: vertical;
  min-height: 100px;
  line-height: 1.6;
}
.modal-footer {
  padding: 12px 16px;
  border-top: 0.5px solid var(--color-border);
  display: flex;
  align-items: center;
  gap: 1rem;
}
.save-btn {
  padding: 7px 20px;
  background: var(--color-teal);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.save-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.error { font-size: 12px; color: #c0392b; }
</style>
