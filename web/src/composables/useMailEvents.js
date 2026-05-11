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

  function canNotify() {
    return (
      settings.notificationsEnabled &&
      typeof Notification !== 'undefined' &&
      Notification.permission === 'granted'
    )
  }

  function fireNotification(msg) {
    const n = new Notification(msg.subject || '(no subject)', {
      body: msg.from || '',
      tag: `letrvu-${msg.uid}`,
    })
    n.onclick = () => window.focus()
  }

  onMounted(() => {
    es = new EventSource('/api/events')

    es.addEventListener('mailbox', async (e) => {
      const data = JSON.parse(e.data || '{}')
      const folder = data.folder || mail.currentFolder

      // Update folder unread count
      const f = mail.folders.find(f => f.name === folder)
      if (f && data.unseen != null) f.unseen = data.unseen

      if (folder === mail.currentFolder) {
        if (folder === 'INBOX' && canNotify()) {
          // Capture known UIDs, fetch, then notify for anything new.
          const knownUids = new Set(mail.messages.map(m => m.uid))
          await mail.fetchMessages('INBOX')
          for (const msg of mail.messages) {
            if (!knownUids.has(msg.uid)) fireNotification(msg)
          }
        } else {
          mail.fetchMessages(mail.currentFolder)
        }
      } else if (folder === 'INBOX' && canNotify()) {
        // New mail in INBOX but user is viewing another folder — generic notification.
        new Notification('New mail', {
          body: 'You have new messages in INBOX',
          tag: 'letrvu-inbox-generic',
        }).onclick = () => window.focus()
      }
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
