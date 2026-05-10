import { describe, it, expect, beforeEach, vi } from 'vitest'
import { apiFetch } from '../api.js'

// apiFetch reads document.cookie directly, so we control it via Object.defineProperty.
function setCookie(value) {
  Object.defineProperty(document, 'cookie', {
    get: () => value,
    configurable: true,
  })
}

beforeEach(() => {
  setCookie('')
  global.fetch = vi.fn().mockResolvedValue({ ok: true })
})

describe('apiFetch — CSRF header', () => {
  it('does not add X-CSRF-Token for GET', async () => {
    setCookie('letrvu_csrf=secret123')
    await apiFetch('/api/test', { method: 'GET' })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBeNull()
  })

  it('does not add X-CSRF-Token for HEAD', async () => {
    setCookie('letrvu_csrf=secret123')
    await apiFetch('/api/test', { method: 'HEAD' })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBeNull()
  })

  it('does not add X-CSRF-Token for OPTIONS', async () => {
    setCookie('letrvu_csrf=secret123')
    await apiFetch('/api/test', { method: 'OPTIONS' })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBeNull()
  })

  it('adds X-CSRF-Token for POST', async () => {
    setCookie('letrvu_csrf=mytoken')
    await apiFetch('/api/test', { method: 'POST' })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBe('mytoken')
  })

  it('adds X-CSRF-Token for PUT', async () => {
    setCookie('letrvu_csrf=mytoken')
    await apiFetch('/api/test', { method: 'PUT' })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBe('mytoken')
  })

  it('adds X-CSRF-Token for PATCH', async () => {
    setCookie('letrvu_csrf=mytoken')
    await apiFetch('/api/test', { method: 'PATCH' })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBe('mytoken')
  })

  it('adds X-CSRF-Token for DELETE', async () => {
    setCookie('letrvu_csrf=mytoken')
    await apiFetch('/api/test', { method: 'DELETE' })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBe('mytoken')
  })

  it('defaults to GET when no method provided', async () => {
    setCookie('letrvu_csrf=mytoken')
    await apiFetch('/api/test')
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBeNull()
  })

  it('sends empty string when cookie is absent', async () => {
    setCookie('')
    await apiFetch('/api/test', { method: 'POST' })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBe('')
  })

  it('parses token correctly when multiple cookies present', async () => {
    setCookie('session=abc; letrvu_csrf=tok42; other=xyz')
    await apiFetch('/api/test', { method: 'POST' })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('X-CSRF-Token')).toBe('tok42')
  })

  it('preserves caller-supplied headers', async () => {
    setCookie('letrvu_csrf=tok')
    await apiFetch('/api/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    })
    const headers = global.fetch.mock.calls[0][1].headers
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(headers.get('X-CSRF-Token')).toBe('tok')
  })

  it('passes the correct URL to fetch', async () => {
    await apiFetch('/api/something', { method: 'GET' })
    expect(global.fetch.mock.calls[0][0]).toBe('/api/something')
  })
})
