<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-[100] flex items-end justify-end p-8 pointer-events-none"
  >
    <div class="pointer-events-auto w-[560px] max-h-[620px] flex flex-col rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-2xl" @paste="onPasteImage">

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

      <!-- Attachments -->
      <div
        v-if="attachments.length"
        class="flex shrink-0 flex-wrap gap-1.5 border-t border-[var(--color-border)] px-4 py-2"
      >
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
          @click="togglePlainText"
          :title="plainTextMode ? 'Switch to rich text' : 'Switch to plain text'"
          class="ml-auto text-[12px] text-[var(--color-text-muted)] transition-colors hover:text-[var(--color-text)]"
        >{{ plainTextMode ? 'Rich text' : 'Plain text' }}</button>

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
import { useMailStore } from '../stores/mail'
import { useSettingsStore } from '../stores/settings'
import AddressInput from './AddressInput.vue'

const mail = useMailStore()
const settings = useSettingsStore()

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

// --- Tiptap editor ---

const editor = useEditor({
  extensions: [
    StarterKit,
    Link.configure({ openOnClick: false, autolink: true }),
    Underline,
    Placeholder.configure({ placeholder: 'Write your message…' }),
  ],
  content: '',
  onUpdate: () => scheduleAutoSave(),
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
  }
  if (plainTextMode.value) {
    return { ...base, text: form.plainBody }
  }
  return { ...base, html: editor.value?.getHTML() ?? '' }
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
          html: prefillHtml, body: prefillBody, ...rest } = prefill

  originalDraft.value = (_df && _du != null) ? { folder: _df, uid: _du } : null
  inReplyTo.value = _irt || ''
  references.value = _refs || ''

  Object.assign(form, { fromIndex, to: '', cc: '', subject: '', plainBody: '', ...rest, fromIndex })

  attachments.value = _a ? [..._a] : []

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
  draftSaved.value = false
  attachments.value = []
  originalDraft.value = null
  inReplyTo.value = ''
  references.value = ''
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

function onPasteImage(e) {
  const items = e.clipboardData?.items
  if (!items) return
  for (const item of items) {
    if (!item.type.startsWith('image/')) continue
    const file = item.getAsFile()
    if (!file) continue
    e.preventDefault()
    const ext = item.type.split('/')[1]?.replace('jpeg', 'jpg') ?? 'png'
    const filename = `image-${Date.now()}.${ext}`
    const reader = new FileReader()
    reader.onload = ev => {
      const base64 = ev.target.result.split(',')[1]
      attachments.value.push({ filename, content_type: item.type, data: base64 })
    }
    reader.readAsDataURL(file)
    break // one image per paste
  }
}

// --- send() ---

async function send() {
  sending.value = true
  error.value = ''
  try {
    await mail.sendMessage({
      ...buildPayload(),
      in_reply_to: inReplyTo.value || undefined,
      references: references.value || undefined,
    })
    if (originalDraft.value) {
      await mail.deleteMessage(originalDraft.value.folder, originalDraft.value.uid).catch(() => {})
      originalDraft.value = null
    }
    close()
  } catch {
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
.ProseMirror p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  color: var(--color-text-muted);
  pointer-events: none;
  float: left;
  height: 0;
}
</style>
