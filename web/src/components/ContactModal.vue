<template>
  <div v-if="visible" class="fixed inset-0 bg-black/30 flex items-center justify-center z-[200]" @click.self="close">
    <div class="w-[480px] bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl flex flex-col max-h-[90vh]">

      <!-- Header -->
      <div class="flex justify-between items-center px-4 py-3 border-b border-[var(--color-border)] text-sm font-medium">
        <span>{{ isEdit ? 'Edit contact' : 'New contact' }}</span>
        <button @click="close" class="bg-none border-none text-lg cursor-pointer text-[var(--color-text-muted)]">×</button>
      </div>

      <!-- Body -->
      <div class="px-4 py-4 overflow-y-auto flex-1">
        <div class="mb-4">
          <label class="block text-xs text-[var(--color-text-muted)] mb-1">Name</label>
          <input v-model="form.name" type="text" placeholder="Full name"
            class="w-full px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none box-border focus:border-teal" />
        </div>
        <div class="mb-4">
          <label class="block text-xs text-[var(--color-text-muted)] mb-1">Email addresses</label>
          <div v-for="(e, i) in form.emails" :key="i" class="flex gap-1.5 mb-1.5 items-center">
            <input v-model="e.email" type="email" placeholder="email@example.com"
              class="flex-1 min-w-0 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none focus:border-teal" />
            <input v-model="e.label" type="text" placeholder="Label (e.g. work)"
              class="w-[110px] shrink-0 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none focus:border-teal" />
            <button type="button"
              class="bg-none border border-[var(--color-border)] rounded-md cursor-pointer text-base text-[var(--color-text-muted)] px-2 py-0"
              @click="removeEmail(i)">×</button>
          </div>
          <button type="button"
            class="bg-none border-none cursor-pointer text-xs text-teal p-0"
            @click="addEmail">+ Add email</button>
        </div>
        <div class="mb-4">
          <label class="block text-xs text-[var(--color-text-muted)] mb-1">Notes</label>
          <textarea v-model="form.notes" placeholder="Optional notes…" rows="3"
            class="w-full px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none box-border resize-y focus:border-teal" />
        </div>
      </div>

      <!-- Footer -->
      <div class="px-4 py-2.5 border-t border-[var(--color-border)] flex items-center justify-end gap-4">
        <p v-if="error" class="text-xs text-red-600 flex-1">{{ error }}</p>
        <button @click="save" :disabled="saving"
          class="px-5 py-2 bg-teal text-white border-none rounded-md text-sm font-medium cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed">
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
