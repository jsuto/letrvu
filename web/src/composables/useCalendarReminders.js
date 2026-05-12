import { onMounted, onUnmounted } from 'vue'
import { useSettingsStore } from '../stores/settings'

/**
 * useCalendarReminders polls the calendar API every minute and fires a desktop
 * notification when an event is about to start within the configured reminder
 * window (Settings → Event reminders).
 *
 * Each event occurrence is notified at most once per session, tracked by a
 * key of "id:starts_at". The composable should be mounted once in MailPage so
 * it runs regardless of which page the user is on.
 */
export function useCalendarReminders() {
  const settings = useSettingsStore()
  const notified = new Set()
  let timer = null

  function canNotify() {
    return (
      settings.notificationsEnabled &&
      typeof Notification !== 'undefined' &&
      Notification.permission === 'granted'
    )
  }

  async function check() {
    if (!canNotify()) return
    const minutes = settings.reminderMinutes
    if (minutes <= 0) return

    const now = new Date()
    // Fetch events for the next 25 hours to cover any reminder window up to 1 hour.
    const to = new Date(now.getTime() + 25 * 60 * 60 * 1000)
    const params = new URLSearchParams({ from: now.toISOString(), to: to.toISOString() })

    let events
    try {
      const res = await fetch(`/api/calendar/events?${params}`)
      if (!res.ok) return
      events = await res.json()
    } catch {
      return
    }

    for (const ev of events) {
      const start = new Date(ev.starts_at)
      const minsUntil = (start - now) / 60000
      const key = `${ev.id}:${ev.starts_at}`

      // Fire when the event falls inside the reminder window (with 1-minute
      // grace on each side to absorb timer drift between 60-second ticks).
      if (minsUntil >= -1 && minsUntil <= minutes + 1 && !notified.has(key)) {
        notified.add(key)
        const body = [
          `Starting in ${Math.max(0, Math.round(minsUntil))} minute${Math.round(minsUntil) === 1 ? '' : 's'}`,
          ev.location ? `📍 ${ev.location}` : '',
        ].filter(Boolean).join(' · ')
        new Notification(ev.title || '(no title)', {
          body,
          icon: '/assets/letrvu-logo-stacked.svg',
          tag: key, // prevents duplicate system-level popups
        })
      }
    }
  }

  onMounted(() => {
    check()
    timer = setInterval(check, 60 * 1000)
  })
  onUnmounted(() => clearInterval(timer))
}
