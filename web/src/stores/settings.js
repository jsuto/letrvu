import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
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

  // The authenticated login address, injected by the server into /api/settings.
  const username = computed(() => settings.value.username ?? '')

  // Domains that belong to this organisation, injected by the server.
  // Empty array means the feature is not configured.
  const internalDomains = computed(() => settings.value.internal_domains ?? [])

  // Identities are stored as a JSON array of { name, email } objects.
  const identities = computed(() => {
    try {
      return JSON.parse(settings.value.identities || '[]')
    } catch {
      return []
    }
  })

  // All selectable From options: configured identities first, then the bare
  // login address as a fallback so there is always at least one choice.
  const fromOptions = computed(() => {
    const opts = identities.value.map(id => ({
      label: id.name ? `${id.name} <${id.email}>` : id.email,
      name: id.name,
      email: id.email,
    }))
    const u = username.value
    if (u && !identities.value.some(id => id.email === u)) {
      opts.push({ label: u, name: '', email: u })
    }
    return opts
  })

  // Poll interval in seconds. 0 = disabled. Default 120 (2 minutes).
  const pollInterval = computed(() => {
    const v = parseInt(settings.value.poll_interval, 10)
    return isNaN(v) ? 120 : v
  })

  // Whether the user has opted into desktop notifications.
  const notificationsEnabled = computed(() => settings.value.notifications_enabled === 'true')

  // Minutes before a calendar event to fire a reminder notification. 0 = off.
  const reminderMinutes = computed(() => {
    const v = parseInt(settings.value.calendar_reminder_minutes, 10)
    return isNaN(v) ? 30 : v
  })

  return { settings, loaded, fetchSettings, saveSettings, username, identities, fromOptions, internalDomains, pollInterval, notificationsEnabled, reminderMinutes }
})
