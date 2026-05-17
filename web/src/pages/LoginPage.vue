<template>
  <div class="min-h-screen flex items-center justify-center bg-[var(--color-bg)]">
    <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl px-8 py-10 w-full max-w-[400px]">
      <img src="/assets/letrvu-logo-stacked.svg" alt="letrvu" class="block mx-auto mb-8 h-20" />
      <form @submit.prevent="handleLogin">
        <template v-if="!serverLocked">
          <div class="mb-4">
            <label class="block text-xs text-[var(--color-text-muted)] mb-1">IMAP server</label>
            <div class="flex gap-2">
              <input v-model="form.imapHost" type="text" placeholder="mail.example.com" required
                class="w-full px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <input v-model.number="form.imapPort" type="number" placeholder="993"
                class="w-20 shrink-0 px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
            </div>
          </div>
          <div class="mb-4">
            <label class="block text-xs text-[var(--color-text-muted)] mb-1">SMTP server</label>
            <div class="flex gap-2">
              <input v-model="form.smtpHost" type="text" placeholder="smtp.example.com" required
                class="w-full px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <input v-model.number="form.smtpPort" type="number" placeholder="587"
                class="w-20 shrink-0 px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
            </div>
          </div>
        </template>
        <div class="mb-4">
          <label class="block text-xs text-[var(--color-text-muted)] mb-1">Email address</label>
          <input v-model="form.username" type="email" placeholder="you@example.com" required
            class="w-full px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
        </div>
        <div class="mb-4">
          <label class="block text-xs text-[var(--color-text-muted)] mb-1">Password</label>
          <input v-model="form.password" type="password" placeholder="••••••••" required
            class="w-full px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
        </div>
        <p v-if="error" class="text-xs text-red-600 mt-2">{{ error }}</p>
        <button type="submit" :disabled="loading"
          class="w-full mt-5 py-2.5 bg-teal text-white border-none rounded-md text-sm font-medium cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed">
          {{ loading ? 'Connecting…' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const form = reactive({
  imapHost: '',
  imapPort: 993,
  smtpHost: '',
  smtpPort: 587,
  username: '',
  password: '',
})
const loading = ref(false)
const error = ref('')
const serverLocked = ref(false)

onMounted(async () => {
  try {
    const res = await fetch('/api/config')
    if (res.ok) {
      const cfg = await res.json()
      if (cfg.imap_host) form.imapHost = cfg.imap_host
      if (cfg.imap_port) form.imapPort = cfg.imap_port
      if (cfg.smtp_host) form.smtpHost = cfg.smtp_host
      if (cfg.smtp_port) form.smtpPort = cfg.smtp_port
      serverLocked.value = cfg.server_locked ?? false
    }
  } catch {}
})

async function handleLogin() {
  loading.value = true
  error.value = ''
  try {
    await auth.login(form)
    router.push('/mail')
  } catch {
    error.value = 'Could not connect. Check your server details and credentials.'
  } finally {
    loading.value = false
  }
}
</script>
