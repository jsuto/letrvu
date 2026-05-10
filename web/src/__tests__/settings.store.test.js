import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSettingsStore } from '../stores/settings.js'

beforeEach(() => {
  setActivePinia(createPinia())
  global.fetch = vi.fn()
})

describe('settings store — computed: username', () => {
  it('returns empty string when settings not loaded', () => {
    const store = useSettingsStore()
    expect(store.username).toBe('')
  })

  it('returns username from settings', () => {
    const store = useSettingsStore()
    store.settings.username = 'alice@example.com'
    expect(store.username).toBe('alice@example.com')
  })
})

describe('settings store — computed: identities', () => {
  it('returns empty array when identities not set', () => {
    const store = useSettingsStore()
    expect(store.identities).toEqual([])
  })

  it('parses valid JSON array', () => {
    const store = useSettingsStore()
    store.settings.identities = JSON.stringify([
      { name: 'Alice', email: 'alice@example.com' },
      { name: 'Work', email: 'alice@work.com' },
    ])
    expect(store.identities).toHaveLength(2)
    expect(store.identities[0].name).toBe('Alice')
    expect(store.identities[1].email).toBe('alice@work.com')
  })

  it('returns empty array for invalid JSON', () => {
    const store = useSettingsStore()
    store.settings.identities = 'not-valid-json'
    expect(store.identities).toEqual([])
  })

  it('returns empty array for empty string', () => {
    const store = useSettingsStore()
    store.settings.identities = ''
    expect(store.identities).toEqual([])
  })
})

describe('settings store — computed: fromOptions', () => {
  it('includes bare username as fallback when no identities', () => {
    const store = useSettingsStore()
    store.settings.username = 'alice@example.com'
    store.settings.identities = '[]'
    expect(store.fromOptions).toHaveLength(1)
    expect(store.fromOptions[0].email).toBe('alice@example.com')
    expect(store.fromOptions[0].label).toBe('alice@example.com')
    expect(store.fromOptions[0].name).toBe('')
  })

  it('formats identity label as "Name <email>" when name is set', () => {
    const store = useSettingsStore()
    store.settings.username = 'alice@example.com'
    store.settings.identities = JSON.stringify([
      { name: 'Alice Smith', email: 'alice@example.com' },
    ])
    expect(store.fromOptions[0].label).toBe('Alice Smith <alice@example.com>')
  })

  it('formats identity label as bare email when name is empty', () => {
    const store = useSettingsStore()
    store.settings.username = 'alice@example.com'
    store.settings.identities = JSON.stringify([
      { name: '', email: 'alice@example.com' },
    ])
    expect(store.fromOptions[0].label).toBe('alice@example.com')
  })

  it('does not duplicate username if already in identities', () => {
    const store = useSettingsStore()
    store.settings.username = 'alice@example.com'
    store.settings.identities = JSON.stringify([
      { name: 'Alice', email: 'alice@example.com' },
    ])
    // username is already covered by the identity — should not be appended
    expect(store.fromOptions).toHaveLength(1)
  })

  it('appends username fallback when not covered by any identity', () => {
    const store = useSettingsStore()
    store.settings.username = 'alice@example.com'
    store.settings.identities = JSON.stringify([
      { name: 'Alias', email: 'alias@example.com' },
    ])
    expect(store.fromOptions).toHaveLength(2)
    expect(store.fromOptions[1].email).toBe('alice@example.com')
  })

  it('returns no options when username is empty and no identities', () => {
    const store = useSettingsStore()
    store.settings.username = ''
    store.settings.identities = '[]'
    expect(store.fromOptions).toHaveLength(0)
  })

  it('identities appear before username fallback', () => {
    const store = useSettingsStore()
    store.settings.username = 'alice@example.com'
    store.settings.identities = JSON.stringify([
      { name: 'Work', email: 'alice@work.com' },
    ])
    expect(store.fromOptions[0].email).toBe('alice@work.com')
    expect(store.fromOptions[1].email).toBe('alice@example.com')
  })
})

describe('settings store — fetchSettings', () => {
  it('sets loaded to true on success', async () => {
    const store = useSettingsStore()
    global.fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ username: 'alice@example.com', display_name: 'Alice' }),
    })
    await store.fetchSettings()
    expect(store.loaded).toBe(true)
    expect(store.settings.username).toBe('alice@example.com')
  })

  it('does not set loaded on failure', async () => {
    const store = useSettingsStore()
    global.fetch.mockResolvedValue({ ok: false })
    await store.fetchSettings()
    expect(store.loaded).toBe(false)
  })
})

describe('settings store — saveSettings', () => {
  it('merges saved values into settings', async () => {
    const store = useSettingsStore()
    store.settings.display_name = 'Old Name'
    // mock apiFetch — it calls fetch under the hood
    global.fetch.mockResolvedValue({ ok: true })
    await store.saveSettings({ display_name: 'New Name' })
    expect(store.settings.display_name).toBe('New Name')
  })

  it('throws when response is not ok', async () => {
    const store = useSettingsStore()
    global.fetch.mockResolvedValue({ ok: false })
    await expect(store.saveSettings({ display_name: 'x' })).rejects.toThrow()
  })
})
