<template>
  <div class="flex flex-col flex-1 overflow-hidden">
    <!-- Day-of-week headers -->
    <div class="grid grid-cols-7 border-b border-[var(--color-border)]">
      <span
        v-for="d in ['Mon','Tue','Wed','Thu','Fri','Sat','Sun']"
        :key="d"
        class="py-1.5 text-center text-[11px] font-medium text-[var(--color-text-muted)]"
      >{{ d }}</span>
    </div>
    <!-- 6-week grid -->
    <div class="grid grid-cols-7 flex-1 overflow-hidden" style="grid-template-rows: repeat(6, 1fr)">
      <div
        v-for="cell in cells"
        :key="cell.iso"
        :class="[
          'border-r border-b border-[var(--color-border)] p-1 overflow-hidden cursor-pointer min-h-0 hover:bg-[var(--color-teal-light)]',
          !cell.inMonth ? 'opacity-40' : '',
        ]"
        @click="emit('day-click', cell.date)"
      >
        <span
          :class="[
            'text-xs font-medium mb-0.5 inline-flex w-[22px] h-[22px] items-center justify-center',
            cell.isToday ? 'bg-teal text-white rounded-full' : '',
          ]"
        >{{ cell.day }}</span>
        <div class="flex flex-col gap-px overflow-hidden">
          <div
            v-for="ev in cell.events"
            :key="ev.id"
            class="bg-teal text-white rounded-sm px-1 py-px text-[11px] whitespace-nowrap overflow-hidden text-ellipsis cursor-pointer hover:opacity-85"
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
