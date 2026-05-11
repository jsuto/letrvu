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
      <textarea ref="textareaEl" v-model="form.body" placeholder="Write your message…" />
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
        <p v-if="error" class="error">{{ error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, nextTick, computed, watch, onMounted, onUnmounted } from 'vue'
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

const form = reactive({ fromIndex: 0, to: '', cc: '', subject: '', body: '' })
// Each entry: { filename, content_type, data } where data is base64.
const attachments = ref([])
// When resuming a draft, remember its location so we can delete it after send/save.
const originalDraft = ref(null) // { folder, uid } | null

// Auto-save draft after 30 s of inactivity while compose is open.
let autoSaveTimer = null

function scheduleAutoSave() {
  if (!visible.value) return
  clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(() => {
    if (visible.value) saveDraft()
  }, 30_000)
}

watch(() => [form.to, form.cc, form.subject, form.body], scheduleAutoSave)

async function saveDraft() {
  savingDraft.value = true
  draftSaved.value = false
  try {
    const selectedFrom = fromOptions.value[form.fromIndex] ?? fromOptions.value[0]
    await mail.saveDraft({
      from_name: selectedFrom?.name ?? '',
      from_email: selectedFrom?.email ?? '',
      to: form.to.split(',').map(s => s.trim()).filter(Boolean),
      cc: form.cc.split(',').map(s => s.trim()).filter(Boolean),
      subject: form.subject,
      text: form.body,
      attachments: attachments.value.length ? attachments.value : undefined,
    })
    // New draft saved — remove the original so there's no duplicate.
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

async function open(prefill = {}) {
  if (!settings.loaded) await settings.fetchSettings()

  // Strip a leading "-- " or "--" line the user may have typed themselves,
  // since we always prepend the standard separator.
  const sig = (settings.settings.signature ?? '').replace(/^--\s*\n/, '').trim()
  const sigBlock = sig ? `\n\n-- \n${sig}` : ''

  // When replying, pick the identity whose email matches one of the original
  // To/CC addresses so the user sends from the same address the mail arrived at.
  // When resuming a draft, _fromEmail picks the exact From identity used.
  let fromIndex = 0
  if (prefill._fromEmail) {
    const lc = prefill._fromEmail.toLowerCase()
    const match = fromOptions.value.findIndex(opt => opt.email.toLowerCase() === lc)
    if (match !== -1) fromIndex = match
  } else {
    const recipients = prefill._originalRecipients
    if (recipients?.length) {
      const lc = recipients.map(r => r.toLowerCase())
      const match = fromOptions.value.findIndex(opt =>
        lc.some(r => r.includes(opt.email.toLowerCase()))
      )
      if (match !== -1) fromIndex = match
    }
  }

  // When resuming a draft the body already contains the signature, so skip
  // prepending it again. For new/reply/forward, prepend the signature block.
  const effectiveSigBlock = prefill._noSignature ? '' : sigBlock

  // Signature goes between the user's typing area and any quoted text
  // (prefill.body carries forwarded content). Strip internal hint keys.
  const { _originalRecipients: _r, _attachments: _a, _fromEmail: _fe, _noSignature: _ns, _draftFolder: _df, _draftUid: _du, ...rest } = prefill
  originalDraft.value = (_df && _du != null) ? { folder: _df, uid: _du } : null
  Object.assign(form, {
    fromIndex,
    to: '',
    cc: '',
    subject: '',
    body: effectiveSigBlock + (rest.body ?? ''),
    ...rest,
    // body from prefill is already incorporated above; don't let the spread
    // overwrite our assembled value when body is the only prefilled key.
    body: effectiveSigBlock + (rest.body ?? ''),
    fromIndex, // restore after spread in case prefill had a fromIndex key
  })

  attachments.value = prefill._attachments ? [...prefill._attachments] : []

  visible.value = true

  // Place cursor at the very top so the user types above the signature.
  await nextTick()
  if (textareaEl.value) {
    textareaEl.value.focus()
    textareaEl.value.setSelectionRange(0, 0)
    textareaEl.value.scrollTop = 0
  }
}

function close() {
  clearTimeout(autoSaveTimer)
  visible.value = false
  error.value = ''
  draftSaved.value = false
  attachments.value = []
  originalDraft.value = null
}

function removeAttachment(i) {
  attachments.value.splice(i, 1)
}

async function send() {
  sending.value = true
  error.value = ''
  try {
    const selectedFrom = fromOptions.value[form.fromIndex] ?? fromOptions.value[0]
    await mail.sendMessage({
      from_name: selectedFrom?.name ?? '',
      from_email: selectedFrom?.email ?? '',
      to: form.to.split(',').map(s => s.trim()).filter(Boolean),
      cc: form.cc.split(',').map(s => s.trim()).filter(Boolean),
      subject: form.subject,
      text: form.body,
      attachments: attachments.value.length ? attachments.value : undefined,
    })
    // Message sent — remove the original draft so there's no stale copy.
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

defineExpose({ open, close })
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
}
.compose {
  width: 520px;
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  max-height: 560px;
}
.compose-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  font-weight: 500;
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
}
textarea {
  flex: 1;
  padding: 12px 16px;
  border: none;
  resize: none;
  font-size: 14px;
  font-family: inherit;
  line-height: 1.6;
  outline: none;
  min-height: 200px;
}
.attachment-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px 16px;
  border-top: 0.5px solid var(--color-border);
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
.error { font-size: 12px; color: #c0392b; }
</style>
