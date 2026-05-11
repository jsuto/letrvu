<template>
  <div v-if="visible" class="overlay" @click.self="close">
    <div class="modal">
      <div class="modal-header">
        <span>{{ isEdit ? 'Edit contact' : 'New contact' }}</span>
        <button @click="close" class="close">×</button>
      </div>
      <div class="modal-body">
        <div class="field-group">
          <label>Name</label>
          <input v-model="form.name" type="text" placeholder="Full name" />
        </div>
        <div class="field-group">
          <label>Email addresses</label>
          <div v-for="(e, i) in form.emails" :key="i" class="email-row">
            <input v-model="e.email" type="email" placeholder="email@example.com" />
            <input v-model="e.label" type="text" placeholder="Label (e.g. work)" class="label-input" />
            <button type="button" class="remove-btn" @click="removeEmail(i)">×</button>
          </div>
          <button type="button" class="add-email-btn" @click="addEmail">+ Add email</button>
        </div>
        <div class="field-group">
          <label>Notes</label>
          <textarea v-model="form.notes" placeholder="Optional notes…" rows="3" />
        </div>
      </div>
      <div class="modal-footer">
        <p v-if="error" class="error">{{ error }}</p>
        <button @click="save" :disabled="saving" class="save-btn">
          {{ saving ? 'Saving…' : 'Save' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useContactsStore } from '../stores/contacts'

const contacts = useContactsStore()
const visible = ref(false)
const saving = ref(false)
const error = ref('')
const editId = ref(null)
const isEdit = computed(() => editId.value !== null)

const form = reactive({ name: '', notes: '', emails: [] })

function open(contact = null) {
  editId.value = contact?.id ?? null
  form.name = contact?.name ?? ''
  form.notes = contact?.notes ?? ''
  form.emails = contact?.emails ? contact.emails.map(e => ({ ...e })) : [{ email: '', label: '' }]
  if (form.emails.length === 0) form.emails.push({ email: '', label: '' })
  error.value = ''
  visible.value = true
}

function close() {
  visible.value = false
}

function addEmail() {
  form.emails.push({ email: '', label: '' })
}

function removeEmail(i) {
  form.emails.splice(i, 1)
  if (form.emails.length === 0) form.emails.push({ email: '', label: '' })
}

async function save() {
  const emails = form.emails.filter(e => e.email.trim())
  if (!form.name.trim() && emails.length === 0) {
    error.value = 'Name or at least one email is required.'
    return
  }
  saving.value = true
  error.value = ''
  try {
    const data = { name: form.name.trim(), notes: form.notes.trim(), emails }
    if (isEdit.value) {
      await contacts.updateContact(editId.value, data)
    } else {
      await contacts.createContact(data)
    }
    close()
  } catch (e) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

function onKeydown(e) { if (e.key === 'Escape' && visible.value) close() }
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

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
  z-index: 200;
}
.modal {
  width: 480px;
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  max-height: 90vh;
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
  overflow-y: auto;
  flex: 1;
}
.field-group { margin-bottom: 1rem; }
.field-group label {
  display: block;
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 4px;
}
input[type="text"], input[type="email"], textarea {
  width: 100%;
  padding: 7px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
}
input:focus, textarea:focus { border-color: var(--color-teal); }
textarea { resize: vertical; }
.email-row {
  display: flex;
  gap: 6px;
  margin-bottom: 6px;
  align-items: center;
}
.email-row input[type="email"] {
  flex: 1;
  min-width: 0;
  width: auto;
}
.label-input {
  flex: 0 0 110px;
  width: auto;
}
.remove-btn {
  background: none;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  font-size: 16px;
  color: var(--color-text-muted);
  padding: 0 8px;
}
.add-email-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 12px;
  color: var(--color-teal);
  padding: 0;
}
.modal-footer {
  padding: 10px 16px;
  border-top: 0.5px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 1rem;
}
.save-btn {
  padding: 8px 20px;
  background: var(--color-teal);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.save-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.error { font-size: 12px; color: #c0392b; flex: 1; }
</style>
