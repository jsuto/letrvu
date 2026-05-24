import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '../api'

export const useTemplatesStore = defineStore('templates', () => {
  const templates = ref([])
  const loaded = ref(false)

  async function fetchTemplates() {
    const res = await fetch('/api/templates')
    if (!res.ok) return
    templates.value = await res.json()
    loaded.value = true
  }

  async function createTemplate(t) {
    const res = await apiFetch('/api/templates', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(t),
    })
    if (!res.ok) throw new Error('Failed to create template')
    const created = await res.json()
    templates.value.push(created)
    templates.value.sort((a, b) => a.name.localeCompare(b.name))
    return created
  }

  async function updateTemplate(id, t) {
    const res = await apiFetch(`/api/templates/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(t),
    })
    if (!res.ok) throw new Error('Failed to update template')
    const idx = templates.value.findIndex(x => x.id === id)
    if (idx >= 0) templates.value[idx] = { ...t, id }
    templates.value.sort((a, b) => a.name.localeCompare(b.name))
  }

  async function deleteTemplate(id) {
    const res = await apiFetch(`/api/templates/${id}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('Failed to delete template')
    templates.value = templates.value.filter(x => x.id !== id)
  }

  return { templates, loaded, fetchTemplates, createTemplate, updateTemplate, deleteTemplate }
})
