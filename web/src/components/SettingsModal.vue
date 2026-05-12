<template>
  <div v-if="visible" class="fixed inset-0 bg-black/30 z-[100] flex items-center justify-center" @click.self="close">
    <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl w-[480px] flex flex-col shadow-xl max-h-[90vh] overflow-y-auto">

      <!-- Header -->
      <div class="flex justify-between items-center px-4 py-3 border-b border-[var(--color-border)] text-sm font-medium sticky top-0 bg-[var(--color-surface)] z-[1]">
        <span>Settings</span>
        <button @click="close" class="bg-none border-none text-lg cursor-pointer text-[var(--color-text-muted)]">×</button>
      </div>

      <!-- Body -->
      <div class="px-4 py-4 flex flex-col gap-3.5">
        <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
          Display name
          <input v-model="form.display_name" type="text" placeholder="Your Name"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal" />
        </label>
        <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
          Signature
          <textarea v-model="form.signature" placeholder="Your name&#10;your@email.com"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none resize-y min-h-[100px] leading-relaxed focus:border-teal" />
        </label>

        <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
          Poll interval
          <select v-model.number="form.poll_interval"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal">
            <option :value="0">Off (IMAP IDLE only)</option>
            <option :value="60">1 minute</option>
            <option :value="120">2 minutes</option>
            <option :value="300">5 minutes</option>
            <option :value="600">10 minutes</option>
          </select>
        </label>

        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">Notifications</div>
        <div class="flex items-center gap-2.5 text-sm">
          <span class="text-[var(--color-text)] flex-1">Desktop notifications</span>
          <template v-if="notifPermission === 'denied'">
            <span class="text-xs text-[var(--color-text-muted)]">Blocked — enable in browser settings</span>
          </template>
          <template v-else-if="settings.notificationsEnabled && notifPermission === 'granted'">
            <span class="text-xs text-teal font-medium">On ✓</span>
            <button @click="disableNotifications"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-teal hover:text-teal">Disable</button>
          </template>
          <template v-else>
            <button @click="enableNotifications"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-teal hover:text-teal">Enable</button>
          </template>
        </div>

        <div class="flex items-center gap-2.5">
          <span class="text-sm text-[var(--color-text)] flex-1">Event reminders</span>
          <select v-model.number="form.calendar_reminder_minutes"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal w-auto">
            <option :value="0">Off</option>
            <option :value="5">5 minutes before</option>
            <option :value="10">10 minutes before</option>
            <option :value="15">15 minutes before</option>
            <option :value="30">30 minutes before</option>
            <option :value="60">1 hour before</option>
          </select>
        </div>

        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">Identities (From: addresses)</div>
        <div class="flex flex-col gap-2">
          <div v-for="(id, i) in form.identities" :key="i" class="flex gap-1.5 items-center">
            <input v-model="id.name" type="text" placeholder="Name"
              class="flex-1 px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal" />
            <input v-model="id.email" type="email" placeholder="email@example.com"
              class="flex-[1.5] px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal" />
            <button @click="removeIdentity(i)"
              class="bg-none border-none text-base cursor-pointer text-[var(--color-text-muted)] px-1.5 py-1 shrink-0 rounded hover:bg-[var(--color-teal-light)]">×</button>
          </div>
          <button @click="addIdentity"
            class="bg-none border border-dashed border-[var(--color-border)] rounded-md px-3 py-1.5 text-xs cursor-pointer text-[var(--color-text-muted)] text-left hover:border-teal hover:text-teal">+ Add identity</button>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-4 py-3 border-t border-[var(--color-border)] flex items-center gap-4 sticky bottom-0 bg-[var(--color-surface)]">
        <button @click="save" :disabled="saving"
          class="px-5 py-1.5 bg-teal text-white border-none rounded-md text-sm font-medium cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed">
          {{ saving ? 'Saving…' : saved ? 'Saved ✓' : 'Save' }}
        </button>
        <p v-if="error" class="text-xs text-red-600">{{ error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useSettingsStore } from '../stores/settings'

const settings = useSettingsStore()
const visible = ref(false)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const notifPermission = ref(typeof Notification !== 'undefined' ? Notification.permission : 'denied')

const form = reactive({ display_name: '', signature: '', identities: [], poll_interval: 120, calendar_reminder_minutes: 30 })

async function open() {
  if (!settings.loaded) await settings.fetchSettings()
  form.display_name = settings.settings.display_name ?? ''
  form.signature = settings.settings.signature ?? ''
  form.identities = settings.identities.map(id => ({ ...id }))
  form.poll_interval = settings.pollInterval
  form.calendar_reminder_minutes = settings.reminderMinutes
  notifPermission.value = typeof Notification !== 'undefined' ? Notification.permission : 'denied'
  saved.value = false
  error.value = ''
  visible.value = true
}

function close() {
  visible.value = false
}

async function enableNotifications() {
  const result = await Notification.requestPermission()
  notifPermission.value = result
  if (result === 'granted') {
    await settings.saveSettings({ notifications_enabled: 'true' })
  }
}

async function disableNotifications() {
  await settings.saveSettings({ notifications_enabled: 'false' })
}

function addIdentity() {
  form.identities.push({ name: '', email: '' })
}

function removeIdentity(i) {
  form.identities.splice(i, 1)
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    const validIdentities = form.identities.filter(id => id.email.trim())
    await settings.saveSettings({
      display_name: form.display_name,
      signature: form.signature,
      identities: JSON.stringify(validIdentities),
      poll_interval: String(form.poll_interval),
      calendar_reminder_minutes: String(form.calendar_reminder_minutes),
    })
    saved.value = true
    setTimeout(() => { saved.value = false }, 2000)
  } catch {
    error.value = 'Could not save settings.'
  } finally {
    saving.value = false
  }
}

function onKeydown(e) { if (e.key === 'Escape' && visible.value) close() }
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

defineExpose({ open, close })
</script>
