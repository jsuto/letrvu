<template>
  <nav class="folder-list">
    <div class="logo-row">
      <img src="/assets/letrvu-logo-horizontal.svg" alt="letrvu" class="logo" />
    </div>
    <button class="compose-btn" @click="compose?.open()">Compose</button>
    <RouterLink to="/contacts" class="nav-link">Contacts</RouterLink>
    <ul v-if="mail.folders.length">
      <li
        v-for="folder in mail.folders"
        :key="folder.name"
        :class="{ active: mail.currentFolder === folder.name }"
        @click="mail.fetchMessages(folder.name)"
      >
        {{ folder.name }}
        <span v-if="folder.unseen" class="badge">{{ folder.unseen }}</span>
      </li>
    </ul>
    <p v-else class="empty">Loading folders…</p>
    <div class="bottom">
      <button class="icon-btn" :title="dark ? 'Switch to light mode' : 'Switch to dark mode'" @click="toggleDark">
        {{ dark ? '☀️' : '🌙' }}
      </button>
      <button class="icon-btn logout" title="Sign out" @click="handleLogout">Sign out</button>
    </div>
  </nav>
</template>

<script setup>
import { inject } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useMailStore } from '../stores/mail'
import { useAuthStore } from '../stores/auth'
import { useDarkMode } from '../composables/useDarkMode'

const mail = useMailStore()
const auth = useAuthStore()
const router = useRouter()
const compose = inject('compose')
const { dark, toggle: toggleDark } = useDarkMode()

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.folder-list {
  padding: 1rem 0.75rem;
  display: flex;
  flex-direction: column;
  height: 100%;
}
.logo-row {
  margin-bottom: 1.25rem;
}
.logo {
  height: 28px;
}
.nav-link {
  display: block;
  padding: 6px 8px;
  border-radius: 6px;
  font-size: 13px;
  color: var(--color-text);
  text-decoration: none;
  margin-bottom: 0.5rem;
}
.nav-link:hover { background: var(--color-teal-light); }
.nav-link.router-link-active { background: var(--color-teal-light); font-weight: 500; }
.compose-btn {
  width: 100%;
  padding: 8px;
  background: var(--color-teal);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  margin-bottom: 1rem;
}
ul {
  list-style: none;
  flex: 1;
  overflow-y: auto;
}
li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: var(--color-text);
}
li:hover { background: var(--color-teal-light); }
li.active { background: var(--color-teal-light); font-weight: 500; }
.badge {
  font-size: 11px;
  background: var(--color-teal);
  color: white;
  padding: 1px 6px;
  border-radius: 10px;
}
.empty {
  font-size: 12px;
  color: var(--color-text-muted);
  padding: 8px;
  flex: 1;
}
.bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 0.75rem;
  border-top: 0.5px solid var(--color-border);
  margin-top: 0.75rem;
}
.icon-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 13px;
  color: var(--color-text-muted);
  padding: 4px 6px;
  border-radius: 5px;
}
.icon-btn:hover { background: var(--color-teal-light); }
.logout { color: var(--color-text-muted); }
</style>
