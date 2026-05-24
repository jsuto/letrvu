<template>
  <div v-if="visible" class="fixed inset-0 bg-black/30 z-[200] flex items-center justify-center" @click.self="close">
    <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl w-[600px] max-w-[95vw] flex flex-col shadow-xl max-h-[90vh]">

      <!-- Header -->
      <div class="flex justify-between items-center px-4 py-3 border-b border-[var(--color-border)] text-sm font-medium shrink-0">
        <span>Message Templates</span>
        <button @click="close" class="bg-none border-none text-lg cursor-pointer text-[var(--color-text-muted)]">×</button>
      </div>

      <!-- List -->
      <div class="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-2">
        <p v-if="!templatesStore.loaded && !templatesStore.templates.length" class="text-xs text-[var(--color-text-muted)]">Loading…</p>
        <p v-else-if="templatesStore.templates.length === 0 && !showEditor" class="text-xs text-[var(--color-text-muted)]">
          No templates yet. Create one to quickly insert saved text while composing.
        </p>

        <div v-for="t in templatesStore.templates" :key="t.id"
          class="flex items-start gap-2 p-2.5 rounded-md border border-[var(--color-border)] bg-[var(--color-bg)]">
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium text-[var(--color-text)] truncate">{{ t.name }}</div>
            <div v-if="t.subject" class="text-xs text-[var(--color-text-muted)] truncate mt-0.5">Subject: {{ t.subject }}</div>
          </div>
          <div class="flex items-center gap-1 shrink-0">
            <button @click="startEdit(t)"
              class="px-2 py-1 text-xs border border-[var(--color-border)] rounded bg-[var(--color-surface)] cursor-pointer hover:border-teal hover:text-teal">Edit</button>
            <button @click="removeTemplate(t.id)"
              class="px-2 py-1 text-xs border border-[var(--color-border)] rounded bg-[var(--color-surface)] cursor-pointer text-[var(--color-text-muted)] hover:border-red-500 hover:text-red-600">Delete</button>
          </div>
        </div>

        <!-- Inline editor -->
        <div v-if="showEditor" class="flex flex-col gap-3 p-3 border border-[var(--color-border)] rounded-md bg-[var(--color-bg)]">
          <p class="text-xs font-medium text-[var(--color-text)]">{{ editId ? 'Edit template' : 'New template' }}</p>

          <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
            Name
            <input v-model="form.name" type="text" placeholder="e.g. Meeting request"
              class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-surface)] text-[var(--color-text)] outline-none focus:border-teal" />
          </label>

          <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
            Subject <span class="font-normal">(optional — fills the subject line when inserted)</span>
            <input v-model="form.subject" type="text" placeholder="Leave blank to keep the current subject"
              class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-surface)] text-[var(--color-text)] outline-none focus:border-teal" />
          </label>

          <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
            Body
            <textarea v-model="form.body" rows="8" placeholder="Template text…"
              class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-surface)] text-[var(--color-text)] outline-none resize-y leading-relaxed focus:border-teal font-sans" />
          </label>

          <p v-if="editorError" class="text-xs text-red-600">{{ editorError }}</p>

          <div class="flex gap-2">
            <button @click="saveTemplate" :disabled="editorBusy"
              class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-60">
              {{ editorBusy ? 'Saving…' : 'Save template' }}
            </button>
            <button @click="cancelEdit"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md text-xs cursor-pointer bg-[var(--color-surface)] text-[var(--color-text)]">Cancel</button>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-4 py-3 border-t border-[var(--color-border)] shrink-0">
        <button v-if="!showEditor" @click="startNew"
          class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-sm font-medium cursor-pointer">
          + New template
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useTemplatesStore } from '../stores/templates'

const props = defineProps({ visible: Boolean })
const emit = defineEmits(['close'])

const templatesStore = useTemplatesStore()

const showEditor = ref(false)
const editId = ref(null)
const editorBusy = ref(false)
const editorError = ref('')

const form = reactive({ name: '', subject: '', body: '' })

watch(() => props.visible, async (v) => {
  if (v && !templatesStore.loaded) await templatesStore.fetchTemplates()
})

function close() { emit('close') }

function startNew() {
  editId.value = null
  form.name = ''
  form.subject = ''
  form.body = ''
  editorError.value = ''
  showEditor.value = true
}

function startEdit(t) {
  editId.value = t.id
  form.name = t.name
  form.subject = t.subject
  form.body = t.body
  editorError.value = ''
  showEditor.value = true
}

function cancelEdit() {
  showEditor.value = false
  editId.value = null
}

async function saveTemplate() {
  if (!form.name.trim()) { editorError.value = 'Name is required'; return }
  if (!form.body.trim()) { editorError.value = 'Body is required'; return }
  editorError.value = ''
  editorBusy.value = true
  try {
    const payload = { name: form.name.trim(), subject: form.subject.trim(), body: form.body }
    if (editId.value) {
      await templatesStore.updateTemplate(editId.value, payload)
    } else {
      await templatesStore.createTemplate(payload)
    }
    showEditor.value = false
    editId.value = null
  } catch (e) {
    editorError.value = e.message
  } finally {
    editorBusy.value = false
  }
}

async function removeTemplate(id) {
  try { await templatesStore.deleteTemplate(id) } catch {}
}
</script>
