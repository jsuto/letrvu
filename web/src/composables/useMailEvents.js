import { onMounted, onUnmounted } from 'vue'
import { useMailStore } from '../stores/mail'

/**
 * useMailEvents opens a Server-Sent Events connection to /api/events
 * and reacts to new mail notifications pushed from the Go backend via IMAP IDLE.
 * EventSource automatically reconnects on disconnect.
 */
export function useMailEvents() {
  const mail = useMailStore()
  let es = null

  onMounted(() => {
    es = new EventSource('/api/events')

    es.addEventListener('new_mail', (e) => {
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
  })

  onUnmounted(() => {
    es?.close()
  })
}
