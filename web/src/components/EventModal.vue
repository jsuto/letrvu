<template>
  <div v-if="visible" class="overlay" @click.self="close">
    <div class="modal">
      <div class="modal-header">
        <span>{{ isEdit ? 'Edit event' : 'New event' }}</span>
        <button @click="close" class="close">×</button>
      </div>
      <div class="modal-body">
        <div class="field-group">
          <label>Title</label>
          <input v-model="form.title" type="text" placeholder="Event title" />
        </div>
        <div class="field-group">
          <label class="all-day-label">
            <input v-model="form.all_day" type="checkbox" />
            All-day event
          </label>
        </div>
        <div class="field-row">
          <div class="field-group">
            <label>Start</label>
            <input v-model="form.starts_date" type="date" />
            <input v-if="!form.all_day" v-model="form.starts_time" type="time" class="time-input" />
          </div>
          <div class="field-group">
            <label>End</label>
            <input v-model="form.ends_date" type="date" />
            <input v-if="!form.all_day" v-model="form.ends_time" type="time" class="time-input" />
          </div>
        </div>
        <div class="field-group">
          <label>Location</label>
          <input v-model="form.location" type="text" placeholder="Optional location" />
        </div>
        <div class="field-group">
          <label>Description</label>
          <textarea v-model="form.description" rows="3" placeholder="Optional description" />
        </div>
      </div>
      <div class="modal-footer">
        <button v-if="isEdit" class="delete-btn" @click="remove">Delete</button>
        <span class="spacer" />
        <p v-if="error" class="error">{{ error }}</p>
        <button @click="save" :disabled="saving" class="save-btn">
          {{ saving ? 'Saving…' : 'Save' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { useCalendarStore } from '../stores/calendar'

const emit = defineEmits(['saved', 'deleted'])
const cal = useCalendarStore()

const visible = ref(false)
const saving = ref(false)
const error = ref('')
const editId = ref(null)
const isEdit = computed(() => editId.value !== null)

const form = reactive({
  title: '',
  all_day: false,
  starts_date: '',
  starts_time: '09:00',
  ends_date: '',
  ends_time: '10:00',
  location: '',
  description: '',
})

function open(event = null, defaultDate = null) {
  editId.value = event?.id ?? null
  error.value = ''

  if (event) {
    const s = new Date(event.starts_at)
    const e = new Date(event.ends_at)
    form.title = event.title
    form.all_day = event.all_day
    form.starts_date = dateStr(s)
    form.starts_time = timeStr(s)
    form.ends_date = dateStr(e)
    form.ends_time = timeStr(e)
    form.location = event.location
    form.description = event.description
  } else {
    const d = defaultDate ?? new Date()
    form.title = ''
    form.all_day = false
    form.starts_date = dateStr(d)
    form.starts_time = '09:00'
    form.ends_date = dateStr(d)
    form.ends_time = '10:00'
    form.location = ''
    form.description = ''
  }
  visible.value = true
}

function close() {
  visible.value = false
}

async function save() {
  if (!form.title.trim()) {
    error.value = 'Title is required.'
    return
  }
  saving.value = true
  error.value = ''
  try {
    const data = buildPayload()
    if (isEdit.value) {
      await cal.updateEvent(editId.value, data)
    } else {
      await cal.createEvent(data)
    }
    emit('saved')
    close()
  } catch (e) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!confirm('Delete this event?')) return
  await cal.deleteEvent(editId.value)
  emit('deleted')
  close()
}

function buildPayload() {
  const starts_at = form.all_day
    ? new Date(form.starts_date + 'T00:00:00').toISOString()
    : new Date(form.starts_date + 'T' + form.starts_time).toISOString()
  const ends_at = form.all_day
    ? new Date(form.ends_date + 'T00:00:00').toISOString()
    : new Date(form.ends_date + 'T' + form.ends_time).toISOString()
  return {
    title: form.title.trim(),
    description: form.description.trim(),
    location: form.location.trim(),
    all_day: form.all_day,
    starts_at,
    ends_at,
  }
}

function dateStr(d) {
  return d.toISOString().slice(0, 10)
}
function timeStr(d) {
  return d.toTimeString().slice(0, 5)
}

defineExpose({ open, close })
</script>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}
.modal {
  width: 460px;
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  max-height: 90vh;
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  font-weight: 500;
}
.close { background: none; border: none; font-size: 18px; cursor: pointer; color: var(--color-text-muted); }
.modal-body { padding: 16px; overflow-y: auto; flex: 1; display: flex; flex-direction: column; gap: 12px; }
.field-group { display: flex; flex-direction: column; gap: 4px; }
.field-group label {
  font-size: 12px;
  color: var(--color-text-muted);
}
.all-day-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text);
  cursor: pointer;
}
.all-day-label input[type="checkbox"] { width: auto; }
input[type="text"], input[type="date"], input[type="time"], textarea {
  padding: 7px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 13px;
  outline: none;
  width: 100%;
  box-sizing: border-box;
}
input:focus, textarea:focus { border-color: var(--color-teal); }
textarea { resize: vertical; }
.field-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.time-input { margin-top: 6px; }
.modal-footer {
  padding: 10px 16px;
  border-top: 0.5px solid var(--color-border);
  display: flex;
  align-items: center;
  gap: 10px;
}
.spacer { flex: 1; }
.save-btn {
  padding: 8px 20px;
  background: var(--color-teal);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.save-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.delete-btn {
  padding: 8px 14px;
  background: none;
  border: 0.5px solid #f5c6c6;
  border-radius: 6px;
  font-size: 13px;
  color: #c0392b;
  cursor: pointer;
}
.error { font-size: 12px; color: #c0392b; }
</style>
