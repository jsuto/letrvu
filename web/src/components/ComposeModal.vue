<template>
  <div v-if="visible" class="overlay" @click.self="close">
    <div class="compose">
      <div class="compose-header">
        <span>New message</span>
        <button @click="close" class="close">×</button>
      </div>
      <div class="fields">
        <input v-model="form.to" type="text" placeholder="To" />
        <input v-model="form.cc" type="text" placeholder="CC" />
        <input v-model="form.subject" type="text" placeholder="Subject" />
      </div>
      <textarea v-model="form.body" placeholder="Write your message…" />
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
import { ref, reactive } from 'vue'
import { useMailStore } from '../stores/mail'

const mail = useMailStore()
const visible = ref(false)
const sending = ref(false)
const error = ref('')

const form = reactive({ to: '', cc: '', subject: '', body: '' })

function open(prefill = {}) {
  Object.assign(form, { to: '', cc: '', subject: '', body: '', ...prefill })
  visible.value = true
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
  position: absolute;
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
.fields input {
  display: block;
  width: 100%;
  padding: 8px 16px;
  border: none;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  outline: none;
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
