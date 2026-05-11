<template>
  <div class="message-view">
  <ConfirmDialog
    v-model:visible="confirmDeleteVisible"
    :message="`Delete &quot;${mail.currentMessage?.subject || '(no subject)'}&quot;?`"
    @confirm="doDelete"
  />
    <div v-if="!mail.currentMessage" class="empty-state">
      <p>Select a message to read</p>
    </div>
    <div v-else class="message">
      <div class="header">
        <h2>{{ mail.currentMessage.subject || '(no subject)' }}</h2>
        <div class="meta">
          <span class="from">{{ mail.currentMessage.from }}</span>
          <span v-if="isExternal" class="external-badge" title="Sender is outside your organisation — authentication passed">External</span>
          <span v-else-if="isUnverified" class="unverified-badge" title="Sender appears to be outside your organisation — no authentication results available">Unverified</span>
          <button class="save-contact-btn" title="Save to address book" @click="saveContact">+</button>
          <span class="date">{{ formatDate(mail.currentMessage.date) }}</span>
        </div>
        <div class="actions">
          <button v-if="isDraftsFolder" @click="editDraft" class="edit-draft-btn">Edit Draft</button>
          <div v-if="!isDraftsFolder" class="reply-wrap" ref="replyWrapEl">
            <button @click="replyMenuOpen = !replyMenuOpen">Reply ▾</button>
            <ul v-if="replyMenuOpen" class="reply-dropdown">
              <li @click="reply">Reply</li>
              <li @click="replyAll">Reply All</li>
            </ul>
          </div>
          <div v-if="!isDraftsFolder" class="forward-wrap" ref="forwardWrapEl">
            <button @click="forwardMenuOpen = !forwardMenuOpen">Forward ▾</button>
            <ul v-if="forwardMenuOpen" class="forward-dropdown">
              <li @click="forwardInline">Inline</li>
              <li @click="forwardAsAttachment">As .eml attachment</li>
            </ul>
          </div>
          <button
            :class="{ active: mail.currentMessage.flagged }"
            :title="mail.currentMessage.flagged ? 'Unflag' : 'Flag as important'"
            @click="toggleFlagged"
          >{{ mail.currentMessage.flagged ? '★' : '☆' }} Flag</button>
          <button
            :title="mail.currentMessage.read ? 'Mark as unread' : 'Mark as read'"
            @click="toggleRead"
          >{{ mail.currentMessage.read ? 'Mark unread' : 'Mark read' }}</button>
          <div class="move-wrap" ref="moveWrapEl">
            <button @click="moveOpen = !moveOpen">Move to…</button>
            <ul v-if="moveOpen" class="move-dropdown">
              <li
                v-for="f in otherFolders"
                :key="f.name"
                @click="moveTo(f.name)"
              >{{ f.name }}</li>
            </ul>
          </div>
          <button v-if="!isJunkFolder" @click="spam" title="Move to Junk">Spam</button>
          <button @click="confirmDeleteVisible = true" class="danger">Delete</button>
          <button @click="viewSource" title="View message source" class="source-btn">&lt;/&gt;</button>
        </div>
      </div>

      <!-- Message source modal -->
      <div v-if="sourceOpen" class="source-overlay" @click.self="sourceOpen = false">
        <div class="source-modal">
          <div class="source-toolbar">
            <span class="source-title">Message source</span>
            <button @click="copySource" class="source-copy">{{ sourceCopied ? 'Copied!' : 'Copy' }}</button>
            <button @click="sourceOpen = false" class="source-close">✕</button>
          </div>
          <pre class="source-body">{{ sourceText }}</pre>
        </div>
      </div>

      <!-- Calendar invite banner -->
      <div v-if="mail.currentMessage.ical_invite" class="invite-banner">
        <span>📅 This message contains a calendar invite.</span>
        <button @click="addToCalendar" :disabled="inviteAdding" class="invite-btn">
          {{ inviteAdded ? 'Added ✓' : inviteAdding ? 'Adding…' : 'Add to calendar' }}
        </button>
      </div>

      <!-- Remote image blocking banner -->
      <div v-if="hasRemoteImages && !showRemoteImages" class="remote-images-banner">
        <span>🛡 Remote images blocked to protect your privacy.</span>
        <button @click="showRemoteImages = true" class="show-images-btn">Show images</button>
      </div>

      <!-- Email authentication failure banner (SPF/DKIM/DMARC) -->
      <div v-if="authFailed" class="auth-banner">
        <span>⚠ Authentication failed — this message did not pass {{ authFailedMethods }} checks and may be spoofed or forged.</span>
      </div>

      <!-- Phishing link warning banner -->
      <div v-if="phishingCount > 0" class="phishing-banner">
        <span>⚠ {{ phishingCount }} misleading {{ phishingCount === 1 ? 'link' : 'links' }} detected — the visible text shows a different domain than the actual destination.</span>
      </div>

      <!-- HTML email rendered in a sandboxed iframe to prevent XSS.
           allow-popups: target="_blank" links open in a new tab.
           allow-same-origin: lets the parent read scrollHeight to auto-size
           the iframe so the outer pane scrolls instead of the iframe itself.
           Scripts are still blocked (no allow-scripts). -->
      <iframe
        v-if="mail.currentMessage.html_body"
        ref="iframeEl"
        class="body-frame"
        sandbox="allow-popups allow-same-origin"
        :srcdoc="displayHtml"
        title="Message body"
        @load="resizeIframe"
      />
      <pre v-else class="body-text">{{ mail.currentMessage.text_body }}</pre>
      <div v-if="mail.currentMessage.attachments?.length" class="attachments">
        <p class="attachments-label">Attachments</p>
        <div
          v-for="att in mail.currentMessage.attachments"
          :key="att.index"
          class="attachment"
        >
          <span
            class="att-name"
            :class="{ 'att-previewable': isPreviewable(att) }"
            @click="isPreviewable(att) && openPreview(att)"
          >📎 {{ att.filename || 'attachment' }}</span>
          <span class="att-size">{{ formatSize(att.size) }}</span>
          <a :href="attachmentUrl(att)" download class="att-download" title="Download">↓</a>
        </div>
      </div>

      <!-- Attachment preview modal -->
      <div v-if="previewAtt" class="preview-overlay" @click.self="previewAtt = null">
        <div class="preview-modal">
          <div class="preview-toolbar">
            <span class="preview-title">{{ previewAtt.filename || 'attachment' }}</span>
            <a :href="attachmentUrl(previewAtt)" download class="preview-download">Download</a>
            <button @click="previewAtt = null" class="preview-close">✕</button>
          </div>
          <div class="preview-body">
            <img
              v-if="previewAtt.content_type?.startsWith('image/')"
              :src="attachmentUrl(previewAtt)"
              :alt="previewAtt.filename"
              class="preview-image"
            />
            <iframe
              v-else-if="previewAtt.content_type === 'application/pdf'"
              :src="attachmentUrl(previewAtt)"
              class="preview-pdf"
              title="PDF preview"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, watch, computed, onMounted, onUnmounted } from 'vue'
import DOMPurify from 'dompurify'
import { useMailStore } from '../stores/mail'
import { useContactsStore } from '../stores/contacts'
import { useCalendarStore } from '../stores/calendar'
import { useSettingsStore } from '../stores/settings'
import { useDarkMode } from '../composables/useDarkMode'
import { extractEmail, buildReplyAllCc, isPreviewable } from '../utils/mail.js'
import ConfirmDialog from './ConfirmDialog.vue'

const mail = useMailStore()
const contacts = useContactsStore()
const calendar = useCalendarStore()
const settings = useSettingsStore()
const compose = inject('compose')
const { dark } = useDarkMode()

const confirmDeleteVisible = ref(false)
const inviteAdding = ref(false)
const inviteAdded = ref(false)
const moveOpen = ref(false)
const moveWrapEl = ref(null)
const replyMenuOpen = ref(false)
const replyWrapEl = ref(null)
const forwardMenuOpen = ref(false)
const forwardWrapEl = ref(null)
const showRemoteImages = ref(false)
const hasRemoteImages = ref(false)
const phishingCount = ref(0)
const processedHtml = ref(null)
const sourceOpen = ref(false)
const sourceText = ref('')
const sourceCopied = ref(false)
const iframeEl = ref(null)
const previewAtt = ref(null)

function openPreview(att) {
  previewAtt.value = att
}

// Placeholder: grey box with image icon, sized to replace the original image
const PLACEHOLDER_SRC =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='48' height='48' viewBox='0 0 48 48'%3E%3Crect width='48' height='48' rx='4' fill='%23e8e8e8'/%3E%3Cpath d='M14 34l8-10 6 7 4-5 8 8H14z' fill='%23bbb'/%3E%3Ccircle cx='32' cy='18' r='4' fill='%23bbb'/%3E%3C/svg%3E"

function isRemoteUrl(url) {
  return url && (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('//'))
}

// Returns the registrable domain (last two labels) of a hostname.
// e.g. "mail.google.com" → "google.com"
function registrableDomain(hostname) {
  const parts = hostname.split('.')
  return parts.length >= 2 ? parts.slice(-2).join('.') : hostname
}

function blockRemoteImages(doc) {
  let found = false

  // Block <img src="..."> remote URLs
  for (const img of doc.querySelectorAll('img[src]')) {
    const src = img.getAttribute('src')
    if (isRemoteUrl(src)) {
      img.setAttribute('data-original-src', src)
      img.setAttribute('src', PLACEHOLDER_SRC)
      img.setAttribute('title', src)
      found = true
    }
  }

  // Block inline style background-image: url(http...) / url(//)
  const remoteUrlPattern = /url\(\s*['"]?(?:https?:|\/\/)([^)'"\s]+)['"]?\s*\)/gi
  for (const el of doc.querySelectorAll('[style]')) {
    const style = el.getAttribute('style')
    if (remoteUrlPattern.test(style)) {
      remoteUrlPattern.lastIndex = 0
      el.setAttribute('data-original-style', style)
      el.setAttribute('style', style.replace(remoteUrlPattern, 'none'))
      found = true
    }
    remoteUrlPattern.lastIndex = 0
  }

  // Block <style> blocks with remote url() references
  for (const styleEl of doc.querySelectorAll('style')) {
    if (remoteUrlPattern.test(styleEl.textContent)) {
      remoteUrlPattern.lastIndex = 0
      styleEl.textContent = styleEl.textContent.replace(remoteUrlPattern, 'none')
      found = true
    }
    remoteUrlPattern.lastIndex = 0
  }

  return found
}

function flagPhishingLinks(doc) {
  let count = 0

  for (const a of doc.querySelectorAll('a[href]')) {
    const href = a.getAttribute('href') ?? ''

    // Open all absolute links safely in a new tab
    if (/^https?:\/\//i.test(href)) {
      a.setAttribute('target', '_blank')
      a.setAttribute('rel', 'noopener noreferrer')
    }

    // Phishing check: only applies to http(s) links whose visible text also
    // looks like a URL (starts with http(s):// or www.)
    if (!/^https?:\/\//i.test(href)) continue
    const text = a.textContent.trim()
    if (!text || (!/^https?:\/\//i.test(text) && !/^www\./i.test(text))) continue

    let hrefHost, textHost
    try { hrefHost = new URL(href).hostname.toLowerCase() } catch { continue }
    try { textHost = new URL(/^https?:\/\//i.test(text) ? text : 'https://' + text).hostname.toLowerCase() } catch { continue }

    if (registrableDomain(hrefHost) === registrableDomain(textHost)) continue

    a.classList.add('letrvu-phishing-warn')
    a.setAttribute('title',
      `⚠ Misleading link: displayed as "${registrableDomain(textHost)}" but goes to "${registrableDomain(hrefHost)}"`)
    count++
  }

  if (count > 0) {
    const style = doc.createElement('style')
    style.textContent =
      'a.letrvu-phishing-warn { color: #c0392b !important; text-decoration: underline wavy #c0392b !important; }' +
      "a.letrvu-phishing-warn::after { content: ' ⚠'; }"
    doc.head.appendChild(style)
  }

  return count
}

// Replace cid: references with base64 data URLs supplied by the backend.
function resolveCIDs(html, inlineImages) {
  if (!inlineImages) return html
  return html.replace(/src="cid:([^"]+)"/gi, (match, cid) => {
    const dataUrl = inlineImages[cid]
    return dataUrl ? `src="${dataUrl}"` : match
  })
}

// Full processing pipeline: CID resolution → DOMPurify → image blocking → phishing detection.
// blockImages controls whether remote images are replaced with placeholders.
// isDark injects a low-specificity dark baseline so the srcdoc background
// matches the app theme when the email itself doesn't set an explicit colour.
function processHtml(html, inlineImages, blockImages, isDark = false) {
  const resolved = resolveCIDs(html, inlineImages)
  const sanitized = DOMPurify.sanitize(resolved, { WHOLE_DOCUMENT: true })
  const doc = new DOMParser().parseFromString(sanitized, 'text/html')
  if (isDark) {
    const style = doc.createElement('style')
    // Element-level selectors are overridden by any email-specific inline styles
    // or stylesheet rules that come later in the document.
    style.textContent = 'body{background-color:#141412;color:#e0e0de}a:not([style*="color"]){color:#7ab3ef}'
    doc.head.prepend(style)
  }
  const foundImages = blockImages ? blockRemoteImages(doc) : false
  const foundPhishing = flagPhishingLinks(doc)
  return { html: doc.documentElement.outerHTML, hasRemoteImages: foundImages, phishingCount: foundPhishing }
}

function reprocessCurrentMessage() {
  const msg = mail.currentMessage
  if (!msg?.html_body) {
    processedHtml.value = null
    hasRemoteImages.value = false
    phishingCount.value = 0
    return
  }
  const result = processHtml(msg.html_body, msg.inline_images, true, dark.value)
  processedHtml.value = result.html
  hasRemoteImages.value = result.hasRemoteImages
  phishingCount.value = result.phishingCount
}

watch(
  () => mail.currentMessage?.uid,
  () => {
    showRemoteImages.value = false
    reprocessCurrentMessage()
  },
  { immediate: true }
)

// Re-inject dark/light baseline when the theme is toggled.
watch(dark, reprocessCurrentMessage)

const displayHtml = computed(() => {
  if (!showRemoteImages.value) return processedHtml.value
  const msg = mail.currentMessage
  return processHtml(msg?.html_body ?? '', msg?.inline_images, false, dark.value).html
})

// Resize the iframe to match its content height so the outer pane scrolls
// rather than the iframe itself. Requires allow-same-origin in the sandbox.
function resizeIframe() {
  const frame = iframeEl.value
  if (!frame) return
  const doc = frame.contentDocument
  if (!doc) return
  frame.style.height = doc.documentElement.scrollHeight + 'px'
}

// Reset iframe height when content changes so the @load event fires fresh.
watch(displayHtml, () => {
  if (iframeEl.value) iframeEl.value.style.height = ''
})

const otherFolders = computed(() =>
  mail.folders.filter(f => f.name !== mail.currentFolder)
)

const isDraftsFolder = computed(() =>
  ['drafts', 'draft'].includes(mail.currentFolder.toLowerCase())
)

const isJunkFolder = computed(() =>
  ['junk', 'junk email', 'spam'].includes(mail.currentFolder.toLowerCase())
)

const debugMode = import.meta.env.VITE_LOG_LEVEL === 'debug'
function debugLog(...args) {
  if (debugMode) console.debug('[letrvu]', ...args)
}

// Authentication failure: only warn on explicit hard fails recorded by the
// receiving MTA in Authentication-Results. Softfail/neutral/none are omitted
// intentionally — they cause too many false positives (e.g. forwarded mail).
const authFailed = computed(() => {
  const msg = mail.currentMessage
  if (!msg) return false
  return msg.dmarc === 'fail' || msg.spf === 'fail'
})

const authFailedMethods = computed(() => {
  const msg = mail.currentMessage
  if (!msg) return ''
  const failed = []
  if (msg.spf === 'fail') failed.push('SPF')
  if (msg.dmarc === 'fail') failed.push('DMARC')
  return failed.join(' and ')
})

// Extract the domain from a "Name <user@domain>" or "user@domain" From field.
function senderDomain(msg) {
  const from = msg.from ?? ''
  const match = from.match(/^.*?<(.+?)>\s*$/)
  const email = match ? match[1] : from.trim()
  return email.split('@')[1]?.toLowerCase() ?? ''
}

// isExternal: INTERNAL_DOMAINS configured + auth results present + From domain
// is not internal. We require auth results so the From domain is verified, not
// just claimed.
const isExternal = computed(() => {
  const domains = settings.internalDomains
  debugLog('isExternal: internalDomains =', domains)
  if (!domains.length) {
    debugLog('isExternal: no internal domains configured → skip')
    return false
  }
  const msg = mail.currentMessage
  if (!msg) return false
  debugLog('isExternal: msg.spf =', msg.spf, 'msg.dkim =', msg.dkim, 'msg.dmarc =', msg.dmarc)
  if (!msg.spf && !msg.dkim && !msg.dmarc) {
    debugLog('isExternal: no auth results → defer to isUnverified')
    return false
  }
  const domain = senderDomain(msg)
  debugLog('isExternal: from domain =', domain, '| external =', !domains.includes(domain))
  return domain !== '' && !domains.includes(domain)
})

// isUnverified: INTERNAL_DOMAINS configured + NO auth results + From domain is
// not internal. We can't confirm the sender but the claimed domain is outside
// the organisation. Shown as a neutral grey badge rather than amber.
const isUnverified = computed(() => {
  const domains = settings.internalDomains
  if (!domains.length) return false
  const msg = mail.currentMessage
  if (!msg) return false
  if (msg.spf || msg.dkim || msg.dmarc) return false // auth present → handled by isExternal
  const domain = senderDomain(msg)
  debugLog('isUnverified: from domain =', domain, '| unverified =', !domains.includes(domain))
  return domain !== '' && !domains.includes(domain)
})

function onDocClick(e) {
  if (moveWrapEl.value && !moveWrapEl.value.contains(e.target)) moveOpen.value = false
  if (replyWrapEl.value && !replyWrapEl.value.contains(e.target)) replyMenuOpen.value = false
  if (forwardWrapEl.value && !forwardWrapEl.value.contains(e.target)) forwardMenuOpen.value = false
}
function onKeydown(e) {
  if (e.key !== 'Escape') return
  if (previewAtt.value) { previewAtt.value = null; return }
  if (sourceOpen.value) { sourceOpen.value = false; return }
}
onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onKeydown)
})
onUnmounted(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onKeydown)
})

// Reset invite state when message changes
watch(() => mail.currentMessage?.uid, () => { inviteAdded.value = false })

function formatDate(dateStr) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString()
}

function formatSize(bytes) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function attachmentUrl(att) {
  const folder = encodeURIComponent(mail.currentFolder)
  const uid = mail.currentMessage.uid
  return `/api/folders/${folder}/messages/${uid}/attachments/${att.index}`
}

async function editDraft() {
  const msg = mail.currentMessage
  if (!msg) return

  // Extract the bare email address from "Name <email>" or plain "email".
  const from = msg.from || ''
  const angleMatch = from.match(/<(.+?)>/)
  const fromEmail = angleMatch ? angleMatch[1].trim() : from.trim()

  compose?.value?.open({
    _fromEmail: fromEmail,
    _noSignature: true,
    _draftFolder: mail.currentFolder,
    _draftUid: msg.uid,
    to: (msg.to ?? []).join(', '),
    cc: (msg.cc ?? []).join(', '),
    subject: msg.subject ?? '',
    body: msg.text_body ?? '',
  })
}

function reply() {
  replyMenuOpen.value = false
  const msg = mail.currentMessage
  if (!msg) return

  // Build quoted body from plain text, or fall back to stripping HTML.
  let bodyText = msg.text_body || ''
  if (!bodyText && msg.html_body) {
    const tmp = document.createElement('div')
    tmp.innerHTML = msg.html_body
    for (const el of tmp.querySelectorAll('style, script, head')) el.remove()
    bodyText = tmp.innerText.replace(/\n{3,}/g, '\n\n').trim()
  }

  const date = msg.date ? new Date(msg.date).toLocaleString() : ''
  const quoted = bodyText
    .split('\n')
    .map(line => `> ${line}`)
    .join('\n')

  // Reply-To takes precedence over From when present.
  const replyTo = msg.reply_to || msg.from

  compose?.value?.open({
    to: replyTo,
    subject: msg.subject?.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`,
    body: `\n\nOn ${date}, ${msg.from} wrote:\n${quoted}`,
    // Pass original recipients so ComposeModal can pick the matching identity.
    _originalRecipients: [...(msg.to ?? []), ...(msg.cc ?? [])],
    _inReplyTo: msg.message_id || '',
    _references: [msg.references, msg.message_id].filter(Boolean).join(' '),
  })
}

function replyAll() {
  replyMenuOpen.value = false
  const msg = mail.currentMessage
  if (!msg) return

  let bodyText = msg.text_body || ''
  if (!bodyText && msg.html_body) {
    const tmp = document.createElement('div')
    tmp.innerHTML = msg.html_body
    for (const el of tmp.querySelectorAll('style, script, head')) el.remove()
    bodyText = tmp.innerText.replace(/\n{3,}/g, '\n\n').trim()
  }

  const date = msg.date ? new Date(msg.date).toLocaleString() : ''
  const quoted = bodyText.split('\n').map(line => `> ${line}`).join('\n')
  const replyTo = msg.reply_to || msg.from

  const ownEmails = settings.fromOptions.map(opt => opt.email)
  const cc = buildReplyAllCc(msg.to, msg.cc, replyTo, ownEmails)

  compose?.value?.open({
    to: replyTo,
    cc,
    subject: msg.subject?.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`,
    body: `\n\nOn ${date}, ${msg.from} wrote:\n${quoted}`,
    _originalRecipients: [...(msg.to ?? []), ...(msg.cc ?? [])],
    _inReplyTo: msg.message_id || '',
    _references: [msg.references, msg.message_id].filter(Boolean).join(' '),
  })
}

function forwardInline() {
  forwardMenuOpen.value = false
  const msg = mail.currentMessage
  if (!msg) return

  // Prefer plain text body; fall back to stripping HTML tags for HTML-only emails.
  let bodyText = msg.text_body || ''
  if (!bodyText && msg.html_body) {
    const tmp = document.createElement('div')
    tmp.innerHTML = msg.html_body
    for (const el of tmp.querySelectorAll('style, script, head')) el.remove()
    bodyText = tmp.innerText.replace(/\n{3,}/g, '\n\n').trim()
  }

  const date = msg.date ? new Date(msg.date).toLocaleString() : ''
  const to = Array.isArray(msg.to) ? msg.to.join(', ') : (msg.to || '')

  const header = [
    '--- Forwarded message ---',
    `From: ${msg.from}`,
    date ? `Date: ${date}` : '',
    `Subject: ${msg.subject || ''}`,
    to ? `To: ${to}` : '',
  ].filter(Boolean).join('\n')

  compose?.value?.open({
    subject: `Fwd: ${msg.subject || ''}`,
    body: `\n\n${header}\n\n${bodyText}`,
  })
}

async function forwardAsAttachment() {
  forwardMenuOpen.value = false
  const msg = mail.currentMessage
  if (!msg) return

  const folder = encodeURIComponent(mail.currentFolder)
  const res = await fetch(`/api/folders/${folder}/messages/${msg.uid}/source`)
  if (!res.ok) return

  const buf = await res.arrayBuffer()
  const uint8 = new Uint8Array(buf)
  let binary = ''
  for (const b of uint8) binary += String.fromCharCode(b)
  const base64 = btoa(binary)

  const filename = `${(msg.subject || 'message').replace(/[/\\?%*:|"<>]/g, '_')}.eml`

  compose?.value?.open({
    subject: `Fwd: ${msg.subject || ''}`,
    _attachments: [{ filename, content_type: 'message/rfc822', data: base64 }],
  })
}

async function spam() {
  const msg = mail.currentMessage
  if (!msg) return
  await mail.markAsSpam(mail.currentFolder, [msg.uid])
}

async function doDelete() {
  const msg = mail.currentMessage
  if (!msg) return
  confirmDeleteVisible.value = false
  await mail.deleteMessage(mail.currentFolder, msg.uid)
}

function remove() {
  if (mail.currentMessage) confirmDeleteVisible.value = true
}

async function addToCalendar() {
  inviteAdding.value = true
  try {
    await calendar.importFromInvite(mail.currentMessage.ical_invite)
    inviteAdded.value = true
  } catch {
    alert('Could not add event to calendar.')
  } finally {
    inviteAdding.value = false
  }
}

function toggleFlagged() {
  const msg = mail.currentMessage
  if (!msg) return
  mail.markFlagged(mail.currentFolder, msg.uid, !msg.flagged)
}

function toggleRead() {
  const msg = mail.currentMessage
  if (!msg) return
  mail.markRead(mail.currentFolder, msg.uid, !msg.read)
}

async function moveTo(dest) {
  moveOpen.value = false
  const msg = mail.currentMessage
  if (!msg) return
  await mail.moveMessage(mail.currentFolder, msg.uid, dest)
}

async function viewSource() {
  const msg = mail.currentMessage
  if (!msg) return
  const folder = encodeURIComponent(mail.currentFolder)
  const res = await fetch(`/api/folders/${folder}/messages/${msg.uid}/source`)
  if (!res.ok) { alert('Could not fetch message source.'); return }
  sourceText.value = await res.text()
  sourceCopied.value = false
  sourceOpen.value = true
}

async function copySource() {
  await navigator.clipboard.writeText(sourceText.value)
  sourceCopied.value = true
  setTimeout(() => { sourceCopied.value = false }, 2000)
}

defineExpose({ reply, remove })

async function saveContact() {
  const msg = mail.currentMessage
  if (!msg) return
  // Parse "Name <email>" or plain email from the from field
  const from = msg.from || ''
  const match = from.match(/^(.*?)\s*<(.+?)>$/)
  const name = match ? match[1].trim() : ''
  const email = match ? match[2].trim() : from.trim()
  try {
    await contacts.saveFromMessage(name, email)
    alert(`Saved ${email} to address book.`)
  } catch {
    alert('Could not save contact.')
  }
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
.message { padding: 2rem; max-width: 80%; margin: 0 auto; }
.header { margin-bottom: 1.5rem; }
h2 { font-size: 18px; font-weight: 500; margin-bottom: 0.5rem; }
.meta { font-size: 13px; color: var(--color-text-muted); margin-bottom: 1rem; display: flex; gap: 1rem; align-items: center; }
.external-badge,
.unverified-badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.03em;
  white-space: nowrap;
}
.external-badge {
  background: #fff4e5;
  border: 0.5px solid #e09030;
  color: #7a4000;
}
.unverified-badge {
  background: #f2f2f2;
  border: 0.5px solid #b0b0b0;
  color: #555;
}
.save-contact-btn {
  background: none;
  border: 0.5px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
  padding: 1px 6px;
  color: var(--color-teal);
}
.save-contact-btn:hover { background: var(--color-teal-light); }
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
.edit-draft-btn { color: var(--color-teal); border-color: var(--color-teal) !important; font-weight: 500; }
.actions button.active { color: #e67e22; border-color: #f5c6a0; background: #fef9ec; }
.move-wrap, .reply-wrap, .forward-wrap { position: relative; }
.move-dropdown, .reply-dropdown, .forward-dropdown {
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
  max-height: 260px;
  overflow-y: auto;
  z-index: 50;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}
.move-dropdown li, .reply-dropdown li, .forward-dropdown li {
  padding: 6px 14px;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
}
.move-dropdown li:hover, .reply-dropdown li:hover, .forward-dropdown li:hover { background: var(--color-teal-light); }
.invite-banner, .remote-images-banner, .phishing-banner, .auth-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: 13px;
  margin-bottom: 1rem;
}
.invite-banner {
  background: var(--color-teal-light);
  border: 0.5px solid var(--color-teal);
}
.remote-images-banner {
  background: #fef9ec;
  border: 0.5px solid #e6b84a;
  color: #7a5800;
}
.phishing-banner {
  background: #fdf0f0;
  border: 0.5px solid #e07070;
  color: #7a1a1a;
}
.auth-banner {
  background: #fff4e5;
  border: 0.5px solid #e09030;
  color: #7a4000;
}
.show-images-btn {
  padding: 5px 14px;
  background: #e6b84a;
  color: #3a2800;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
  margin-left: auto;
}
.show-images-btn:hover { background: #d4a830; }
.invite-btn {
  padding: 5px 14px;
  background: var(--color-teal);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
}
.invite-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.body-frame {
  width: 100%;
  min-height: 200px;
  border: 0.5px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-bg);
  display: block;
}
.body-text {
  white-space: pre-wrap;
  font-family: inherit;
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text);
}
.attachments {
  margin-top: 1.5rem;
  border-top: 0.5px solid var(--color-border);
  padding-top: 1rem;
}
.attachments-label {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 0.5rem;
}
.attachment {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 12px;
  text-decoration: none;
  color: var(--color-text);
  margin-right: 8px;
  margin-bottom: 6px;
}
.attachment:hover { background: var(--color-bg); }
.att-size { color: var(--color-text-muted); }
.att-name { cursor: default; }
.att-previewable { cursor: pointer; color: var(--color-teal); text-decoration: underline; }
.att-previewable:hover { opacity: 0.8; }
.att-download {
  color: var(--color-text-muted);
  text-decoration: none;
  font-size: 14px;
  padding: 0 2px;
}
.att-download:hover { color: var(--color-text); }
.preview-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.65);
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
}
.preview-modal {
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  width: min(900px, 94vw);
  height: min(700px, 90vh);
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.25);
}
.preview-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 0.5px solid var(--color-border);
  flex-shrink: 0;
}
.preview-title { font-size: 13px; font-weight: 500; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.preview-download, .preview-close {
  padding: 4px 12px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  font-size: 12px;
  cursor: pointer;
  text-decoration: none;
  color: var(--color-text);
}
.preview-download:hover, .preview-close:hover { background: var(--color-bg); }
.preview-body {
  flex: 1;
  overflow: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
  background: var(--color-bg);
  border-radius: 0 0 10px 10px;
}
.preview-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 4px;
}
.preview-pdf {
  width: 100%;
  height: 100%;
  border: none;
}
.source-btn {
  padding: 6px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  font-size: 12px;
  font-family: monospace;
  cursor: pointer;
  color: var(--color-text-muted);
  margin-left: auto;
}
.source-btn:hover { background: var(--color-bg); color: var(--color-text); }
.source-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
}
.source-modal {
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  width: min(860px, 92vw);
  height: min(640px, 85vh);
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.18);
}
.source-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 0.5px solid var(--color-border);
  flex-shrink: 0;
}
.source-title { font-size: 13px; font-weight: 500; flex: 1; }
.source-copy, .source-close {
  padding: 4px 12px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  font-size: 12px;
  cursor: pointer;
}
.source-copy:hover, .source-close:hover { background: var(--color-bg); }
.source-body {
  flex: 1;
  overflow: auto;
  margin: 0;
  padding: 14px 16px;
  font-family: monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--color-text);
}
</style>
