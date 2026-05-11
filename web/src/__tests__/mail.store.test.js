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
