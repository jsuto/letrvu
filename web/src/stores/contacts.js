import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '../api'

export const useContactsStore = defineStore('contacts', () => {
  const contacts = ref([])
  const loading = ref(false)

  async function fetchContacts() {
    loading.value = true
    try {
      const res = await fetch('/api/contacts')
      if (!res.ok) throw new Error('Failed to load contacts')
      contacts.value = await res.json()
    } finally {
      loading.value = false
    }
  }

  async function createContact(data) {
    const res = await apiFetch('/api/contacts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    if (!res.ok) throw new Error('Failed to create contact')
    const contact = await res.json()
    contacts.value.push(contact)
    contacts.value.sort((a, b) => a.name.localeCompare(b.name))
    return contact
  }

  async function updateContact(id, data) {
    const res = await apiFetch(`/api/contacts/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    if (!res.ok) throw new Error('Failed to update contact')
    const contact = await res.json()
    const idx = contacts.value.findIndex(c => c.id === id)
    if (idx !== -1) contacts.value[idx] = contact
    contacts.value.sort((a, b) => a.name.localeCompare(b.name))
    return contact
  }

  async function deleteContact(id) {
    const res = await apiFetch(`/api/contacts/${id}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('Failed to delete contact')
    contacts.value = contacts.value.filter(c => c.id !== id)
  }

  async function autocomplete(q) {
    if (!q) return []
    const res = await fetch(`/api/contacts/autocomplete?q=${encodeURIComponent(q)}`)
    if (!res.ok) return []
    return res.json()
  }

  async function saveFromMessage(name, email) {
    const res = await apiFetch('/api/contacts/save-from-message', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, email }),
    })
    if (!res.ok) throw new Error('Failed to save contact')
    const contact = await res.json()
    const idx = contacts.value.findIndex(c => c.id === contact.id)
    if (idx === -1) {
      contacts.value.push(contact)
      contacts.value.sort((a, b) => a.name.localeCompare(b.name))
    }
    return contact
  }

  return { contacts, loading, fetchContacts, createContact, updateContact, deleteContact, autocomplete, saveFromMessage }
})
