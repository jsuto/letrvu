<template>
  <div v-if="visible" class="overlay" @click.self="close">
    <div class="compose">
      <div class="compose-header">
        <span>New message</span>
        <button @click="close" class="close">×</button>
      </div>
      <div class="fields">
        <AddressInput v-model="form.to" placeholder="To" />
        <AddressInput v-model="form.cc" placeholder="CC" />
        <input v-model="form.subject" type="text" placeholder="Subject" class="subject-input" />
      </div>
      <textarea ref="textareaEl" v-model="form.body" placeholder="Write your message…" />
      <div class="compose-footer">
        <button @click="send" :disabled="sending" class="send-btn">
          {{ sending ? 'Sending…' : 'Send' }}
        </button>
        <p v-if="error" class="error">{{ error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, nextTick } from 'vue'
import { useMailStore } from '../stores/mail'
import { useSettingsStore } from '../stores/settings'
import AddressInput from './AddressInput.vue'

const mail = useMailStore()
const settings = useSettingsStore()

const visible = ref(false)
const sending = ref(false)
const error = ref('')
const textareaEl = ref(null)

const form = reactive({ to: '', cc: '', subject: '', body: '' })

async function open(prefill = {}) {
  if (!settings.loaded) await settings.fetchSettings()

  // Strip a leading "-- " or "--" line the user may have typed themselves,
  // since we always prepend the standard separator.
  const sig = (settings.settings.signature ?? '').replace(/^--\s*\n/, '').trim()
  const sigBlock = sig ? `\n\n-- \n${sig}` : ''

  // Signature goes between the user's typing area and any quoted text
  // (prefill.body carries forwarded content).
  Object.assign(form, {
    to: '',
    cc: '',
    subject: '',
    body: sigBlock + (prefill.body ?? ''),
    ...prefill,
    // body from prefill is already incorporated above; don't let the spread
    // overwrite our assembled value when body is the only prefilled key.
    body: sigBlock + (prefill.body ?? ''),
  })

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
  visible.value = false
  error.value = ''
}

async function send() {
  sending.value = true
  error.value = ''
  try {
    await mail.sendMessage({
      to: form.to.split(',').map(s => s.trim()).filter(Boolean),
      cc: form.cc.split(',').map(s => s.trim()).filter(Boolean),
      subject: form.subject,
      text: form.body,
    })
    close()
  } catch {
    error.value = 'Failed to send. Please try again.'
  } finally {
    sending.value = false
  }
}

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
  max-height: 520px;
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
.error { font-size: 12px; color: #c0392b; }
</style>
