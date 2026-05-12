<template>
  <div class="h-full flex flex-col">
    <ConfirmDialog
      v-model:visible="confirmBulkDeleteVisible"
      :message="`Delete ${mail.selectedUids.size} selected ${mail.selectedUids.size === 1 ? 'message' : 'messages'}?`"
      @confirm="doBulkDelete"
    />
    <div class="px-4 py-2 border-b border-[var(--color-border)] flex items-center gap-2">
      <span class="text-sm font-medium whitespace-nowrap">{{ mail.globalSearchMode ? 'All folders' : mail.currentFolder }}</span>
      <form class="flex-1 flex gap-1" @submit.prevent="onSearch">
        <input
          v-model="query"
          type="search"
          placeholder="Search…"
          class="flex-1 px-2 py-1.5 border border-[var(--color-border)] rounded-md text-xs outline-none bg-[var(--color-bg)] focus:border-teal"
          @input="onSearchInput"
        />
        <button
          type="button"
          :class="[
            'px-2 py-1 border rounded-md text-[11px] cursor-pointer whitespace-nowrap shrink-0',
            searchAllFolders
              ? 'bg-teal text-white border-teal'
              : 'bg-[var(--color-surface)] text-[var(--color-text-muted)] border-[var(--color-border)]',
          ]"
          @click="searchAllFolders = !searchAllFolders"
          title="Search all folders"
        >All</button>
      </form>
    </div>

    <!-- Bulk action bar -->
    <div v-if="mail.selectedUids.size > 0" class="flex items-center gap-1.5 px-2.5 py-1.5 bg-[var(--color-teal-light)] border-b border-teal text-xs shrink-0">
      <label class="flex items-center cursor-pointer">
        <input type="checkbox" :checked="allSelected" @change="toggleSelectAll" class="[accent-color:var(--color-teal)] cursor-pointer" />
      </label>
      <span class="text-teal font-medium whitespace-nowrap mr-1">{{ mail.selectedUids.size }}</span>
      <button class="px-2.5 py-1 border border-teal rounded bg-transparent text-xs cursor-pointer text-teal whitespace-nowrap hover:bg-teal hover:text-white" @click="bulkMarkRead(true)" title="Mark as read">✓</button>
      <button class="px-2.5 py-1 border border-teal rounded bg-transparent text-xs cursor-pointer text-teal whitespace-nowrap hover:bg-teal hover:text-white" @click="bulkMarkRead(false)" title="Mark as unread">◯</button>
      <div class="relative" ref="bulkMoveWrapEl">
        <button class="px-2.5 py-1 border border-teal rounded bg-transparent text-xs cursor-pointer text-teal whitespace-nowrap hover:bg-teal hover:text-white" @click="bulkMoveOpen = !bulkMoveOpen" title="Move to…">⤷</button>
        <ul v-if="bulkMoveOpen" class="absolute top-[calc(100%+4px)] left-0 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-md list-none m-0 py-1 min-w-[160px] max-h-60 overflow-y-auto z-50 shadow-lg">
          <li v-for="f in otherFolders" :key="f.name" class="px-3.5 py-1.5 text-sm cursor-pointer whitespace-nowrap hover:bg-[var(--color-teal-light)]" @click="bulkMove(f.name)">{{ f.name }}</li>
        </ul>
      </div>
      <button v-if="!isJunkFolder" class="px-2.5 py-1 border border-teal rounded bg-transparent text-xs cursor-pointer text-teal whitespace-nowrap hover:bg-teal hover:text-white" @click="bulkSpam" title="Mark as spam">⊘</button>
      <button class="px-2.5 py-1 border border-red-300 rounded bg-transparent text-xs cursor-pointer text-red-600 whitespace-nowrap hover:bg-red-600 hover:text-white hover:border-red-600" @click="confirmBulkDeleteVisible = true" title="Delete">🗑</button>
      <button class="px-2.5 py-1 border-transparent border rounded bg-transparent text-xs cursor-pointer text-[var(--color-text-muted)] whitespace-nowrap ml-auto hover:text-[var(--color-text)]" @click="mail.clearSelection()" title="Clear selection">✕</button>
    </div>

    <div v-if="mail.loading" class="py-8 px-4 text-[var(--color-text-muted)] text-sm text-center">Loading…</div>
    <div v-else-if="!mail.messages.length" class="py-8 px-4 text-[var(--color-text-muted)] text-sm text-center">No messages</div>
    <ul v-else class="list-none flex-1 overflow-y-auto">
      <li
        v-for="(thread, i) in mail.threads"
        :key="thread.id"
        :class="[
          'flex items-start gap-2 px-3 py-2.5 border-b border-[var(--color-border)] cursor-pointer select-none hover:bg-[var(--color-bg)]',
          isThreadActive(thread) ? 'bg-[var(--color-teal-light)]' : '',
          isThreadSelected(thread) ? '!bg-teal/10' : '',
        ]"
        draggable="true"
        @click="onRowClick($event, thread, i)"
        @dragstart="onDragStart($event, thread)"
        @dragend="onDragEnd"
      >
        <input
          type="checkbox"
          :checked="isThreadSelected(thread)"
          class="mt-0.5 shrink-0 [accent-color:var(--color-teal)] w-3.5 h-3.5 cursor-pointer opacity-0 transition-opacity group-hover:opacity-100"
          :class="{ '!opacity-100': isThreadSelected(thread) }"
          style="opacity: 0; transition: opacity 0.1s"
          :ref="el => { if (el) el.style.opacity = isThreadSelected(thread) ? '1' : '' }"
          @click.stop="toggleThreadSelect(thread)"
        />
        <div class="flex-1 min-w-0">
          <div class="flex items-center justify-between gap-1.5 mb-0.5">
            <span :class="['text-sm text-[var(--color-text)] whitespace-nowrap overflow-hidden text-ellipsis min-w-0', thread.hasUnread ? 'font-bold' : '']">{{ threadSenders(thread) }}</span>
            <span class="flex items-center gap-1 shrink-0">
              <span v-if="thread.messages.some(m => m.flagged)" class="text-orange-400 text-xs">★</span>
              <span v-if="thread.messages.some(m => m.has_attachments)" class="text-xs">📎</span>
              <span v-if="thread.messages.length > 1" class="text-[10px] bg-teal text-white rounded-full px-1.5 py-px font-semibold">{{ thread.messages.length }}</span>
              <span class="text-[11px] text-[var(--color-text-muted)]">{{ formatDate(thread.latestDate) }}</span>
            </span>
          </div>
          <div :class="['text-sm text-[var(--color-text-muted)] whitespace-nowrap overflow-hidden text-ellipsis', thread.hasUnread ? 'font-semibold text-[var(--color-text)]' : '']">{{ thread.latest.subject || '(no subject)' }}</div>
          <div v-if="mail.globalSearchMode && thread.latest.folder" class="text-[10px] text-[var(--color-text-muted)] bg-[var(--color-bg)] border border-[var(--color-border)] rounded px-1.5 py-px inline-block mt-0.5 max-w-full overflow-hidden text-ellipsis whitespace-nowrap">{{ thread.latest.folder }}</div>
        </div>
      </li>
    </ul>
    <div v-if="!searching && (mail.page > 1 || mail.hasMore)" class="flex items-center justify-between px-4 py-2 border-t border-[var(--color-border)] text-xs text-[var(--color-text-muted)] shrink-0">
      <button :disabled="mail.page <= 1" @click="changePage(-1)"
        class="px-2.5 py-1 border border-[var(--color-border)] rounded bg-[var(--color-surface)] text-xs cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed hover:enabled:bg-[var(--color-bg)]">← Newer</button>
      <span>Page {{ mail.page }}</span>
      <button :disabled="!mail.hasMore" @click="changePage(1)"
        class="px-2.5 py-1 border border-[var(--color-border)] rounded bg-[var(--color-surface)] text-xs cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed hover:enabled:bg-[var(--color-bg)]">Older →</button>
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
