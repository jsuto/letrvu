<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-[100] flex items-end justify-end p-8 pointer-events-none"
  >
    <div class="pointer-events-auto w-[560px] max-h-[620px] flex flex-col rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-2xl">

      <!-- Header -->
      <div class="flex shrink-0 items-center justify-between border-b border-[var(--color-border)] px-4 py-3">
        <span class="text-[13px] font-medium text-[var(--color-text)]">New message</span>
        <button
          @click="close"
          class="flex h-7 w-7 items-center justify-center rounded-md text-lg text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-bg)] hover:text-[var(--color-text)]"
        >×</button>
      </div>

      <!-- Address fields -->
      <div class="shrink-0">
        <div class="flex items-center border-b border-[var(--color-border)]">
          <span class="shrink-0 px-4 py-2 text-[13px] text-[var(--color-text-muted)]">From</span>
          <select
            v-model="form.fromIndex"
            class="flex-1 cursor-pointer bg-transparent py-2 pr-2 text-[13px] text-[var(--color-text)] outline-none"
          >
            <option v-for="(opt, i) in fromOptions" :key="i" :value="i">{{ opt.label }}</option>
          </select>
        </div>
        <AddressInput v-model="form.to" placeholder="To" />
        <AddressInput v-model="form.cc" placeholder="CC" />
        <input
          v-model="form.subject"
          type="text"
          placeholder="Subject"
          class="block w-full border-b border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-2 text-[13px] text-[var(--color-text)] outline-none placeholder:text-[var(--color-text-muted)]"
        />
      </div>

      <!-- Rich text toolbar -->
      <div
        v-if="!plainTextMode"
        class="flex shrink-0 flex-wrap items-center gap-0.5 border-b border-[var(--color-border)] px-2.5 py-1.5"
      >
        <template v-for="item in toolbarItems" :key="item.title ?? item.sep">
          <span v-if="item.sep" class="mx-1 h-4 w-px shrink-0 bg-[var(--color-border)]" />
          <button
            v-else
            @mousedown.prevent="item.action()"
            :title="item.title"
            :class="[
              'min-w-[26px] rounded px-1.5 py-0.5 text-[13px] leading-snug transition-colors',
              item.active?.()
                ? 'bg-teal-light text-teal'
                : 'text-[var(--color-text)] hover:bg-[var(--color-bg)]'
            ]"
            v-html="item.label"
          />
        </template>
      </div>

      <!-- Rich editor -->
      <EditorContent v-if="!plainTextMode" :editor="editor" class="min-h-[200px] flex-1 overflow-y-auto" />

      <!-- Plain text fallback -->
      <textarea
        v-else
        ref="textareaEl"
        v-model="form.plainBody"
        placeholder="Write your message…"
        class="min-h-[200px] flex-1 resize-none bg-[var(--color-surface)] px-4 py-3 text-[14px] leading-relaxed text-[var(--color-text)] outline-none placeholder:text-[var(--color-text-muted)]"
      />

      <!-- Attachments + pending invite badge -->
      <div
        v-if="attachments.length || pendingInvite"
        class="flex shrink-0 flex-wrap gap-1.5 border-t border-[var(--color-border)] px-4 py-2"
      >
        <div
          v-if="pendingInvite"
          class="flex items-center gap-1 rounded-full border border-teal bg-[var(--color-teal-light)] px-2 py-0.5 text-[12px] text-teal"
        >
          <span>📅</span>
          <span class="max-w-[180px] overflow-hidden text-ellipsis whitespace-nowrap">{{ pendingInvite.title }}.ics</span>
          <button
            @click="pendingInvite = null"
            class="ml-0.5 transition-colors hover:text-red-600"
            title="Remove invite"
          >×</button>
        </div>
        <div
          v-for="(att, i) in attachments"
          :key="i"
          class="flex items-center gap-1 rounded-full border border-[var(--color-border)] bg-[var(--color-bg)] px-2 py-0.5 text-[12px] text-[var(--color-text)]"
        >
          <span>📎</span>
          <span class="max-w-[180px] overflow-hidden text-ellipsis whitespace-nowrap">{{ att.filename }}</span>
          <button
            @click="removeAttachment(i)"
            class="ml-0.5 text-[var(--color-text-muted)] transition-colors hover:text-red-600"
            title="Remove"
          >×</button>
        </div>
      </div>

      <!-- Event picker panel -->
      <div v-if="showEventPicker" class="shrink-0 border-t border-[var(--color-border)] bg-[var(--color-bg)] max-h-[180px] overflow-y-auto">
        <div v-if="pickerLoading" class="px-4 py-3 text-xs text-[var(--color-text-muted)]">Loading events…</div>
        <div v-else-if="!pickerEvents.length" class="px-4 py-3 text-xs text-[var(--color-text-muted)]">No upcoming events in the next 60 days.</div>
        <button
          v-for="ev in pickerEvents"
          :key="ev.id"
          @click="attachEvent(ev)"
          class="w-full text-left px-4 py-2 text-[13px] hover:bg-[var(--color-teal-light)] flex items-center gap-2 border-none bg-transparent cursor-pointer"
        >
          <span class="font-medium text-[var(--color-text)] truncate flex-1">{{ ev.title }}</span>
          <span class="shrink-0 text-xs text-[var(--color-text-muted)]">{{ formatEventDate(ev.starts_at) }}</span>
        </button>
      </div>

      <!-- Footer -->
      <div class="flex shrink-0 items-center gap-3 border-t border-[var(--color-border)] px-4 py-2.5">
        <button
          @click="send"
          :disabled="sending || savingDraft"
          class="rounded-md bg-teal px-5 py-2 text-[13px] font-medium text-white transition-colors hover:bg-teal/90 disabled:cursor-not-allowed disabled:opacity-60"
        >{{ sending ? 'Sending…' : 'Send' }}</button>

        <button
          @click="saveDraftManual"
          :disabled="sending || savingDraft"
          class="rounded-md border border-[var(--color-border)] px-4 py-2 text-[13px] font-medium text-[var(--color-text-muted)] transition-colors hover:text-[var(--color-text)] disabled:cursor-not-allowed disabled:opacity-60"
        >{{ savingDraft ? 'Saving…' : 'Save Draft' }}</button>

        <span v-if="draftSaved && !savingDraft" class="text-[12px] text-[var(--color-text-muted)]">Draft saved</span>

        <button
          @click="fileInputEl.click()"
          title="Attach file"
          class="text-[18px] leading-none text-[var(--color-text-muted)] transition-colors hover:text-[var(--color-text)]"
        >📎</button>
        <input ref="fileInputEl" type="file" multiple class="hidden" @change="onFileInput" />

        <button
          @click="toggleEventPicker"
          title="Attach calendar invite"
          :class="['text-[18px] leading-none transition-colors', showEventPicker ? 'text-teal' : 'text-[var(--color-text-muted)] hover:text-[var(--color-text)]']"
        >📅</button>

        <div v-if="templatesStore.templates.length" class="relative">
          <button
            @click="showTemplatePicker = !showTemplatePicker"
            title="Insert template"
            :class="['text-[12px] transition-colors px-2 py-1 rounded border', showTemplatePicker ? 'border-teal text-teal bg-[var(--color-teal-light)]' : 'border-[var(--color-border)] text-[var(--color-text-muted)] hover:text-[var(--color-text)]']"
          >📋 Templates</button>
          <div v-if="showTemplatePicker"
            class="absolute bottom-full mb-1 left-0 z-50 min-w-[180px] bg-[var(--color-surface)] border border-[var(--color-border)] rounded-md shadow-lg py-1">
            <button
              v-for="t in templatesStore.templates" :key="t.id"
              @click="insertTemplate(t)"
              class="w-full text-left px-3 py-1.5 text-[13px] text-[var(--color-text)] hover:bg-[var(--color-bg)] border-none bg-transparent cursor-pointer truncate"
            >{{ t.name }}</button>
          </div>
        </div>

        <button
          v-if="pgp.isUnlocked"
          @click="pgpSign = !pgpSign"
          :title="pgpSign ? 'Signing enabled — click to disable' : 'Sign this message'"
          :class="['text-[12px] transition-colors px-2 py-1 rounded border', pgpSign ? 'border-teal text-teal bg-[var(--color-teal-light)]' : 'border-[var(--color-border)] text-[var(--color-text-muted)] hover:text-[var(--color-text)]']"
        >✍ Sign</button>

        <button
          v-if="pgp.isUnlocked"
          @click="pgpEncrypt = !pgpEncrypt"
          :disabled="!pgpEncryptable && !pgpEncrypt"
          :title="pgpEncryptable ? (pgpEncrypt ? 'Encryption enabled — click to disable' : 'Encrypt this message') : 'No public key found for all recipients (checked contacts and WKD)'"
          :class="['text-[12px] transition-colors px-2 py-1 rounded border', pgpEncrypt ? 'border-teal text-teal bg-[var(--color-teal-light)]' : 'border-[var(--color-border)] text-[var(--color-text-muted)] hover:text-[var(--color-text)] disabled:opacity-40 disabled:cursor-not-allowed']"
        >🔐 Encrypt</button>

        <button
          @click="requestReadReceipt = !requestReadReceipt"
          :title="requestReadReceipt ? 'Read receipt requested — click to cancel' : 'Request read receipt'"
          :class="['text-[12px] transition-colors px-2 py-1 rounded border', requestReadReceipt ? 'border-teal text-teal bg-[var(--color-teal-light)]' : 'border-[var(--color-border)] text-[var(--color-text-muted)] hover:text-[var(--color-text)]']"
        >✉ Receipt</button>

        <button
          @click="togglePlainText"
          :title="plainTextMode ? 'Switch to rich text' : 'Switch to plain text'"
          class="ml-auto text-[12px] text-[var(--color-text-muted)] transition-colors hover:text-[var(--color-text)]"
        >{{ plainTextMode ? 'Rich text' : 'Plain text' }}</button>

        <p v-if="pgpError" class="text-[12px] text-orange-600">{{ pgpError }}</p>
        <p v-if="error" class="text-[12px] text-red-600">{{ error }}</p>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, reactive, nextTick, computed, watch, onMounted, onUnmounted, onBeforeUnmount } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Underline from '@tiptap/extension-underline'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import { useMailStore } from '../stores/mail'
import { useSettingsStore } from '../stores/settings'
import { useCalendarStore } from '../stores/calendar'
import { usePGPStore } from '../stores/pgp'
import { useTemplatesStore } from '../stores/templates'
import { apiFetch } from '../api'
import { useUndoSend } from '../composables/useUndoSend'
import AddressInput from './AddressInput.vue'

const mail = useMailStore()
const settings = useSettingsStore()
const calendarStore = useCalendarStore()
const pgp = usePGPStore()
const templatesStore = useTemplatesStore()
const undoSend = useUndoSend()

const visible = ref(false)
const sending = ref(false)
const savingDraft = ref(false)
const draftSaved = ref(false)
const error = ref('')
const textareaEl = ref(null)
const fileInputEl = ref(null)
const plainTextMode = ref(false)

const form = reactive({ fromIndex: 0, to: '', cc: '', subject: '', plainBody: '' })
const attachments = ref([])
const originalDraft = ref(null)
const inReplyTo = ref('')
const references = ref('')
// pendingInvite holds { id, title } when a calendar event is queued as an invite.
// The actual .ics is generated at send time using the final To/CC addresses.
const pendingInvite = ref(null)

// PGP toggles
const pgpSign = ref(false)
const pgpEncrypt = ref(false)

const requestReadReceipt = ref(false)
const pgpEncryptable = ref(false)   // true when all recipients have stored public keys
const pgpError = ref('')

const showEventPicker = ref(false)
const pickerEvents = ref([])
const pickerLoading = ref(false)

// --- Tiptap editor ---

const editor = useEditor({
  extensions: [
    StarterKit,
    Link.configure({ openOnClick: false, autolink: true }),
    Underline,
    Placeholder.configure({ placeholder: 'Write your message…' }),
    Image.configure({ inline: true, allowBase64: true }),
  ],
  content: '',
  onUpdate: () => scheduleAutoSave(),
  editorProps: {
    handlePaste(_, event) {
      const items = event.clipboardData?.items
      if (!items) return false
      for (const item of Array.from(items)) {
        if (item.type.startsWith('image/')) {
          const file = item.getAsFile()
          if (!file) continue
          const reader = new FileReader()
          reader.onload = e => editor.value?.chain().focus().setImage({ src: e.target.result }).run()
          reader.readAsDataURL(file)
          return true
        }
      }
      return false
    },
    handleDrop(_, event) {
      const files = event.dataTransfer?.files
      if (!files?.length) return false
      let handled = false
      for (const file of Array.from(files)) {
        if (file.type.startsWith('image/')) {
          const reader = new FileReader()
          reader.onload = e => editor.value?.chain().focus().setImage({ src: e.target.result }).run()
          reader.readAsDataURL(file)
          handled = true
        }
      }
      return handled
    },
  },
})

onBeforeUnmount(() => editor.value?.destroy())

function setLink() {
  const prev = editor.value?.getAttributes('link').href ?? ''
  const url = window.prompt('URL', prev)
  if (url === null) return
  if (url === '') {
    editor.value?.chain().focus().unsetLink().run()
  } else {
    editor.value?.chain().focus().setLink({ href: url }).run()
  }
}

function togglePlainText() {
  if (!plainTextMode.value) {
    // Rich → plain: extract text from editor
    form.plainBody = htmlToPlain(editor.value?.getHTML() ?? '')
    plainTextMode.value = true
  } else {
    // Plain → rich: set editor content from textarea
    editor.value?.commands.setContent(plainToHtml(form.plainBody))
    plainTextMode.value = false
    nextTick(() => editor.value?.commands.focus())
  }
}

// --- Signature helpers ---

function sigHtml(sig) {
  if (!sig) return ''
  const lines = sig.split('\n').map(l => `<p>${escHtml(l) || '<br>'}</p>`).join('')
  return `<p>-- </p>${lines}`
}

function escHtml(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// Convert plain text to basic HTML paragraphs.
function plainToHtml(text) {
  if (!text) return ''
  return text.split('\n').map(l => `<p>${escHtml(l) || '<br>'}</p>`).join('')
}

// Strip HTML tags to plain text.
function htmlToPlain(html) {
  const el = document.createElement('div')
  el.innerHTML = html
  return el.innerText
}

// --- Auto-save ---

let autoSaveTimer = null

function scheduleAutoSave() {
  if (!visible.value) return
  clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(() => { if (visible.value) saveDraft() }, 30_000)
}

watch(() => [form.to, form.cc, form.subject, form.plainBody], scheduleAutoSave)

// Resolve a public key for a single recipient: stored contact key first, WKD fallback.
async function resolveRecipientKey(email) {
  const stored = await pgp.getKeyForEmail(email)
  if (stored) return stored
  return pgp.wkdLookup(email)
}

// Check whether all recipients have a resolvable public key (enables Encrypt toggle).
watch(() => [form.to, form.cc], async () => {
  if (!pgp.isUnlocked) return
  const recipients = [
    ...form.to.split(',').map(s => s.trim()).filter(Boolean),
    ...form.cc.split(',').map(s => s.trim()).filter(Boolean),
  ]
  if (!recipients.length) { pgpEncryptable.value = false; return }
  const results = await Promise.all(recipients.map(r => resolveRecipientKey(r)))
  pgpEncryptable.value = results.every(k => k !== null)
})

// --- Draft save / send helpers ---

function buildPayload() {
  const selectedFrom = fromOptions.value[form.fromIndex] ?? fromOptions.value[0]
  const base = {
    from_name: selectedFrom?.name ?? '',
    from_email: selectedFrom?.email ?? '',
    to: form.to.split(',').map(s => s.trim()).filter(Boolean),
    cc: form.cc.split(',').map(s => s.trim()).filter(Boolean),
    subject: form.subject,
    attachments: attachments.value.length ? attachments.value : undefined,
    disposition_notification_to: requestReadReceipt.value ? (selectedFrom?.email ?? '') : undefined,
  }
  if (plainTextMode.value) {
    return { ...base, text: form.plainBody }
  }

  // Extract inline images (data: URIs) from the editor HTML before sending.
  // Replace each with a cid: reference and pass the binary data separately so
  // the backend can build a proper multipart/related MIME structure.
  const inlineImages = []
  const html = (editor.value?.getHTML() ?? '').replace(
    /src="data:([^;]+);base64,([^"]+)"/g,
    (_, contentType, b64) => {
      const n = inlineImages.length + 1
      const cid = `img${String(n).padStart(3, '0')}@letrvu`
      inlineImages.push({ content_id: cid, content_type: contentType, data: b64 })
      return `src="cid:${cid}"`
    }
  )

  return {
    ...base,
    html,
    inline_images: inlineImages.length ? inlineImages : undefined,
  }
}

async function saveDraft() {
  savingDraft.value = true
  draftSaved.value = false
  try {
    await mail.saveDraft(buildPayload())
    if (originalDraft.value) {
      await mail.deleteMessage(originalDraft.value.folder, originalDraft.value.uid).catch(() => {})
      originalDraft.value = null
    }
    draftSaved.value = true
  } catch {
    // Silent — draft save failure is non-critical
  } finally {
    savingDraft.value = false
  }
}

async function saveDraftManual() {
  await saveDraft()
}

const fromOptions = computed(() => settings.fromOptions)

// Toolbar items — buttons and separators in one flat list, rendered with v-if in the template.
const showTemplatePicker = ref(false)

function insertTemplate(t) {
  showTemplatePicker.value = false
  // Fill subject only when the field is currently empty.
  if (t.subject && !form.subject) form.subject = t.subject
  // Prepend the template body above the current content (signature + quoted text).
  const current = editor.value?.getHTML() ?? ''
  const html = `<p>${t.body.replace(/\n/g, '</p><p>')}</p>`
  editor.value?.commands.setContent(html + current)
  editor.value?.commands.setTextSelection(0)
}

const toolbarItems = computed(() => [
  { label: '<b>B</b>',  title: 'Bold',           active: () => editor.value?.isActive('bold'),        action: () => editor.value?.chain().focus().toggleBold().run() },
  { label: '<i>I</i>',  title: 'Italic',          active: () => editor.value?.isActive('italic'),      action: () => editor.value?.chain().focus().toggleItalic().run() },
  { label: '<u>U</u>',  title: 'Underline',       active: () => editor.value?.isActive('underline'),   action: () => editor.value?.chain().focus().toggleUnderline().run() },
  { label: '<s>S</s>',  title: 'Strikethrough',   active: () => editor.value?.isActive('strike'),      action: () => editor.value?.chain().focus().toggleStrike().run() },
  { sep: 'a' },
  { label: '🔗',        title: 'Link',             active: () => editor.value?.isActive('link'),        action: () => setLink() },
  { sep: 'b' },
  { label: '≡',         title: 'Bullet list',      active: () => editor.value?.isActive('bulletList'),  action: () => editor.value?.chain().focus().toggleBulletList().run() },
  { label: '1.',        title: 'Ordered list',     active: () => editor.value?.isActive('orderedList'), action: () => editor.value?.chain().focus().toggleOrderedList().run() },
  { label: '❝',         title: 'Blockquote',       active: () => editor.value?.isActive('blockquote'),  action: () => editor.value?.chain().focus().toggleBlockquote().run() },
  { sep: 'c' },
  { label: '—',         title: 'Horizontal rule',  active: null,                                        action: () => editor.value?.chain().focus().setHorizontalRule().run() },
])

// --- open() ---

async function open(prefill = {}) {
  if (!settings.loaded) await settings.fetchSettings()
  if (!templatesStore.loaded) templatesStore.fetchTemplates()

  const sig = (settings.settings.signature ?? '').replace(/^--\s*\n/, '').trim()

  let fromIndex = 0
  if (prefill._fromEmail) {
    const lc = prefill._fromEmail.toLowerCase()
    const match = fromOptions.value.findIndex(opt => opt.email.toLowerCase() === lc)
    if (match !== -1) fromIndex = match
  } else if (prefill._originalRecipients?.length) {
    const lc = prefill._originalRecipients.map(r => r.toLowerCase())
    const match = fromOptions.value.findIndex(opt =>
      lc.some(r => r.includes(opt.email.toLowerCase()))
    )
    if (match !== -1) fromIndex = match
  }

  const { _originalRecipients: _r, _attachments: _a, _fromEmail: _fe, _noSignature: _ns,
          _draftFolder: _df, _draftUid: _du, _inReplyTo: _irt, _references: _refs,
          _inviteEvent: _inv,
          html: prefillHtml, body: prefillBody, ...rest } = prefill

  originalDraft.value = (_df && _du != null) ? { folder: _df, uid: _du } : null
  inReplyTo.value = _irt || ''
  references.value = _refs || ''

  Object.assign(form, { fromIndex, to: '', cc: '', subject: '', plainBody: '', ...rest, fromIndex })

  attachments.value = _a ? [..._a] : []
  pendingInvite.value = _inv ?? null
  requestReadReceipt.value = false

  // Build the HTML content for the editor.
  // Layout: [cursor] [sig block] [quoted content if reply/forward]
  const sigBlock = (!_ns && sig) ? sigHtml(sig) : ''
  let contentHtml = ''

  if (prefillHtml) {
    // Reply / forward: quoted HTML passed directly
    contentHtml = `<p></p>${sigBlock}${prefillHtml}`
  } else if (prefillBody) {
    // Plain text prefill (e.g. draft with text_body only)
    contentHtml = `<p></p>${sigBlock}${plainToHtml(prefillBody)}`
  } else {
    // Fresh compose
    contentHtml = `<p></p>${sigBlock}`
  }

  visible.value = true
  plainTextMode.value = false

  await nextTick()
  editor.value?.commands.setContent(contentHtml)
  // Place cursor at the very start (above the signature).
  editor.value?.commands.focus()
  editor.value?.commands.setTextSelection(0)
}

// --- close() ---

function close() {
  clearTimeout(autoSaveTimer)
  visible.value = false
  error.value = ''
  pgpError.value = ''
  draftSaved.value = false
  attachments.value = []
  pendingInvite.value = null
  showEventPicker.value = false
  pickerEvents.value = []
  originalDraft.value = null
  inReplyTo.value = ''
  references.value = ''
  pgpSign.value = false
  pgpEncrypt.value = false
  pgpEncryptable.value = false
  editor.value?.commands.setContent('')
}

function removeAttachment(i) {
  attachments.value.splice(i, 1)
}

function onFileInput(e) {
  const files = Array.from(e.target.files ?? [])
  for (const file of files) {
    const reader = new FileReader()
    reader.onload = ev => {
      const base64 = ev.target.result.split(',')[1]
      attachments.value.push({ filename: file.name, content_type: file.type || 'application/octet-stream', data: base64 })
    }
    reader.readAsDataURL(file)
  }
  e.target.value = '' // reset so the same file can be picked again
}


// --- event picker ---

async function toggleEventPicker() {
  showEventPicker.value = !showEventPicker.value
  if (showEventPicker.value && !pickerEvents.value.length) {
    pickerLoading.value = true
    try {
      pickerEvents.value = await calendarStore.fetchUpcoming(60)
    } finally {
      pickerLoading.value = false
    }
  }
}

function attachEvent(ev) {
  pendingInvite.value = { id: ev.id, title: ev.title }
  showEventPicker.value = false
}

function formatEventDate(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleString('default', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// --- invite helpers ---

// Fetches a METHOD:REQUEST .ics for the pending invite and returns it as an
// attachment object, or null if the fetch fails. Attendees are the final To+CC
// addresses so they appear in the ATTENDEE fields of the iCal.
async function buildInviteAttachment() {
  if (!pendingInvite.value) return null
  const payload = buildPayload()
  const attendees = [
    ...payload.to ?? [],
    ...payload.cc ?? [],
  ]
  try {
    const res = await apiFetch(`/api/calendar/events/${pendingInvite.value.id}/ical`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ attendees }),
    })
    if (!res.ok) return null
    const icsText = await res.text()
    const b64 = btoa(unescape(encodeURIComponent(icsText)))
    const filename = (pendingInvite.value.title || 'invite').replace(/[/\\:*?"<>|]/g, '_') + '.ics'
    return { filename, content_type: 'text/calendar; method=REQUEST', data: b64 }
  } catch {
    return null
  }
}

// --- send() ---

async function send() {
  sending.value = true
  error.value = ''
  pgpError.value = ''
  try {
    const inviteAtt = await buildInviteAttachment()
    const payload = buildPayload()
    if (inviteAtt) {
      payload.attachments = [...(payload.attachments ?? []), inviteAtt]
    }

    // PGP: sign and/or encrypt the message body before sending.
    if (pgpSign.value || pgpEncrypt.value) {
      // Both operations require plain text body.
      const bodyText = plainTextMode.value
        ? form.plainBody
        : htmlToPlain(editor.value?.getHTML() ?? '')

      if (pgpEncrypt.value) {
        // Inline PGP encryption (replaces the body with an encrypted block).
        const allRecipients = [...(payload.to ?? []), ...(payload.cc ?? [])]
        const keys = await Promise.all(allRecipients.map(r => resolveRecipientKey(r)))
        if (keys.some(k => !k)) {
          pgpError.value = 'Missing public key for one or more recipients. Encryption aborted.'
          return
        }
        const armoredMsg = await pgp.encryptText(bodyText, keys, pgpSign.value)
        payload.text = armoredMsg
        delete payload.html
      } else if (pgpSign.value) {
        // PGP/MIME detached signature (RFC 3156 multipart/signed).
        const { text, signature, micalg } = await pgp.signMIME(bodyText)
        payload.text = text
        delete payload.html
        payload.pgp_mime_sig = { signature, micalg }
      }
    }

    const finalPayload = {
      ...payload,
      in_reply_to: inReplyTo.value || undefined,
      references: references.value || undefined,
    }
    const draft = originalDraft.value

    const delay = settings.undoSendDelay
    if (delay > 0) {
      // Capture enough state to reopen compose if the user hits Undo.
      // We save the raw editor HTML (not the processed payload HTML which has
      // cid: references instead of data: URIs) so images are editable again.
      const undoState = {
        to: form.to,
        cc: form.cc,
        subject: form.subject,
        html: editor.value?.getHTML() ?? '',
        _fromEmail: fromOptions.value[form.fromIndex]?.email,
        _inReplyTo: inReplyTo.value || undefined,
        _references: references.value || undefined,
        _attachments: attachments.value.length ? [...attachments.value] : undefined,
        _noSignature: true, // signature already in the body — don't add another
        _draftFolder: draft?.folder,
        _draftUid: draft?.uid,
      }
      close()
      sending.value = false
      try {
        await undoSend.schedule(delay)
      } catch (e) {
        if (e.message === 'undo') {
          open(undoState)
        }
        return
      }
      // Timer expired — send for real (outside sending spinner, best-effort)
      try {
        await mail.sendMessage(finalPayload)
        if (draft) await mail.deleteMessage(draft.folder, draft.uid).catch(() => {})
      } catch {
        // Reopen compose so the user doesn't lose their message
        open(undoState)
      }
      return
    }

    await mail.sendMessage(finalPayload)
    if (draft) {
      await mail.deleteMessage(draft.folder, draft.uid).catch(() => {})
      originalDraft.value = null
    }
    close()
  } catch (e) {
    if (pgpError.value) return  // already set above
    error.value = 'Failed to send. Please try again.'
  } finally {
    sending.value = false
  }
}

function onKeydown(e) { if (e.key === 'Escape' && visible.value) close() }
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

defineExpose({ open, close, visible })
</script>

<!-- ProseMirror styles — must be global (not scoped) to reach inside Tiptap's rendered HTML -->
<style>
.ProseMirror {
  padding: 12px 16px;
  outline: none;
  min-height: 200px;
  color: var(--color-text);
  background: var(--color-surface);
  font-size: 14px;
  line-height: 1.6;
}
.ProseMirror p { margin: 0 0 0.3em; }
.ProseMirror blockquote {
  border-left: 3px solid var(--color-border);
  margin: 0.5em 0;
  padding-left: 1em;
  color: var(--color-text-muted);
}
.ProseMirror ul, .ProseMirror ol { padding-left: 1.5em; margin: 0.3em 0; }
.ProseMirror hr { border: none; border-top: 0.5px solid var(--color-border); margin: 0.8em 0; }
.ProseMirror a { color: var(--color-teal); }
.ProseMirror img { max-width: 100%; height: auto; display: inline-block; }
.ProseMirror p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  color: var(--color-text-muted);
  pointer-events: none;
  float: left;
  height: 0;
}
</style>
