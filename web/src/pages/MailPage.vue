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
    <ComposeModal ref="composeModal" />
  </div>
</template>

<script setup>
import { ref, provide, onMounted } from 'vue'
import { useMailStore } from '../stores/mail'
import { useMailEvents } from '../composables/useMailEvents'
import FolderList from '../components/FolderList.vue'
import MessageList from '../components/MessageList.vue'
import MessageView from '../components/MessageView.vue'
import ComposeModal from '../components/ComposeModal.vue'

const mail = useMailStore()
const composeModal = ref(null)

// Provide compose modal to all descendants so FolderList and MessageView can open it.
provide('compose', composeModal)

useMailEvents()

onMounted(async () => {
  await mail.fetchFolders()
  if (!mail.messages.length) {
    await mail.fetchMessages(mail.currentFolder)
  }
})
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
