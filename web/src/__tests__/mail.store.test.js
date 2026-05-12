import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMailStore } from '../stores/mail.js'

beforeEach(() => {
  setActivePinia(createPinia())
  global.fetch = vi.fn()
})

// Seed the store with a set of messages for convenience.
function seedMessages(store) {
  store.messages = [
    { uid: 1, subject: 'First', read: false, flagged: false },
    { uid: 2, subject: 'Second', read: true, flagged: false },
    { uid: 3, subject: 'Third', read: false, flagged: true },
  ]
}

describe('mail store — toggleSelect', () => {
  it('selects an unselected uid', () => {
    const store = useMailStore()
    store.toggleSelect(42)
    expect(store.selectedUids.has(42)).toBe(true)
  })

  it('deselects an already-selected uid', () => {
    const store = useMailStore()
    store.toggleSelect(42)
    store.toggleSelect(42)
    expect(store.selectedUids.has(42)).toBe(false)
  })

  it('can select multiple uids independently', () => {
    const store = useMailStore()
    store.toggleSelect(1)
    store.toggleSelect(2)
    store.toggleSelect(3)
    expect(store.selectedUids.size).toBe(3)
  })

  it('deselecting one does not affect others', () => {
    const store = useMailStore()
    store.toggleSelect(1)
    store.toggleSelect(2)
    store.toggleSelect(1)
    expect(store.selectedUids.has(1)).toBe(false)
    expect(store.selectedUids.has(2)).toBe(true)
  })
})

describe('mail store — clearSelection', () => {
  it('removes all selected uids', () => {
    const store = useMailStore()
    store.toggleSelect(1)
    store.toggleSelect(2)
    store.clearSelection()
    expect(store.selectedUids.size).toBe(0)
  })

  it('is safe to call when nothing is selected', () => {
    const store = useMailStore()
    expect(() => store.clearSelection()).not.toThrow()
  })
})

describe('mail store — deleteMessage', () => {
  it('removes the message from the list', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.deleteMessage('INBOX', 2)
    expect(store.messages.find(m => m.uid === 2)).toBeUndefined()
    expect(store.messages).toHaveLength(2)
  })

  it('clears currentMessage when it is the deleted one', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.currentMessage = { uid: 1 }
    global.fetch.mockResolvedValue({ ok: true })
    await store.deleteMessage('INBOX', 1)
    expect(store.currentMessage).toBeNull()
  })

  it('does not clear currentMessage when a different message is deleted', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.currentMessage = { uid: 3 }
    global.fetch.mockResolvedValue({ ok: true })
    await store.deleteMessage('INBOX', 1)
    expect(store.currentMessage).not.toBeNull()
  })
})

describe('mail store — markRead', () => {
  it('updates read flag in the message list', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.markRead('INBOX', 1, true)
    expect(store.messages.find(m => m.uid === 1).read).toBe(true)
  })

  it('updates currentMessage when it matches', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.currentMessage = { uid: 1, read: false }
    global.fetch.mockResolvedValue({ ok: true })
    await store.markRead('INBOX', 1, true)
    expect(store.currentMessage.read).toBe(true)
  })

  it('can mark as unread', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.markRead('INBOX', 2, false)
    expect(store.messages.find(m => m.uid === 2).read).toBe(false)
  })
})

describe('mail store — markFlagged', () => {
  it('updates flagged flag in the message list', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.markFlagged('INBOX', 1, true)
    expect(store.messages.find(m => m.uid === 1).flagged).toBe(true)
  })

  it('updates currentMessage when it matches', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.currentMessage = { uid: 3, flagged: true }
    global.fetch.mockResolvedValue({ ok: true })
    await store.markFlagged('INBOX', 3, false)
    expect(store.currentMessage.flagged).toBe(false)
  })
})

describe('mail store — moveMessagesTo', () => {
  it('removes moved messages from the list', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.moveMessagesTo('INBOX', [1, 2], 'Archive')
    expect(store.messages.find(m => m.uid === 1)).toBeUndefined()
    expect(store.messages.find(m => m.uid === 2)).toBeUndefined()
    expect(store.messages.find(m => m.uid === 3)).toBeDefined()
  })

  it('clears currentMessage when it is moved', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.currentMessage = { uid: 2 }
    global.fetch.mockResolvedValue({ ok: true })
    await store.moveMessagesTo('INBOX', [2], 'Archive')
    expect(store.currentMessage).toBeNull()
  })

  it('removes moved uids from selectedUids', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.toggleSelect(1)
    store.toggleSelect(2)
    global.fetch.mockResolvedValue({ ok: true })
    await store.moveMessagesTo('INBOX', [1], 'Archive')
    expect(store.selectedUids.has(1)).toBe(false)
    expect(store.selectedUids.has(2)).toBe(true)
  })

  it('throws when server returns an error', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.moveMessagesTo('INBOX', [1], 'Archive')).rejects.toThrow()
  })
})

describe('mail store — fetchMessages', () => {
  it('sets currentFolder', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.fetchMessages('Sent')
    expect(store.currentFolder).toBe('Sent')
  })

  it('sets loading to false after fetch', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.fetchMessages('INBOX')
    expect(store.loading).toBe(false)
  })

  it('clears selectedUids on folder change', async () => {
    const store = useMailStore()
    store.toggleSelect(99)
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.fetchMessages('INBOX')
    expect(store.selectedUids.size).toBe(0)
  })

  it('sets hasMore true when a full page is returned', async () => {
    const store = useMailStore()
    const fullPage = Array.from({ length: 50 }, (_, i) => ({ uid: i + 1 }))
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve(fullPage) })
    await store.fetchMessages('INBOX')
    expect(store.hasMore).toBe(true)
  })

  it('sets hasMore false when fewer than a full page returned', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([{ uid: 1 }]) })
    await store.fetchMessages('INBOX')
    expect(store.hasMore).toBe(false)
  })
})

describe('mail store — sendMessage', () => {
  it('throws when server returns error', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.sendMessage({ to: ['x@example.com'], subject: 'hi' })).rejects.toThrow()
  })

  it('resolves when server returns ok', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true })
    await expect(store.sendMessage({ to: ['x@example.com'] })).resolves.not.toThrow()
  })
})

describe('mail store — subscribeFolder / unsubscribeFolder', () => {
  function seedFolders(store) {
    store.folders = [
      { name: 'INBOX', subscribed: true },
      { name: 'Drafts', subscribed: true },
      { name: 'OldProject', subscribed: false },
    ]
  }

  it('subscribeFolder sets subscribed=true on the matching folder', async () => {
    const store = useMailStore()
    seedFolders(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.subscribeFolder('OldProject')
    expect(store.folders.find(f => f.name === 'OldProject').subscribed).toBe(true)
  })

  it('unsubscribeFolder sets subscribed=false on the matching folder', async () => {
    const store = useMailStore()
    seedFolders(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.unsubscribeFolder('Drafts')
    expect(store.folders.find(f => f.name === 'Drafts').subscribed).toBe(false)
  })

  it('subscribeFolder POSTs to /api/folders/{folder}/subscribe', async () => {
    const store = useMailStore()
    seedFolders(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.subscribeFolder('OldProject')
    const [url, opts] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders/OldProject/subscribe')
    expect(opts.method).toBe('POST')
  })

  it('unsubscribeFolder sends DELETE to /api/folders/{folder}/subscribe', async () => {
    const store = useMailStore()
    seedFolders(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.unsubscribeFolder('Drafts')
    const [url, opts] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders/Drafts/subscribe')
    expect(opts.method).toBe('DELETE')
  })

  it('subscribeFolder throws when server returns error', async () => {
    const store = useMailStore()
    seedFolders(store)
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.subscribeFolder('OldProject')).rejects.toThrow()
  })

  it('unsubscribeFolder throws when server returns error', async () => {
    const store = useMailStore()
    seedFolders(store)
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.unsubscribeFolder('INBOX')).rejects.toThrow()
  })

  it('subscribeFolder encodes folder names with special characters', async () => {
    const store = useMailStore()
    store.folders = [{ name: 'My Folder/Sub', subscribed: false }]
    global.fetch.mockResolvedValue({ ok: true })
    await store.subscribeFolder('My Folder/Sub')
    const [url] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders/My%20Folder%2FSub/subscribe')
  })

  it('does not mutate other folders when subscribing one', async () => {
    const store = useMailStore()
    seedFolders(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.subscribeFolder('OldProject')
    expect(store.folders.find(f => f.name === 'INBOX').subscribed).toBe(true)
    expect(store.folders.find(f => f.name === 'Drafts').subscribed).toBe(true)
  })
})

describe('mail store — saveDraft', () => {
  it('resolves when server returns ok', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true })
    await expect(store.saveDraft({ to: ['x@example.com'], subject: 'draft' })).resolves.not.toThrow()
  })

  it('throws when server returns error', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.saveDraft({ to: ['x@example.com'], subject: 'draft' })).rejects.toThrow()
  })

  it('posts to /api/draft', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true })
    await store.saveDraft({ to: ['x@example.com'], subject: 'my draft', text: 'hello' })
    const [url, opts] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/draft')
    expect(opts.method).toBe('POST')
  })

  it('sends the payload as JSON', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true })
    const payload = { to: ['a@example.com'], subject: 'test', text: 'body' }
    await store.saveDraft(payload)
    const [, opts] = global.fetch.mock.calls[0]
    expect(JSON.parse(opts.body)).toEqual(payload)
  })

  it('does not modify the messages list', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.saveDraft({ subject: 'draft' })
    expect(store.messages).toHaveLength(3)
  })
})

describe('mail store — deleteMessages (bulk)', () => {
  it('POSTs uids to /messages/delete', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true })
    await store.deleteMessages('INBOX', [1, 2])
    const [url, opts] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders/INBOX/messages/delete')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body)).toEqual({ uids: [1, 2] })
  })

  it('removes all deleted messages from the list', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.deleteMessages('INBOX', [1, 3])
    expect(store.messages.map(m => m.uid)).toEqual([2])
  })

  it('clears currentMessage if it is among deleted', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.currentMessage = { uid: 2 }
    global.fetch.mockResolvedValue({ ok: true })
    await store.deleteMessages('INBOX', [2])
    expect(store.currentMessage).toBeNull()
  })

  it('clears selectedUids after delete', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.toggleSelect(1)
    store.toggleSelect(2)
    global.fetch.mockResolvedValue({ ok: true })
    await store.deleteMessages('INBOX', [1, 2])
    expect(store.selectedUids.size).toBe(0)
  })

  it('throws when server returns error', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.deleteMessages('INBOX', [1])).rejects.toThrow()
  })
})

describe('mail store — markReadMessages (bulk)', () => {
  it('POSTs uids and read flag to /messages/read', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true })
    await store.markReadMessages('INBOX', [1, 2], true)
    const [url, opts] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders/INBOX/messages/read')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body)).toEqual({ uids: [1, 2], read: true })
  })

  it('updates read flag on all matching messages', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.markReadMessages('INBOX', [1, 3], true)
    expect(store.messages.find(m => m.uid === 1).read).toBe(true)
    expect(store.messages.find(m => m.uid === 3).read).toBe(true)
    expect(store.messages.find(m => m.uid === 2).read).toBe(true) // was already true
  })

  it('marks messages as unread', async () => {
    const store = useMailStore()
    seedMessages(store)
    global.fetch.mockResolvedValue({ ok: true })
    await store.markReadMessages('INBOX', [2], false)
    expect(store.messages.find(m => m.uid === 2).read).toBe(false)
  })

  it('updates currentMessage when it is in the uid list', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.currentMessage = { uid: 1, read: false }
    global.fetch.mockResolvedValue({ ok: true })
    await store.markReadMessages('INBOX', [1], true)
    expect(store.currentMessage.read).toBe(true)
  })

  it('clears selectedUids after marking', async () => {
    const store = useMailStore()
    seedMessages(store)
    store.toggleSelect(1)
    global.fetch.mockResolvedValue({ ok: true })
    await store.markReadMessages('INBOX', [1], true)
    expect(store.selectedUids.size).toBe(0)
  })

  it('throws when server returns error', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.markReadMessages('INBOX', [1], true)).rejects.toThrow()
  })
})

describe('mail store — createFolder', () => {
  it('POSTs to /api/folders with the folder name', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.createFolder('Archive')
    const [url, opts] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body)).toEqual({ name: 'Archive' })
  })

  it('refreshes folder list after creation', async () => {
    const store = useMailStore()
    const newFolders = [{ name: 'INBOX' }, { name: 'Archive' }]
    global.fetch
      .mockResolvedValueOnce({ ok: true })               // POST create
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(newFolders) }) // GET folders
    await store.createFolder('Archive')
    expect(store.folders).toHaveLength(2)
  })

  it('throws when server returns error', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.createFolder('Bad')).rejects.toThrow()
  })
})

describe('mail store — renameFolder', () => {
  it('PATCHes /api/folders/{name} with new_name', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.renameFolder('OldName', 'NewName')
    const [url, opts] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders/OldName')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body)).toEqual({ new_name: 'NewName' })
  })

  it('URL-encodes folder names with special characters', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.renameFolder('My Folder/Sub', 'Archive')
    const [url] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders/My%20Folder%2FSub')
  })

  it('updates currentFolder when the renamed folder is open', async () => {
    const store = useMailStore()
    store.currentFolder = 'OldName'
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.renameFolder('OldName', 'NewName')
    expect(store.currentFolder).toBe('NewName')
  })

  it('does not change currentFolder when a different folder is renamed', async () => {
    const store = useMailStore()
    store.currentFolder = 'INBOX'
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.renameFolder('OldName', 'NewName')
    expect(store.currentFolder).toBe('INBOX')
  })

  it('throws when server returns error', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.renameFolder('A', 'B')).rejects.toThrow()
  })
})

describe('mail store — deleteFolder', () => {
  it('DELETEs /api/folders/{name}', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.deleteFolder('Archive')
    const [url, opts] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders/Archive')
    expect(opts.method).toBe('DELETE')
  })

  it('URL-encodes special characters in folder name', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.deleteFolder('My Folder/Sub')
    const [url] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/folders/My%20Folder%2FSub')
  })

  it('navigates to INBOX when the deleted folder is currently open', async () => {
    const store = useMailStore()
    store.currentFolder = 'Archive'
    const inboxMessages = [{ uid: 1, subject: 'hello' }]
    global.fetch
      .mockResolvedValueOnce({ ok: true })  // DELETE
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([]) })  // fetchFolders
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(inboxMessages) }) // fetchMessages INBOX
    await store.deleteFolder('Archive')
    expect(store.currentFolder).toBe('INBOX')
  })

  it('does not change currentFolder when a non-open folder is deleted', async () => {
    const store = useMailStore()
    store.currentFolder = 'INBOX'
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.deleteFolder('Archive')
    expect(store.currentFolder).toBe('INBOX')
  })

  it('throws when server returns error', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.deleteFolder('Archive')).rejects.toThrow()
  })
})

// ---------------------------------------------------------------------------
// threads computed
// ---------------------------------------------------------------------------

function makeMsg(overrides) {
  return {
    uid: 0,
    subject: 'Test',
    from: 'sender@example.com',
    date: new Date('2024-01-01T10:00:00Z').toISOString(),
    read: true,
    flagged: false,
    has_attachments: false,
    message_id: '',
    in_reply_to: '',
    references: '',
    ...overrides,
  }
}

describe('mail store — threads computed', () => {
  it('wraps a single message in its own thread', () => {
    const store = useMailStore()
    store.messages = [makeMsg({ uid: 1, message_id: '<a@x>' })]
    expect(store.threads).toHaveLength(1)
    expect(store.threads[0].messages).toHaveLength(1)
    expect(store.threads[0].id).toBe(1)
  })

  it('groups two messages linked by In-Reply-To into one thread', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', date: '2024-01-01T10:00:00Z' }),
      makeMsg({ uid: 2, message_id: '<b@x>', in_reply_to: '<a@x>', date: '2024-01-01T11:00:00Z' }),
    ]
    expect(store.threads).toHaveLength(1)
    expect(store.threads[0].messages).toHaveLength(2)
  })

  it('groups three messages linked via References into one thread', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', date: '2024-01-01T10:00:00Z' }),
      makeMsg({ uid: 2, message_id: '<b@x>', references: '<a@x>', date: '2024-01-01T11:00:00Z' }),
      makeMsg({ uid: 3, message_id: '<c@x>', references: '<a@x> <b@x>', date: '2024-01-01T12:00:00Z' }),
    ]
    expect(store.threads).toHaveLength(1)
    expect(store.threads[0].messages).toHaveLength(3)
  })

  it('keeps two unrelated messages as separate threads', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', subject: 'Topic A', date: '2024-01-01T10:00:00Z' }),
      makeMsg({ uid: 2, message_id: '<b@x>', subject: 'Topic B', date: '2024-01-01T11:00:00Z' }),
    ]
    expect(store.threads).toHaveLength(2)
  })

  it('does NOT group messages just because they share a subject', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', subject: 'Hello' }),
      makeMsg({ uid: 2, message_id: '<b@x>', subject: 'Hello' }),
    ]
    expect(store.threads).toHaveLength(2)
  })

  it('sorts threads newest-first by latest message date', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', date: '2024-01-01T10:00:00Z' }),
      makeMsg({ uid: 2, message_id: '<b@x>', date: '2024-01-03T10:00:00Z' }),
      makeMsg({ uid: 3, message_id: '<c@x>', date: '2024-01-02T10:00:00Z' }),
    ]
    const ids = store.threads.map(t => t.id)
    expect(ids).toEqual([2, 3, 1])
  })

  it('reports hasUnread true when any message in thread is unread', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', read: true, date: '2024-01-01T10:00:00Z' }),
      makeMsg({ uid: 2, message_id: '<b@x>', in_reply_to: '<a@x>', read: false, date: '2024-01-01T11:00:00Z' }),
    ]
    expect(store.threads[0].hasUnread).toBe(true)
  })

  it('reports hasUnread false when all messages in thread are read', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', read: true, date: '2024-01-01T10:00:00Z' }),
      makeMsg({ uid: 2, message_id: '<b@x>', in_reply_to: '<a@x>', read: true, date: '2024-01-01T11:00:00Z' }),
    ]
    expect(store.threads[0].hasUnread).toBe(false)
  })

  it('exposes the latest message as thread.latest', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', date: '2024-01-01T10:00:00Z' }),
      makeMsg({ uid: 2, message_id: '<b@x>', in_reply_to: '<a@x>', date: '2024-01-01T12:00:00Z' }),
      makeMsg({ uid: 3, message_id: '<c@x>', in_reply_to: '<b@x>', date: '2024-01-01T11:00:00Z' }),
    ]
    expect(store.threads[0].latest.uid).toBe(2)
  })

  it('returns an empty array when there are no messages', () => {
    const store = useMailStore()
    store.messages = []
    expect(store.threads).toEqual([])
  })

  it('handles messages missing message_id gracefully', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '' }),
      makeMsg({ uid: 2, message_id: '' }),
    ]
    // No crash; two separate threads since there are no IDs to link them.
    expect(store.threads).toHaveLength(2)
  })

  it('does not infinitely recurse on a self-referencing message', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', references: '<a@x>' }),
    ]
    expect(() => store.threads).not.toThrow()
    expect(store.threads).toHaveLength(1)
  })

  it('does not infinitely recurse on a circular reference chain', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, message_id: '<a@x>', in_reply_to: '<b@x>' }),
      makeMsg({ uid: 2, message_id: '<b@x>', in_reply_to: '<a@x>' }),
    ]
    expect(() => store.threads).not.toThrow()
    // Cycle is broken at the visited check; each message becomes its own root.
    // The important thing is no crash and a finite result.
    expect(store.threads.length).toBeGreaterThan(0)
  })
})
