import { defineStore } from 'pinia'
import { ref } from 'vue'

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
    const res = await fetch('/api/auth/login', {
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
    if (!res.ok) throw new Error('Login failed')
    loggedIn.value = true
  }

  async function logout() {
    await fetch('/api/auth/logout', { method: 'POST' })
    loggedIn.value = false
  }

  return { loggedIn, checkSession, login, logout }
})
