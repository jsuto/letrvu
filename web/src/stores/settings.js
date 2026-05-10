import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '../api'

export const useSettingsStore = defineStore('settings', () => {
  const settings = ref({})
  const loaded = ref(false)

  async function fetchSettings() {
    const res = await fetch('/api/settings')
    if (!res.ok) return
    settings.value = await res.json()
    loaded.value = true
  }

  async function saveSettings(values) {
    const res = await apiFetch('/api/settings', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(values),
    })
    if (!res.ok) throw new Error('Failed to save settings')
    Object.assign(settings.value, values)
  }

  return { settings, loaded, fetchSettings, saveSettings }
})
