<template>
  <div class="grid h-screen overflow-hidden bg-[var(--color-bg)]" style="grid-template-columns: 200px 1fr">
    <aside class="border-r border-[var(--color-border)] overflow-y-auto">
      <FolderList />
    </aside>

    <div class="flex flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="flex items-center justify-between px-4 py-2.5 border-b border-[var(--color-border)] shrink-0 gap-2">
        <div class="flex items-center gap-2">
          <button class="bg-none border border-[var(--color-border)] rounded-md text-base cursor-pointer px-2.5 py-0.5 text-[var(--color-text)] hover:bg-[var(--color-teal-light)]" @click="prev">‹</button>
          <button class="bg-none border border-[var(--color-border)] rounded-md text-base cursor-pointer px-2.5 py-0.5 text-[var(--color-text)] hover:bg-[var(--color-teal-light)]" @click="next">›</button>
          <button class="px-3 py-1.5 text-xs border border-[var(--color-border)] rounded-md cursor-pointer bg-[var(--color-surface)] text-[var(--color-text)] hover:bg-[var(--color-teal-light)]" @click="goToday">Today</button>
          <span class="text-sm font-medium">{{ periodLabel }}</span>
        </div>
        <div class="flex items-center gap-2">
          <label class="px-2.5 py-1.5 text-xs border border-[var(--color-border)] rounded-md cursor-pointer bg-[var(--color-surface)] text-[var(--color-text)] hover:bg-[var(--color-teal-light)]" title="Import .ics">
            Import
            <input type="file" accept=".ics" @change="importIcs" hidden />
          </label>
          <a href="/api/calendar/events/export" download="calendar.ics"
            class="px-2.5 py-1.5 text-xs border border-[var(--color-border)] rounded-md cursor-pointer bg-[var(--color-surface)] text-[var(--color-text)] no-underline hover:bg-[var(--color-teal-light)]">Export</a>
          <div class="flex border border-[var(--color-border)] rounded-md overflow-hidden">
            <button
              :class="['px-3 py-1.5 text-xs border-none cursor-pointer', view === 'month' ? 'bg-teal text-white' : 'bg-[var(--color-surface)] text-[var(--color-text)]']"
              @click="view = 'month'"
            >Month</button>
            <button
              :class="['px-3 py-1.5 text-xs border-none cursor-pointer', view === 'week' ? 'bg-teal text-white' : 'bg-[var(--color-surface)] text-[var(--color-text)]']"
              @click="view = 'week'"
            >Week</button>
          </div>
          <button class="px-3 py-1.5 text-xs bg-teal text-white border-none rounded-md cursor-pointer" @click="eventModal?.open(null, new Date())">+ New</button>
        </div>
      </div>

      <!-- Views -->
      <CalendarMonth
        v-if="view === 'month'"
        :year="cursor.year"
        :month="cursor.month"
        :events="cal.events"
        @day-click="d => eventModal?.open(null, d)"
        @event-click="ev => eventModal?.open(ev)"
      />
      <CalendarWeek
        v-else
        :week-start="weekStart"
        :events="cal.events"
        @day-click="d => eventModal?.open(null, d)"
        @event-click="ev => eventModal?.open(ev)"
      />
    </div>
  </div>

  <EventModal ref="eventModal" @saved="onSaved" @deleted="onSaved" />
  <ComposeModal ref="composeModal" />
</template>

<script setup>
import { ref, reactive, computed, provide, onMounted, watch } from 'vue'
import FolderList from '../components/FolderList.vue'
import CalendarMonth from '../components/CalendarMonth.vue'
import CalendarWeek from '../components/CalendarWeek.vue'
import EventModal from '../components/EventModal.vue'
import ComposeModal from '../components/ComposeModal.vue'
import { useCalendarStore } from '../stores/calendar'
import { apiFetch } from '../api'

const cal = useCalendarStore()
const eventModal = ref(null)
const composeModal = ref(null)
provide('compose', composeModal)

const view = ref('month')

// cursor tracks current month (month view) and current week (week view)
const now = new Date()
const cursor = reactive({ year: now.getFullYear(), month: now.getMonth() })
const weekCursor = ref(mondayOf(now))

const periodLabel = computed(() => {
  if (view.value === 'month') {
    return new Date(cursor.year, cursor.month, 1)
      .toLocaleString('default', { month: 'long', year: 'numeric' })
  }
  const end = new Date(weekCursor.value)
  end.setDate(end.getDate() + 6)
  return `${fmt(weekCursor.value)} – ${fmt(end)}`
})

const weekStart = computed(() => weekCursor.value)

function prev() {
  if (view.value === 'month') {
    if (cursor.month === 0) { cursor.year--; cursor.month = 11 }
    else cursor.month--
  } else {
    const d = new Date(weekCursor.value)
    d.setDate(d.getDate() - 7)
    weekCursor.value = d
  }
}

function next() {
  if (view.value === 'month') {
    if (cursor.month === 11) { cursor.year++; cursor.month = 0 }
    else cursor.month++
  } else {
    const d = new Date(weekCursor.value)
    d.setDate(d.getDate() + 7)
    weekCursor.value = d
  }
}

function goToday() {
  const n = new Date()
  cursor.year = n.getFullYear()
  cursor.month = n.getMonth()
  weekCursor.value = mondayOf(n)
}

function fetchVisible() {
  let from, to
  if (view.value === 'month') {
    from = new Date(cursor.year, cursor.month, 1)
    to = new Date(cursor.year, cursor.month + 1, 1)
  } else {
    from = new Date(weekCursor.value)
    to = new Date(weekCursor.value)
    to.setDate(to.getDate() + 7)
  }
  cal.fetchEvents(from, to)
}

onMounted(fetchVisible)
watch([view, () => cursor.year, () => cursor.month, weekCursor], fetchVisible)

function onSaved() {
  fetchVisible()
}

async function importIcs(e) {
  const file = e.target.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  const res = await apiFetch('/api/calendar/events/import', { method: 'POST', body: fd })
  if (res.ok) {
    const { imported } = await res.json()
    alert(`Imported ${imported} event(s).`)
    fetchVisible()
  } else {
    alert('Import failed.')
  }
  e.target.value = ''
}

function mondayOf(date) {
  const d = new Date(date)
  d.setHours(0, 0, 0, 0)
  const dow = (d.getDay() + 6) % 7 // 0=Mon
  d.setDate(d.getDate() - dow)
  return d
}

function fmt(d) {
  return d.toLocaleString('default', { month: 'short', day: 'numeric' })
}
</script>
