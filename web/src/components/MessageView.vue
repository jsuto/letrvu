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
          <button @click="remove" class="danger">Delete</button>
        </div>
      </div>
      <!-- HTML email rendered in a sandboxed iframe to prevent XSS -->
      <iframe
        v-if="mail.currentMessage.html_body"
        class="body-frame"
        sandbox="allow-same-origin"
        :srcdoc="mail.currentMessage.html_body"
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
import { inject } from 'vue'
import { useMailStore } from '../stores/mail'
import { useContactsStore } from '../stores/contacts'

const mail = useMailStore()
const contacts = useContactsStore()
const compose = inject('compose')

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
</style>
