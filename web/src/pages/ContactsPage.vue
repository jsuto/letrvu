<template>
  <div class="grid h-screen overflow-hidden bg-[var(--color-bg)]" style="grid-template-columns: 200px 260px 1fr">
    <!-- Sidebar -->
    <aside class="border-r border-[var(--color-border)] overflow-y-auto">
      <FolderList />
    </aside>

    <!-- Contact/Group list panel -->
    <div class="border-r border-[var(--color-border)] flex flex-col overflow-hidden">
      <div class="flex items-center justify-between px-3.5 py-2 border-b border-[var(--color-border)] gap-2 shrink-0">
        <div class="flex gap-0.5">
          <button
            :class="['px-2.5 py-1 text-xs border rounded cursor-pointer', view === 'contacts' ? 'bg-[var(--color-teal-light)] text-teal border-teal' : 'bg-[var(--color-surface)] text-[var(--color-text-muted)] border-[var(--color-border)]']"
            @click="view = 'contacts'"
          >Contacts</button>
          <button
            :class="['px-2.5 py-1 text-xs border rounded cursor-pointer', view === 'groups' ? 'bg-[var(--color-teal-light)] text-teal border-teal' : 'bg-[var(--color-surface)] text-[var(--color-text-muted)] border-[var(--color-border)]']"
            @click="view = 'groups'"
          >Groups</button>
        </div>
        <div class="flex gap-1.5 items-center">
          <template v-if="view === 'contacts'">
            <label class="px-2.5 py-1.5 text-xs border border-[var(--color-border)] rounded-md cursor-pointer bg-[var(--color-surface)] text-[var(--color-text)]" title="Import vCard (.vcf)">
              Import
              <input type="file" accept=".vcf" @change="importVCard" hidden />
            </label>
            <a :href="exportUrl" download="contacts.vcf" class="px-2.5 py-1.5 text-xs border border-[var(--color-border)] rounded-md cursor-pointer bg-[var(--color-surface)] text-[var(--color-text)] no-underline">Export</a>
          </template>
          <button class="px-2.5 py-1.5 text-xs bg-teal text-white border-none border-teal rounded-md cursor-pointer"
            @click="view === 'contacts' ? contactModal?.open() : openNewGroup()">+ New</button>
        </div>
      </div>

      <!-- Contacts list -->
      <template v-if="view === 'contacts'">
        <div v-if="contacts.loading" class="flex items-center justify-center h-full text-[var(--color-text-muted)] text-sm">Loading…</div>
        <div v-else-if="contacts.contacts.length === 0" class="flex items-center justify-center h-full text-[var(--color-text-muted)] text-sm">No contacts yet.</div>
        <ul v-else class="list-none flex-1 overflow-y-auto">
          <li
            v-for="c in contacts.contacts"
            :key="c.id"
            :class="['flex items-center px-3.5 py-2 cursor-pointer border-b border-[var(--color-border)] hover:bg-[var(--color-teal-light)]', selected?.id === c.id ? 'bg-[var(--color-teal-light)]' : '']"
            @click="selected = c"
          >
            <div class="flex flex-col flex-1 min-w-0">
              <span class="text-sm font-medium">{{ c.name || c.emails?.[0]?.email || '—' }}</span>
              <span class="text-[11px] text-[var(--color-text-muted)]">{{ c.emails?.[0]?.email }}</span>
            </div>
            <button class="bg-none border-none cursor-pointer text-[var(--color-text-muted)] text-xs px-1.5 py-1 rounded opacity-0 shrink-0 hover:bg-[#fde8e8] hover:text-red-600 [li:hover_&]:opacity-100" title="Delete" @click.stop="confirmDelete(c)">✕</button>
          </li>
        </ul>
      </template>

      <!-- Groups list -->
      <template v-else>
        <div v-if="contacts.loading" class="flex items-center justify-center h-full text-[var(--color-text-muted)] text-sm">Loading…</div>
        <div v-else-if="contacts.groups.length === 0" class="flex items-center justify-center h-full text-[var(--color-text-muted)] text-sm">No groups yet.</div>
        <ul v-else class="list-none flex-1 overflow-y-auto">
          <li
            v-for="g in contacts.groups"
            :key="g.id"
            :class="['flex items-center px-3.5 py-2 cursor-pointer border-b border-[var(--color-border)] hover:bg-[var(--color-teal-light)]', selectedGroup?.id === g.id ? 'bg-[var(--color-teal-light)]' : '']"
            @click="selectedGroup = g"
          >
            <div class="flex flex-col flex-1 min-w-0">
              <span class="text-sm font-medium">{{ g.name }}</span>
              <span class="text-[11px] text-[var(--color-text-muted)]">{{ g.members?.length ?? 0 }} member{{ g.members?.length === 1 ? '' : 's' }}</span>
            </div>
            <button class="bg-none border-none cursor-pointer text-[var(--color-text-muted)] text-xs px-1.5 py-1 rounded opacity-0 shrink-0 hover:bg-[#fde8e8] hover:text-red-600 [li:hover_&]:opacity-100" title="Delete" @click.stop="confirmDeleteGroup(g)">✕</button>
          </li>
        </ul>
      </template>
    </div>

    <!-- Contact detail panel -->
    <div class="overflow-y-auto p-8" v-if="view === 'contacts'">
      <div v-if="!selected" class="flex items-center justify-center h-full text-[var(--color-text-muted)] text-sm">Select a contact</div>
      <div v-else>
        <div class="flex items-start justify-between mb-4 gap-3">
          <h2 class="text-xl font-medium">{{ selected.name || selected.emails?.[0]?.email || '—' }}</h2>
          <div class="flex gap-2 shrink-0">
            <button class="px-3.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm cursor-pointer hover:bg-[var(--color-bg)]" @click="contactModal?.open(selected)">Edit</button>
            <button class="px-3.5 py-1.5 border border-red-200 rounded-md bg-[var(--color-surface)] text-sm cursor-pointer text-red-600 hover:bg-[var(--color-bg)]" @click="confirmDelete(selected)">Delete</button>
          </div>
        </div>
        <div v-if="selected.notes" class="text-sm text-[var(--color-text-muted)] mb-4 whitespace-pre-wrap">{{ selected.notes }}</div>
        <ul class="list-none">
          <li v-for="e in selected.emails" :key="e.id" class="flex items-center gap-2 py-1.5 border-b border-[var(--color-border)] text-sm">
            <span>{{ e.email }}</span>
            <span v-if="e.label" class="text-[11px] bg-[var(--color-teal-light)] border border-teal rounded px-1.5 py-px text-teal">{{ e.label }}</span>
          </li>
        </ul>
      </div>
    </div>

    <!-- Group detail panel -->
    <div class="overflow-y-auto p-8" v-else>
      <div v-if="!selectedGroup" class="flex items-center justify-center h-full text-[var(--color-text-muted)] text-sm">Select a group</div>
      <div v-else>
        <div class="flex items-start justify-between mb-4 gap-3">
          <div class="flex-1 min-w-0">
            <h2 v-if="!editingGroupName" class="text-xl font-medium">{{ selectedGroup.name }}</h2>
            <input
              v-else
              ref="groupNameInput"
              v-model="groupNameDraft"
              class="text-xl font-medium border-none border-b-2 border-teal bg-transparent outline-none w-full py-0.5"
              @keydown.enter="saveGroupName"
              @keydown.escape="editingGroupName = false"
            />
          </div>
          <div class="flex gap-2 shrink-0">
            <button v-if="!editingGroupName" class="px-3.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm cursor-pointer hover:bg-[var(--color-bg)]" @click="startEditGroupName">Rename</button>
            <button v-else class="px-3.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm cursor-pointer hover:bg-[var(--color-bg)]" @click="saveGroupName">Save</button>
            <button class="px-3.5 py-1.5 border border-red-200 rounded-md bg-[var(--color-surface)] text-sm cursor-pointer text-red-600 hover:bg-[var(--color-bg)]" @click="confirmDeleteGroup(selectedGroup)">Delete</button>
          </div>
        </div>

        <!-- Add member search -->
        <div class="mb-4">
          <div class="relative">
            <div class="flex gap-1.5">
              <input
                v-model="memberSearch"
                placeholder="Search contacts or type an email address…"
                class="flex-1 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-sm outline-none min-w-0 focus:border-teal"
                @input="onMemberSearch"
                @keydown.enter.prevent="addFirstSuggestion"
                @keydown.escape="memberSuggestions = []"
                autocomplete="off"
              />
              <button
                class="px-3.5 py-1.5 text-sm border border-teal rounded-md bg-teal text-white cursor-pointer shrink-0 disabled:opacity-40 disabled:cursor-default"
                :disabled="!memberSuggestions.length && !isRawEmail"
                @click="addFirstSuggestion"
              >Add</button>
            </div>
            <div v-if="memberSearch && !memberSuggestions.length && !isRawEmail" class="text-xs text-[var(--color-text-muted)] mt-1">
              No matching contacts. Type a full email address to add directly.
            </div>
            <ul v-if="memberSuggestions.length" class="absolute top-[calc(100%+4px)] left-0 right-0 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-md list-none m-0 py-1 z-50 shadow-lg">
              <li
                v-for="c in memberSuggestions"
                :key="c.id"
                class="px-3 py-1.5 cursor-pointer text-sm flex gap-2 items-center hover:bg-[var(--color-teal-light)]"
                @mousedown.prevent="addMember(c)"
              >
                <span class="font-medium">{{ c.name || c.emails?.[0]?.email }}</span>
                <span class="text-[var(--color-text-muted)] text-xs">{{ c.emails?.[0]?.email }}</span>
              </li>
            </ul>
          </div>
        </div>

        <!-- Member list -->
        <div v-if="!selectedGroup.members?.length" class="text-sm text-[var(--color-text-muted)] py-2">No members yet.</div>
        <ul v-else class="list-none">
          <li v-for="m in selectedGroup.members" :key="m.contact_id" class="flex items-center py-2 border-b border-[var(--color-border)] hover:[&_.m-remove]:opacity-100">
            <div class="flex flex-col flex-1 min-w-0">
              <span class="text-sm font-medium">{{ m.name || m.email }}</span>
              <span class="text-[11px] text-[var(--color-text-muted)]">{{ m.email }}</span>
            </div>
            <button class="m-remove bg-none border-none cursor-pointer text-[var(--color-text-muted)] text-xs px-1.5 py-1 rounded opacity-0 shrink-0 hover:bg-[#fde8e8] hover:text-red-600" title="Remove" @click="removeMember(m.contact_id)">✕</button>
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
