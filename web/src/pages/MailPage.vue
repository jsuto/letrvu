<template>
  <div class="mail-layout">
    <aside class="sidebar">
      <FolderList />
    </aside>
    <section class="message-list-panel">
      <MessageList />
    </section>
    <main class="message-view-panel">
      <ThreadView v-if="mail.currentThread" />
      <MessageView v-else ref="messageView" />
    </main>
    <ComposeModal ref="composeModal" />
    <KeyboardShortcutsModal />
  </div>
</template>

<script setup>
import { ref, provide, onMounted, onUnmounted } from 'vue'
import { useMailStore } from '../stores/mail'
import { useSettingsStore } from '../stores/settings'
import { useMailEvents } from '../composables/useMailEvents'
import { useTabTitle } from '../composables/useTabTitle'
import { useCalendarReminders } from '../composables/useCalendarReminders'
import FolderList from '../components/FolderList.vue'
import MessageList from '../components/MessageList.vue'
import MessageView from '../components/MessageView.vue'
import ThreadView from '../components/ThreadView.vue'
import ComposeModal from '../components/ComposeModal.vue'
import KeyboardShortcutsModal from '../components/KeyboardShortcutsModal.vue'

const mail = useMailStore()
const settings = useSettingsStore()
const composeModal = ref(null)
const messageView = ref(null)

// Provide compose modal to all descendants so FolderList and MessageView can open it.
provide('compose', composeModal)

useMailEvents()
useTabTitle()
useCalendarReminders()

onMounted(async () => {
  await Promise.all([mail.fetchFolders(), settings.fetchSettings()])
  if (!mail.messages.length) {
    await mail.fetchMessages(mail.currentFolder)
  }
  document.addEventListener('keydown', onKeydown)
})
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

function onKeydown(e) {
  // Ignore shortcuts when the user is typing, a modal is open, or a modifier key is held.
  const tag = document.activeElement?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || document.activeElement?.isContentEditable) return
  if (composeModal.value?.visible) return
  if (e.metaKey || e.ctrlKey || e.altKey) return

  switch (e.key) {
    case 'c':
      e.preventDefault()
      composeModal.value?.open({})
      break
    case 'r':
      if (mail.currentMessage) {
        e.preventDefault()
        messageView.value?.reply()
      }
      break
    case 'd':
      if (mail.currentMessage) {
        e.preventDefault()
        messageView.value?.remove()
      }
      break
    case 'n':
      e.preventDefault()
      navigateMessage(1)
      break
    case 'p':
      e.preventDefault()
      navigateMessage(-1)
      break
  }
}

function navigateMessage(delta) {
  const msgs = mail.messages
  if (!msgs.length) return
  const idx = mail.currentMessage
    ? msgs.findIndex(m => m.uid === mail.currentMessage.uid)
    : -1
  const next = idx === -1
    ? (delta > 0 ? 0 : msgs.length - 1)
    : Math.max(0, Math.min(msgs.length - 1, idx + delta))
  const msg = msgs[next]
  if (msg && msg.uid !== mail.currentMessage?.uid) {
    mail.fetchMessage(mail.currentFolder, msg.uid)
    if (!msg.read) mail.markRead(mail.currentFolder, msg.uid, true)
  }
}
</script>

<style scoped>
.mail-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  position: relative;
}
.sidebar {
  width: var(--sidebar-width);
  flex-shrink: 0;
  border-right: 0.5px solid var(--color-border);
  background: var(--color-surface);
  overflow: hidden;
}
.message-list-panel {
  width: var(--list-width);
  flex-shrink: 0;
  border-right: 0.5px solid var(--color-border);
  background: var(--color-surface);
  overflow-y: auto;
}
.message-view-panel {
  flex: 1;
  overflow-y: auto;
  background: var(--color-bg);
}
</style>
