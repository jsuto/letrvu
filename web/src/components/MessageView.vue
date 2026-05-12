<template>
  <div class="h-full overflow-y-auto">
    <ConfirmDialog
      v-model:visible="confirmDeleteVisible"
      :message="`Delete &quot;${mail.currentMessage?.subject || '(no subject)'}&quot;?`"
      @confirm="doDelete"
    />
    <div v-if="!mail.currentMessage" class="h-full flex items-center justify-center text-[var(--color-text-muted)] text-sm">
      <p>Select a message to read</p>
    </div>
    <div v-else class="px-8 py-8 max-w-[80%] mx-auto">
      <div class="mb-6">
        <h2 class="text-lg font-medium mb-2">{{ mail.currentMessage.subject || '(no subject)' }}</h2>
        <div class="text-sm text-[var(--color-text-muted)] mb-4 flex gap-4 items-center">
          <span>{{ mail.currentMessage.from }}</span>
          <span v-if="isExternal" class="inline-block px-1.5 py-px rounded text-[11px] font-medium tracking-wide whitespace-nowrap bg-[#fff4e5] border border-[#e09030] text-[#7a4000]" title="Sender is outside your organisation — authentication passed">External</span>
          <span v-else-if="isUnverified" class="inline-block px-1.5 py-px rounded text-[11px] font-medium tracking-wide whitespace-nowrap bg-[#f2f2f2] border border-[#b0b0b0] text-[#555]" title="Sender appears to be outside your organisation — no authentication results available">Unverified</span>
          <button class="bg-none border border-[var(--color-border)] rounded px-1.5 py-px cursor-pointer text-sm font-semibold text-teal hover:bg-[var(--color-teal-light)]" title="Save to address book" @click="saveContact">+</button>
          <span>{{ formatDate(mail.currentMessage.date) }}</span>
        </div>
        <div class="flex gap-2 flex-wrap">
          <button v-if="isDraftsFolder" @click="editDraft"
            class="px-3.5 py-1.5 border border-teal rounded-md bg-[var(--color-surface)] text-sm cursor-pointer text-teal font-medium">Edit Draft</button>
          <div v-if="!isDraftsFolder" class="relative" ref="replyWrapEl">
            <button @click="replyMenuOpen = !replyMenuOpen" class="px-3.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm cursor-pointer hover:bg-[var(--color-bg)]">Reply ▾</button>
            <ul v-if="replyMenuOpen" class="absolute top-[calc(100%+4px)] left-0 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-md list-none m-0 py-1 min-w-[160px] z-50 shadow-lg">
              <li @click="reply" class="px-3.5 py-1.5 text-sm cursor-pointer whitespace-nowrap hover:bg-[var(--color-teal-light)]">Reply</li>
              <li @click="replyAll" class="px-3.5 py-1.5 text-sm cursor-pointer whitespace-nowrap hover:bg-[var(--color-teal-light)]">Reply All</li>
            </ul>
          </div>
          <div v-if="!isDraftsFolder" class="relative" ref="forwardWrapEl">
            <button @click="forwardMenuOpen = !forwardMenuOpen" class="px-3.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm cursor-pointer hover:bg-[var(--color-bg)]">Forward ▾</button>
            <ul v-if="forwardMenuOpen" class="absolute top-[calc(100%+4px)] left-0 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-md list-none m-0 py-1 min-w-[160px] z-50 shadow-lg">
              <li @click="forwardInline" class="px-3.5 py-1.5 text-sm cursor-pointer whitespace-nowrap hover:bg-[var(--color-teal-light)]">Inline</li>
              <li @click="forwardAsAttachment" class="px-3.5 py-1.5 text-sm cursor-pointer whitespace-nowrap hover:bg-[var(--color-teal-light)]">As .eml attachment</li>
            </ul>
          </div>
          <button
            :class="['px-3.5 py-1.5 border rounded-md bg-[var(--color-surface)] text-sm cursor-pointer hover:bg-[var(--color-bg)]', mail.currentMessage.flagged ? 'text-orange-400 border-[#f5c6a0] bg-[#fef9ec]' : 'border-[var(--color-border)]']"
            :title="mail.currentMessage.flagged ? 'Unflag' : 'Flag as important'"
            @click="toggleFlagged"
          >{{ mail.currentMessage.flagged ? '★' : '☆' }} Flag</button>
          <button
            :title="mail.currentMessage.read ? 'Mark as unread' : 'Mark as read'"
            @click="toggleRead"
            class="px-3.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm cursor-pointer hover:bg-[var(--color-bg)]"
          >{{ mail.currentMessage.read ? 'Mark unread' : 'Mark read' }}</button>
          <div class="relative" ref="moveWrapEl">
            <button @click="moveOpen = !moveOpen" class="px-3.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm cursor-pointer hover:bg-[var(--color-bg)]">Move to…</button>
            <ul v-if="moveOpen" class="absolute top-[calc(100%+4px)] left-0 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-md list-none m-0 py-1 min-w-[160px] max-h-[260px] overflow-y-auto z-50 shadow-lg">
              <li v-for="f in otherFolders" :key="f.name" @click="moveTo(f.name)" class="px-3.5 py-1.5 text-sm cursor-pointer whitespace-nowrap hover:bg-[var(--color-teal-light)]">{{ f.name }}</li>
            </ul>
          </div>
          <button v-if="!isJunkFolder" @click="spam" title="Move to Junk" class="px-3.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm cursor-pointer hover:bg-[var(--color-bg)]">Spam</button>
          <button @click="confirmDeleteVisible = true" class="px-3.5 py-1.5 border border-red-200 rounded-md bg-[var(--color-surface)] text-sm cursor-pointer text-red-600 hover:bg-[var(--color-bg)]">Delete</button>
          <button @click="viewSource" title="View message source" class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs font-mono cursor-pointer text-[var(--color-text-muted)] ml-auto hover:bg-[var(--color-bg)] hover:text-[var(--color-text)]">&lt;/&gt;</button>
        </div>
      </div>

      <!-- Message source modal -->
      <div v-if="sourceOpen" class="fixed inset-0 bg-black/45 z-[200] flex items-center justify-center" @click.self="sourceOpen = false">
        <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl flex flex-col shadow-2xl" style="width: min(860px, 92vw); height: min(640px, 85vh)">
          <div class="flex items-center gap-2 px-3.5 py-2.5 border-b border-[var(--color-border)] shrink-0">
            <span class="text-sm font-medium flex-1">Message source</span>
            <button @click="copySource" class="px-3 py-1 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:bg-[var(--color-bg)]">{{ sourceCopied ? 'Copied!' : 'Copy' }}</button>
            <button @click="sourceOpen = false" class="px-3 py-1 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:bg-[var(--color-bg)]">✕</button>
          </div>
          <pre class="flex-1 overflow-auto m-0 px-4 py-3.5 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all text-[var(--color-text)]">{{ sourceText }}</pre>
        </div>
      </div>

      <!-- Calendar invite banner -->
      <div v-if="mail.currentMessage.ical_invite" class="flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm mb-4 bg-[var(--color-teal-light)] border border-teal">
        <span>📅 This message contains a calendar invite.</span>
        <button @click="addToCalendar" :disabled="inviteAdding"
          class="px-3.5 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer whitespace-nowrap disabled:opacity-60 disabled:cursor-not-allowed ml-auto">
          {{ inviteAdded ? 'Added ✓' : inviteAdding ? 'Adding…' : 'Add to calendar' }}
        </button>
      </div>

      <!-- Remote image blocking banner -->
      <div v-if="hasRemoteImages && !showRemoteImages" class="flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm mb-4 bg-[#fef9ec] border border-[#e6b84a] text-[#7a5800]">
        <span>🛡 Remote images blocked to protect your privacy.</span>
        <button @click="showRemoteImages = true"
          class="px-3.5 py-1.5 bg-[#e6b84a] text-[#3a2800] border-none rounded-md text-xs cursor-pointer whitespace-nowrap ml-auto hover:bg-[#d4a830]">Show images</button>
      </div>

      <!-- Email authentication failure banner (SPF/DKIM/DMARC) -->
      <div v-if="authFailed" class="flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm mb-4 bg-[#fff4e5] border border-[#e09030] text-[#7a4000]">
        <span>⚠ Authentication failed — this message did not pass {{ authFailedMethods }} checks and may be spoofed or forged.</span>
      </div>

      <!-- Phishing link warning banner -->
      <div v-if="phishingCount > 0" class="flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm mb-4 bg-[#fdf0f0] border border-[#e07070] text-[#7a1a1a]">
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
        class="w-full min-h-[200px] border border-[var(--color-border)] rounded-lg bg-[var(--color-bg)] block"
        sandbox="allow-popups allow-same-origin"
        :srcdoc="displayHtml"
        title="Message body"
        @load="resizeIframe"
      />
      <pre v-else class="whitespace-pre-wrap text-sm leading-7 text-[var(--color-text)]">{{ mail.currentMessage.text_body }}</pre>

      <div v-if="mail.currentMessage.attachments?.length" class="mt-6 border-t border-[var(--color-border)] pt-4">
        <p class="text-xs text-[var(--color-text-muted)] mb-2">Attachments</p>
        <div
          v-for="att in mail.currentMessage.attachments"
          :key="att.index"
          class="inline-flex items-center gap-1.5 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-xs text-[var(--color-text)] mr-2 mb-1.5 hover:bg-[var(--color-bg)]"
        >
          <span
            :class="['cursor-default', isPreviewable(att) ? 'cursor-pointer text-teal underline hover:opacity-80' : '']"
            @click="isPreviewable(att) && openPreview(att)"
          >📎 {{ att.filename || 'attachment' }}</span>
          <span class="text-[var(--color-text-muted)]">{{ formatSize(att.size) }}</span>
          <a :href="attachmentUrl(att)" download class="text-[var(--color-text-muted)] no-underline text-sm px-0.5 hover:text-[var(--color-text)]" title="Download">↓</a>
        </div>
      </div>

      <!-- Attachment preview modal -->
      <div v-if="previewAtt" class="fixed inset-0 bg-black/65 z-[200] flex items-center justify-center" @click.self="previewAtt = null">
        <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl flex flex-col shadow-2xl" style="width: min(900px, 94vw); height: min(700px, 90vh)">
          <div class="flex items-center gap-2 px-3.5 py-2.5 border-b border-[var(--color-border)] shrink-0">
            <span class="text-sm font-medium flex-1 overflow-hidden text-ellipsis whitespace-nowrap">{{ previewAtt.filename || 'attachment' }}</span>
            <a :href="attachmentUrl(previewAtt)" download class="px-3 py-1 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer no-underline text-[var(--color-text)] hover:bg-[var(--color-bg)]">Download</a>
            <button @click="previewAtt = null" class="px-3 py-1 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:bg-[var(--color-bg)]">✕</button>
          </div>
          <div class="flex-1 overflow-auto flex items-center justify-center p-3 bg-[var(--color-bg)] rounded-b-xl">
            <img
              v-if="previewAtt.content_type?.startsWith('image/')"
              :src="attachmentUrl(previewAtt)"
              :alt="previewAtt.filename"
              class="max-w-full max-h-full object-contain rounded"
            />
            <iframe
              v-else-if="previewAtt.content_type === 'application/pdf'"
              :src="attachmentUrl(previewAtt)"
              class="w-full h-full border-none"
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

// --- Helpers used by reply / forward ---

function escHtml(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function plainToHtml(text) {
  if (!text) return ''
  return text.split('\n').map(l => `<p>${escHtml(l) || '<br>'}</p>`).join('')
}

function buildQuoteHtml(msg, date) {
  const bodyHtml = msg.html_body || plainToHtml(msg.text_body || '')
  return `<p>On ${escHtml(date)}, ${escHtml(msg.from || '')} wrote:</p><blockquote>${bodyHtml}</blockquote>`
}

async function editDraft() {
  const msg = mail.currentMessage
  if (!msg) return

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
    // Prefer saved HTML body; fall back to plain text.
    html: msg.html_body || undefined,
    body: !msg.html_body ? (msg.text_body ?? '') : undefined,
  })
}

function reply() {
  replyMenuOpen.value = false
  const msg = mail.currentMessage
  if (!msg) return

  const date = msg.date ? new Date(msg.date).toLocaleString() : ''
  const replyTo = msg.reply_to || msg.from
  const quotedHtml = buildQuoteHtml(msg, date)

  compose?.value?.open({
    to: replyTo,
    subject: msg.subject?.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`,
    html: quotedHtml,
    _originalRecipients: [...(msg.to ?? []), ...(msg.cc ?? [])],
    _inReplyTo: msg.message_id || '',
    _references: [msg.references, msg.message_id].filter(Boolean).join(' '),
  })
}

function replyAll() {
  replyMenuOpen.value = false
  const msg = mail.currentMessage
  if (!msg) return

  const date = msg.date ? new Date(msg.date).toLocaleString() : ''
  const replyTo = msg.reply_to || msg.from
  const quotedHtml = buildQuoteHtml(msg, date)

  const ownEmails = settings.fromOptions.map(opt => opt.email)
  const cc = buildReplyAllCc(msg.to, msg.cc, replyTo, ownEmails)

  compose?.value?.open({
    to: replyTo,
    cc,
    subject: msg.subject?.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`,
    html: quotedHtml,
    _originalRecipients: [...(msg.to ?? []), ...(msg.cc ?? [])],
    _inReplyTo: msg.message_id || '',
    _references: [msg.references, msg.message_id].filter(Boolean).join(' '),
  })
}

function forwardInline() {
  forwardMenuOpen.value = false
  const msg = mail.currentMessage
  if (!msg) return

  const date = msg.date ? new Date(msg.date).toLocaleString() : ''
  const to = Array.isArray(msg.to) ? msg.to.join(', ') : (msg.to || '')

  const headerHtml = [
    '<p><strong>--- Forwarded message ---</strong></p>',
    `<p><strong>From:</strong> ${escHtml(msg.from || '')}</p>`,
    date ? `<p><strong>Date:</strong> ${escHtml(date)}</p>` : '',
    `<p><strong>Subject:</strong> ${escHtml(msg.subject || '')}</p>`,
    to ? `<p><strong>To:</strong> ${escHtml(to)}</p>` : '',
  ].filter(Boolean).join('')

  const bodyHtml = msg.html_body || plainToHtml(msg.text_body || '')

  compose?.value?.open({
    subject: `Fwd: ${msg.subject || ''}`,
    html: `<blockquote>${headerHtml}${bodyHtml}</blockquote>`,
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
