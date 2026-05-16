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

        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">Security</div>
        <div class="flex flex-col gap-2">
          <div v-if="sessionsLoading" class="text-xs text-[var(--color-text-muted)]">Loading sessions…</div>
          <div v-else-if="sessions.length" class="flex flex-col gap-1.5">
            <div v-for="s in sessions" :key="s.id"
              class="flex items-start gap-2 px-2.5 py-2 rounded-md bg-[var(--color-bg)] border border-[var(--color-border)]">
              <div class="flex-1 min-w-0">
                <div class="text-xs font-medium text-[var(--color-text)] truncate">{{ browserName(s.user_agent) }}<span v-if="s.current" class="ml-1.5 text-[10px] text-teal font-semibold">(this device)</span></div>
                <div class="text-[10px] text-[var(--color-text-muted)] mt-0.5">
                  Signed in {{ formatDate(s.created_at) }} · Last seen {{ formatDate(s.last_activity_at) }}
                </div>
              </div>
            </div>
          </div>
          <div class="flex gap-2 flex-wrap">
            <button @click="logoutOtherDevices" :disabled="logoutAllBusy || sessions.filter(s => !s.current).length === 0"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-red-500 hover:text-red-600 disabled:opacity-40 disabled:cursor-not-allowed">
              Logout other devices
            </button>
            <button @click="logoutEverywhere" :disabled="logoutAllBusy"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-red-500 hover:text-red-600 disabled:opacity-40 disabled:cursor-not-allowed">
              Logout everywhere
            </button>
          </div>
          <p v-if="logoutAllError" class="text-xs text-red-600">{{ logoutAllError }}</p>
        </div>

        <!-- PGP -->
        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">PGP Key</div>
        <div class="flex flex-col gap-2">

          <!-- No key stored -->
          <template v-if="!pgp.hasKey">
            <p class="text-xs text-[var(--color-text-muted)]">No PGP key configured. Generate a new key pair or import an existing one.</p>
            <div class="flex gap-2 flex-wrap">
              <button @click="pgpMode = 'generate'"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:border-teal hover:text-teal">Generate key</button>
              <button @click="pgpMode = 'import'"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:border-teal hover:text-teal">Import key</button>
            </div>

            <!-- Generate form -->
            <div v-if="pgpMode === 'generate'" class="flex flex-col gap-2 p-3 bg-[var(--color-bg)] rounded-md border border-[var(--color-border)]">
              <p class="text-xs font-medium text-[var(--color-text)]">Generate new ECC (Curve25519) key pair</p>
              <input v-model="pgpForm.name" type="text" placeholder="Your name"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <input v-model="pgpForm.email" type="email" placeholder="your@email.com"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <input v-model="pgpForm.passphrase" type="password" placeholder="Passphrase (used to protect the key)"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <input v-model="pgpForm.passphrase2" type="password" placeholder="Confirm passphrase"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <p v-if="pgpError" class="text-xs text-red-600">{{ pgpError }}</p>
              <div class="flex gap-2">
                <button @click="generatePGPKey" :disabled="pgpBusy"
                  class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-60">{{ pgpBusy ? 'Generating…' : 'Generate' }}</button>
                <button @click="pgpMode = null" class="px-3 py-1.5 border border-[var(--color-border)] rounded-md text-xs cursor-pointer">Cancel</button>
              </div>
            </div>

            <!-- Import form -->
            <div v-if="pgpMode === 'import'" class="flex flex-col gap-2 p-3 bg-[var(--color-bg)] rounded-md border border-[var(--color-border)]">
              <p class="text-xs font-medium text-[var(--color-text)]">Import existing private key</p>
              <textarea v-model="pgpForm.armoredKey" rows="5" placeholder="-----BEGIN PGP PRIVATE KEY BLOCK-----"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-xs font-mono outline-none bg-[var(--color-surface)] resize-y focus:border-teal" />
              <input v-model="pgpForm.passphrase" type="password" placeholder="Passphrase"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <p v-if="pgpError" class="text-xs text-red-600">{{ pgpError }}</p>
              <div class="flex gap-2">
                <button @click="importPGPKey" :disabled="pgpBusy"
                  class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-60">{{ pgpBusy ? 'Importing…' : 'Import' }}</button>
                <button @click="pgpMode = null" class="px-3 py-1.5 border border-[var(--color-border)] rounded-md text-xs cursor-pointer">Cancel</button>
              </div>
            </div>
          </template>

          <!-- Key stored but locked -->
          <template v-else-if="pgp.isLocked">
            <p class="text-xs text-[var(--color-text-muted)]">Key stored on server. Enter your passphrase to unlock for this session.</p>
            <div class="flex gap-2">
              <input v-model="pgpForm.passphrase" type="password" placeholder="Passphrase"
                class="flex-1 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal"
                @keydown.enter="unlockPGPKey" />
              <button @click="unlockPGPKey" :disabled="pgpBusy"
                class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-60">{{ pgpBusy ? 'Unlocking…' : 'Unlock' }}</button>
            </div>
            <p v-if="pgpError" class="text-xs text-red-600">{{ pgpError }}</p>
            <button @click="confirmDeletePGPKey" class="self-start text-xs text-red-600 bg-none border-none cursor-pointer p-0 hover:underline">Delete key…</button>
          </template>

          <!-- Key unlocked -->
          <template v-else-if="pgp.isUnlocked">
            <div class="px-3 py-2.5 bg-[var(--color-bg)] rounded-md border border-[var(--color-border)] flex flex-col gap-1">
              <p class="text-xs font-medium text-[var(--color-text)]">🔑 Key unlocked</p>
              <p class="text-[11px] text-[var(--color-text-muted)] font-mono break-all">{{ pgp.fingerprint }}</p>
              <p v-if="pgp.userId" class="text-xs text-[var(--color-text-muted)]">{{ pgp.userId }}</p>
            </div>
            <div class="flex gap-2 flex-wrap">
              <button @click="exportPublicKey"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:border-teal hover:text-teal">Export public key</button>
              <button @click="exportPrivateKey"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:border-teal hover:text-teal">Export private key</button>
              <button @click="pgp.lock()"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:bg-[var(--color-bg)]">Lock</button>
              <button @click="confirmDeletePGPKey"
                class="px-3 py-1.5 border border-red-200 rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-red-600 hover:bg-[var(--color-bg)]">Delete key</button>
            </div>
          </template>

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
import { useAuthStore } from '../stores/auth'
import { usePGPStore } from '../stores/pgp'
import { apiFetch } from '../api'

const settings = useSettingsStore()
const auth = useAuthStore()
const pgp = usePGPStore()
const visible = ref(false)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const notifPermission = ref(typeof Notification !== 'undefined' ? Notification.permission : 'denied')

const sessions = ref([])
const sessionsLoading = ref(false)
const logoutAllBusy = ref(false)
const logoutAllError = ref('')

// PGP state
const pgpMode = ref(null)  // null | 'generate' | 'import'
const pgpBusy = ref(false)
const pgpError = ref('')
const pgpForm = reactive({ name: '', email: '', passphrase: '', passphrase2: '', armoredKey: '' })

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
  logoutAllError.value = ''
  pgpMode.value = null
  pgpError.value = ''
  pgpForm.passphrase = ''
  pgpForm.passphrase2 = ''
  // Pre-fill generate form with account info
  pgpForm.name = settings.settings.display_name ?? ''
  pgpForm.email = auth.user?.username ?? ''
  visible.value = true
  fetchSessions()
  pgp.fetchKey()
}

async function fetchSessions() {
  sessionsLoading.value = true
  try {
    const res = await apiFetch('/api/auth/sessions')
    if (res.ok) sessions.value = await res.json()
  } finally {
    sessionsLoading.value = false
  }
}

async function logoutOtherDevices() {
  logoutAllBusy.value = true
  logoutAllError.value = ''
  try {
    const res = await apiFetch('/api/auth/sessions', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ include_current: false }),
    })
    if (!res.ok) throw new Error()
    await fetchSessions()
  } catch {
    logoutAllError.value = 'Could not logout other devices.'
  } finally {
    logoutAllBusy.value = false
  }
}

async function logoutEverywhere() {
  logoutAllBusy.value = true
  logoutAllError.value = ''
  try {
    const res = await apiFetch('/api/auth/sessions', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ include_current: true }),
    })
    if (!res.ok) throw new Error()
    await auth.logout()
  } catch {
    logoutAllError.value = 'Could not logout everywhere.'
    logoutAllBusy.value = false
  }
}

function browserName(ua) {
  if (!ua) return 'Unknown browser'
  if (/Edg\//.test(ua)) return 'Microsoft Edge'
  if (/Firefox\//.test(ua)) return 'Firefox'
  if (/Chrome\//.test(ua)) return 'Chrome'
  if (/Safari\//.test(ua)) return 'Safari'
  return ua.length > 60 ? ua.slice(0, 60) + '…' : ua
}

function formatDate(iso) {
  if (!iso) return 'never'
  const d = new Date(iso)
  const now = new Date()
  const diffMs = now - d
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return 'just now'
  if (diffMin < 60) return `${diffMin}m ago`
  const diffH = Math.floor(diffMin / 60)
  if (diffH < 24) return `${diffH}h ago`
  const diffD = Math.floor(diffH / 24)
  if (diffD < 7) return `${diffD}d ago`
  return d.toLocaleDateString()
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

// ── PGP helpers ───────────────────────────────────────────────────────────

async function generatePGPKey() {
  pgpError.value = ''
  if (!pgpForm.name.trim() || !pgpForm.email.trim()) { pgpError.value = 'Name and email are required.'; return }
  if (!pgpForm.passphrase) { pgpError.value = 'Passphrase is required.'; return }
  if (pgpForm.passphrase !== pgpForm.passphrase2) { pgpError.value = 'Passphrases do not match.'; return }
  pgpBusy.value = true
  try {
    await pgp.generateKey(pgpForm.name.trim(), pgpForm.email.trim(), pgpForm.passphrase)
    pgpMode.value = null
    pgpForm.passphrase = ''
    pgpForm.passphrase2 = ''
  } catch (e) {
    pgpError.value = e.message || 'Key generation failed.'
  } finally {
    pgpBusy.value = false
  }
}

async function importPGPKey() {
  pgpError.value = ''
  if (!pgpForm.armoredKey.trim()) { pgpError.value = 'Paste your armored private key.'; return }
  if (!pgpForm.passphrase) { pgpError.value = 'Passphrase is required.'; return }
  pgpBusy.value = true
  try {
    await pgp.importKey(pgpForm.armoredKey.trim(), pgpForm.passphrase)
    pgpMode.value = null
    pgpForm.armoredKey = ''
    pgpForm.passphrase = ''
  } catch (e) {
    pgpError.value = e.message || 'Import failed. Check your key and passphrase.'
  } finally {
    pgpBusy.value = false
  }
}

async function unlockPGPKey() {
  pgpError.value = ''
  if (!pgpForm.passphrase) { pgpError.value = 'Enter your passphrase.'; return }
  pgpBusy.value = true
  try {
    await pgp.unlock(pgpForm.passphrase)
    pgpForm.passphrase = ''
  } catch {
    pgpError.value = 'Wrong passphrase.'
  } finally {
    pgpBusy.value = false
  }
}

async function confirmDeletePGPKey() {
  if (!confirm('Delete your PGP key from the server? This cannot be undone.')) return
  await pgp.deleteKey()
}

function downloadText(text, filename, mime = 'text/plain') {
  const a = document.createElement('a')
  a.href = URL.createObjectURL(new Blob([text], { type: mime }))
  a.download = filename
  a.click()
  URL.revokeObjectURL(a.href)
}

function exportPublicKey() {
  const armored = pgp.armoredPublicKey()
  if (armored) downloadText(armored, 'publickey.asc', 'application/pgp-keys')
}

function exportPrivateKey() {
  if (!pgp.encryptedKey) return
  downloadText(pgp.encryptedKey, 'privatekey.asc', 'application/pgp-keys')
}

defineExpose({ open, close })
</script>
