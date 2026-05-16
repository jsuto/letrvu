<template>
  <div class="mail-layout">
    <aside class="sidebar" :class="{ 'mobile-hidden': mobilePanel !== 'folders' }">
      <FolderList />
    </aside>
    <section class="message-list-panel" :class="{ 'mobile-hidden': mobilePanel !== 'list' }">
      <MessageList />
    </section>
    <main class="message-view-panel" :class="{ 'mobile-hidden': mobilePanel !== 'view' }">
      <ThreadView v-if="mail.currentThread" />
      <MessageView v-else ref="messageView" />
    </main>

    <!-- Mobile bottom navigation -->
    <nav class="mobile-nav">
      <button
        :class="['mobile-nav-btn', mobilePanel === 'folders' ? 'mobile-nav-active' : '']"
        @click="mobilePanel = 'folders'"
      >
        <span class="mobile-nav-icon">☰</span>
        <span class="mobile-nav-label">Folders</span>
      </button>
      <button
        :class="['mobile-nav-btn', mobilePanel === 'list' ? 'mobile-nav-active' : '']"
        @click="mobilePanel = 'list'"
      >
        <span class="mobile-nav-icon">✉</span>
        <span class="mobile-nav-label">Messages</span>
      </button>
      <button
        :class="['mobile-nav-btn', mobilePanel === 'view' ? 'mobile-nav-active' : '']"
        :disabled="!mail.currentMessage && !mail.currentThread"
        @click="mobilePanel = 'view'"
      >
        <span class="mobile-nav-icon">📖</span>
        <span class="mobile-nav-label">Read</span>
      </button>
    </nav>

    <ComposeModal ref="composeModal" />
    <KeyboardShortcutsModal />
  </div>
</template>

<script setup>
import { ref, provide, watch, onMounted, onUnmounted } from 'vue'
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
const mobilePanel = ref('list')

// Provide compose modal to all descendants so FolderList and MessageView can open it.
provide('compose', composeModal)
provide('setMobilePanel', (panel) => { mobilePanel.value = panel })

// Auto-advance panels on mobile when store state changes.
watch(() => mail.currentMessage, (msg) => { if (msg) mobilePanel.value = 'view' })
watch(() => mail.currentThread,  (t)   => { if (t)   mobilePanel.value = 'view' })
watch(() => mail.currentFolder,  ()    => { mobilePanel.value = 'list' })

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
.mobile-nav {
  display: none;
}

@media (max-width: 767px) {
  .sidebar,
  .message-list-panel,
  .message-view-panel {
    position: absolute;
    inset: 0;
    bottom: 52px;
    width: 100%;
    border-right: none;
  }
  .mobile-hidden {
    display: none !important;
  }
  .mobile-nav {
    display: flex;
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 52px;
    border-top: 0.5px solid var(--color-border);
    background: var(--color-surface);
  }
  .mobile-nav-btn {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    border: none;
    background: none;
    cursor: pointer;
    color: var(--color-text-muted);
    padding: 0;
  }
  .mobile-nav-btn:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }
  .mobile-nav-btn.mobile-nav-active {
    color: var(--color-teal);
  }
  .mobile-nav-icon {
    font-size: 16px;
    line-height: 1;
  }
  .mobile-nav-label {
    font-size: 10px;
  }
}
</style>
