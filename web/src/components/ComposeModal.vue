<template>
  <div v-if="visible" class="overlay" @click.self="close">
    <div class="compose">
      <div class="compose-header">
        <span>New message</span>
        <button @click="close" class="close">×</button>
      </div>
      <div class="fields">
        <div class="from-row">
          <span class="from-label">From</span>
          <select v-model="form.fromIndex" class="from-select">
            <option v-for="(opt, i) in fromOptions" :key="i" :value="i">{{ opt.label }}</option>
          </select>
        </div>
        <AddressInput v-model="form.to" placeholder="To" />
        <AddressInput v-model="form.cc" placeholder="CC" />
        <input v-model="form.subject" type="text" placeholder="Subject" class="subject-input" />
      </div>

      <!-- Rich text toolbar -->
      <div v-if="!plainTextMode" class="toolbar">
        <button class="tb" :class="{ active: editor?.isActive('bold') }"
          @mousedown.prevent="editor?.chain().focus().toggleBold().run()" title="Bold"><b>B</b></button>
        <button class="tb" :class="{ active: editor?.isActive('italic') }"
          @mousedown.prevent="editor?.chain().focus().toggleItalic().run()" title="Italic"><i>I</i></button>
        <button class="tb" :class="{ active: editor?.isActive('underline') }"
          @mousedown.prevent="editor?.chain().focus().toggleUnderline().run()" title="Underline"><u>U</u></button>
        <button class="tb" :class="{ active: editor?.isActive('strike') }"
          @mousedown.prevent="editor?.chain().focus().toggleStrike().run()" title="Strikethrough"><s>S</s></button>
        <span class="tb-sep" />
        <button class="tb" :class="{ active: editor?.isActive('link') }"
          @mousedown.prevent="setLink()" title="Link">🔗</button>
        <span class="tb-sep" />
        <button class="tb" :class="{ active: editor?.isActive('bulletList') }"
          @mousedown.prevent="editor?.chain().focus().toggleBulletList().run()" title="Bullet list">≡</button>
        <button class="tb" :class="{ active: editor?.isActive('orderedList') }"
          @mousedown.prevent="editor?.chain().focus().toggleOrderedList().run()" title="Ordered list">1.</button>
        <button class="tb" :class="{ active: editor?.isActive('blockquote') }"
          @mousedown.prevent="editor?.chain().focus().toggleBlockquote().run()" title="Blockquote">❝</button>
        <span class="tb-sep" />
        <button class="tb"
          @mousedown.prevent="editor?.chain().focus().setHorizontalRule().run()" title="Horizontal rule">—</button>
      </div>

      <!-- Rich editor -->
      <EditorContent v-if="!plainTextMode" :editor="editor" class="editor" />

      <!-- Plain text fallback -->
      <textarea v-else ref="textareaEl" v-model="form.plainBody" placeholder="Write your message…" class="plain-textarea" />

      <div v-if="attachments.length" class="attachment-list">
        <div v-for="(att, i) in attachments" :key="i" class="attachment-chip">
          <span class="att-icon">📎</span>
          <span class="att-name">{{ att.filename }}</span>
          <button @click="removeAttachment(i)" class="att-remove" title="Remove">×</button>
        </div>
      </div>
      <div class="compose-footer">
        <button @click="send" :disabled="sending || savingDraft" class="send-btn">
          {{ sending ? 'Sending…' : 'Send' }}
        </button>
        <button @click="saveDraftManual" :disabled="sending || savingDraft" class="draft-btn">
          {{ savingDraft ? 'Saving…' : 'Save Draft' }}
        </button>
        <span v-if="draftSaved && !savingDraft" class="draft-status">Draft saved</span>
        <button class="plain-toggle" @click="togglePlainText" :title="plainTextMode ? 'Switch to rich text' : 'Switch to plain text'">
          {{ plainTextMode ? 'Rich text' : 'Plain text' }}
        </button>
        <p v-if="error" class="error">{{ error }}</p>
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

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.2);
  display: flex;
  align-items: flex-end;
  justify-content: flex-end;
  padding: 2rem;
  z-index: 100;
}
.compose {
  width: 560px;
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  max-height: 620px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.15);
}
.compose-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  font-weight: 500;
  flex-shrink: 0;
}
.close { background: none; border: none; font-size: 18px; cursor: pointer; color: var(--color-text-muted); }
.from-row {
  display: flex;
  align-items: center;
  border-bottom: 0.5px solid var(--color-border);
}
.from-label {
  padding: 8px 16px;
  font-size: 13px;
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.from-select {
  flex: 1;
  padding: 8px 8px 8px 0;
  border: none;
  font-size: 13px;
  font-family: inherit;
  background: transparent;
  color: var(--color-text);
  outline: none;
  cursor: pointer;
}
.fields input, .subject-input {
  display: block;
  width: 100%;
  padding: 8px 16px;
  border: none;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
  background: var(--color-surface);
  color: var(--color-text);
}

/* Toolbar */
.toolbar {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 5px 10px;
  border-bottom: 0.5px solid var(--color-border);
  flex-shrink: 0;
  flex-wrap: wrap;
}
.tb {
  padding: 3px 7px;
  border: none;
  border-radius: 4px;
  background: transparent;
  font-size: 13px;
  cursor: pointer;
  color: var(--color-text);
  line-height: 1.4;
  min-width: 26px;
}
.tb:hover { background: var(--color-bg); }
.tb.active { background: var(--color-teal-light); color: var(--color-teal); }
.tb-sep { width: 0.5px; height: 16px; background: var(--color-border); margin: 0 4px; flex-shrink: 0; }

/* Rich editor area */
.editor {
  flex: 1;
  overflow-y: auto;
  min-height: 200px;
  font-size: 14px;
  font-family: inherit;
  line-height: 1.6;
}

.plain-textarea {
  flex: 1;
  padding: 12px 16px;
  border: none;
  resize: none;
  font-size: 14px;
  font-family: inherit;
  line-height: 1.6;
  outline: none;
  min-height: 200px;
  background: var(--color-surface);
  color: var(--color-text);
}

.attachment-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px 16px;
  border-top: 0.5px solid var(--color-border);
  flex-shrink: 0;
}
.attachment-chip {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--color-bg);
  border: 0.5px solid var(--color-border);
  border-radius: 20px;
  padding: 3px 8px 3px 6px;
  font-size: 12px;
  color: var(--color-text);
}
.att-icon { font-size: 13px; }
.att-name { max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.att-remove {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  color: var(--color-text-muted);
  padding: 0 0 0 2px;
}
.att-remove:hover { color: #c0392b; }
.compose-footer {
  padding: 10px 16px;
  border-top: 0.5px solid var(--color-border);
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-shrink: 0;
}
.send-btn {
  padding: 8px 20px;
  background: var(--color-teal);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.send-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.draft-btn {
  padding: 8px 16px;
  background: transparent;
  color: var(--color-text-muted);
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.draft-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.draft-status { font-size: 12px; color: var(--color-text-muted); }
.plain-toggle {
  margin-left: auto;
  background: none;
  border: none;
  font-size: 12px;
  color: var(--color-text-muted);
  cursor: pointer;
  padding: 0;
}
.plain-toggle:hover { color: var(--color-text); }
.error { font-size: 12px; color: #c0392b; }
</style>

<!-- ProseMirror styles — not scoped so they reach inside the editor shadow DOM -->
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
