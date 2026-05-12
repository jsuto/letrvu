import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '../api'

export const useContactsStore = defineStore('contacts', () => {
  const contacts = ref([])
  const groups = ref([])
  const loading = ref(false)

  async function fetchContacts() {
    loading.value = true
    try {
      const [cRes, gRes] = await Promise.all([fetch('/api/contacts'), fetch('/api/contact-groups')])
      if (cRes.ok) contacts.value = await cRes.json()
      if (gRes.ok) groups.value = await gRes.json()
    } finally {
      loading.value = false
    }
  }

  async function fetchGroups() {
    const res = await fetch('/api/contact-groups')
    if (!res.ok) return
    groups.value = await res.json()
  }

  async function createGroup(name) {
    const res = await apiFetch('/api/contact-groups', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    if (!res.ok) throw new Error('Failed to create group')
    const group = await res.json()
    groups.value.push(group)
    groups.value.sort((a, b) => a.name.localeCompare(b.name))
    return group
  }

  async function updateGroup(id, name) {
    const res = await apiFetch(`/api/contact-groups/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    if (!res.ok) throw new Error('Failed to update group')
    const group = await res.json()
    const idx = groups.value.findIndex(g => g.id === id)
    if (idx !== -1) groups.value[idx] = group
    groups.value.sort((a, b) => a.name.localeCompare(b.name))
    return group
  }

  async function deleteGroup(id) {
    const res = await apiFetch(`/api/contact-groups/${id}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('Failed to delete group')
    groups.value = groups.value.filter(g => g.id !== id)
  }

  async function addGroupMember(groupId, contactId) {
    const res = await apiFetch(`/api/contact-groups/${groupId}/members`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ contact_id: contactId }),
    })
    if (!res.ok) throw new Error('Failed to add member')
    const group = await res.json()
    const idx = groups.value.findIndex(g => g.id === groupId)
    if (idx !== -1) groups.value[idx] = group
    return group
  }

  async function removeGroupMember(groupId, contactId) {
    const res = await apiFetch(`/api/contact-groups/${groupId}/members/${contactId}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('Failed to remove member')
    const group = await res.json()
    const idx = groups.value.findIndex(g => g.id === groupId)
    if (idx !== -1) groups.value[idx] = group
    return group
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

  return {
    contacts, groups, loading,
    fetchContacts, fetchGroups,
    createContact, updateContact, deleteContact,
    createGroup, updateGroup, deleteGroup,
    addGroupMember, removeGroupMember,
    autocomplete, saveFromMessage,
  }
})
