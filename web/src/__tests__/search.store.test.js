import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMailStore } from '../stores/mail.js'

beforeEach(() => {
  setActivePinia(createPinia())
  global.fetch = vi.fn()
})

function makeMsg(overrides) {
  return {
    uid: 1,
    subject: 'Test',
    from: 'a@example.com',
    date: new Date('2024-01-01T10:00:00Z').toISOString(),
    read: false,
    flagged: false,
    has_attachments: false,
    message_id: '',
    in_reply_to: '',
    references: '',
    folder: 'INBOX',
    ...overrides,
  }
}

// ---------------------------------------------------------------------------
// searchAllFolders — state transitions
// ---------------------------------------------------------------------------

describe('mail store — searchAllFolders', () => {
  it('sets globalSearchMode to true', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.searchAllFolders('budget')
    expect(store.globalSearchMode).toBe(true)
  })

  it('sets loading to false after fetch completes', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.searchAllFolders('budget')
    expect(store.loading).toBe(false)
  })

  it('populates messages with API response', async () => {
    const store = useMailStore()
    const results = [
      makeMsg({ uid: 1, subject: 'Q1 budget', folder: 'INBOX' }),
      makeMsg({ uid: 5, subject: 'Budget review', folder: 'Archive' }),
    ]
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve(results) })
    await store.searchAllFolders('budget')
    expect(store.messages).toHaveLength(2)
    expect(store.messages[0].subject).toBe('Q1 budget')
    expect(store.messages[1].folder).toBe('Archive')
  })

  it('clears selectedUids before fetching', async () => {
    const store = useMailStore()
    store.toggleSelect(99)
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.searchAllFolders('anything')
    expect(store.selectedUids.size).toBe(0)
  })

  it('clears currentThread before fetching', async () => {
    const store = useMailStore()
    store.currentThread = { id: 1, messages: [] }
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.searchAllFolders('anything')
    expect(store.currentThread).toBeNull()
  })

  it('calls GET /api/search with encoded query', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.searchAllFolders('hello world')
    const [url] = global.fetch.mock.calls[0]
    expect(url).toBe('/api/search?q=hello%20world')
  })

  it('does not change messages when server returns not-ok', async () => {
    const store = useMailStore()
    store.messages = [makeMsg({ uid: 10 })]
    global.fetch.mockResolvedValue({ ok: false })
    await store.searchAllFolders('query')
    // messages unchanged when response is not ok
    expect(store.messages).toHaveLength(1)
  })

  it('still sets loading to false when server returns not-ok', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: false })
    await store.searchAllFolders('query')
    expect(store.loading).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// fetchMessages — resets globalSearchMode
// ---------------------------------------------------------------------------

describe('mail store — fetchMessages resets globalSearchMode', () => {
  it('sets globalSearchMode to false after fetchMessages', async () => {
    const store = useMailStore()
    // Simulate being in global search mode first.
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.searchAllFolders('budget')
    expect(store.globalSearchMode).toBe(true)

    await store.fetchMessages('INBOX')
    expect(store.globalSearchMode).toBe(false)
  })

  it('does not affect messages or folder on global search error then folder switch', async () => {
    const store = useMailStore()
    global.fetch
      .mockResolvedValueOnce({ ok: false }) // global search fails
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([makeMsg({ uid: 1 })]) })
    await store.searchAllFolders('bad')
    await store.fetchMessages('Sent')
    expect(store.globalSearchMode).toBe(false)
    expect(store.currentFolder).toBe('Sent')
    expect(store.messages).toHaveLength(1)
  })
})

// ---------------------------------------------------------------------------
// searchMessages — keeps globalSearchMode false
// ---------------------------------------------------------------------------

describe('mail store — searchMessages keeps globalSearchMode false', () => {
  it('does not set globalSearchMode to true', async () => {
    const store = useMailStore()
    global.fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    await store.searchMessages('INBOX', 'hello')
    expect(store.globalSearchMode).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// threads — works with cross-folder messages
// ---------------------------------------------------------------------------

describe('mail store — threads with cross-folder results', () => {
  it('threads messages from different folders if they share message-id references', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, folder: 'INBOX', message_id: '<a@x>', date: '2024-01-01T10:00:00Z' }),
      makeMsg({ uid: 2, folder: 'Sent', message_id: '<b@x>', in_reply_to: '<a@x>', date: '2024-01-01T11:00:00Z' }),
    ]
    expect(store.threads).toHaveLength(1)
    expect(store.threads[0].messages).toHaveLength(2)
  })

  it('exposes folder on each message in a thread', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, folder: 'INBOX', message_id: '<a@x>' }),
      makeMsg({ uid: 2, folder: 'Sent', message_id: '<b@x>', in_reply_to: '<a@x>' }),
    ]
    const msgs = store.threads[0].messages
    const folders = msgs.map(m => m.folder)
    expect(folders).toContain('INBOX')
    expect(folders).toContain('Sent')
  })

  it('keeps messages from unrelated threads separate even across folders', () => {
    const store = useMailStore()
    store.messages = [
      makeMsg({ uid: 1, folder: 'INBOX', message_id: '<a@x>', subject: 'Topic A' }),
      makeMsg({ uid: 2, folder: 'Archive', message_id: '<b@x>', subject: 'Topic B' }),
    ]
    expect(store.threads).toHaveLength(2)
  })
})
