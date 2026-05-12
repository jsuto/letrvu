<template>
  <div class="contacts-layout">
    <!-- Sidebar -->
    <aside class="sidebar">
      <FolderList />
    </aside>

    <!-- Contact/Group list panel -->
    <div class="contact-list-panel">
      <div class="panel-header">
        <div class="tab-row">
          <button :class="{ active: view === 'contacts' }" @click="view = 'contacts'">Contacts</button>
          <button :class="{ active: view === 'groups' }" @click="view = 'groups'">Groups</button>
        </div>
        <div class="panel-actions" v-if="view === 'contacts'">
          <label class="import-btn" title="Import vCard (.vcf)">
            Import
            <input type="file" accept=".vcf" @change="importVCard" hidden />
          </label>
          <a :href="exportUrl" download="contacts.vcf" class="export-btn">Export</a>
          <button class="new-btn" @click="contactModal?.open()">+ New</button>
        </div>
        <div class="panel-actions" v-else>
          <button class="new-btn" @click="openNewGroup">+ New</button>
        </div>
      </div>

      <!-- Contacts list -->
      <template v-if="view === 'contacts'">
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
      </template>

      <!-- Groups list -->
      <template v-else>
        <div v-if="contacts.loading" class="empty-state">Loading…</div>
        <div v-else-if="contacts.groups.length === 0" class="empty-state">No groups yet.</div>
        <ul v-else class="contact-list">
          <li
            v-for="g in contacts.groups"
            :key="g.id"
            :class="{ active: selectedGroup?.id === g.id }"
            @click="selectedGroup = g"
          >
            <div class="c-info">
              <span class="c-name">{{ g.name }}</span>
              <span class="c-email">{{ g.members?.length ?? 0 }} member{{ g.members?.length === 1 ? '' : 's' }}</span>
            </div>
            <button class="c-delete" title="Delete" @click.stop="confirmDeleteGroup(g)">✕</button>
          </li>
        </ul>
      </template>
    </div>

    <!-- Contact detail panel -->
    <div class="contact-detail-panel" v-if="view === 'contacts'">
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

    <!-- Group detail panel -->
    <div class="contact-detail-panel" v-else>
      <div v-if="!selectedGroup" class="empty-state">Select a group</div>
      <div v-else class="contact-detail">
        <div class="detail-header">
          <div class="group-name-row">
            <h2 v-if="!editingGroupName">{{ selectedGroup.name }}</h2>
            <input
              v-else
              ref="groupNameInput"
              v-model="groupNameDraft"
              class="group-name-input"
              @keydown.enter="saveGroupName"
              @keydown.escape="editingGroupName = false"
            />
          </div>
          <div class="detail-actions">
            <button v-if="!editingGroupName" @click="startEditGroupName">Rename</button>
            <button v-else @click="saveGroupName">Save</button>
            <button class="danger" @click="confirmDeleteGroup(selectedGroup)">Delete</button>
          </div>
        </div>

        <!-- Add member search -->
        <div class="add-member-row">
          <div class="add-member-wrap">
            <div class="add-member-input-row">
              <input
                v-model="memberSearch"
                placeholder="Search contacts or type an email address…"
                class="member-search-input"
                @input="onMemberSearch"
                @keydown.enter.prevent="addFirstSuggestion"
                @keydown.escape="memberSuggestions = []"
                autocomplete="off"
              />
              <button
                class="add-member-btn"
                :disabled="!memberSuggestions.length && !isRawEmail"
                @click="addFirstSuggestion"
              >Add</button>
            </div>
            <div v-if="memberSearch && !memberSuggestions.length && !isRawEmail" class="member-hint">
              No matching contacts. Type a full email address to add directly.
            </div>
            <ul v-if="memberSuggestions.length" class="member-suggestions">
              <li
                v-for="c in memberSuggestions"
                :key="c.id"
                @mousedown.prevent="addMember(c)"
              >
                <span class="sug-name">{{ c.name || c.emails?.[0]?.email }}</span>
                <span class="sug-email">{{ c.emails?.[0]?.email }}</span>
              </li>
            </ul>
          </div>
        </div>

        <!-- Member list -->
        <div v-if="!selectedGroup.members?.length" class="empty-members">No members yet.</div>
        <ul v-else class="member-list">
          <li v-for="m in selectedGroup.members" :key="m.contact_id">
            <div class="m-info">
              <span class="m-name">{{ m.name || m.email }}</span>
              <span class="m-email">{{ m.email }}</span>
            </div>
            <button class="m-remove" title="Remove" @click="removeMember(m.contact_id)">✕</button>
          </li>
        </ul>
      </div>
    </div>
  </div>

  <ContactModal ref="contactModal" @close="onModalClose" />
  <ComposeModal ref="composeModal" />
</template>

<script setup>
import { ref, computed, nextTick, onMounted, watch, provide } from 'vue'
import FolderList from '../components/FolderList.vue'
import ContactModal from '../components/ContactModal.vue'
import ComposeModal from '../components/ComposeModal.vue'
import { useContactsStore } from '../stores/contacts'
import { apiFetch } from '../api'

const contacts = useContactsStore()
const contactModal = ref(null)
const composeModal = ref(null)
const selected = ref(null)
const selectedGroup = ref(null)
const exportUrl = '/api/contacts/export'
const view = ref('contacts')

// Group editing state
const editingGroupName = ref(false)
const groupNameDraft = ref('')
const groupNameInput = ref(null)

// Add-member search
const memberSearch = ref('')
const memberSuggestions = ref([])
let memberDebounce = null
const isRawEmail = computed(() => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(memberSearch.value.trim()))

provide('compose', composeModal)

onMounted(() => contacts.fetchContacts())

watch(() => contacts.contacts, list => {
  if (selected.value) {
    selected.value = list.find(c => c.id === selected.value.id) ?? null
  }
})

watch(() => contacts.groups, list => {
  if (selectedGroup.value) {
    selectedGroup.value = list.find(g => g.id === selectedGroup.value.id) ?? null
  }
})

watch(view, () => {
  selected.value = null
  selectedGroup.value = null
  editingGroupName.value = false
})

function onModalClose() {}

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
  const res = await apiFetch('/api/contacts/import', { method: 'POST', body: fd })
  if (res.ok) {
    const { imported } = await res.json()
    alert(`Imported ${imported} contact(s).`)
    await contacts.fetchContacts()
  } else {
    alert('Import failed.')
  }
  e.target.value = ''
}

// Groups
function openNewGroup() {
  const name = prompt('Group name:')
  if (!name?.trim()) return
  contacts.createGroup(name.trim()).then(g => { selectedGroup.value = g })
}

async function confirmDeleteGroup(g) {
  if (!confirm(`Delete group "${g.name}"?`)) return
  await contacts.deleteGroup(g.id)
  selectedGroup.value = null
}

function startEditGroupName() {
  groupNameDraft.value = selectedGroup.value.name
  editingGroupName.value = true
  nextTick(() => groupNameInput.value?.focus())
}

async function saveGroupName() {
  if (!groupNameDraft.value.trim()) { editingGroupName.value = false; return }
  const g = await contacts.updateGroup(selectedGroup.value.id, groupNameDraft.value.trim())
  selectedGroup.value = g
  editingGroupName.value = false
}

function onMemberSearch() {
  clearTimeout(memberDebounce)
  const q = memberSearch.value.trim()
  if (!q) { memberSuggestions.value = []; return }
  memberDebounce = setTimeout(() => {
    // Filter from local contacts list, excluding existing members
    const existingIds = new Set((selectedGroup.value?.members || []).map(m => m.contact_id))
    memberSuggestions.value = contacts.contacts
      .filter(c => {
        if (existingIds.has(c.id)) return false
        const ql = q.toLowerCase()
        return c.name.toLowerCase().includes(ql) ||
          c.emails.some(e => e.email.toLowerCase().includes(ql))
      })
      .slice(0, 8)
  }, 150)
}

async function addFirstSuggestion() {
  if (memberSuggestions.value.length) {
    addMember(memberSuggestions.value[0])
  } else if (isRawEmail.value) {
    // Auto-create a contact for the typed email, then add to group.
    const email = memberSearch.value.trim()
    const contact = await contacts.createContact({ name: '', notes: '', emails: [{ email, label: '' }] })
    addMember(contact)
  }
}

async function addMember(c) {
  memberSearch.value = ''
  memberSuggestions.value = []
  const g = await contacts.addGroupMember(selectedGroup.value.id, c.id)
  selectedGroup.value = g
}

async function removeMember(contactId) {
  const g = await contacts.removeGroupMember(selectedGroup.value.id, contactId)
  selectedGroup.value = g
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
  padding: 8px 14px;
  border-bottom: 0.5px solid var(--color-border);
  gap: 8px;
  flex-shrink: 0;
}
.tab-row {
  display: flex;
  gap: 2px;
}
.tab-row button {
  padding: 4px 10px;
  font-size: 12px;
  border: 0.5px solid var(--color-border);
  border-radius: 5px;
  cursor: pointer;
  background: var(--color-surface);
  color: var(--color-text-muted);
}
.tab-row button.active {
  background: var(--color-teal-light);
  color: var(--color-teal);
  border-color: var(--color-teal);
}
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
  gap: 12px;
}
.group-name-row { flex: 1; min-width: 0; }
h2 { font-size: 20px; font-weight: 500; }
.group-name-input {
  font-size: 20px;
  font-weight: 500;
  border: none;
  border-bottom: 1.5px solid var(--color-teal);
  background: transparent;
  outline: none;
  width: 100%;
  padding: 2px 0;
}
.detail-actions { display: flex; gap: 8px; flex-shrink: 0; }
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

/* Groups */
.add-member-row {
  margin-bottom: 1rem;
}
.add-member-wrap {
  position: relative;
}
.add-member-input-row {
  display: flex;
  gap: 6px;
}
.member-search-input {
  flex: 1;
  padding: 7px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  font-size: 13px;
  outline: none;
  min-width: 0;
}
.member-search-input:focus { border-color: var(--color-teal); }
.add-member-btn {
  padding: 7px 14px;
  font-size: 13px;
  border: 0.5px solid var(--color-teal);
  border-radius: 6px;
  background: var(--color-teal);
  color: white;
  cursor: pointer;
  flex-shrink: 0;
}
.add-member-btn:disabled {
  opacity: 0.4;
  cursor: default;
}
.member-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-top: 4px;
}
.member-hint a { color: var(--color-teal); }
.member-suggestions {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  list-style: none;
  margin: 0;
  padding: 4px 0;
  z-index: 50;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}
.member-suggestions li {
  padding: 6px 12px;
  cursor: pointer;
  font-size: 13px;
  display: flex;
  gap: 8px;
  align-items: center;
}
.member-suggestions li:hover { background: var(--color-teal-light); }
.sug-name { font-weight: 500; }
.sug-email { color: var(--color-text-muted); font-size: 12px; }
.empty-members {
  font-size: 13px;
  color: var(--color-text-muted);
  padding: 8px 0;
}
.member-list { list-style: none; }
.member-list li {
  display: flex;
  align-items: center;
  padding: 8px 0;
  border-bottom: 0.5px solid var(--color-border);
}
.m-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}
.m-name { font-size: 13px; font-weight: 500; }
.m-email { font-size: 11px; color: var(--color-text-muted); }
.m-remove {
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
.member-list li:hover .m-remove { opacity: 1; }
.m-remove:hover { background: #fde8e8; color: #c0392b; }
</style>
