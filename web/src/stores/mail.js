import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '../api'

export const useMailStore = defineStore('mail', () => {
  const folders = ref([])
  const messages = ref([])
  const currentMessage = ref(null)
  const currentFolder = ref('INBOX')
  const loading = ref(false)
  const page = ref(1)
  const pageSize = 50
  const hasMore = ref(false)
  const selectedUids = ref(new Set())

  async function fetchFolders() {
    const res = await fetch('/api/folders')
    if (!res.ok) return
    folders.value = await res.json()
  }

  function toggleSelect(uid) {
    const s = new Set(selectedUids.value)
    if (s.has(uid)) s.delete(uid)
    else s.add(uid)
    selectedUids.value = s
  }

  function clearSelection() {
    selectedUids.value = new Set()
  }

  async function fetchMessages(folder, p = 1) {
    currentFolder.value = folder
    page.value = p
    selectedUids.value = new Set()
    loading.value = true
    try {
      const res = await fetch(
        `/api/folders/${encodeURIComponent(folder)}/messages?page=${p}&page_size=${pageSize}`,
      )
      if (!res.ok) return
      const data = await res.json()
      messages.value = data
      // If we got a full page there may be more.
      hasMore.value = data.length === pageSize
    } finally {
      loading.value = false
    }
  }

  async function searchMessages(folder, query) {
    currentFolder.value = folder
    loading.value = true
    try {
      const res = await fetch(
        `/api/folders/${encodeURIComponent(folder)}/messages?q=${encodeURIComponent(query)}`,
      )
      if (!res.ok) return
      messages.value = await res.json()
    } finally {
      loading.value = false
    }
  }

  async function fetchMessage(folder, uid) {
    const res = await fetch(`/api/folders/${encodeURIComponent(folder)}/messages/${uid}`)
    if (!res.ok) return
    currentMessage.value = await res.json()
  }

  async function deleteMessage(folder, uid) {
    await apiFetch(`/api/folders/${encodeURIComponent(folder)}/messages/${uid}`, {
      method: 'DELETE',
    })
    messages.value = messages.value.filter(m => m.uid !== uid)
    if (currentMessage.value?.uid === uid) currentMessage.value = null
  }

  async function markRead(folder, uid, read = true) {
    await apiFetch(`/api/folders/${encodeURIComponent(folder)}/messages/${uid}/read`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ read }),
    })
    const msg = messages.value.find(m => m.uid === uid)
    if (msg) msg.read = read
    if (currentMessage.value?.uid === uid) currentMessage.value = { ...currentMessage.value, read }
  }

  async function markFlagged(folder, uid, flagged) {
    await apiFetch(`/api/folders/${encodeURIComponent(folder)}/messages/${uid}/flagged`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ flagged }),
    })
    const msg = messages.value.find(m => m.uid === uid)
    if (msg) msg.flagged = flagged
    if (currentMessage.value?.uid === uid) currentMessage.value = { ...currentMessage.value, flagged }
  }

  async function moveMessage(folder, uid, dest) {
    return moveMessagesTo(folder, [uid], dest)
  }

  async function moveMessagesTo(folder, uids, dest) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(folder)}/messages/move`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ uids, dest }),
    })
    if (!res.ok) throw new Error('Move failed')
    const uidSet = new Set(uids)
    messages.value = messages.value.filter(m => !uidSet.has(m.uid))
    if (currentMessage.value && uidSet.has(currentMessage.value.uid)) currentMessage.value = null
    selectedUids.value = new Set([...selectedUids.value].filter(u => !uidSet.has(u)))
  }

  async function sendMessage(payload) {
    const res = await apiFetch('/api/send', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!res.ok) throw new Error('Send failed')
  }

  async function subscribeFolder(folder) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(folder)}/subscribe`, { method: 'POST' })
    if (!res.ok) throw new Error('Subscribe failed')
    const f = folders.value.find(f => f.name === folder)
    if (f) f.subscribed = true
  }

  async function unsubscribeFolder(folder) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(folder)}/subscribe`, { method: 'DELETE' })
    if (!res.ok) throw new Error('Unsubscribe failed')
    const f = folders.value.find(f => f.name === folder)
    if (f) f.subscribed = false
  }

  async function createFolder(name) {
    const res = await apiFetch('/api/folders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    if (!res.ok) throw new Error('Create folder failed')
    await fetchFolders()
  }

  async function renameFolder(oldName, newName) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(oldName)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ new_name: newName }),
    })
    if (!res.ok) throw new Error('Rename folder failed')
    await fetchFolders()
    // If the renamed folder is currently open, navigate to the new name.
    if (currentFolder.value === oldName) currentFolder.value = newName
  }

  async function deleteFolder(name) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(name)}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('Delete folder failed')
    await fetchFolders()
    if (currentFolder.value === name) {
      currentFolder.value = 'INBOX'
      await fetchMessages('INBOX')
    }
  }

  async function saveDraft(payload) {
    const res = await apiFetch('/api/draft', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!res.ok) throw new Error('Save draft failed')
  }

  return {
    folders,
    messages,
    currentMessage,
    currentFolder,
    loading,
    page,
    hasMore,
    fetchFolders,
    fetchMessages,
    searchMessages,
    fetchMessage,
    selectedUids,
    toggleSelect,
    clearSelection,
    deleteMessage,
    moveMessage,
    moveMessagesTo,
    markRead,
    markFlagged,
    sendMessage,
    saveDraft,
    subscribeFolder,
    unsubscribeFolder,
    createFolder,
    renameFolder,
    deleteFolder,
  }
})
