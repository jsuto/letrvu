<template>
  <div class="address-input" @click="focusInput">
    <span
      v-for="(token, i) in tokens"
      :key="i"
      class="token"
    >
      {{ token }}
      <button type="button" class="token-remove" @click.stop="removeToken(i)">×</button>
    </span>
    <div class="input-wrap">
      <input
        ref="inputEl"
        v-model="inputVal"
        type="text"
        :placeholder="tokens.length === 0 ? placeholder : ''"
        @keydown.enter.prevent="commitInput"
        @keydown.tab.prevent="commitInput"
        @keydown.backspace="onBackspace"
        @keydown.comma.prevent="commitInput"
        @input="onInput"
        @blur="onBlur"
        autocomplete="off"
      />
      <ul v-if="suggestions.length" class="suggestions">
        <li
          v-for="s in suggestions"
          :key="s.contact_id + s.email"
          @mousedown.prevent="selectSuggestion(s)"
        >
          <span class="sug-name">{{ s.name || s.email }}</span>
          <span v-if="s.name" class="sug-email">{{ s.email }}</span>
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
  const label = s.name ? `${s.name} <${s.email}>` : s.email
  tokens.value.push(label)
  emitValue()
  inputVal.value = ''
  suggestions.value = []
}

function focusInput() {
  inputEl.value?.focus()
}
</script>

<style scoped>
.address-input {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px 16px;
  border-bottom: 0.5px solid var(--color-border);
  cursor: text;
  min-height: 36px;
  align-items: center;
}
.token {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--color-teal-light);
  border: 0.5px solid var(--color-teal);
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 12px;
}
.token-remove {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  color: var(--color-text-muted);
  padding: 0;
}
.input-wrap {
  position: relative;
  flex: 1;
  min-width: 80px;
}
input {
  width: 100%;
  border: none;
  outline: none;
  font-size: 13px;
  background: transparent;
  padding: 2px 0;
}
.suggestions {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  list-style: none;
  margin: 0;
  padding: 4px 0;
  z-index: 100;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}
.suggestions li {
  padding: 6px 12px;
  cursor: pointer;
  font-size: 13px;
  display: flex;
  gap: 8px;
  align-items: center;
}
.suggestions li:hover { background: var(--color-teal-light); }
.sug-name { font-weight: 500; }
.sug-email { color: var(--color-text-muted); font-size: 12px; }
</style>
