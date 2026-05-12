<template>
  <div v-if="visible" class="fixed inset-0 bg-black/30 flex items-center justify-center z-[200]" @click.self="close">
    <div class="w-[460px] bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl flex flex-col max-h-[90vh]">

      <!-- Header -->
      <div class="flex justify-between items-center px-4 py-3 border-b border-[var(--color-border)] text-sm font-medium">
        <span>{{ isEdit ? 'Edit event' : 'New event' }}</span>
        <button @click="close" class="bg-none border-none text-lg cursor-pointer text-[var(--color-text-muted)]">×</button>
      </div>

      <!-- Body -->
      <div class="px-4 py-4 overflow-y-auto flex-1 flex flex-col gap-3">
        <div class="flex flex-col gap-1">
          <label class="text-xs text-[var(--color-text-muted)]">Title</label>
          <input v-model="form.title" type="text" placeholder="Event title"
            class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none w-full box-border focus:border-teal" />
        </div>
        <div class="flex flex-col gap-1">
          <label class="flex items-center gap-1.5 text-sm text-[var(--color-text)] cursor-pointer">
            <input v-model="form.all_day" type="checkbox" class="w-auto" />
            All-day event
          </label>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div class="flex flex-col gap-1">
            <label class="text-xs text-[var(--color-text-muted)]">Start</label>
            <input v-model="form.starts_date" type="date"
              class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none w-full box-border focus:border-teal" />
            <input v-if="!form.all_day" v-model="form.starts_time" type="time"
              class="mt-1.5 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none w-full box-border focus:border-teal" />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-xs text-[var(--color-text-muted)]">End</label>
            <input v-model="form.ends_date" type="date"
              class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none w-full box-border focus:border-teal" />
            <input v-if="!form.all_day" v-model="form.ends_time" type="time"
              class="mt-1.5 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none w-full box-border focus:border-teal" />
          </div>
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-xs text-[var(--color-text-muted)]">Location</label>
          <input v-model="form.location" type="text" placeholder="Optional location"
            class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none w-full box-border focus:border-teal" />
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-xs text-[var(--color-text-muted)]">Description</label>
          <textarea v-model="form.description" rows="3" placeholder="Optional description"
            class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none w-full box-border resize-y focus:border-teal" />
        </div>

        <!-- Recurrence -->
        <div class="flex flex-col gap-1">
          <label class="text-xs text-[var(--color-text-muted)]">Repeat</label>
          <select v-model="form.rrule_freq" @change="onFreqChange"
            class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] text-[var(--color-text)] w-full box-border focus:border-teal">
            <option value="">Does not repeat</option>
            <option value="DAILY">Daily</option>
            <option value="WEEKLY">Weekly</option>
            <option value="MONTHLY">Monthly</option>
            <option value="YEARLY">Yearly</option>
          </select>
        </div>
        <template v-if="form.rrule_freq">
          <div class="flex flex-col gap-1">
            <label class="text-xs text-[var(--color-text-muted)] mb-1">Every</label>
            <div class="flex items-center gap-2">
              <input v-model.number="form.rrule_interval" type="number" min="1" max="99"
                class="w-14 text-center px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none focus:border-teal" />
              <span class="text-sm text-[var(--color-text)]">{{ freqLabel }}</span>
            </div>
          </div>
          <div v-if="form.rrule_freq === 'WEEKLY'" class="flex flex-col gap-1">
            <label class="text-xs text-[var(--color-text-muted)]">On</label>
            <div class="flex gap-1 flex-wrap">
              <button
                v-for="d in weekdays"
                :key="d.code"
                type="button"
                :class="[
                  'px-2 py-1 text-xs border rounded cursor-pointer',
                  form.rrule_byday.includes(d.code)
                    ? 'bg-teal text-white border-teal'
                    : 'bg-[var(--color-surface)] text-[var(--color-text)] border-[var(--color-border)]',
                ]"
                @click="toggleDay(d.code)"
              >{{ d.label }}</button>
            </div>
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-xs text-[var(--color-text-muted)]">Ends</label>
            <div class="flex flex-col gap-1.5">
              <label class="flex items-center gap-1.5 text-sm cursor-pointer">
                <input type="radio" v-model="form.rrule_end" value="never" class="w-auto" /> Never
              </label>
              <label class="flex items-center gap-1.5 text-sm cursor-pointer">
                <input type="radio" v-model="form.rrule_end" value="count" class="w-auto" />
                After
                <input
                  v-model.number="form.rrule_count"
                  type="number" min="1" max="999"
                  class="w-14 text-center px-1.5 py-1 border border-[var(--color-border)] rounded-md text-sm outline-none focus:border-teal"
                  @focus="form.rrule_end = 'count'"
                />
                occurrence{{ form.rrule_count === 1 ? '' : 's' }}
              </label>
              <label class="flex items-center gap-1.5 text-sm cursor-pointer">
                <input type="radio" v-model="form.rrule_end" value="until" class="w-auto" />
                On
                <input
                  type="date"
                  v-model="form.rrule_until"
                  class="flex-1 px-1.5 py-1 border border-[var(--color-border)] rounded-md text-sm outline-none focus:border-teal"
                  @focus="form.rrule_end = 'until'"
                />
              </label>
            </div>
          </div>
          <div v-if="isEdit" class="text-[11px] text-[var(--color-text-muted)] bg-[var(--color-teal-light)] border border-teal rounded px-2 py-1">Editing changes all occurrences.</div>
        </template>
      </div>

      <ConfirmDialog
        :visible="confirmDeleteVisible"
        message="Delete this event?"
        @confirm="doDelete"
        @cancel="confirmDeleteVisible = false"
        @update:visible="confirmDeleteVisible = $event"
      />

      <!-- Footer -->
      <div class="px-4 py-2.5 border-t border-[var(--color-border)] flex items-center gap-2.5">
        <button v-if="isEdit"
          class="px-3.5 py-2 bg-none border border-red-200 rounded-md text-sm text-red-600 cursor-pointer"
          @click="confirmDeleteVisible = true">Delete</button>
        <span class="flex-1" />
        <p v-if="error" class="text-xs text-red-600">{{ error }}</p>
        <button @click="save" :disabled="saving"
          class="px-5 py-2 bg-teal text-white border-none rounded-md text-sm font-medium cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed">
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
