import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useFiltersStore } from '../stores/filters'

// Mock the api module
vi.mock('../api', () => ({
  apiFetch: vi.fn(),
}))

import { apiFetch } from '../api'

describe('useFiltersStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
  })

  it('fetchFilters populates the list', async () => {
    const mockFilters = [
      { id: 1, name: 'Invoices', match_all: true, conditions: [], actions: [], enabled: true, position: 0 },
    ]
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockFilters,
    })

    const store = useFiltersStore()
    await store.fetchFilters()
    expect(store.filters).toEqual(mockFilters)
    expect(store.loaded).toBe(true)
  })

  it('createFilter appends to list', async () => {
    const newFilter = { id: 2, name: 'Spam', match_all: false, conditions: [], actions: [], enabled: true }
    apiFetch.mockResolvedValue({ ok: true, json: async () => newFilter })

    const store = useFiltersStore()
    const result = await store.createFilter({ name: 'Spam', match_all: false })
    expect(store.filters).toContainEqual(newFilter)
    expect(result).toEqual(newFilter)
  })

  it('deleteFilter removes from list', async () => {
    const store = useFiltersStore()
    store.filters = [{ id: 1, name: 'Old', enabled: true }]
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })

    await store.deleteFilter(1)
    expect(store.filters).toHaveLength(0)
  })

  it('updateFilter updates in place', async () => {
    const store = useFiltersStore()
    store.filters = [{ id: 1, name: 'Old', enabled: true, conditions: [], actions: [] }]
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })

    await store.updateFilter(1, { name: 'New', enabled: false })
    expect(store.filters[0].name).toBe('New')
    expect(store.filters[0].enabled).toBe(false)
  })

  it('throws on createFilter failure', async () => {
    apiFetch.mockResolvedValue({ ok: false })
    const store = useFiltersStore()
    await expect(store.createFilter({ name: 'X' })).rejects.toThrow('Failed to create filter')
  })
})
