<template>
  <div class="h-full overflow-y-auto">
    <div class="flex items-center gap-2.5 px-5 py-3.5 border-b border-[var(--color-border)] sticky top-0 bg-[var(--color-bg)] z-[1]">
      <button
        class="md:hidden bg-none border border-[var(--color-border)] rounded-md px-2.5 py-1 text-sm cursor-pointer text-[var(--color-text-muted)] shrink-0 hover:bg-[var(--color-surface)] hover:text-[var(--color-text)]"
        @click="setMobilePanel('list')"
        title="Back to messages"
      >←</button>
      <button
        class="hidden md:block bg-none border border-[var(--color-border)] rounded-md px-2.5 py-1 text-sm cursor-pointer text-[var(--color-text-muted)] shrink-0 hover:bg-[var(--color-surface)] hover:text-[var(--color-text)]"
        @click="mail.currentThread = null"
        title="Back to message"
      >←</button>
      <h2 class="text-[15px] font-semibold flex-1 m-0 overflow-hidden text-ellipsis whitespace-nowrap">{{ thread.latest.subject || '(no subject)' }}</h2>
      <span class="text-xs text-[var(--color-text-muted)] shrink-0">{{ thread.messages.length }} messages</span>
    </div>

    <div class="px-5 py-3 flex flex-col gap-2">
      <div
        v-for="msg in thread.messages"
        :key="msg.uid"
        :class="['border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)] overflow-hidden', expandedUids.has(msg.uid) ? 'shadow-sm' : '']"
      >
        <!-- Collapsed header — always visible -->
        <div
          class="flex items-center gap-2 px-3.5 py-2.5 cursor-pointer select-none hover:bg-[var(--color-bg)]"
          @click="toggle(msg)"
        >
          <span :class="['text-sm flex-1 overflow-hidden text-ellipsis whitespace-nowrap', !msg.read ? 'font-bold text-[var(--color-text)]' : '']">{{ msg.from }}</span>
          <span class="flex gap-1 shrink-0">
            <span v-if="msg.flagged" class="text-orange-400 text-xs">★</span>
            <span v-if="msg.has_attachments" class="text-xs">📎</span>
          </span>
          <span class="text-[11px] text-[var(--color-text-muted)] shrink-0">{{ formatDate(msg.date) }}</span>
          <span class="text-[11px] text-[var(--color-text-muted)] shrink-0 w-3.5">{{ expandedUids.has(msg.uid) ? '▾' : '▸' }}</span>
        </div>

        <!-- Expanded body -->
        <div v-if="expandedUids.has(msg.uid)" class="border-t border-[var(--color-border)]">
          <div v-if="loadingUids.has(msg.uid)" class="px-3.5 py-4 text-sm text-[var(--color-text-muted)]">Loading…</div>
          <template v-else-if="fullMessages[msg.uid]">
            <div class="px-3.5 pt-2 text-xs text-[var(--color-text-muted)]">
              <span v-if="fullMessages[msg.uid].to?.length">
                <strong>To:</strong> {{ fullMessages[msg.uid].to.join(', ') }}
              </span>
            </div>

            <!-- Auth / phishing banners reused from MessageView -->
            <div v-if="authFailed(msg.uid)" class="mx-3.5 my-2 px-3 py-2 bg-[#fff4e5] rounded-md text-xs text-[#856404]">
              ⚠ Authentication failed — this message may be spoofed.
            </div>

            <iframe
              v-if="fullMessages[msg.uid].html_body"
              :ref="el => setIframe(el, msg.uid)"
              class="block w-full border-none min-h-[60px]"
              sandbox="allow-popups allow-same-origin"
              :srcdoc="processedHtml(msg.uid)"
              :title="`Message from ${msg.from}`"
              @load="resizeIframe(msg.uid)"
            />
            <pre v-else class="px-3.5 py-3 text-sm leading-relaxed whitespace-pre-wrap break-words m-0">{{ fullMessages[msg.uid].text_body }}</pre>

            <div v-if="fullMessages[msg.uid].attachments?.length" class="px-3.5 py-2 border-t border-[var(--color-border)]">
              <p class="text-[11px] text-[var(--color-text-muted)] mb-1.5">Attachments</p>
              <div v-for="att in fullMessages[msg.uid].attachments" :key="att.index" class="flex items-center gap-2 text-xs">
                <span class="flex-1 overflow-hidden text-ellipsis whitespace-nowrap">📎 {{ att.filename || 'attachment' }}</span>
                <a :href="attachmentUrl(msg.uid, att)" download class="text-teal no-underline">↓</a>
              </div>
            </div>

            <div class="flex gap-2 px-3.5 py-2.5 border-t border-[var(--color-border)]">
              <button
                v-for="(action, idx) in messageActions(msg)"
                :key="idx"
                :class="[
                  'px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] hover:bg-[var(--color-bg)]',
                  action.danger ? 'text-red-600 border-red-200 hover:bg-red-600 hover:text-white hover:border-red-600' : '',
                ]"
                @click="action.handler(msg)"
              >{{ action.label }}</button>
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
const setMobilePanel = inject('setMobilePanel', () => {})
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

function messageActions(msg) {
  const actions = [
    { label: 'Reply', handler: reply },
    { label: 'Reply All', handler: replyAll },
  ]
  if (!isJunkFolder.value) actions.push({ label: 'Spam', handler: spam })
  actions.push({ label: 'Delete', handler: requestDelete, danger: true })
  return actions
}

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
    s.textContent = 'html{filter:invert(1) hue-rotate(180deg) !important;background:#fff}img,video,picture,canvas,svg image{filter:invert(1) hue-rotate(180deg)}'
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
