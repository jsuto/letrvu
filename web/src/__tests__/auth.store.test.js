import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../stores/auth.js'

beforeEach(() => {
  setActivePinia(createPinia())
  global.fetch = vi.fn()
})

describe('auth store — initial state', () => {
  it('starts as not logged in', () => {
    const store = useAuthStore()
    expect(store.loggedIn).toBe(false)
  })
})

describe('auth store — checkSession', () => {
  it('sets loggedIn true when /api/folders returns ok', async () => {
    const store = useAuthStore()
    global.fetch.mockResolvedValue({ ok: true })
    const result = await store.checkSession()
    expect(store.loggedIn).toBe(true)
    expect(result).toBe(true)
  })

  it('sets loggedIn false when /api/folders returns not ok', async () => {
    const store = useAuthStore()
    global.fetch.mockResolvedValue({ ok: false })
    const result = await store.checkSession()
    expect(store.loggedIn).toBe(false)
    expect(result).toBe(false)
  })

  it('sets loggedIn false when fetch throws', async () => {
    const store = useAuthStore()
    global.fetch.mockRejectedValue(new Error('network error'))
    const result = await store.checkSession()
    expect(store.loggedIn).toBe(false)
    expect(result).toBe(false)
  })

  it('returns the current loggedIn value', async () => {
    const store = useAuthStore()
    global.fetch.mockResolvedValue({ ok: true })
    const returned = await store.checkSession()
    expect(returned).toBe(store.loggedIn)
  })
})

describe('auth store — login', () => {
  it('sets loggedIn true on success', async () => {
    const store = useAuthStore()
    global.fetch.mockResolvedValue({ ok: true })
    await store.login({ username: 'alice@example.com', password: 'secret' })
    expect(store.loggedIn).toBe(true)
  })

  it('throws and does not set loggedIn on failure', async () => {
    const store = useAuthStore()
    global.fetch.mockResolvedValue({ ok: false })
    await expect(
      store.login({ username: 'alice@example.com', password: 'wrong' })
    ).rejects.toThrow('Login failed')
    expect(store.loggedIn).toBe(false)
  })

  it('sends credentials as JSON', async () => {
    const store = useAuthStore()
    global.fetch.mockResolvedValue({ ok: true })
    await store.login({
      imapHost: 'imap.example.com',
      imapPort: 993,
      smtpHost: 'smtp.example.com',
      smtpPort: 587,
      username: 'alice@example.com',
      password: 'secret',
    })
    const body = JSON.parse(global.fetch.mock.calls[0][1].body)
    expect(body.imap_host).toBe('imap.example.com')
    expect(body.smtp_host).toBe('smtp.example.com')
    expect(body.username).toBe('alice@example.com')
    expect(body.password).toBe('secret')
  })

  it('defaults imap_port to 993 when not provided', async () => {
    const store = useAuthStore()
    global.fetch.mockResolvedValue({ ok: true })
    await store.login({ username: 'alice@example.com', password: 'x' })
    const body = JSON.parse(global.fetch.mock.calls[0][1].body)
    expect(body.imap_port).toBe(993)
  })

  it('defaults smtp_port to 587 when not provided', async () => {
    const store = useAuthStore()
    global.fetch.mockResolvedValue({ ok: true })
    await store.login({ username: 'alice@example.com', password: 'x' })
    const body = JSON.parse(global.fetch.mock.calls[0][1].body)
    expect(body.smtp_port).toBe(587)
  })
})

describe('auth store — logout', () => {
  it('sets loggedIn false', async () => {
    const store = useAuthStore()
    // Start logged in
    global.fetch.mockResolvedValue({ ok: true })
    await store.login({ username: 'alice@example.com', password: 'x' })
    expect(store.loggedIn).toBe(true)

    global.fetch.mockResolvedValue({ ok: true })
    await store.logout()
    expect(store.loggedIn).toBe(false)
  })

  it('sets loggedIn false even when server returns error', async () => {
    const store = useAuthStore()
    store.loggedIn = true
    global.fetch.mockResolvedValue({ ok: false })
    await store.logout()
    expect(store.loggedIn).toBe(false)
  })
})
