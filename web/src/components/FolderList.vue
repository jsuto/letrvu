<template>
  <nav class="folder-list">
    <div class="logo-row">
      <img src="/assets/letrvu-logo-horizontal.svg" alt="letrvu" class="logo" />
    </div>
    <button class="compose-btn" @click="compose?.open()">Compose</button>
    <RouterLink to="/contacts" class="nav-link">Contacts</RouterLink>
    <RouterLink to="/calendar" class="nav-link">Calendar</RouterLink>
    <ul v-if="mail.folders.length">
      <li
        v-for="folder in mail.folders"
        :key="folder.name"
        :class="{
          active: mail.currentFolder === folder.name,
          'drop-target': dragOver === folder.name,
        }"
        @click="openFolder(folder.name)"
        @dragover.prevent="dragOver = folder.name"
        @dragleave="dragOver = null"
        @drop.prevent="onDrop($event, folder.name)"
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
      <button class="icon-btn" title="Settings" @click="settingsModal?.open()">⚙</button>
      <button class="icon-btn logout" title="Sign out" @click="handleLogout">Sign out</button>
    </div>
  </nav>
  <SettingsModal ref="settingsModal" />
</template>

<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useMailStore } from '../stores/mail'
import { useAuthStore } from '../stores/auth'
import { useDarkMode } from '../composables/useDarkMode'
import SettingsModal from './SettingsModal.vue'

const mail = useMailStore()
const auth = useAuthStore()
const router = useRouter()
const compose = inject('compose', null)
const { dark, toggle: toggleDark } = useDarkMode()
const dragOver = ref(null)
const settingsModal = ref(null)

onMounted(async () => {
  if (!mail.folders.length) {
    await mail.fetchFolders()
  }
})

async function onDrop(e, destFolder) {
  dragOver.value = null
  const uids = JSON.parse(e.dataTransfer.getData('application/x-letrvu-uids') || '[]')
  const srcFolder = e.dataTransfer.getData('application/x-letrvu-folder')
  if (!uids.length || !srcFolder || srcFolder === destFolder) return
  await mail.moveMessagesTo(srcFolder, uids, destFolder)
}

async function openFolder(name) {
  await mail.fetchMessages(name)
  router.push('/mail')
}

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
li.drop-target {
  background: var(--color-teal);
  color: white;
}
li.drop-target .badge { background: white; color: var(--color-teal); }
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
