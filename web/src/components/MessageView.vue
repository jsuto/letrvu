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
          <button class="save-contact-btn" title="Save to address book" @click="saveContact">+</button>
          <span class="date">{{ formatDate(mail.currentMessage.date) }}</span>
        </div>
        <div class="actions">
          <button @click="reply">Reply</button>
          <button @click="forward">Forward</button>
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
          <button @click="remove" class="danger">Delete</button>
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

      <!-- Phishing link warning banner -->
      <div v-if="phishingCount > 0" class="phishing-banner">
        <span>⚠ {{ phishingCount }} misleading {{ phishingCount === 1 ? 'link' : 'links' }} detected — the visible text shows a different domain than the actual destination.</span>
      </div>

      <!-- HTML email rendered in a sandboxed iframe to prevent XSS.
           allow-popups is required so target="_blank" links can open in a new tab. -->
      <iframe
        v-if="mail.currentMessage.html_body"
        class="body-frame"
        sandbox="allow-popups"
        :srcdoc="displayHtml"
        title="Message body"
      />
      <pre v-else class="body-text">{{ mail.currentMessage.text_body }}</pre>
      <div v-if="mail.currentMessage.attachments?.length" class="attachments">
        <p class="attachments-label">Attachments</p>
        <a
          v-for="att in mail.currentMessage.attachments"
          :key="att.index"
          :href="attachmentUrl(att)"
          download
          class="attachment"
        >
          📎 {{ att.filename || 'attachment' }}
          <span class="att-size">{{ formatSize(att.size) }}</span>
        </a>
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

const mail = useMailStore()
const contacts = useContactsStore()
const calendar = useCalendarStore()
const compose = inject('compose')

const inviteAdding = ref(false)
const inviteAdded = ref(false)
const moveOpen = ref(false)
const moveWrapEl = ref(null)
const showRemoteImages = ref(false)
const hasRemoteImages = ref(false)
const phishingCount = ref(0)
const processedHtml = ref(null)
const sourceOpen = ref(false)
const sourceText = ref('')
const sourceCopied = ref(false)

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
function processHtml(html, inlineImages, blockImages) {
  const resolved = resolveCIDs(html, inlineImages)
  const sanitized = DOMPurify.sanitize(resolved, { WHOLE_DOCUMENT: true })
  const doc = new DOMParser().parseFromString(sanitized, 'text/html')
  const foundImages = blockImages ? blockRemoteImages(doc) : false
  const foundPhishing = flagPhishingLinks(doc)
  return { html: doc.documentElement.outerHTML, hasRemoteImages: foundImages, phishingCount: foundPhishing }
}

watch(
  () => mail.currentMessage?.uid,
  () => {
    showRemoteImages.value = false
    const msg = mail.currentMessage
    if (!msg?.html_body) {
      processedHtml.value = null
      hasRemoteImages.value = false
      phishingCount.value = 0
      return
    }
    const result = processHtml(msg.html_body, msg.inline_images, true)
    processedHtml.value = result.html
    hasRemoteImages.value = result.hasRemoteImages
    phishingCount.value = result.phishingCount
  },
  { immediate: true }
)

const displayHtml = computed(() => {
  if (!showRemoteImages.value) return processedHtml.value
  const msg = mail.currentMessage
  return processHtml(msg?.html_body ?? '', msg?.inline_images, false).html
})

const otherFolders = computed(() =>
  mail.folders.filter(f => f.name !== mail.currentFolder)
)

function onDocClick(e) {
  if (moveWrapEl.value && !moveWrapEl.value.contains(e.target)) moveOpen.value = false
}
onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))

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

function reply() {
  const msg = mail.currentMessage
  if (!msg) return
  compose?.value?.open({
    to: msg.from,
    subject: msg.subject?.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`,
  })
}

function forward() {
  const msg = mail.currentMessage
  if (!msg) return
  compose?.value?.open({
    subject: `Fwd: ${msg.subject || ''}`,
    body: `\n\n--- Forwarded message ---\nFrom: ${msg.from}\n\n${msg.text_body || ''}`,
  })
}

async function remove() {
  const msg = mail.currentMessage
  if (!msg) return
  await mail.deleteMessage(mail.currentFolder, msg.uid)
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
.message { padding: 2rem; max-width: 780px; margin: 0 auto; }
.header { margin-bottom: 1.5rem; }
h2 { font-size: 18px; font-weight: 500; margin-bottom: 0.5rem; }
.meta { font-size: 13px; color: var(--color-text-muted); margin-bottom: 1rem; display: flex; gap: 1rem; align-items: center; }
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
.actions button.active { color: #e67e22; border-color: #f5c6a0; background: #fef9ec; }
.move-wrap { position: relative; }
.move-dropdown {
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
.move-dropdown li {
  padding: 6px 14px;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
}
.move-dropdown li:hover { background: var(--color-teal-light); }
.invite-banner, .remote-images-banner, .phishing-banner {
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
