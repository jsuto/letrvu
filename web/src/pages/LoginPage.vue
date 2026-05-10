<template>
  <div class="login-page">
    <div class="login-card">
      <img src="/assets/letrvu-logo-stacked.svg" alt="letrvu" class="logo" />
      <form @submit.prevent="handleLogin">
        <div class="field-group">
          <label>IMAP server</label>
          <div class="row">
            <input v-model="form.imapHost" type="text" placeholder="mail.example.com" required />
            <input v-model.number="form.imapPort" type="number" placeholder="993" class="port" />
          </div>
        </div>
        <div class="field-group">
          <label>SMTP server</label>
          <div class="row">
            <input v-model="form.smtpHost" type="text" placeholder="smtp.example.com" required />
            <input v-model.number="form.smtpPort" type="number" placeholder="587" class="port" />
          </div>
        </div>
        <div class="field-group">
          <label>Email address</label>
          <input v-model="form.username" type="email" placeholder="you@example.com" required />
        </div>
        <div class="field-group">
          <label>Password</label>
          <input v-model="form.password" type="password" placeholder="••••••••" required />
        </div>
        <p v-if="error" class="error">{{ error }}</p>
        <button type="submit" :disabled="loading" class="submit">
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

onMounted(async () => {
  try {
    const res = await fetch('/api/config')
    if (res.ok) {
      const cfg = await res.json()
      if (cfg.imap_host) form.imapHost = cfg.imap_host
      if (cfg.imap_port) form.imapPort = cfg.imap_port
      if (cfg.smtp_host) form.smtpHost = cfg.smtp_host
      if (cfg.smtp_port) form.smtpPort = cfg.smtp_port
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

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
}
.login-card {
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 12px;
  padding: 2.5rem 2rem;
  width: 100%;
  max-width: 400px;
}
.logo {
  display: block;
  margin: 0 auto 2rem;
  height: 80px;
}
.field-group {
  margin-bottom: 1rem;
}
.field-group label {
  display: block;
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 4px;
}
.row {
  display: flex;
  gap: 8px;
}
.port {
  width: 80px;
  flex-shrink: 0;
}
input {
  width: 100%;
  padding: 8px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 14px;
  outline: none;
  background: var(--color-surface);
}
input:focus {
  border-color: var(--color-teal);
}
.submit {
  width: 100%;
  margin-top: 1.25rem;
  padding: 10px;
  background: var(--color-teal);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}
.submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.error {
  font-size: 12px;
  color: #c0392b;
  margin-top: 8px;
}
</style>
