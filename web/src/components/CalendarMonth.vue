<template>
  <div class="month-view">
    <!-- Day-of-week headers -->
    <div class="dow-header">
      <span v-for="d in ['Mon','Tue','Wed','Thu','Fri','Sat','Sun']" :key="d">{{ d }}</span>
    </div>
    <!-- 6-week grid -->
    <div class="grid">
      <div
        v-for="cell in cells"
        :key="cell.iso"
        class="cell"
        :class="{
          'other-month': !cell.inMonth,
          'today': cell.isToday,
        }"
        @click="emit('day-click', cell.date)"
      >
        <span class="day-num">{{ cell.day }}</span>
        <div class="events">
          <div
            v-for="ev in cell.events"
            :key="ev.id"
            class="event-chip"
            :title="ev.title"
            @click.stop="emit('event-click', ev)"
          >
            {{ ev.all_day ? '' : formatTime(ev.starts_at) + ' ' }}{{ ev.title }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  year: { type: Number, required: true },
  month: { type: Number, required: true }, // 0-indexed
  events: { type: Array, default: () => [] },
})
const emit = defineEmits(['day-click', 'event-click'])

const today = new Date()
today.setHours(0, 0, 0, 0)

const cells = computed(() => {
  // First day of the month
  const first = new Date(props.year, props.month, 1)
  // Start from Monday of that week (ISO week)
  const startDow = (first.getDay() + 6) % 7 // 0=Mon
  const start = new Date(first)
  start.setDate(start.getDate() - startDow)

  const result = []
  for (let i = 0; i < 42; i++) {
    const date = new Date(start)
    date.setDate(start.getDate() + i)
    const iso = date.toISOString().slice(0, 10)
    result.push({
      date,
      iso,
      day: date.getDate(),
      inMonth: date.getMonth() === props.month,
      isToday: date.getTime() === today.getTime(),
      events: eventsForDay(date),
    })
  }
  return result
})

function eventsForDay(date) {
  const iso = date.toISOString().slice(0, 10)
  return props.events.filter(ev => {
    const s = ev.starts_at.slice(0, 10)
    const e = ev.ends_at.slice(0, 10)
    return iso >= s && iso <= e
  })
}

function formatTime(iso) {
  const d = new Date(iso)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.month-view { display: flex; flex-direction: column; flex: 1; overflow: hidden; }
.dow-header {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  border-bottom: 0.5px solid var(--color-border);
}
.dow-header span {
  padding: 6px;
  text-align: center;
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text-muted);
}
.grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  grid-template-rows: repeat(6, 1fr);
  flex: 1;
  overflow: hidden;
}
.cell {
  border-right: 0.5px solid var(--color-border);
  border-bottom: 0.5px solid var(--color-border);
  padding: 4px;
  overflow: hidden;
  cursor: pointer;
  min-height: 0;
}
.cell:hover { background: var(--color-teal-light); }
.cell.other-month { opacity: 0.4; }
.cell.today .day-num {
  background: var(--color-teal);
  color: white;
  border-radius: 50%;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.day-num {
  font-size: 12px;
  font-weight: 500;
  margin-bottom: 2px;
  display: inline-flex;
  width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
}
.events { display: flex; flex-direction: column; gap: 1px; overflow: hidden; }
.event-chip {
  background: var(--color-teal);
  color: white;
  border-radius: 3px;
  padding: 1px 4px;
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
}
.event-chip:hover { opacity: 0.85; }
</style>
