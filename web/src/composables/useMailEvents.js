import { onMounted, onUnmounted, watch } from 'vue'
import { useMailStore } from '../stores/mail'
import { useSettingsStore } from '../stores/settings'

/**
 * useMailEvents opens a Server-Sent Events connection to /api/events
 * and reacts to new mail notifications pushed from the Go backend via IMAP IDLE.
 * EventSource automatically reconnects on disconnect.
 *
 * A periodic poll runs in parallel as a safety net for when IDLE drops or the
 * server does not support it. The interval is configured in Settings.
 */
export function useMailEvents() {
  const mail = useMailStore()
  const settings = useSettingsStore()
  let es = null
  let pollTimer = null

  function poll() {
    mail.fetchFolders()
    mail.fetchMessages(mail.currentFolder)
  }

  function startPoll(intervalSecs) {
    clearInterval(pollTimer)
    if (intervalSecs > 0) {
      pollTimer = setInterval(poll, intervalSecs * 1000)
    }
  }

  onMounted(() => {
    es = new EventSource('/api/events')

    es.addEventListener('mailbox', (e) => {
      const data = JSON.parse(e.data || '{}')
      const folder = data.folder || mail.currentFolder
      if (folder === mail.currentFolder) {
        mail.fetchMessages(mail.currentFolder)
      }
      // Update folder unread count if folders are loaded
      const f = mail.folders.find(f => f.name === folder)
      if (f && data.unseen != null) f.unseen = data.unseen
    })

    es.onerror = () => {
      // EventSource handles reconnection automatically — no action needed
    }

    startPoll(settings.pollInterval)
  })

  // Restart the timer whenever the user changes the poll interval in settings.
  watch(() => settings.pollInterval, startPoll)

  onUnmounted(() => {
    es?.close()
    clearInterval(pollTimer)
  })
}
