<template>
  <nav class="flex flex-col h-full py-4 px-3">
    <div class="mb-5">
      <img src="/assets/letrvu-logo-horizontal.svg" alt="letrvu" class="h-7" />
    </div>
    <button
      class="w-full py-2 bg-teal text-white border-none rounded-md text-sm font-medium cursor-pointer mb-4 hover:opacity-90"
      @click="compose?.open()"
    >Compose</button>
    <RouterLink
      to="/contacts"
      class="block px-2 py-1.5 rounded-md text-sm text-[var(--color-text)] no-underline mb-2 hover:bg-[var(--color-teal-light)]"
      active-class="!bg-[var(--color-teal-light)] font-medium"
    >Contacts</RouterLink>
    <RouterLink
      to="/calendar"
      class="block px-2 py-1.5 rounded-md text-sm text-[var(--color-text)] no-underline mb-2 hover:bg-[var(--color-teal-light)]"
      active-class="!bg-[var(--color-teal-light)] font-medium"
    >Calendar</RouterLink>
    <ul v-if="visibleFolders.length" class="list-none flex-1 overflow-y-auto mt-1">
      <li
        v-for="folder in visibleFolders"
        :key="folder.name"
        :class="[
          'flex justify-between items-center pl-3 pr-2 py-1.5 rounded-md cursor-pointer text-sm text-[var(--color-text)] hover:bg-[var(--color-teal-light)]',
          mail.currentFolder === folder.name ? 'bg-[var(--color-teal-light)] font-medium' : '',
          dragOver === folder.name ? '!bg-teal !text-white' : '',
          folder.unseen > 0 ? 'font-semibold' : '',
        ]"
        @click="openFolder(folder.name)"
        @dragover.prevent="dragOver = folder.name"
        @dragleave="dragOver = null"
        @drop.prevent="onDrop($event, folder.name)"
      >
        {{ folder.name }}
        <span
          v-if="folder.unseen"
          :class="[
            'text-[11px] px-1.5 py-px rounded-full',
            dragOver === folder.name
              ? 'bg-white text-teal'
              : 'bg-teal text-white',
          ]"
        >{{ folder.unseen }}</span>
      </li>
    </ul>
    <p v-else class="text-xs text-[var(--color-text-muted)] px-2 flex-1">Loading folders…</p>
    <button
      class="w-full py-1.5 px-2 bg-none border border-[var(--color-border)] rounded-md text-xs text-[var(--color-text-muted)] cursor-pointer mb-2 text-left hover:bg-[var(--color-teal-light)] hover:text-[var(--color-text)]"
      @click="manageFoldersModal?.open()"
    >Manage folders</button>
    <div class="flex items-center justify-between pt-3 border-t border-[var(--color-border)] mt-3">
      <button
        class="bg-none border-none cursor-pointer text-sm text-[var(--color-text-muted)] px-1.5 py-1 rounded hover:bg-[var(--color-teal-light)]"
        :title="dark ? 'Switch to light mode' : 'Switch to dark mode'"
        @click="toggleDark"
      >{{ dark ? '☀️' : '🌙' }}</button>
      <button
        class="bg-none border-none cursor-pointer text-sm text-[var(--color-text-muted)] px-1.5 py-1 rounded hover:bg-[var(--color-teal-light)]"
        title="Settings"
        @click="settingsModal?.open()"
      >⚙</button>
      <button
        class="bg-none border-none cursor-pointer text-sm text-[var(--color-text-muted)] px-1.5 py-1 rounded hover:bg-[var(--color-teal-light)]"
        title="Sign out"
        @click="handleLogout"
      >Sign out</button>
    </div>
  </nav>
  <SettingsModal ref="settingsModal" />
  <ManageFoldersModal ref="manageFoldersModal" />
</template>

<script setup>
import { inject, onMounted, ref, computed } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useMailStore } from '../stores/mail'
import { useAuthStore } from '../stores/auth'
import { useDarkMode } from '../composables/useDarkMode'
import SettingsModal from './SettingsModal.vue'
import ManageFoldersModal from './ManageFoldersModal.vue'

const mail = useMailStore()
const auth = useAuthStore()
const router = useRouter()
const compose = inject('compose', null)
const { dark, toggle: toggleDark } = useDarkMode()
const dragOver = ref(null)
const settingsModal = ref(null)
const manageFoldersModal = ref(null)

// Show only subscribed folders; fall back to all if the server doesn't report
// subscription status (i.e. none are marked subscribed).
const visibleFolders = computed(() => {
  const subscribed = mail.folders.filter(f => f.subscribed)
  return subscribed.length ? subscribed : mail.folders
})

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
