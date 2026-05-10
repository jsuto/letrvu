<template>
  <div class="calendar-layout">
    <aside class="sidebar">
      <FolderList />
    </aside>

    <div class="calendar-panel">
      <!-- Toolbar -->
      <div class="toolbar">
        <div class="toolbar-left">
          <button class="nav-btn" @click="prev">‹</button>
          <button class="nav-btn" @click="next">›</button>
          <button class="today-btn" @click="goToday">Today</button>
          <span class="period-label">{{ periodLabel }}</span>
        </div>
        <div class="toolbar-right">
          <label class="import-btn" title="Import .ics">
            Import
            <input type="file" accept=".ics" @change="importIcs" hidden />
          </label>
          <a href="/api/calendar/events/export" download="calendar.ics" class="export-btn">Export</a>
          <div class="view-toggle">
            <button :class="{ active: view === 'month' }" @click="view = 'month'">Month</button>
            <button :class="{ active: view === 'week' }" @click="view = 'week'">Week</button>
          </div>
          <button class="new-btn" @click="eventModal?.open(null, new Date())">+ New</button>
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

<style scoped>
.calendar-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  height: 100vh;
  overflow: hidden;
  background: var(--color-bg);
}
.sidebar {
  border-right: 0.5px solid var(--color-border);
  overflow-y: auto;
}
.calendar-panel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 0.5px solid var(--color-border);
  flex-shrink: 0;
  gap: 8px;
}
.toolbar-left, .toolbar-right { display: flex; align-items: center; gap: 8px; }
.nav-btn {
  background: none;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 16px;
  cursor: pointer;
  padding: 2px 10px;
  color: var(--color-text);
}
.nav-btn:hover { background: var(--color-teal-light); }
.today-btn {
  padding: 5px 12px;
  font-size: 12px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  background: var(--color-surface);
  color: var(--color-text);
}
.today-btn:hover { background: var(--color-teal-light); }
.period-label { font-size: 14px; font-weight: 500; }
.view-toggle {
  display: flex;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  overflow: hidden;
}
.view-toggle button {
  padding: 5px 12px;
  font-size: 12px;
  border: none;
  background: var(--color-surface);
  cursor: pointer;
  color: var(--color-text);
}
.view-toggle button.active {
  background: var(--color-teal);
  color: white;
}
.new-btn {
  padding: 5px 12px;
  font-size: 12px;
  background: var(--color-teal);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}
.import-btn, .export-btn {
  padding: 5px 10px;
  font-size: 12px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  background: var(--color-surface);
  color: var(--color-text);
  text-decoration: none;
}
</style>
