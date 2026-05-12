<template>
  <div class="flex flex-col flex-1 overflow-hidden">
    <!-- Header row: day columns -->
    <div class="grid border-b border-[var(--color-border)] shrink-0" style="grid-template-columns: 52px repeat(7, 1fr)">
      <div class="w-[52px] shrink-0" />
      <div
        v-for="d in days"
        :key="d.iso"
        :class="['flex flex-col items-center py-1.5 text-xs', d.isToday ? 'text-teal' : 'text-[var(--color-text-muted)]']"
      >
        <span class="font-medium">{{ d.dow }}</span>
        <span
          :class="[
            'w-[26px] h-[26px] flex items-center justify-center rounded-full text-[13px]',
            d.isToday ? 'bg-teal text-white' : '',
          ]"
        >{{ d.day }}</span>
      </div>
    </div>

    <!-- Scrollable time grid -->
    <div class="flex-1 overflow-y-auto" ref="gridEl">
      <!-- All-day events row -->
      <div class="grid border-b border-[var(--color-border)] min-h-8" style="grid-template-columns: 52px repeat(7, 1fr)">
        <div class="w-[52px] text-[10px] text-[var(--color-text-muted)] flex items-center justify-center">all-day</div>
        <div v-for="d in days" :key="d.iso" class="p-0.5 flex flex-col gap-px">
          <div
            v-for="ev in d.allDayEvents"
            :key="ev.id"
            class="bg-teal text-white rounded-sm px-1 py-px text-[11px] whitespace-nowrap overflow-hidden text-ellipsis cursor-pointer"
            @click="emit('event-click', ev)"
          >{{ ev.title }}</div>
        </div>
      </div>

      <!-- Hourly rows -->
      <div class="flex">
        <div class="w-[52px] shrink-0">
          <div
            v-for="h in 24"
            :key="h"
            class="h-14 text-[10px] text-[var(--color-text-muted)] text-right pr-1.5 pt-0.5 border-t border-[var(--color-border)] box-border"
          >{{ formatHour(h - 1) }}</div>
        </div>
        <div class="grid flex-1" style="grid-template-columns: repeat(7, 1fr)">
          <div
            v-for="d in days"
            :key="d.iso"
            :class="['relative border-l border-[var(--color-border)] cursor-pointer', d.isToday ? 'bg-teal/[0.04]' : '']"
            @click="e => onColClick(e, d.date)"
          >
            <div class="absolute inset-0 pointer-events-none">
              <div v-for="h in 24" :key="h" class="h-14 border-t border-[var(--color-border)] box-border" />
            </div>
            <div
              v-for="ev in d.timedEvents"
              :key="ev.id"
              class="absolute left-0.5 right-0.5 bg-teal text-white rounded px-1 py-0.5 text-[11px] overflow-hidden cursor-pointer box-border hover:opacity-85"
              :style="eventStyle(ev)"
              @click.stop="emit('event-click', ev)"
            >
              <span class="block font-semibold">{{ formatTime(ev.starts_at) }}</span>
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
