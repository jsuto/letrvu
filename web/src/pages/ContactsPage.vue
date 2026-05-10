<template>
  <div class="contacts-layout">
    <!-- Sidebar -->
    <aside class="sidebar">
      <FolderList />
    </aside>

    <!-- Contact list panel -->
    <div class="contact-list-panel">
      <div class="panel-header">
        <span class="panel-title">Contacts</span>
        <div class="panel-actions">
          <label class="import-btn" title="Import vCard (.vcf)">
            Import
            <input type="file" accept=".vcf" @change="importVCard" hidden />
          </label>
          <a :href="exportUrl" download="contacts.vcf" class="export-btn">Export</a>
          <button class="new-btn" @click="contactModal?.open()">+ New</button>
        </div>
      </div>

      <div v-if="contacts.loading" class="empty-state">Loading…</div>
      <div v-else-if="contacts.contacts.length === 0" class="empty-state">No contacts yet.</div>
      <ul v-else class="contact-list">
        <li
          v-for="c in contacts.contacts"
          :key="c.id"
          :class="{ active: selected?.id === c.id }"
          @click="selected = c"
        >
          <div class="c-info">
            <span class="c-name">{{ c.name || c.emails?.[0]?.email || '—' }}</span>
            <span class="c-email">{{ c.emails?.[0]?.email }}</span>
          </div>
          <button class="c-delete" title="Delete" @click.stop="confirmDelete(c)">✕</button>
        </li>
      </ul>
    </div>

    <!-- Contact detail panel -->
    <div class="contact-detail-panel">
      <div v-if="!selected" class="empty-state">Select a contact</div>
      <div v-else class="contact-detail">
        <div class="detail-header">
          <h2>{{ selected.name || selected.emails?.[0]?.email || '—' }}</h2>
          <div class="detail-actions">
            <button @click="contactModal?.open(selected)">Edit</button>
            <button class="danger" @click="confirmDelete(selected)">Delete</button>
          </div>
        </div>
        <div v-if="selected.notes" class="detail-notes">{{ selected.notes }}</div>
        <ul class="email-list">
          <li v-for="e in selected.emails" :key="e.id">
            <span class="email-addr">{{ e.email }}</span>
            <span v-if="e.label" class="email-label">{{ e.label }}</span>
          </li>
        </ul>
      </div>
    </div>
  </div>

  <ContactModal ref="contactModal" @close="onModalClose" />
  <ComposeModal ref="composeModal" />
</template>

<script setup>
import { ref, onMounted, watch, provide } from 'vue'
import FolderList from '../components/FolderList.vue'
import ContactModal from '../components/ContactModal.vue'
import ComposeModal from '../components/ComposeModal.vue'
import { useContactsStore } from '../stores/contacts'

const contacts = useContactsStore()
const contactModal = ref(null)
const composeModal = ref(null)
const selected = ref(null)
const exportUrl = '/api/contacts/export'

provide('compose', composeModal)

onMounted(() => contacts.fetchContacts())

watch(() => contacts.contacts, list => {
  // Keep selected in sync after edit/delete
  if (selected.value) {
    selected.value = list.find(c => c.id === selected.value.id) ?? null
  }
})

function onModalClose() {
  // selected will be updated by the watcher above
}

async function confirmDelete(c) {
  if (!confirm(`Delete "${c.name || c.emails?.[0]?.email}"?`)) return
  await contacts.deleteContact(c.id)
  selected.value = null
}

async function importVCard(e) {
  const file = e.target.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  const res = await fetch('/api/contacts/import', { method: 'POST', body: fd })
  if (res.ok) {
    const { imported } = await res.json()
    alert(`Imported ${imported} contact(s).`)
    await contacts.fetchContacts()
  } else {
    alert('Import failed.')
  }
  e.target.value = ''
}
</script>

<style scoped>
.contacts-layout {
  display: grid;
  grid-template-columns: 200px 260px 1fr;
  height: 100vh;
  overflow: hidden;
  background: var(--color-bg);
}
.sidebar {
  border-right: 0.5px solid var(--color-border);
  overflow-y: auto;
}
.contact-list-panel {
  border-right: 0.5px solid var(--color-border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 0.5px solid var(--color-border);
  gap: 8px;
}
.panel-title { font-size: 13px; font-weight: 500; }
.panel-actions { display: flex; gap: 6px; align-items: center; }
.new-btn, .export-btn, .import-btn {
  padding: 5px 10px;
  font-size: 12px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  background: var(--color-surface);
  color: var(--color-text);
  text-decoration: none;
}
.new-btn { background: var(--color-teal); color: white; border-color: var(--color-teal); }
.contact-list {
  list-style: none;
  flex: 1;
  overflow-y: auto;
}
.contact-list li {
  display: flex;
  align-items: center;
  padding: 8px 14px;
  cursor: pointer;
  border-bottom: 0.5px solid var(--color-border);
}
.contact-list li:hover, .contact-list li.active { background: var(--color-teal-light); }
.c-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}
.c-name { font-size: 13px; font-weight: 500; }
.c-email { font-size: 11px; color: var(--color-text-muted); }
.c-delete {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-muted);
  font-size: 12px;
  padding: 4px 6px;
  border-radius: 4px;
  opacity: 0;
  flex-shrink: 0;
}
.contact-list li:hover .c-delete { opacity: 1; }
.c-delete:hover { background: #fde8e8; color: #c0392b; }
.contact-detail-panel { overflow-y: auto; padding: 2rem; }
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-muted);
  font-size: 13px;
}
.detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 1rem;
}
h2 { font-size: 20px; font-weight: 500; }
.detail-actions { display: flex; gap: 8px; }
.detail-actions button {
  padding: 6px 14px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  font-size: 13px;
  cursor: pointer;
}
.detail-actions button:hover { background: var(--color-bg); }
.detail-actions button.danger { color: #c0392b; border-color: #f5c6c6; }
.detail-notes {
  font-size: 13px;
  color: var(--color-text-muted);
  margin-bottom: 1rem;
  white-space: pre-wrap;
}
.email-list { list-style: none; }
.email-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
}
.email-label {
  font-size: 11px;
  background: var(--color-teal-light);
  border: 0.5px solid var(--color-teal);
  border-radius: 4px;
  padding: 1px 6px;
  color: var(--color-teal);
}
</style>
