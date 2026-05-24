import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiFetch } from '../api'

export const useAuthStore = defineStore('auth', () => {
  const loggedIn = ref(false)

  async function checkSession() {
    try {
      const res = await fetch('/api/folders')
      loggedIn.value = res.ok
    } catch {
      loggedIn.value = false
    }
    return loggedIn.value
  }

  async function login({ imapHost, imapPort, smtpHost, smtpPort, username, password }) {
    const res = await apiFetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        imap_host: imapHost,
        imap_port: imapPort || 993,
        smtp_host: smtpHost,
        smtp_port: smtpPort || 587,
        username,
        password,
      }),
    })
    if (res.status === 202) {
      const data = await res.json()
      if (data.totp_required) return { totpRequired: true }
    }
    if (!res.ok) throw new Error('Login failed')
    loggedIn.value = true
    return { totpRequired: false }
  }

  async function verifyTOTP(code) {
    const res = await apiFetch('/api/auth/totp/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.error || 'Invalid code')
    }
    loggedIn.value = true
  }

  async function logout() {
    await apiFetch('/api/auth/logout', { method: 'POST' })
    loggedIn.value = false
  }

  return { loggedIn, checkSession, login, verifyTOTP, logout }
})
