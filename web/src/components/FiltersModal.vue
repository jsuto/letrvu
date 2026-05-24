<template>
  <div v-if="visible" class="fixed inset-0 bg-black/30 z-[200] flex items-center justify-center" @click.self="close">
    <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl w-[680px] max-w-[95vw] flex flex-col shadow-xl max-h-[90vh]">

      <!-- Header -->
      <div class="flex justify-between items-center px-4 py-3 border-b border-[var(--color-border)] text-sm font-medium shrink-0">
        <span>Mail Filters</span>
        <button @click="close" class="bg-none border-none text-lg cursor-pointer text-[var(--color-text-muted)]">×</button>
      </div>

      <!-- Filter list -->
      <div class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-2">

        <p v-if="!filters.loaded && !filtersStore.filters.length" class="text-xs text-[var(--color-text-muted)]">Loading…</p>
        <p v-else-if="filtersStore.filters.length === 0 && !showEditor" class="text-xs text-[var(--color-text-muted)]">
          No filters yet. Filters run in order — first matching filter wins.
        </p>

        <!-- Filter rows -->
        <div v-for="(f, idx) in filtersStore.filters" :key="f.id"
          class="flex items-start gap-2 p-2.5 rounded-md border border-[var(--color-border)] bg-[var(--color-bg)]">

          <!-- Reorder arrows -->
          <div class="flex flex-col gap-0.5 shrink-0 mt-0.5">
            <button @click="moveUp(idx)" :disabled="idx === 0"
              class="text-[10px] leading-none px-0.5 py-0 border border-[var(--color-border)] rounded bg-[var(--color-surface)] cursor-pointer disabled:opacity-30 hover:border-teal hover:text-teal">▲</button>
            <button @click="moveDown(idx)" :disabled="idx === filtersStore.filters.length - 1"
              class="text-[10px] leading-none px-0.5 py-0 border border-[var(--color-border)] rounded bg-[var(--color-surface)] cursor-pointer disabled:opacity-30 hover:border-teal hover:text-teal">▼</button>
          </div>

          <!-- Body -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-[var(--color-text)] truncate">{{ f.name }}</span>
              <span v-if="!f.enabled" class="text-[10px] text-[var(--color-text-muted)] border border-[var(--color-border)] rounded px-1">disabled</span>
            </div>
            <div class="text-xs text-[var(--color-text-muted)] mt-0.5">
              {{ summaryText(f) }}
            </div>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-1 shrink-0">
            <button @click="startEdit(f)"
              class="px-2 py-1 text-xs border border-[var(--color-border)] rounded bg-[var(--color-surface)] cursor-pointer hover:border-teal hover:text-teal">Edit</button>
            <button @click="removeFilter(f.id)"
              class="px-2 py-1 text-xs border border-[var(--color-border)] rounded bg-[var(--color-surface)] cursor-pointer text-[var(--color-text-muted)] hover:border-red-500 hover:text-red-600">Delete</button>
          </div>
        </div>

        <!-- Inline editor -->
        <div v-if="showEditor" class="flex flex-col gap-3 p-3 border border-[var(--color-border)] rounded-md bg-[var(--color-bg)]">
          <p class="text-xs font-medium text-[var(--color-text)]">{{ editId ? 'Edit filter' : 'New filter' }}</p>

          <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
            Name
            <input v-model="form.name" type="text" placeholder="e.g. Invoices"
              class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-surface)] text-[var(--color-text)] outline-none focus:border-teal" />
          </label>

          <div class="flex items-center gap-2 text-sm">
            <span class="text-[var(--color-text)]">Match</span>
            <select v-model="form.match_all"
              class="px-2 py-1.5 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-surface)] text-[var(--color-text)] outline-none focus:border-teal">
              <option :value="true">all conditions (AND)</option>
              <option :value="false">any condition (OR)</option>
            </select>
          </div>

          <!-- Conditions -->
          <div class="flex flex-col gap-1.5">
            <div class="text-xs text-[var(--color-text-muted)] font-medium">Conditions</div>
            <div v-for="(c, i) in form.conditions" :key="i" class="flex items-center gap-1.5 flex-wrap">
              <select v-model="c.field"
                class="px-2 py-1.5 border border-[var(--color-border)] rounded-md text-xs bg-[var(--color-surface)] text-[var(--color-text)] outline-none focus:border-teal">
                <option value="subject">Subject</option>
                <option value="from">From</option>
                <option value="to">To</option>
                <option value="body">Body</option>
                <option value="has_attachment">Has attachment</option>
              </select>
              <select v-model="c.op"
                class="px-2 py-1.5 border border-[var(--color-border)] rounded-md text-xs bg-[var(--color-surface)] text-[var(--color-text)] outline-none focus:border-teal">
                <option value="contains">contains</option>
                <option value="not_contains">does not contain</option>
                <option value="equals">equals</option>
                <option value="not_equals">does not equal</option>
                <option value="matches">matches regex</option>
              </select>
              <input v-if="c.field !== 'has_attachment'" v-model="c.value" type="text" placeholder="value"
                class="flex-1 min-w-[100px] px-2 py-1.5 border border-[var(--color-border)] rounded-md text-xs bg-[var(--color-surface)] text-[var(--color-text)] outline-none focus:border-teal" />
              <button @click="removeCondition(i)"
                class="px-1.5 py-1 border border-[var(--color-border)] rounded text-xs cursor-pointer text-[var(--color-text-muted)] hover:border-red-500 hover:text-red-600 bg-transparent">×</button>
            </div>
            <button @click="addCondition"
              class="self-start text-xs border border-dashed border-[var(--color-border)] rounded-md px-2 py-1 cursor-pointer text-[var(--color-text-muted)] bg-transparent hover:border-teal hover:text-teal">+ Add condition</button>
          </div>

          <!-- Actions -->
          <div class="flex flex-col gap-1.5">
            <div class="text-xs text-[var(--color-text-muted)] font-medium">Actions</div>
            <div v-for="(a, i) in form.actions" :key="i" class="flex items-center gap-1.5">
              <select v-model="a.type"
                class="px-2 py-1.5 border border-[var(--color-border)] rounded-md text-xs bg-[var(--color-surface)] text-[var(--color-text)] outline-none focus:border-teal">
                <option value="move">Move to folder</option>
                <option value="mark_read">Mark as read</option>
                <option value="mark_flagged">Mark as flagged</option>
                <option value="delete">Delete (discard)</option>
                <option value="stop">Stop processing</option>
              </select>
              <input v-if="a.type === 'move'" v-model="a.value" type="text" placeholder="Folder name"
                class="flex-1 px-2 py-1.5 border border-[var(--color-border)] rounded-md text-xs bg-[var(--color-surface)] text-[var(--color-text)] outline-none focus:border-teal" />
              <button @click="removeAction(i)"
                class="px-1.5 py-1 border border-[var(--color-border)] rounded text-xs cursor-pointer text-[var(--color-text-muted)] hover:border-red-500 hover:text-red-600 bg-transparent">×</button>
            </div>
            <button @click="addAction"
              class="self-start text-xs border border-dashed border-[var(--color-border)] rounded-md px-2 py-1 cursor-pointer text-[var(--color-text-muted)] bg-transparent hover:border-teal hover:text-teal">+ Add action</button>
          </div>

          <!-- Enabled toggle -->
          <div class="flex items-center gap-2">
            <span class="text-sm text-[var(--color-text)]">Enabled</span>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="form.enabled" class="sr-only peer" />
              <div class="w-9 h-5 bg-[var(--color-border)] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-teal"></div>
            </label>
          </div>

          <p v-if="editorError" class="text-xs text-red-600">{{ editorError }}</p>

          <div class="flex gap-2">
            <button @click="saveFilter" :disabled="editorBusy"
              class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-60">
              {{ editorBusy ? 'Saving…' : 'Save filter' }}
            </button>
            <button @click="cancelEdit"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md text-xs cursor-pointer bg-[var(--color-surface)] text-[var(--color-text)]">Cancel</button>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-4 py-3 border-t border-[var(--color-border)] flex items-center gap-3 shrink-0">
        <button v-if="!showEditor" @click="startNew"
          class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-sm font-medium cursor-pointer">
          + New filter
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useFiltersStore } from '../stores/filters'

const props = defineProps({
  visible: Boolean,
})
const emit = defineEmits(['close'])

const filtersStore = useFiltersStore()
const filters = ref({ loaded: false })

const showEditor = ref(false)
const editId = ref(null)
const editorBusy = ref(false)
const editorError = ref('')

const form = reactive({
  name: '',
  match_all: true,
  conditions: [],
  actions: [],
  enabled: true,
})

watch(() => props.visible, async (v) => {
  if (v && !filtersStore.loaded) {
    await filtersStore.fetchFilters()
    filters.value.loaded = true
  }
})

function close() {
  emit('close')
}

function summaryText(f) {
  const cond = f.conditions?.length ?? 0
  const act = f.actions?.length ?? 0
  const logic = f.match_all ? 'all' : 'any'
  return `${cond} condition${cond !== 1 ? 's' : ''} (${logic}) → ${act} action${act !== 1 ? 's' : ''}`
}

function startNew() {
  editId.value = null
  form.name = ''
  form.match_all = true
  form.conditions = [{ field: 'subject', op: 'contains', value: '' }]
  form.actions = [{ type: 'move', value: '' }]
  form.enabled = true
  editorError.value = ''
  showEditor.value = true
}

function startEdit(f) {
  editId.value = f.id
  form.name = f.name
  form.match_all = f.match_all
  form.conditions = JSON.parse(JSON.stringify(f.conditions ?? []))
  form.actions = JSON.parse(JSON.stringify(f.actions ?? []))
  form.enabled = f.enabled
  editorError.value = ''
  showEditor.value = true
}

function cancelEdit() {
  showEditor.value = false
  editId.value = null
}

function addCondition() {
  form.conditions.push({ field: 'subject', op: 'contains', value: '' })
}
function removeCondition(i) {
  form.conditions.splice(i, 1)
}

function addAction() {
  form.actions.push({ type: 'move', value: '' })
}
function removeAction(i) {
  form.actions.splice(i, 1)
}

async function saveFilter() {
  if (!form.name.trim()) {
    editorError.value = 'Name is required'
    return
  }
  if (form.conditions.length === 0) {
    editorError.value = 'At least one condition is required'
    return
  }
  if (form.actions.length === 0) {
    editorError.value = 'At least one action is required'
    return
  }
  editorError.value = ''
  editorBusy.value = true
  try {
    const payload = {
      name: form.name.trim(),
      match_all: form.match_all,
      conditions: form.conditions.map(c => ({ ...c })),
      actions: form.actions.map(a => ({ ...a })),
      enabled: form.enabled,
    }
    if (editId.value) {
      await filtersStore.updateFilter(editId.value, payload)
    } else {
      await filtersStore.createFilter(payload)
    }
    showEditor.value = false
    editId.value = null
  } catch (e) {
    editorError.value = e.message
  } finally {
    editorBusy.value = false
  }
}

async function removeFilter(id) {
  try {
    await filtersStore.deleteFilter(id)
  } catch (e) {
    // best-effort
  }
}

async function moveUp(idx) {
  if (idx === 0) return
  const arr = filtersStore.filters
  const newOrder = arr.map(f => f.id)
  ;[newOrder[idx - 1], newOrder[idx]] = [newOrder[idx], newOrder[idx - 1]]
  await filtersStore.reorderFilters(newOrder)
  await filtersStore.fetchFilters()
}

async function moveDown(idx) {
  const arr = filtersStore.filters
  if (idx >= arr.length - 1) return
  const newOrder = arr.map(f => f.id)
  ;[newOrder[idx], newOrder[idx + 1]] = [newOrder[idx + 1], newOrder[idx]]
  await filtersStore.reorderFilters(newOrder)
  await filtersStore.fetchFilters()
}

</script>
