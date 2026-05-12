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

        <!-- Recurrence -->
        <div class="field-group">
          <label>Repeat</label>
          <select v-model="form.rrule_freq" @change="onFreqChange">
            <option value="">Does not repeat</option>
            <option value="DAILY">Daily</option>
            <option value="WEEKLY">Weekly</option>
            <option value="MONTHLY">Monthly</option>
            <option value="YEARLY">Yearly</option>
          </select>
        </div>
        <template v-if="form.rrule_freq">
          <div class="field-group interval-row">
            <label>Every</label>
            <div class="interval-wrap">
              <input v-model.number="form.rrule_interval" type="number" min="1" max="99" class="interval-input" />
              <span>{{ freqLabel }}</span>
            </div>
          </div>
          <div v-if="form.rrule_freq === 'WEEKLY'" class="field-group">
            <label>On</label>
            <div class="byday-row">
              <button
                v-for="d in weekdays"
                :key="d.code"
                type="button"
                :class="['day-btn', { active: form.rrule_byday.includes(d.code) }]"
                @click="toggleDay(d.code)"
              >{{ d.label }}</button>
            </div>
          </div>
          <div class="field-group">
            <label>Ends</label>
            <div class="ends-row">
              <label class="radio-label">
                <input type="radio" v-model="form.rrule_end" value="never" /> Never
              </label>
              <label class="radio-label">
                <input type="radio" v-model="form.rrule_end" value="count" />
                After
                <input
                  v-model.number="form.rrule_count"
                  type="number" min="1" max="999"
                  class="count-input"
                  @focus="form.rrule_end = 'count'"
                />
                occurrence{{ form.rrule_count === 1 ? '' : 's' }}
              </label>
              <label class="radio-label">
                <input type="radio" v-model="form.rrule_end" value="until" />
                On
                <input
                  type="date"
                  v-model="form.rrule_until"
                  class="until-input"
                  @focus="form.rrule_end = 'until'"
                />
              </label>
            </div>
          </div>
          <div v-if="isEdit" class="recur-notice">Editing changes all occurrences.</div>
        </template>
      </div>
      <ConfirmDialog
        :visible="confirmDeleteVisible"
        message="Delete this event?"
        @confirm="doDelete"
        @cancel="confirmDeleteVisible = false"
        @update:visible="confirmDeleteVisible = $event"
      />
      <div class="modal-footer">
        <button v-if="isEdit" class="delete-btn" @click="confirmDeleteVisible = true">Delete</button>
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
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useCalendarStore } from '../stores/calendar'
import ConfirmDialog from './ConfirmDialog.vue'

const emit = defineEmits(['saved', 'deleted'])
const cal = useCalendarStore()

const visible = ref(false)
const saving = ref(false)
const error = ref('')
const editId = ref(null)
const confirmDeleteVisible = ref(false)
const isEdit = computed(() => editId.value !== null)

const weekdays = [
  { code: 'MO', label: 'Mo' }, { code: 'TU', label: 'Tu' }, { code: 'WE', label: 'We' },
  { code: 'TH', label: 'Th' }, { code: 'FR', label: 'Fr' }, { code: 'SA', label: 'Sa' },
  { code: 'SU', label: 'Su' },
]

const freqLabel = computed(() => {
  const map = { DAILY: 'day(s)', WEEKLY: 'week(s)', MONTHLY: 'month(s)', YEARLY: 'year(s)' }
  return map[form.rrule_freq] ?? ''
})

const form = reactive({
  title: '',
  all_day: false,
  starts_date: '',
  starts_time: '09:00',
  ends_date: '',
  ends_time: '10:00',
  location: '',
  description: '',
  rrule_freq: '',
  rrule_interval: 1,
  rrule_byday: [],
  rrule_end: 'never',
  rrule_count: 10,
  rrule_until: '',
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
    parseRrule(event.rrule || '')
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
    parseRrule('')
  }
  visible.value = true
}

function onFreqChange() {
  // When switching to weekly, default to the day of the event's start.
  if (form.rrule_freq === 'WEEKLY' && form.rrule_byday.length === 0) {
    const codes = ['SU', 'MO', 'TU', 'WE', 'TH', 'FR', 'SA']
    const d = form.starts_date ? new Date(form.starts_date + 'T12:00:00') : new Date()
    form.rrule_byday = [codes[d.getDay()]]
  }
}

function toggleDay(code) {
  const i = form.rrule_byday.indexOf(code)
  if (i >= 0) {
    if (form.rrule_byday.length > 1) form.rrule_byday.splice(i, 1) // keep at least one
  } else {
    form.rrule_byday.push(code)
  }
}

function buildRrule() {
  if (!form.rrule_freq) return ''
  const parts = [`FREQ=${form.rrule_freq}`]
  if (form.rrule_interval > 1) parts.push(`INTERVAL=${form.rrule_interval}`)
  if (form.rrule_freq === 'WEEKLY' && form.rrule_byday.length > 0) {
    // Sort in weekday order
    const order = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU']
    const sorted = [...form.rrule_byday].sort((a, b) => order.indexOf(a) - order.indexOf(b))
    parts.push(`BYDAY=${sorted.join(',')}`)
  }
  if (form.rrule_end === 'count' && form.rrule_count > 0) {
    parts.push(`COUNT=${form.rrule_count}`)
  } else if (form.rrule_end === 'until' && form.rrule_until) {
    const d = new Date(form.rrule_until + 'T00:00:00Z')
    parts.push(`UNTIL=${d.toISOString().replace(/[-:.]/g, '').slice(0, 15)}Z`)
  }
  return parts.join(';')
}

function parseRrule(rrule) {
  if (!rrule) {
    form.rrule_freq = ''; form.rrule_interval = 1; form.rrule_byday = []
    form.rrule_end = 'never'; form.rrule_count = 10; form.rrule_until = ''
    return
  }
  const params = Object.fromEntries(rrule.split(';').map(p => p.split('=')))
  form.rrule_freq = params.FREQ || ''
  form.rrule_interval = parseInt(params.INTERVAL || '1')
  form.rrule_byday = params.BYDAY ? params.BYDAY.split(',') : []
  if (params.COUNT) {
    form.rrule_end = 'count'; form.rrule_count = parseInt(params.COUNT)
  } else if (params.UNTIL) {
    form.rrule_end = 'until'
    const s = params.UNTIL.replace('Z', '')
    form.rrule_until = `${s.slice(0,4)}-${s.slice(4,6)}-${s.slice(6,8)}`
  } else {
    form.rrule_end = 'never'; form.rrule_count = 10; form.rrule_until = ''
  }
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

async function doDelete() {
  confirmDeleteVisible.value = false
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
    rrule: buildRrule(),
  }
}

function dateStr(d) {
  return d.toISOString().slice(0, 10)
}
function timeStr(d) {
  return d.toTimeString().slice(0, 5)
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

select {
  padding: 7px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 13px;
  outline: none;
  background: var(--color-surface);
  color: var(--color-text);
  width: 100%;
  box-sizing: border-box;
}
select:focus { border-color: var(--color-teal); }
.interval-row label { margin-bottom: 4px; }
.interval-wrap { display: flex; align-items: center; gap: 8px; }
.interval-input { width: 56px; text-align: center; }
.byday-row { display: flex; gap: 4px; flex-wrap: wrap; }
.day-btn {
  padding: 4px 8px;
  font-size: 12px;
  border: 0.5px solid var(--color-border);
  border-radius: 5px;
  background: var(--color-surface);
  cursor: pointer;
  color: var(--color-text);
}
.day-btn.active {
  background: var(--color-teal);
  color: white;
  border-color: var(--color-teal);
}
.ends-row { display: flex; flex-direction: column; gap: 6px; }
.radio-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  cursor: pointer;
}
.radio-label input[type="radio"] { width: auto; }
.count-input { width: 56px; text-align: center; padding: 4px 6px; }
.until-input { flex: 1; padding: 4px 6px; }
.recur-notice {
  font-size: 11px;
  color: var(--color-text-muted);
  background: var(--color-teal-light);
  border: 0.5px solid var(--color-teal);
  border-radius: 5px;
  padding: 4px 8px;
}
</style>
