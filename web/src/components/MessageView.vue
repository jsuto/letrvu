<template>
  <div class="message-view">
    <div v-if="!mail.currentMessage" class="empty-state">
      <p>Select a message to read</p>
    </div>
    <div v-else class="message">
      <div class="header">
        <h2>{{ mail.currentMessage.subject || '(no subject)' }}</h2>
        <div class="meta">
          <span class="from">{{ mail.currentMessage.from }}</span>
          <span class="date">{{ formatDate(mail.currentMessage.date) }}</span>
        </div>
        <div class="actions">
          <button @click="reply">Reply</button>
          <button @click="forward">Forward</button>
          <button @click="remove" class="danger">Delete</button>
        </div>
      </div>
      <!-- HTML email rendered in a sandboxed iframe to prevent XSS -->
      <iframe
        v-if="mail.currentMessage.html_body"
        class="body-frame"
        sandbox="allow-same-origin"
        :srcdoc="mail.currentMessage.html_body"
        title="Message body"
      />
      <pre v-else class="body-text">{{ mail.currentMessage.text_body }}</pre>
    </div>
  </div>
</template>

<script setup>
import { useMailStore } from '../stores/mail'
const mail = useMailStore()

function formatDate(dateStr) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString()
}

function reply() {
  // TODO: open ComposeModal pre-filled with reply headers
}

function forward() {
  // TODO: open ComposeModal pre-filled with forwarded body
}

async function remove() {
  const msg = mail.currentMessage
  if (!msg) return
  await mail.deleteMessage(mail.currentFolder, msg.uid)
}
</script>

<style scoped>
.message-view { height: 100%; overflow-y: auto; }
.empty-state {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 13px;
}
.message { padding: 2rem; max-width: 780px; margin: 0 auto; }
.header { margin-bottom: 1.5rem; }
h2 { font-size: 18px; font-weight: 500; margin-bottom: 0.5rem; }
.meta { font-size: 13px; color: var(--color-text-muted); margin-bottom: 1rem; display: flex; gap: 1rem; }
.actions { display: flex; gap: 8px; }
.actions button {
  padding: 6px 14px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  font-size: 13px;
  cursor: pointer;
}
.actions button:hover { background: var(--color-bg); }
.actions button.danger { color: #c0392b; border-color: #f5c6c6; }
.body-frame {
  width: 100%;
  min-height: 400px;
  border: 0.5px solid var(--color-border);
  border-radius: 8px;
  background: white;
}
.body-text {
  white-space: pre-wrap;
  font-family: inherit;
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text);
}
</style>
