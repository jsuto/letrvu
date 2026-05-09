<template>
  <div class="mail-layout">
    <aside class="sidebar">
      <FolderList />
    </aside>
    <section class="message-list-panel">
      <MessageList />
    </section>
    <main class="message-view-panel">
      <MessageView />
    </main>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useMailStore } from '../stores/mail'
import { useMailEvents } from '../composables/useMailEvents'
import FolderList from '../components/FolderList.vue'
import MessageList from '../components/MessageList.vue'
import MessageView from '../components/MessageView.vue'

const mail = useMailStore()

// Wire up IMAP IDLE push notifications
useMailEvents()

onMounted(async () => {
  await mail.fetchFolders()
  await mail.fetchMessages('INBOX')
})
</script>

<style scoped>
.mail-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}
.sidebar {
  width: var(--sidebar-width);
  flex-shrink: 0;
  border-right: 0.5px solid var(--color-border);
  background: var(--color-surface);
  overflow-y: auto;
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
