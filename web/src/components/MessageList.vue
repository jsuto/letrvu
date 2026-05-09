<template>
  <div class="message-list">
    <div class="toolbar">
      <span class="folder-name">{{ mail.currentFolder }}</span>
      <form class="search-form" @submit.prevent="onSearch">
        <input
          v-model="query"
          type="search"
          placeholder="Search…"
          class="search-input"
          @input="onSearchInput"
        />
      </form>
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
    <div v-if="!searching && (mail.page > 1 || mail.hasMore)" class="pagination">
      <button :disabled="mail.page <= 1" @click="changePage(-1)">← Newer</button>
      <span>Page {{ mail.page }}</span>
      <button :disabled="!mail.hasMore" @click="changePage(1)">Older →</button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useMailStore } from '../stores/mail'

const mail = useMailStore()
const query = ref('')
const searching = ref(false)

function selectMessage(msg) {
  mail.fetchMessage(mail.currentFolder, msg.uid)
  if (!msg.read) mail.markRead(mail.currentFolder, msg.uid, true)
}

function onSearch() {
  if (query.value.trim()) {
    searching.value = true
    mail.searchMessages(mail.currentFolder, query.value.trim())
  } else {
    searching.value = false
    mail.fetchMessages(mail.currentFolder)
  }
}

// Clear search and reload when the field is cleared.
function onSearchInput() {
  if (query.value === '') {
    searching.value = false
    mail.fetchMessages(mail.currentFolder)
  }
}

function changePage(delta) {
  mail.fetchMessages(mail.currentFolder, mail.page + delta)
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
  padding: 8px 16px;
  border-bottom: 0.5px solid var(--color-border);
  display: flex;
  align-items: center;
  gap: 8px;
}
.folder-name {
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
}
.search-form { flex: 1; }
.search-input {
  width: 100%;
  padding: 5px 8px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 12px;
  outline: none;
  background: var(--color-bg);
}
.search-input:focus { border-color: var(--color-teal); }
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
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  border-top: 0.5px solid var(--color-border);
  font-size: 12px;
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.pagination button {
  padding: 4px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 5px;
  background: var(--color-surface);
  font-size: 12px;
  cursor: pointer;
}
.pagination button:disabled { opacity: 0.4; cursor: not-allowed; }
.pagination button:not(:disabled):hover { background: var(--color-bg); }
</style>
