<template>
  <div
    class="flex flex-wrap gap-1 px-4 py-1 border-b border-[var(--color-border)] cursor-text min-h-9 items-center"
    @click="focusInput"
  >
    <span
      v-for="(token, i) in tokens"
      :key="i"
      class="inline-flex items-center gap-1 bg-[var(--color-teal-light)] border border-teal rounded px-1.5 py-0.5 text-xs"
    >
      {{ token }}
      <button type="button" class="bg-none border-none cursor-pointer text-sm leading-none text-[var(--color-text-muted)] p-0" @click.stop="removeToken(i)">×</button>
    </span>
    <div class="relative flex-1 min-w-[80px]">
      <input
        ref="inputEl"
        v-model="inputVal"
        type="text"
        :placeholder="tokens.length === 0 ? placeholder : ''"
        class="w-full border-none outline-none text-sm bg-transparent py-0.5"
        @keydown.enter.prevent="commitInput"
        @keydown.tab.prevent="commitInput"
        @keydown.backspace="onBackspace"
        @keydown.comma.prevent="commitInput"
        @input="onInput"
        @blur="onBlur"
        autocomplete="off"
      />
      <ul v-if="suggestions.length" class="absolute top-[calc(100%+4px)] left-0 right-0 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-md list-none m-0 py-1 z-[100] shadow-lg">
        <li
          v-for="s in suggestions"
          :key="(s.type === 'group' ? 'g' + s.group_id : 'c' + s.contact_id) + s.email"
          class="px-3 py-1.5 cursor-pointer text-sm flex gap-2 items-center hover:bg-[var(--color-teal-light)]"
          @mousedown.prevent="selectSuggestion(s)"
        >
          <span v-if="s.type === 'group'" class="text-[10px] font-semibold bg-[var(--color-teal-light)] border border-teal rounded px-1 py-px text-teal shrink-0">Group</span>
          <span class="font-medium">{{ s.name }}</span>
          <span v-if="s.type === 'contact' && s.name" class="text-[var(--color-text-muted)] text-xs flex-1 overflow-hidden text-ellipsis whitespace-nowrap">{{ s.email }}</span>
          <span v-if="s.type === 'group'" class="text-[var(--color-text-muted)] text-xs flex-1 overflow-hidden text-ellipsis whitespace-nowrap">{{ (s.emails || []).slice(0, 3).join(', ') }}{{ (s.emails || []).length > 3 ? '…' : '' }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useContactsStore } from '../stores/contacts'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const contacts = useContactsStore()
const inputEl = ref(null)
const inputVal = ref('')
const tokens = ref(props.modelValue ? props.modelValue.split(',').map(s => s.trim()).filter(Boolean) : [])
const suggestions = ref([])
let debounceTimer = null

watch(() => props.modelValue, val => {
  tokens.value = val ? val.split(',').map(s => s.trim()).filter(Boolean) : []
})

function emitValue() {
  emit('update:modelValue', tokens.value.join(', '))
}

function commitInput() {
  const val = inputVal.value.trim()
  if (val) {
    tokens.value.push(val)
    emitValue()
  }
  inputVal.value = ''
  suggestions.value = []
}

function removeToken(i) {
  tokens.value.splice(i, 1)
  emitValue()
}

function onBackspace() {
  if (inputVal.value === '' && tokens.value.length) {
    tokens.value.pop()
    emitValue()
  }
}

function onBlur() {
  setTimeout(() => {
    commitInput()
    suggestions.value = []
  }, 150)
}

function onInput() {
  clearTimeout(debounceTimer)
  const q = inputVal.value.trim()
  if (q.length < 1) {
    suggestions.value = []
    return
  }
  debounceTimer = setTimeout(async () => {
    suggestions.value = await contacts.autocomplete(q)
  }, 200)
}

function selectSuggestion(s) {
  if (s.type === 'group') {
    // Expand all group members as individual tokens.
    for (const email of s.emails || []) {
      tokens.value.push(email)
    }
  } else {
    const label = s.name ? `${s.name} <${s.email}>` : s.email
    tokens.value.push(label)
  }
  emitValue()
  inputVal.value = ''
  suggestions.value = []
}

function focusInput() {
  inputEl.value?.focus()
}
</script>
