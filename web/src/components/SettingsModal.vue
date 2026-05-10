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

        <div class="section-title">Identities (From: addresses)</div>
        <div class="identity-list">
          <div v-for="(id, i) in form.identities" :key="i" class="identity-row">
            <input v-model="id.name" type="text" placeholder="Name" class="id-name" />
            <input v-model="id.email" type="email" placeholder="email@example.com" class="id-email" />
            <button @click="removeIdentity(i)" class="remove-btn" title="Remove">×</button>
          </div>
          <button @click="addIdentity" class="add-btn">+ Add identity</button>
        </div>
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

const form = reactive({ display_name: '', signature: '', identities: [] })

async function open() {
  if (!settings.loaded) await settings.fetchSettings()
  form.display_name = settings.settings.display_name ?? ''
  form.signature = settings.settings.signature ?? ''
  form.identities = settings.identities.map(id => ({ ...id }))
  saved.value = false
  error.value = ''
  visible.value = true
}

function close() {
  visible.value = false
}

function addIdentity() {
  form.identities.push({ name: '', email: '' })
}

function removeIdentity(i) {
  form.identities.splice(i, 1)
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    const validIdentities = form.identities.filter(id => id.email.trim())
    await settings.saveSettings({
      display_name: form.display_name,
      signature: form.signature,
      identities: JSON.stringify(validIdentities),
    })
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
  width: 480px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.15);
  max-height: 90vh;
  overflow-y: auto;
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  font-weight: 500;
  position: sticky;
  top: 0;
  background: var(--color-surface);
  z-index: 1;
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
.section-title {
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 500;
  padding-top: 4px;
  border-top: 0.5px solid var(--color-border);
  margin-top: 4px;
}
.identity-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.identity-row {
  display: flex;
  gap: 6px;
  align-items: center;
}
.id-name { flex: 1; }
.id-email { flex: 1.5; }
.remove-btn {
  background: none;
  border: none;
  font-size: 16px;
  cursor: pointer;
  color: var(--color-text-muted);
  padding: 4px 6px;
  flex-shrink: 0;
  border-radius: 4px;
}
.remove-btn:hover { background: var(--color-teal-light); }
.add-btn {
  background: none;
  border: 0.5px dashed var(--color-border);
  border-radius: 6px;
  padding: 6px 12px;
  font-size: 12px;
  cursor: pointer;
  color: var(--color-text-muted);
  text-align: left;
}
.add-btn:hover { border-color: var(--color-teal); color: var(--color-teal); }
.modal-footer {
  padding: 12px 16px;
  border-top: 0.5px solid var(--color-border);
  display: flex;
  align-items: center;
  gap: 1rem;
  position: sticky;
  bottom: 0;
  background: var(--color-surface);
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
