import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '../api'

export const useFiltersStore = defineStore('filters', () => {
  const filters = ref([])
  const loaded = ref(false)

  async function fetchFilters() {
    const res = await fetch('/api/filters')
    if (!res.ok) return
    filters.value = await res.json()
    loaded.value = true
  }

  async function createFilter(f) {
    const res = await apiFetch('/api/filters', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(f),
    })
    if (!res.ok) throw new Error('Failed to create filter')
    const created = await res.json()
    filters.value.push(created)
    return created
  }

  async function updateFilter(id, f) {
    const res = await apiFetch(`/api/filters/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(f),
    })
    if (!res.ok) throw new Error('Failed to update filter')
    const idx = filters.value.findIndex(x => x.id === id)
    if (idx >= 0) filters.value[idx] = { ...filters.value[idx], ...f, id }
  }

  async function deleteFilter(id) {
    const res = await apiFetch(`/api/filters/${id}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('Failed to delete filter')
    filters.value = filters.value.filter(x => x.id !== id)
  }

  async function reorderFilters(ids) {
    const res = await apiFetch('/api/filters/reorder', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids }),
    })
    if (!res.ok) throw new Error('Failed to reorder filters')
  }

  async function applyFilters(folder) {
    const res = await apiFetch('/api/filters/apply', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ folder }),
    })
    if (!res.ok) throw new Error('Failed to apply filters')
    return await res.json()
  }

  return { filters, loaded, fetchFilters, createFilter, updateFilter, deleteFilter, reorderFilters, applyFilters }
})
