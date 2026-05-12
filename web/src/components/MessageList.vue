<template>
  <div class="message-list">
  <ConfirmDialog
    v-model:visible="confirmBulkDeleteVisible"
    :message="`Delete ${mail.selectedUids.size} selected ${mail.selectedUids.size === 1 ? 'message' : 'messages'}?`"
    @confirm="doBulkDelete"
  />
    <div class="toolbar">
      <span class="folder-name">{{ mail.globalSearchMode ? 'All folders' : mail.currentFolder }}</span>
      <form class="search-form" @submit.prevent="onSearch">
        <input
          v-model="query"
          type="search"
          placeholder="Search…"
          class="search-input"
          @input="onSearchInput"
        />
        <button
          type="button"
          :class="['scope-btn', { active: searchAllFolders }]"
          @click="searchAllFolders = !searchAllFolders"
          title="Search all folders"
        >All</button>
      </form>
    </div>

    <!-- Bulk action bar -->
    <div v-if="mail.selectedUids.size > 0" class="selection-bar">
      <label class="select-all-wrap" title="Select / deselect all on this page">
        <input type="checkbox" :checked="allSelected" @change="toggleSelectAll" />
      </label>
      <span class="sel-count">{{ mail.selectedUids.size }}</span>
      <button class="icon-btn" @click="bulkMarkRead(true)" title="Mark as read">✓</button>
      <button class="icon-btn" @click="bulkMarkRead(false)" title="Mark as unread">◯</button>
      <div class="move-wrap" ref="bulkMoveWrapEl">
        <button class="icon-btn" @click="bulkMoveOpen = !bulkMoveOpen" title="Move to…">⤷</button>
        <ul v-if="bulkMoveOpen" class="bulk-move-dropdown">
          <li v-for="f in otherFolders" :key="f.name" @click="bulkMove(f.name)">{{ f.name }}</li>
        </ul>
      </div>
      <button v-if="!isJunkFolder" class="icon-btn" @click="bulkSpam" title="Mark as spam">⊘</button>
      <button class="icon-btn danger" @click="confirmBulkDeleteVisible = true" title="Delete">🗑</button>
      <button class="icon-btn clear-btn" @click="mail.clearSelection()" title="Clear selection">✕</button>
    </div>

    <div v-if="mail.loading" class="state">Loading…</div>
    <div v-else-if="!mail.messages.length" class="state">No messages</div>
    <ul v-else>
      <li
        v-for="(thread, i) in mail.threads"
        :key="thread.id"
        :class="{
          unread: thread.hasUnread,
          active: isThreadActive(thread),
          selected: isThreadSelected(thread),
        }"
        draggable="true"
        @click="onRowClick($event, thread, i)"
        @dragstart="onDragStart($event, thread)"
        @dragend="onDragEnd"
      >
        <input
          type="checkbox"
          class="row-check"
          :checked="isThreadSelected(thread)"
          @click.stop="toggleThreadSelect(thread)"
        />
        <div class="row-content">
          <div class="row-top">
            <span class="from">{{ threadSenders(thread) }}</span>
            <span class="row-icons">
              <span v-if="thread.messages.some(m => m.flagged)" class="icon-flag" title="Flagged">★</span>
              <span v-if="thread.messages.some(m => m.has_attachments)" class="icon-attach" title="Has attachments">📎</span>
              <span v-if="thread.messages.length > 1" class="thread-badge">{{ thread.messages.length }}</span>
              <span class="date">{{ formatDate(thread.latestDate) }}</span>
            </span>
          </div>
          <div class="subject">{{ thread.latest.subject || '(no subject)' }}</div>
          <div v-if="mail.globalSearchMode && thread.latest.folder" class="folder-badge">{{ thread.latest.folder }}</div>
        </div>
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
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useMailStore } from '../stores/mail'
import ConfirmDialog from './ConfirmDialog.vue'

const mail = useMailStore()
const query = ref('')
const confirmBulkDeleteVisible = ref(false)
const searching = ref(false)
const searchAllFolders = ref(false)
const anchorIndex = ref(null)
const bulkMoveOpen = ref(false)
const bulkMoveWrapEl = ref(null)
watch(() => mail.currentFolder, () => { anchorIndex.value = null })

const otherFolders = computed(() =>
  mail.folders.filter(f => f.name !== mail.currentFolder)
)

const isJunkFolder = computed(() =>
  ['junk', 'junk email', 'spam'].includes(mail.currentFolder.toLowerCase())
)

// --- Thread helpers ---

function isThreadActive(thread) {
  if (mail.currentThread) return mail.currentThread.id === thread.id
  return thread.messages.some(m => m.uid === mail.currentMessage?.uid)
}

function isThreadSelected(thread) {
  return thread.messages.some(m => mail.selectedUids.has(m.uid))
}

function toggleThreadSelect(thread) {
  const allSel = thread.messages.every(m => mail.selectedUids.has(m.uid))
  thread.messages.forEach(m => {
    if (allSel) {
      const s = new Set(mail.selectedUids); s.delete(m.uid); mail.selectedUids = s
    } else if (!mail.selectedUids.has(m.uid)) {
      mail.toggleSelect(m.uid)
    }
  })
}

// Returns unique senders for the thread (up to 3).
function threadSenders(thread) {
  const seen = new Set()
  const names = []
  for (const m of thread.messages) {
    const name = m.from?.replace(/<[^>]+>/, '').trim() || m.from || ''
    if (!seen.has(name)) { seen.add(name); names.push(name) }
    if (names.length === 3) break
  }
  return names.join(', ')
}

const allSelected = computed(() =>
  mail.messages.length > 0 && mail.messages.every(m => mail.selectedUids.has(m.uid))
)

function toggleSelectAll() {
  if (allSelected.value) {
    mail.clearSelection()
  } else {
    mail.messages.forEach(m => {
      if (!mail.selectedUids.has(m.uid)) mail.toggleSelect(m.uid)
    })
  }
}

async function bulkMarkRead(read) {
  const uids = [...mail.selectedUids]
  await mail.markReadMessages(mail.currentFolder, uids, read)
}

async function bulkMove(dest) {
  bulkMoveOpen.value = false
  const uids = [...mail.selectedUids]
  await mail.moveMessagesTo(mail.currentFolder, uids, dest)
}

async function bulkSpam() {
  const uids = [...mail.selectedUids]
  await mail.markAsSpam(mail.currentFolder, uids)
}

async function doBulkDelete() {
  confirmBulkDeleteVisible.value = false
  const uids = [...mail.selectedUids]
  await mail.deleteMessages(mail.currentFolder, uids)
}

function onDocClick(e) {
  if (bulkMoveWrapEl.value && !bulkMoveWrapEl.value.contains(e.target)) {
    bulkMoveOpen.value = false
  }
}
onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))

function onRowClick(e, thread, i) {
  if (e.shiftKey && anchorIndex.value !== null) {
    const lo = Math.min(anchorIndex.value, i)
    const hi = Math.max(anchorIndex.value, i)
    const uids = mail.threads.slice(lo, hi + 1).flatMap(t => t.messages.map(m => m.uid))
    mail.selectedUids = new Set(uids)
    return
  }

  if (e.metaKey || e.ctrlKey) {
    anchorIndex.value = i
    toggleThreadSelect(thread)
    return
  }

  anchorIndex.value = i
  mail.clearSelection()

  if (thread.messages.length === 1) {
    // Single-message thread — open directly in MessageView, no thread pane.
    mail.currentThread = null
    const msg = thread.messages[0]
    const folder = msg.folder || mail.currentFolder
    mail.fetchMessage(folder, msg.uid)
    if (!msg.read) mail.markRead(folder, msg.uid, true)
  } else {
    mail.openThread(thread)
  }
}

// --- Drag and drop ---

function onDragStart(e, thread) {
  // Determine which UIDs are being dragged — all messages in the thread,
  // or all selected UIDs if the user has a selection.
  const threadUids = thread.messages.map(m => m.uid)
  const hasSelection = threadUids.some(uid => mail.selectedUids.has(uid))
  const uids = hasSelection ? [...mail.selectedUids] : threadUids

  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('application/x-letrvu-uids', JSON.stringify(uids))
  e.dataTransfer.setData('application/x-letrvu-folder', mail.currentFolder)

  // Custom drag image showing count.
  const ghost = document.createElement('div')
  ghost.className = 'drag-ghost'
  ghost.textContent = uids.length === 1 ? '1 message' : `${uids.length} messages`
  document.body.appendChild(ghost)
  e.dataTransfer.setDragImage(ghost, 0, 0)
  setTimeout(() => ghost.remove(), 0)
}

function onDragEnd() {
  // Nothing to clean up — ghost was removed immediately.
}

// --- Search / pagination ---

function onSearch() {
  if (query.value.trim()) {
    searching.value = true
    if (searchAllFolders.value) {
      mail.searchAllFolders(query.value.trim())
    } else {
      mail.searchMessages(mail.currentFolder, query.value.trim())
    }
  } else {
    searching.value = false
    mail.fetchMessages(mail.currentFolder)
  }
}

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
.folder-name { font-size: 13px; font-weight: 500; white-space: nowrap; }
.search-form { flex: 1; display: flex; gap: 4px; }
.search-input {
  flex: 1;
  padding: 5px 8px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 12px;
  outline: none;
  background: var(--color-bg);
}
.search-input:focus { border-color: var(--color-teal); }
.scope-btn {
  padding: 4px 8px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 11px;
  background: var(--color-surface);
  color: var(--color-text-muted);
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
}
.scope-btn.active {
  background: var(--color-teal);
  color: white;
  border-color: var(--color-teal);
}
.folder-badge {
  font-size: 10px;
  color: var(--color-text-muted);
  background: var(--color-bg);
  border: 0.5px solid var(--color-border);
  border-radius: 4px;
  padding: 1px 5px;
  display: inline-block;
  margin-top: 2px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selection-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  background: var(--color-teal-light);
  border-bottom: 0.5px solid var(--color-teal);
  font-size: 12px;
  flex-shrink: 0;
}
.select-all-wrap { display: flex; align-items: center; cursor: pointer; }
.select-all-wrap input { accent-color: var(--color-teal); cursor: pointer; }
.sel-count { color: var(--color-teal); font-weight: 500; white-space: nowrap; margin-right: 4px; }
.selection-bar button {
  padding: 3px 10px;
  border: 0.5px solid var(--color-teal);
  border-radius: 5px;
  background: transparent;
  font-size: 12px;
  cursor: pointer;
  color: var(--color-teal);
  white-space: nowrap;
}
.selection-bar button:hover { background: var(--color-teal); color: white; }
.selection-bar button.danger { border-color: #e07070; color: #c0392b; }
.selection-bar button.danger:hover { background: #c0392b; color: white; border-color: #c0392b; }
.selection-bar button.clear-btn { border-color: transparent; color: var(--color-text-muted); margin-left: auto; }
.selection-bar button.clear-btn:hover { background: transparent; color: var(--color-text); }
.move-wrap { position: relative; }
.bulk-move-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  list-style: none;
  margin: 0;
  padding: 4px 0;
  min-width: 160px;
  max-height: 240px;
  overflow-y: auto;
  z-index: 50;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}
.bulk-move-dropdown li {
  padding: 6px 14px;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
}
.bulk-move-dropdown li:hover { background: var(--color-teal-light); }

.state {
  padding: 2rem 1rem;
  color: var(--color-text-muted);
  font-size: 13px;
  text-align: center;
}
ul { list-style: none; flex: 1; overflow-y: auto; }
li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 0.5px solid var(--color-border);
  cursor: pointer;
  user-select: none;
}
li:hover { background: var(--color-bg); }
li.active { background: var(--color-teal-light); }
li.selected { background: color-mix(in srgb, var(--color-teal) 12%, transparent); }
li.unread .from { font-weight: 700; color: var(--color-text); }
li.unread .subject { font-weight: 600; color: var(--color-text); }

.row-check {
  margin-top: 3px;
  flex-shrink: 0;
  accent-color: var(--color-teal);
  width: 14px;
  height: 14px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.1s;
}
li:hover .row-check,
li.selected .row-check { opacity: 1; }

.row-content { flex: 1; min-width: 0; }
.row-top { display: flex; align-items: center; justify-content: space-between; gap: 6px; margin-bottom: 2px; }
.from { font-size: 13px; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; min-width: 0; }
.row-icons { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
.icon-flag { color: #e67e22; font-size: 12px; }
.icon-attach { font-size: 12px; }
.thread-badge {
  font-size: 10px;
  background: var(--color-teal);
  color: white;
  border-radius: 8px;
  padding: 1px 5px;
  font-weight: 600;
  flex-shrink: 0;
}
.date { font-size: 11px; color: var(--color-text-muted); }
.subject { font-size: 13px; color: var(--color-text-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

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

<!-- Global drag ghost style (not scoped) -->
<style>
.drag-ghost {
  position: fixed;
  top: -100px;
  left: 0;
  background: var(--color-teal);
  color: white;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-family: inherit;
  pointer-events: none;
  white-space: nowrap;
}
</style>
