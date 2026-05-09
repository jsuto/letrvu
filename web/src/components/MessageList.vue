<template>
  <div class="message-list">
    <div class="toolbar">
      <span class="folder-name">{{ mail.currentFolder }}</span>
    </div>
    <div v-if="mail.loading" class="state">Loading…</div>
    <div v-else-if="!mail.messages.length" class="state">No messages</div>
    <ul v-else>
      <li
        v-for="msg in mail.messages"
        :key="msg.uid"
        :class="{ unread: !msg.read, active: mail.currentMessage?.uid === msg.uid }"
        @click="selectMessage(msg)"
      >
        <div class="from">{{ msg.from }}</div>
        <div class="subject">{{ msg.subject || '(no subject)' }}</div>
        <div class="date">{{ formatDate(msg.date) }}</div>
      </li>
    </ul>
  </div>
</template>

<script setup>
import { useMailStore } from '../stores/mail'
const mail = useMailStore()

function selectMessage(msg) {
  mail.fetchMessage(mail.currentFolder, msg.uid)
  if (!msg.read) mail.markRead(mail.currentFolder, msg.uid, true)
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}
</script>

<style scoped>
.message-list { height: 100%; display: flex; flex-direction: column; }
.toolbar {
  padding: 12px 16px;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  font-weight: 500;
}
.state {
  padding: 2rem 1rem;
  color: var(--color-text-muted);
  font-size: 13px;
  text-align: center;
}
ul { list-style: none; flex: 1; overflow-y: auto; }
li {
  padding: 12px 16px;
  border-bottom: 0.5px solid var(--color-border);
  cursor: pointer;
}
li:hover { background: var(--color-bg); }
li.active { background: var(--color-teal-light); }
li.unread .from, li.unread .subject { font-weight: 500; }
.from { font-size: 13px; color: var(--color-text); margin-bottom: 2px; }
.subject { font-size: 13px; color: var(--color-text-muted); margin-bottom: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.date { font-size: 11px; color: var(--color-text-muted); }
</style>
