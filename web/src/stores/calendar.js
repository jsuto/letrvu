import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '../api'

export const useCalendarStore = defineStore('calendar', () => {
  const events = ref([])
  const loading = ref(false)

  async function fetchEvents(from, to) {
    loading.value = true
    try {
      const params = new URLSearchParams({
        from: from.toISOString(),
        to: to.toISOString(),
      })
      const res = await fetch(`/api/calendar/events?${params}`)
      if (!res.ok) throw new Error('Failed to load events')
      events.value = await res.json()
    } finally {
      loading.value = false
    }
  }

  async function createEvent(data) {
    const res = await apiFetch('/api/calendar/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    if (!res.ok) throw new Error('Failed to create event')
    const ev = await res.json()
    events.value.push(ev)
    return ev
  }

  async function updateEvent(id, data) {
    const res = await apiFetch(`/api/calendar/events/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    if (!res.ok) throw new Error('Failed to update event')
    const ev = await res.json()
    const idx = events.value.findIndex(e => e.id === id)
    if (idx !== -1) events.value[idx] = ev
    return ev
  }

  async function deleteEvent(id) {
    const res = await apiFetch(`/api/calendar/events/${id}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('Failed to delete event')
    events.value = events.value.filter(e => e.id !== id)
  }

  async function importFromInvite(ical) {
    const res = await apiFetch('/api/calendar/events/import-invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ical }),
    })
    if (!res.ok) throw new Error('Failed to import invite')
    return res.json()
  }

  return { events, loading, fetchEvents, createEvent, updateEvent, deleteEvent, importFromInvite }
})
