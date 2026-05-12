<template>
  <div class="thread-view">
    <div class="thread-header">
      <button class="back-btn" @click="mail.currentThread = null" title="Back to message">←</button>
      <h2 class="thread-subject">{{ thread.latest.subject || '(no subject)' }}</h2>
      <span class="thread-count">{{ thread.messages.length }} messages</span>
    </div>

    <div class="thread-list">
      <div
        v-for="msg in thread.messages"
        :key="msg.uid"
        class="thread-item"
        :class="{ expanded: expandedUids.has(msg.uid), unread: !msg.read }"
      >
        <!-- Collapsed header — always visible -->
        <div class="item-header" @click="toggle(msg)">
          <span class="item-from">{{ msg.from }}</span>
          <span class="item-icons">
            <span v-if="msg.flagged" class="icon-flag">★</span>
            <span v-if="msg.has_attachments" class="icon-attach">📎</span>
          </span>
          <span class="item-date">{{ formatDate(msg.date) }}</span>
          <span class="item-chevron">{{ expandedUids.has(msg.uid) ? '▾' : '▸' }}</span>
        </div>

        <!-- Expanded body -->
        <div v-if="expandedUids.has(msg.uid)" class="item-body">
          <div v-if="loadingUids.has(msg.uid)" class="item-loading">Loading…</div>
          <template v-else-if="fullMessages[msg.uid]">
            <div class="item-meta">
              <span v-if="fullMessages[msg.uid].to?.length">
                <strong>To:</strong> {{ fullMessages[msg.uid].to.join(', ') }}
              </span>
            </div>

            <!-- Auth / phishing banners reused from MessageView -->
            <div v-if="authFailed(msg.uid)" class="auth-banner">
              ⚠ Authentication failed — this message may be spoofed.
            </div>

            <iframe
              v-if="fullMessages[msg.uid].html_body"
              :ref="el => setIframe(el, msg.uid)"
              class="body-frame"
              sandbox="allow-popups allow-same-origin"
              :srcdoc="processedHtml(msg.uid)"
              :title="`Message from ${msg.from}`"
              @load="resizeIframe(msg.uid)"
            />
            <pre v-else class="body-text">{{ fullMessages[msg.uid].text_body }}</pre>

            <div v-if="fullMessages[msg.uid].attachments?.length" class="attachments">
              <p class="att-label">Attachments</p>
              <div v-for="att in fullMessages[msg.uid].attachments" :key="att.index" class="attachment">
                <span class="att-name">📎 {{ att.filename || 'attachment' }}</span>
                <a :href="attachmentUrl(msg.uid, att)" download class="att-download">↓</a>
              </div>
            </div>

            <div class="item-actions">
              <button @click="reply(msg)">Reply</button>
              <button @click="replyAll(msg)">Reply All</button>
              <button v-if="!isJunkFolder" @click="spam(msg)">Spam</button>
              <button @click="requestDelete(msg)" class="danger">Delete</button>
            </div>
          </template>
        </div>
      </div>
    </div>

    <ConfirmDialog
      v-model:visible="confirmDeleteVisible"
      :message="`Delete this message?`"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup>
import { ref, computed, inject } from 'vue'
import DOMPurify from 'dompurify'
import { useMailStore } from '../stores/mail'
import { useSettingsStore } from '../stores/settings'
import { useDarkMode } from '../composables/useDarkMode'
import { extractEmail, buildReplyAllCc } from '../utils/mail.js'
import ConfirmDialog from './ConfirmDialog.vue'

const mail = useMailStore()
const settings = useSettingsStore()
const compose = inject('compose')
const { dark } = useDarkMode()

const thread = computed(() => mail.currentThread)

// Set of UIDs currently expanded
const expandedUids = ref(new Set())
// Full message data keyed by UID
const fullMessages = ref({})
// UIDs currently being fetched
const loadingUids = ref(new Set())
// iframe element refs keyed by UID
const iframeRefs = ref({})

const isJunkFolder = computed(() =>
  ['junk', 'junk email', 'spam'].includes(mail.currentFolder.toLowerCase())
)

// Auto-expand the latest unread or latest message when the thread changes
import { watch } from 'vue'
watch(
  () => mail.currentThread,
  (t) => {
    expandedUids.value = new Set()
    fullMessages.value = {}
    if (!t) return
    const toExpand = t.messages.find(m => !m.read) ?? t.messages[t.messages.length - 1]
    expand(toExpand)
  },
  { immediate: true }
)

async function expand(msg) {
  expandedUids.value = new Set([...expandedUids.value, msg.uid])
  if (fullMessages.value[msg.uid]) return
  loadingUids.value = new Set([...loadingUids.value, msg.uid])
  try {
    const folder = encodeURIComponent(mail.currentFolder)
    const res = await fetch(`/api/folders/${folder}/messages/${msg.uid}`)
    if (res.ok) {
      const data = await res.json()
      fullMessages.value = { ...fullMessages.value, [msg.uid]: data }
      // Mark as read
      if (!msg.read) mail.markRead(mail.currentFolder, msg.uid, true)
    }
  } finally {
    const s = new Set(loadingUids.value)
    s.delete(msg.uid)
    loadingUids.value = s
  }
}

function collapse(msg) {
  const s = new Set(expandedUids.value)
  s.delete(msg.uid)
  expandedUids.value = s
}

function toggle(msg) {
  if (expandedUids.value.has(msg.uid)) collapse(msg)
  else expand(msg)
}

// --- HTML processing (mirrors MessageView) ---

function processedHtml(uid) {
  const full = fullMessages.value[uid]
  if (!full?.html_body) return ''
  const sanitized = DOMPurify.sanitize(full.html_body, { WHOLE_DOCUMENT: true })
  const doc = new DOMParser().parseFromString(sanitized, 'text/html')
  if (dark.value) {
    const s = doc.createElement('style')
    s.textContent = 'body{background-color:#141412;color:#e0e0de}a:not([style*="color"]){color:#7ab3ef}'
    doc.head.prepend(s)
  }
  return doc.documentElement.outerHTML
}

function authFailed(uid) {
  const full = fullMessages.value[uid]
  return full && (full.dmarc === 'fail' || full.spf === 'fail')
}

// --- iframe resizing ---

function setIframe(el, uid) {
  if (el) iframeRefs.value[uid] = el
}

function resizeIframe(uid) {
  const frame = iframeRefs.value[uid]
  if (!frame) return
  const doc = frame.contentDocument
  if (doc) frame.style.height = doc.documentElement.scrollHeight + 'px'
}

function attachmentUrl(uid, att) {
  const folder = encodeURIComponent(mail.currentFolder)
  return `/api/folders/${folder}/messages/${uid}/attachments/${att.index}`
}

// --- Actions ---

function escHtml(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function buildQuoteHtml(msg, full, date) {
  const bodyHtml = full.html_body || plainToHtml(full.text_body || '')
  return `<p>On ${escHtml(date)}, ${escHtml(msg.from || '')} wrote:</p><blockquote>${bodyHtml}</blockquote>`
}

function plainToHtml(text) {
  if (!text) return ''
  return text.split('\n').map(l => `<p>${escHtml(l) || '<br>'}</p>`).join('')
}

function reply(msg) {
  const full = fullMessages.value[msg.uid]
  if (!full) return
  const date = msg.date ? new Date(msg.date).toLocaleString() : ''
  compose?.value?.open({
    to: full.reply_to || msg.from,
    subject: msg.subject?.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`,
    html: buildQuoteHtml(msg, full, date),
    _originalRecipients: [...(full.to ?? []), ...(full.cc ?? [])],
    _inReplyTo: full.message_id || '',
    _references: [full.references, full.message_id].filter(Boolean).join(' '),
  })
}

function replyAll(msg) {
  const full = fullMessages.value[msg.uid]
  if (!full) return
  const date = msg.date ? new Date(msg.date).toLocaleString() : ''
  const replyTo = full.reply_to || msg.from
  const ownEmails = settings.fromOptions.map(o => o.email)
  const cc = buildReplyAllCc(full.to, full.cc, replyTo, ownEmails)
  compose?.value?.open({
    to: replyTo,
    cc,
    subject: msg.subject?.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`,
    html: buildQuoteHtml(msg, full, date),
    _originalRecipients: [...(full.to ?? []), ...(full.cc ?? [])],
    _inReplyTo: full.message_id || '',
    _references: [full.references, full.message_id].filter(Boolean).join(' '),
  })
}

async function spam(msg) {
  await mail.markAsSpam(mail.currentFolder, [msg.uid])
}

const deleteTarget = ref(null)
const confirmDeleteVisible = ref(false)

function requestDelete(msg) {
  deleteTarget.value = msg
  confirmDeleteVisible.value = true
}

async function doDelete() {
  if (!deleteTarget.value) return
  confirmDeleteVisible.value = false
  await mail.deleteMessage(mail.currentFolder, deleteTarget.value.uid)
  deleteTarget.value = null
  // If the thread is now empty, close it
  if (!mail.currentThread?.messages.length) mail.currentThread = null
}

// --- Formatting ---

function formatDate(dateStr) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString()
}
</script>

<style scoped>
.thread-view { height: 100%; overflow-y: auto; }

.thread-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 20px;
  border-bottom: 0.5px solid var(--color-border);
  position: sticky;
  top: 0;
  background: var(--color-bg);
  z-index: 1;
}
.back-btn {
  background: none;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  padding: 4px 10px;
  font-size: 14px;
  cursor: pointer;
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.back-btn:hover { background: var(--color-surface); color: var(--color-text); }
.thread-subject {
  font-size: 15px;
  font-weight: 600;
  flex: 1;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.thread-count {
  font-size: 12px;
  color: var(--color-text-muted);
  flex-shrink: 0;
}

.thread-list { padding: 12px 20px; display: flex; flex-direction: column; gap: 8px; }

.thread-item {
  border: 0.5px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-surface);
  overflow: hidden;
}
.thread-item.expanded { box-shadow: 0 2px 8px rgba(0,0,0,0.06); }

.item-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  cursor: pointer;
  user-select: none;
}
.item-header:hover { background: var(--color-bg); }
.item-from { font-size: 13px; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.thread-item.unread .item-from { font-weight: 700; color: var(--color-text); }
.item-icons { display: flex; gap: 4px; flex-shrink: 0; }
.icon-flag { color: #e67e22; font-size: 12px; }
.icon-attach { font-size: 12px; }
.item-date { font-size: 11px; color: var(--color-text-muted); flex-shrink: 0; }
.item-chevron { font-size: 11px; color: var(--color-text-muted); flex-shrink: 0; width: 14px; }

.item-body { border-top: 0.5px solid var(--color-border); }
.item-loading { padding: 16px; font-size: 13px; color: var(--color-text-muted); }
.item-meta { padding: 8px 14px 0; font-size: 12px; color: var(--color-text-muted); }

.auth-banner {
  margin: 8px 14px;
  padding: 8px 12px;
  background: #fff3cd;
  border-radius: 6px;
  font-size: 12px;
  color: #856404;
}

.body-frame {
  display: block;
  width: 100%;
  border: none;
  min-height: 60px;
}
.body-text {
  padding: 12px 14px;
  font-size: 13px;
  font-family: inherit;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
}

.attachments { padding: 8px 14px; border-top: 0.5px solid var(--color-border); }
.att-label { font-size: 11px; color: var(--color-text-muted); margin: 0 0 6px; }
.attachment { display: flex; align-items: center; gap: 8px; font-size: 12px; }
.att-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.att-download { color: var(--color-teal); text-decoration: none; }

.item-actions {
  display: flex;
  gap: 8px;
  padding: 10px 14px;
  border-top: 0.5px solid var(--color-border);
}
.item-actions button {
  padding: 5px 12px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  font-size: 12px;
  cursor: pointer;
  color: var(--color-text);
}
.item-actions button:hover { background: var(--color-bg); }
.item-actions button.danger { color: #c0392b; border-color: #f5c6c6; }
.item-actions button.danger:hover { background: #c0392b; color: white; border-color: #c0392b; }
</style>
