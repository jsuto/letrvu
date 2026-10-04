import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiFetch } from '../api'

export const useMailStore = defineStore('mail', () => {
  const folders = ref([])
  const messages = ref([])
  const currentMessage = ref(null)
  const currentMessageFolder = ref(null)
  const currentFolder = ref('INBOX')
  const loading = ref(false)
  const page = ref(1)
  const pageSize = 50
  const hasMore = ref(false)
  const selectedUids = ref(new Set())
  // true when messages are cross-folder search results
  const globalSearchMode = ref(false)
  // Non-empty while the list shows search results (folder or global).
  // Background refreshes leave the list alone while a search is active.
  const searchQuery = ref('')
  // Bumped whenever the list is replaced by a user action (folder change,
  // page change, search). A background refresh that started under an older
  // generation discards its result instead of clobbering the newer list.
  let listGeneration = 0
  // Folder whose first page was last loaded successfully; refreshMessages
  // only reports "new" messages relative to a list it has actually seen.
  let loadedFolder = null

  const quota = ref({ used: 0, limit: 0 })

  async function fetchQuota() {
    const res = await fetch('/api/quota')
    if (!res.ok) return
    quota.value = await res.json()
  }

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
    listGeneration++
    currentFolder.value = folder
    globalSearchMode.value = false
    searchQuery.value = ''
    page.value = p
    selectedUids.value = new Set()
    currentThread.value = null
    loading.value = true
    try {
      const res = await fetch(
        `/api/folders/${encodeURIComponent(folder)}/messages?page=${p}&page_size=${pageSize}`,
      )
      if (!res.ok) return
      const data = await res.json()
      messages.value = data
      loadedFolder = folder
      // If we got a full page there may be more.
      hasMore.value = data.length === pageSize
      // Sync folder unseen count from loaded messages. Some IMAP servers
      // don't return a correct STATUS count for certain folders (e.g. INBOX),
      // so we derive it from the message list as a reliable fallback.
      const f = folders.value.find(f => f.name === folder)
      if (f) f.unseen = data.filter(m => !m.read).length
    } finally {
      loading.value = false
    }
  }

  // refreshMessages re-fetches the first page of the current folder in the
  // background (poll / IDLE push) and merges it into the list without
  // showing the loading state or resetting selection and the open thread.
  // Existing message objects are updated in place so the list only renders
  // the rows that actually changed. Does nothing while search results or a
  // later page are displayed.
  //
  // Returns the messages that arrived since the last load (UID greater than
  // any previously listed UID), or [] when nothing was refreshed.
  async function refreshMessages() {
    if (searchQuery.value || globalSearchMode.value || page.value !== 1) return []
    const folder = currentFolder.value
    const gen = listGeneration
    const res = await fetch(
      `/api/folders/${encodeURIComponent(folder)}/messages?page=1&page_size=${pageSize}`,
    )
    if (!res.ok) return []
    const data = await res.json()
    // The user changed folder, page or started a search meanwhile.
    if (gen !== listGeneration || loading.value) return []

    const existing = new Map(messages.value.map(m => [m.uid, m]))
    const maxKnownUid = messages.value.reduce((max, m) => Math.max(max, m.uid), 0)
    const canReportNew = loadedFolder === folder
    const fresh = []
    messages.value = data.map(d => {
      const m = existing.get(d.uid)
      if (m) return Object.assign(m, d)
      if (canReportNew && d.uid > maxKnownUid) fresh.push(d)
      return d
    })
    loadedFolder = folder
    hasMore.value = data.length === pageSize

    const present = new Set(data.map(m => m.uid))
    if ([...selectedUids.value].some(u => !present.has(u))) {
      selectedUids.value = new Set([...selectedUids.value].filter(u => present.has(u)))
    }
    const f = folders.value.find(f => f.name === folder)
    if (f) f.unseen = data.filter(m => !m.read).length
    // Return the reactive objects now held in the list.
    const freshUids = new Set(fresh.map(m => m.uid))
    return messages.value.filter(m => freshUids.has(m.uid))
  }

  async function searchMessages(folder, query) {
    listGeneration++
    currentFolder.value = folder
    globalSearchMode.value = false
    searchQuery.value = query
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

  async function searchAllFolders(query) {
    listGeneration++
    globalSearchMode.value = true
    searchQuery.value = query
    currentThread.value = null
    selectedUids.value = new Set()
    loading.value = true
    try {
      const res = await fetch(`/api/search?q=${encodeURIComponent(query)}`)
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
    currentMessageFolder.value = folder
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
    if (msg && msg.read !== read) {
      msg.read = read
      const f = folders.value.find(f => f.name === folder)
      if (f) f.unseen = Math.max(0, f.unseen + (read ? -1 : 1))
    }
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

  async function deleteMessages(folder, uids) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(folder)}/messages/delete`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ uids }),
    })
    if (!res.ok) throw new Error('Delete failed')
    const uidSet = new Set(uids)
    messages.value = messages.value.filter(m => !uidSet.has(m.uid))
    if (currentMessage.value && uidSet.has(currentMessage.value.uid)) currentMessage.value = null
    selectedUids.value = new Set()
  }

  async function markAsSpam(folder, uids) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(folder)}/messages/spam`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ uids }),
    })
    if (!res.ok) throw new Error('Mark as spam failed')
    const uidSet = new Set(uids)
    messages.value = messages.value.filter(m => !uidSet.has(m.uid))
    if (currentMessage.value && uidSet.has(currentMessage.value.uid)) currentMessage.value = null
    selectedUids.value = new Set()
  }

  async function archiveMessages(folder, uids) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(folder)}/messages/archive`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ uids }),
    })
    if (!res.ok) throw new Error('Archive failed')
    const uidSet = new Set(uids)
    messages.value = messages.value.filter(m => !uidSet.has(m.uid))
    if (currentMessage.value && uidSet.has(currentMessage.value.uid)) currentMessage.value = null
    selectedUids.value = new Set()
  }

  async function markAsNotSpam(folder, uids) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(folder)}/messages/notspam`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ uids }),
    })
    if (!res.ok) throw new Error('Mark as not spam failed')
    const uidSet = new Set(uids)
    messages.value = messages.value.filter(m => !uidSet.has(m.uid))
    if (currentMessage.value && uidSet.has(currentMessage.value.uid)) currentMessage.value = null
    selectedUids.value = new Set()
  }

  async function markReadMessages(folder, uids, read) {
    const res = await apiFetch(`/api/folders/${encodeURIComponent(folder)}/messages/read`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ uids, read }),
    })
    if (!res.ok) throw new Error('Mark read failed')
    const uidSet = new Set(uids)
    let delta = 0
    for (const m of messages.value) {
      if (uidSet.has(m.uid) && m.read !== read) {
        m.read = read
        delta += read ? -1 : 1
      }
    }
    if (delta !== 0) {
      const f = folders.value.find(f => f.name === folder)
      if (f) f.unseen = Math.max(0, f.unseen + delta)
    }
    if (currentMessage.value && uidSet.has(currentMessage.value.uid)) {
      currentMessage.value = { ...currentMessage.value, read }
    }
    selectedUids.value = new Set()
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

  // --- Thread grouping ---

  // currentThread: the array of messages in the thread the user opened.
  // null means no thread is open (single-message view or nothing selected).
  const currentThread = ref(null)

  // threads: messages grouped into conversation threads, sorted by the date
  // of the most recent message (newest thread first).
  //
  // Threading algorithm:
  //  1. Build a map of message_id → message.
  //  2. For each message, walk its References chain to find the oldest
  //     ancestor that is also present in the current page. That becomes the
  //     thread root.
  //  3. Group by root UID; sort threads by latest message date.
  const threads = computed(() => {
    const msgs = messages.value
    if (!msgs.length) return []

    // Index messages by message_id for O(1) ancestor lookup.
    const byId = new Map()
    for (const m of msgs) {
      if (m.message_id) byId.set(m.message_id, m)
    }

    // Find the root of a message by walking References, then In-Reply-To.
    // visited guards against circular chains (self-references, A→B→A, etc.).
    function findRoot(msg, visited = new Set()) {
      if (visited.has(msg.uid)) return msg
      visited.add(msg.uid)
      // Walk the References chain (oldest ancestor is first in the list).
      if (msg.references) {
        const refs = msg.references.trim().split(/\s+/)
        for (const ref of refs) {
          if (byId.has(ref)) return findRoot(byId.get(ref), visited)
        }
      }
      if (msg.in_reply_to) {
        const parent = byId.get(msg.in_reply_to)
        if (parent) return findRoot(parent, visited)
      }
      return msg
    }

    // Group messages by their root UID.
    const groups = new Map() // rootUid → Message[]
    for (const m of msgs) {
      const root = findRoot(m)
      const key = root.uid
      if (!groups.has(key)) groups.set(key, [])
      groups.get(key).push(m)
    }

    // Build thread objects, sorted by latest message date descending.
    const result = []
    for (const [rootUid, threadMsgs] of groups) {
      // Sort within the thread oldest→newest.
      threadMsgs.sort((a, b) => new Date(a.date) - new Date(b.date))
      const latest = threadMsgs[threadMsgs.length - 1]
      const hasUnread = threadMsgs.some(m => !m.read)
      result.push({
        id: rootUid,
        messages: threadMsgs,
        latest,
        hasUnread,
        latestDate: latest.date,
      })
    }
    result.sort((a, b) => new Date(b.latestDate) - new Date(a.latestDate))
    return result
  })

  function openThread(thread) {
    currentThread.value = thread
    // Mark the latest unread message as the current message so MessageView
    // knows what to show first. fetchMessage will be called per-message
    // inside ThreadView as each one is expanded.
    const toOpen = thread.messages.find(m => !m.read) ?? thread.messages[thread.messages.length - 1]
    fetchMessage(toOpen.folder || currentFolder.value, toOpen.uid)
  }

  return {
    folders,
    messages,
    threads,
    currentThread,
    currentMessage,
    currentMessageFolder,
    currentFolder,
    globalSearchMode,
    loading,
    page,
    hasMore,
    fetchFolders,
    fetchMessages,
    refreshMessages,
    searchQuery,
    searchMessages,
    searchAllFolders,
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
    deleteMessages,
    markAsSpam,
    markAsNotSpam,
    archiveMessages,
    markReadMessages,
    subscribeFolder,
    unsubscribeFolder,
    createFolder,
    openThread,
    renameFolder,
    deleteFolder,
    quota,
    fetchQuota,
  }
})
