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
        <div class="mb-2">
          <label class="block text-xs text-[var(--color-text-muted)] mb-1">PGP Public Key</label>
          <div v-if="pgpKeyInfo" class="flex items-center gap-2 mb-1.5">
            <span class="text-xs font-mono text-[var(--color-text-muted)] truncate flex-1">{{ pgpKeyInfo }}</span>
            <button type="button" @click="removePGPKey"
              class="text-xs text-red-600 bg-none border-none cursor-pointer hover:underline shrink-0">Remove</button>
          </div>
          <div v-else>
            <textarea v-model="form.pgpKey" placeholder="Paste armored public key (-----BEGIN PGP PUBLIC KEY BLOCK-----)" rows="4"
              class="w-full px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-xs font-mono outline-none box-border resize-y focus:border-teal" />
            <button type="button" @click="fetchWKD" :disabled="wkdBusy || !form.emails[0]?.email"
              class="mt-1 bg-none border border-[var(--color-border)] rounded-md px-3 py-1 text-xs cursor-pointer text-[var(--color-text-muted)] hover:border-teal hover:text-teal disabled:opacity-40 disabled:cursor-not-allowed">
              {{ wkdBusy ? 'Looking up…' : '🔍 Look up via WKD' }}
            </button>
            <span v-if="wkdError" class="ml-2 text-xs text-[var(--color-text-muted)]">{{ wkdError }}</span>
          </div>
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
import * as openpgp from 'openpgp'
import { useContactsStore } from '../stores/contacts'
import { usePGPStore } from '../stores/pgp'
import { apiFetch } from '../api'

const contacts = useContactsStore()
const pgp = usePGPStore()
const visible = ref(false)
const saving = ref(false)
const error = ref('')
const editId = ref(null)
const isEdit = computed(() => editId.value !== null)

const form = reactive({ name: '', notes: '', emails: [], pgpKey: '' })

// Fingerprint of the currently stored PGP key (fetched when editing a contact with has_pgp_key)
const pgpKeyInfo = ref(null)   // e.g. "Fingerprint: ABCD 1234 …" or null
const wkdBusy = ref(false)
const wkdError = ref('')

async function open(contact = null) {
  editId.value = contact?.id ?? null
  form.name = contact?.name ?? ''
  form.notes = contact?.notes ?? ''
  form.emails = contact?.emails ? contact.emails.map(e => ({ ...e })) : [{ email: '', label: '' }]
  if (form.emails.length === 0) form.emails.push({ email: '', label: '' })
  form.pgpKey = ''
  pgpKeyInfo.value = null
  wkdError.value = ''
  error.value = ''
  visible.value = true

  // Load existing PGP key info if the contact already has one
  if (contact?.id && contact.has_pgp_key) {
    try {
      const res = await apiFetch(`/api/contacts/${contact.id}/pgpkey`)
      if (res.ok) {
        const data = await res.json()
        const key = await openpgp.readKey({ armoredKey: data.key })
        const fp = key.getFingerprint().toUpperCase().match(/.{4}/g)?.join(' ')
        pgpKeyInfo.value = fp ?? 'Key stored'
      }
    } catch { /* ignore */ }
  }
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

async function removePGPKey() {
  if (!editId.value) { pgpKeyInfo.value = null; return }
  await apiFetch(`/api/contacts/${editId.value}/pgpkey`, { method: 'DELETE' })
  pgpKeyInfo.value = null
  await contacts.fetchContacts()
}

async function fetchWKD() {
  wkdError.value = ''
  const email = form.emails[0]?.email?.trim()
  if (!email) return
  wkdBusy.value = true
  try {
    const armored = await pgp.wkdLookup(email)
    if (armored) {
      form.pgpKey = armored
      wkdError.value = ''
    } else {
      wkdError.value = 'No key found via WKD.'
    }
  } catch {
    wkdError.value = 'WKD lookup failed.'
  } finally {
    wkdBusy.value = false
  }
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
    let savedContact
    if (isEdit.value) {
      savedContact = await contacts.updateContact(editId.value, data)
    } else {
      savedContact = await contacts.createContact(data)
    }
    // Save PGP public key if provided
    const keyText = form.pgpKey.trim()
    if (keyText && savedContact?.id) {
      await apiFetch(`/api/contacts/${savedContact.id}/pgpkey`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key: keyText }),
      })
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
