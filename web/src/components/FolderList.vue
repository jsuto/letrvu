<template>
  <nav class="folder-list">
    <div class="logo-row">
      <img src="/assets/letrvu-logo-horizontal.svg" alt="letrvu" class="logo" />
    </div>
    <button class="compose-btn">Compose</button>
    <!-- TODO: render mail.folders list with unread counts -->
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
  </nav>
</template>

<script setup>
import { useMailStore } from '../stores/mail'
const mail = useMailStore()
</script>

<style scoped>
.folder-list {
  padding: 1rem 0.75rem;
}
.logo-row {
  margin-bottom: 1.25rem;
}
.logo {
  height: 28px;
}
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
}
</style>
