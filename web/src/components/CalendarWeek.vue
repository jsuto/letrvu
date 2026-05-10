<template>
  <div class="week-view">
    <!-- Header row: day columns -->
    <div class="week-header">
      <div class="time-gutter" />
      <div
        v-for="d in days"
        :key="d.iso"
        class="day-col-header"
        :class="{ today: d.isToday }"
      >
        <span class="dow">{{ d.dow }}</span>
        <span class="day-num" :class="{ 'today-num': d.isToday }">{{ d.day }}</span>
      </div>
    </div>

    <!-- Scrollable time grid -->
    <div class="time-grid" ref="gridEl">
      <!-- All-day events row -->
      <div class="allday-row">
        <div class="time-gutter allday-label">all-day</div>
        <div v-for="d in days" :key="d.iso" class="allday-col">
          <div
            v-for="ev in d.allDayEvents"
            :key="ev.id"
            class="event-chip allday-chip"
            @click="emit('event-click', ev)"
          >{{ ev.title }}</div>
        </div>
      </div>

      <!-- Hourly rows -->
      <div class="hours">
        <div class="time-col">
          <div v-for="h in 24" :key="h" class="hour-label">
            {{ formatHour(h - 1) }}
          </div>
        </div>
        <div class="day-cols">
          <div
            v-for="d in days"
            :key="d.iso"
            class="day-col"
            :class="{ today: d.isToday }"
            @click="e => onColClick(e, d.date)"
          >
            <div class="hour-lines">
              <div v-for="h in 24" :key="h" class="hour-line" />
            </div>
            <div
              v-for="ev in d.timedEvents"
              :key="ev.id"
              class="event-block"
              :style="eventStyle(ev)"
              @click.stop="emit('event-click', ev)"
            >
              <span class="ev-time">{{ formatTime(ev.starts_at) }}</span>
              {{ ev.title }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'

const props = defineProps({
  weekStart: { type: Date, required: true }, // Monday of the displayed week
  events: { type: Array, default: () => [] },
})
const emit = defineEmits(['event-click', 'day-click'])

const gridEl = ref(null)
const today = new Date()
today.setHours(0, 0, 0, 0)

onMounted(() => {
  // Scroll to 7am on mount
  if (gridEl.value) gridEl.value.scrollTop = 7 * 56
})

const days = computed(() => {
  return Array.from({ length: 7 }, (_, i) => {
    const date = new Date(props.weekStart)
    date.setDate(date.getDate() + i)
    date.setHours(0, 0, 0, 0)
    const iso = date.toISOString().slice(0, 10)
    const isToday = date.getTime() === today.getTime()
    return {
      date,
      iso,
      dow: ['Mon','Tue','Wed','Thu','Fri','Sat','Sun'][i],
      day: date.getDate(),
      isToday,
      allDayEvents: props.events.filter(ev => ev.all_day && ev.starts_at.slice(0,10) <= iso && ev.ends_at.slice(0,10) >= iso),
      timedEvents: props.events.filter(ev => !ev.all_day && ev.starts_at.slice(0,10) === iso),
    }
  })
})

function eventStyle(ev) {
  const s = new Date(ev.starts_at)
  const e = new Date(ev.ends_at)
  const startMin = s.getHours() * 60 + s.getMinutes()
  const endMin = e.getHours() * 60 + e.getMinutes()
  const top = (startMin / 60) * 56
  const height = Math.max(((endMin - startMin) / 60) * 56, 20)
  return { top: `${top}px`, height: `${height}px` }
}

function formatHour(h) {
  if (h === 0) return ''
  return h < 12 ? `${h} am` : h === 12 ? '12 pm' : `${h - 12} pm`
}

function formatTime(iso) {
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function onColClick(e, date) {
  const col = e.currentTarget.getBoundingClientRect()
  const relY = e.clientY - col.top + gridEl.value.scrollTop - 56 // subtract allday row
  const hour = Math.floor(relY / 56)
  const d = new Date(date)
  d.setHours(Math.max(0, Math.min(23, hour)), 0, 0, 0)
  emit('day-click', d)
}
</script>

<style scoped>
.week-view { display: flex; flex-direction: column; flex: 1; overflow: hidden; }
.week-header {
  display: grid;
  grid-template-columns: 52px repeat(7, 1fr);
  border-bottom: 0.5px solid var(--color-border);
  flex-shrink: 0;
}
.time-gutter { width: 52px; flex-shrink: 0; }
.day-col-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 6px 0;
  font-size: 12px;
  color: var(--color-text-muted);
}
.day-col-header.today { color: var(--color-teal); }
.dow { font-weight: 500; }
.day-num {
  width: 26px; height: 26px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 50%;
  font-size: 13px;
}
.today-num { background: var(--color-teal); color: white; }

.time-grid { flex: 1; overflow-y: auto; }

.allday-row {
  display: grid;
  grid-template-columns: 52px repeat(7, 1fr);
  border-bottom: 0.5px solid var(--color-border);
  min-height: 32px;
}
.allday-label {
  font-size: 10px;
  color: var(--color-text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
}
.allday-col { padding: 2px; display: flex; flex-direction: column; gap: 1px; }
.allday-chip { font-size: 11px; }

.hours { display: flex; }
.time-col { width: 52px; flex-shrink: 0; }
.hour-label {
  height: 56px;
  font-size: 10px;
  color: var(--color-text-muted);
  text-align: right;
  padding-right: 6px;
  padding-top: 2px;
  border-top: 0.5px solid var(--color-border);
  box-sizing: border-box;
}
.day-cols { display: grid; grid-template-columns: repeat(7, 1fr); flex: 1; }
.day-col {
  position: relative;
  border-left: 0.5px solid var(--color-border);
  cursor: pointer;
}
.day-col.today { background: color-mix(in srgb, var(--color-teal) 4%, transparent); }
.hour-lines { position: absolute; inset: 0; pointer-events: none; }
.hour-line { height: 56px; border-top: 0.5px solid var(--color-border); box-sizing: border-box; }
.event-block {
  position: absolute;
  left: 2px; right: 2px;
  background: var(--color-teal);
  color: white;
  border-radius: 4px;
  font-size: 11px;
  padding: 2px 4px;
  overflow: hidden;
  cursor: pointer;
  box-sizing: border-box;
}
.event-block:hover { opacity: 0.85; }
.ev-time { display: block; font-weight: 600; }
.event-chip {
  background: var(--color-teal);
  color: white;
  border-radius: 3px;
  padding: 1px 5px;
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
}
</style>
