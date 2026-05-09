import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useMailStore = defineStore('mail', () => {
  const folders = ref([])
  const messages = ref([])
  const currentMessage = ref(null)
  const currentFolder = ref('INBOX')
  const loading = ref(false)

  async function fetchFolders() {
    const res = await fetch('/api/folders')
    if (!res.ok) return
    folders.value = await res.json()
  }

  async function fetchMessages(folder) {
    currentFolder.value = folder
    loading.value = true
    try {
      const res = await fetch(`/api/folders/${encodeURIComponent(folder)}/messages`)
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
    await fetch(`/api/folders/${encodeURIComponent(folder)}/messages/${uid}`, {
      method: 'DELETE',
    })
    messages.value = messages.value.filter(m => m.uid !== uid)
    if (currentMessage.value?.uid === uid) currentMessage.value = null
  }

  async function markRead(folder, uid, read = true) {
    await fetch(`/api/folders/${encodeURIComponent(folder)}/messages/${uid}/read`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ read }),
    })
    const msg = messages.value.find(m => m.uid === uid)
    if (msg) msg.read = read
  }

  async function sendMessage(payload) {
    const res = await fetch('/api/send', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!res.ok) throw new Error('Send failed')
  }

  return {
    folders,
    messages,
    currentMessage,
    currentFolder,
    loading,
    fetchFolders,
    fetchMessages,
    fetchMessage,
    deleteMessage,
    markRead,
    sendMessage,
  }
})
