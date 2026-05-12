<template>
  <div v-if="visible" class="fixed inset-0 bg-black/30 flex items-center justify-center z-[100]" @click.self="close">
    <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl w-[440px] max-h-[75vh] flex flex-col shadow-xl">

      <!-- Header -->
      <div class="flex justify-between items-center px-4 py-3 border-b border-[var(--color-border)] text-sm font-medium shrink-0">
        <span>Manage folders</span>
        <button @click="close" class="bg-none border-none text-lg cursor-pointer text-[var(--color-text-muted)]">×</button>
      </div>

      <!-- Body -->
      <div class="overflow-y-auto px-4 py-3 flex-1">

        <!-- Create new folder -->
        <div class="flex gap-2 mb-3 flex-wrap">
          <select v-model="newParent" class="px-2 py-1.5 border border-[var(--color-border)] rounded-md text-xs bg-[var(--color-bg)] text-[var(--color-text)] outline-none max-w-[140px] focus:border-teal">
            <option value="">— Top level —</option>
            <option v-for="f in mail.folders" :key="f.name" :value="f.name">{{ f.name }}</option>
          </select>
          <input
            v-model="newName"
            type="text"
            placeholder="New folder name…"
            class="flex-1 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal"
            @keydown.enter="create"
          />
          <button @click="create" :disabled="!newName.trim() || !!busy"
            class="px-3.5 py-1.5 bg-teal text-white border-none rounded-md text-sm cursor-pointer whitespace-nowrap disabled:opacity-50 disabled:cursor-not-allowed">Create</button>
        </div>

        <p v-if="error" class="text-xs text-red-600 mb-2.5">{{ error }}</p>

        <ul class="list-none m-0 p-0">
          <li v-for="folder in mail.folders" :key="folder.name"
            class="flex items-center justify-between py-1.5 border-b border-[var(--color-border)] gap-2 last:border-b-0">

            <!-- Normal view -->
            <template v-if="renamingFolder !== folder.name">
              <span
                class="text-sm flex-1 overflow-hidden text-ellipsis whitespace-nowrap"
                :style="{ paddingLeft: folderDepth(folder) * 14 + 'px' }"
              >{{ folderBasename(folder) }}</span>
              <div class="flex gap-1 items-center shrink-0">
                <button
                  :class="[
                    'px-3 py-1 rounded text-xs cursor-pointer border min-w-[90px]',
                    folder.subscribed
                      ? 'bg-[var(--color-teal-light)] border-teal text-teal font-medium'
                      : 'bg-transparent border-[var(--color-border)] text-[var(--color-text-muted)]',
                    busy === folder.name ? 'opacity-50 cursor-not-allowed' : '',
                  ]"
                  :disabled="busy === folder.name"
                  @click="toggle(folder)"
                  :title="folder.subscribed ? 'Unsubscribe (hide from sidebar)' : 'Subscribe (show in sidebar)'"
                >{{ busy === folder.name ? '…' : folder.subscribed ? 'Subscribed' : 'Subscribe' }}</button>
                <button
                  class="px-1.5 py-1 border border-[var(--color-border)] rounded text-sm bg-transparent cursor-pointer text-[var(--color-text-muted)] hover:bg-[var(--color-bg)] hover:text-[var(--color-text)] disabled:opacity-40 disabled:cursor-not-allowed"
                  title="Rename"
                  :disabled="!!busy"
                  @click="startRename(folder.name)"
                >✎</button>
                <button
                  class="px-1.5 py-1 border border-[var(--color-border)] rounded text-sm bg-transparent cursor-pointer text-[var(--color-text-muted)] hover:bg-[var(--color-bg)] hover:text-red-600 hover:border-red-200 disabled:opacity-40 disabled:cursor-not-allowed"
                  title="Delete folder"
                  :disabled="!!busy"
                  @click="confirmDelete(folder.name)"
                >🗑</button>
              </div>
            </template>

            <!-- Inline rename -->
            <template v-else>
              <input
                v-model="renameValue"
                class="flex-1 px-2 py-1.5 border border-teal rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none"
                @keydown.enter="commitRename"
                @keydown.escape="cancelRename"
                ref="renameInputEl"
              />
              <div class="flex gap-1 items-center">
                <button class="px-1.5 py-1 border border-[var(--color-border)] rounded text-sm bg-transparent cursor-pointer text-[var(--color-text-muted)] hover:bg-[var(--color-bg)] hover:text-[var(--color-text)] disabled:opacity-40" title="Save" :disabled="!renameValue.trim()" @click="commitRename">✓</button>
                <button class="px-1.5 py-1 border border-[var(--color-border)] rounded text-sm bg-transparent cursor-pointer text-[var(--color-text-muted)] hover:bg-[var(--color-bg)] hover:text-[var(--color-text)]" title="Cancel" @click="cancelRename">✕</button>
              </div>
            </template>
          </li>
        </ul>
      </div>
    </div>
  </div>

  <ConfirmDialog
    :visible="!!deleteTarget"
    :message="`Delete &quot;${deleteTarget}&quot; and all its messages?`"
    :busy="!!busy"
    @confirm="doDelete"
    @cancel="deleteTarget = null"
    @update:visible="v => { if (!v) deleteTarget = null }"
  />
</template>

<script setup>
import { ref, nextTick, onMounted, onUnmounted } from 'vue'
import { useMailStore } from '../stores/mail'
import ConfirmDialog from './ConfirmDialog.vue'

const mail = useMailStore()
const visible = ref(false)
const busy = ref(null)
const error = ref('')

// Create
const newName = ref('')
const newParent = ref('')

// Rename
const renamingFolder = ref(null)
const renameValue = ref('')
const renameInputEl = ref(null)

// Delete
const deleteTarget = ref(null)

function folderDepth(folder) {
  if (!folder.delimiter) return 0
  return folder.name.split(folder.delimiter).length - 1
}

function folderBasename(folder) {
  if (!folder.delimiter) return folder.name
  const parts = folder.name.split(folder.delimiter)
  return parts[parts.length - 1]
}

async function open() {
  error.value = ''
  newName.value = ''
  newParent.value = ''
  renamingFolder.value = null
  deleteTarget.value = null
  visible.value = true
  await mail.fetchFolders()
}

function close() {
  visible.value = false
}

// --- subscribe / unsubscribe -------------------------------------------------

async function toggle(folder) {
  busy.value = folder.name
  error.value = ''
  try {
    if (folder.subscribed) {
      await mail.unsubscribeFolder(folder.name)
    } else {
      await mail.subscribeFolder(folder.name)
    }
  } catch {
    error.value = `Could not update subscription for "${folder.name}".`
  } finally {
    busy.value = null
  }
}

// --- create ------------------------------------------------------------------

async function create() {
  const name = newName.value.trim()
  if (!name) return
  let fullName = name
  if (newParent.value) {
    const parent = mail.folders.find(f => f.name === newParent.value)
    const delim = parent?.delimiter || '/'
    fullName = newParent.value + delim + name
  }
  busy.value = '__create__'
  error.value = ''
  try {
    await mail.createFolder(fullName)
    newName.value = ''
    newParent.value = ''
  } catch (e) {
    error.value = `Could not create folder: ${e.message}`
  } finally {
    busy.value = null
  }
}

// --- rename ------------------------------------------------------------------

async function startRename(name) {
  renamingFolder.value = name
  renameValue.value = name
  await nextTick()
  renameInputEl.value?.focus()
  renameInputEl.value?.select()
}

function cancelRename() {
  renamingFolder.value = null
  renameValue.value = ''
}

async function commitRename() {
  const newVal = renameValue.value.trim()
  if (!newVal || newVal === renamingFolder.value) { cancelRename(); return }
  busy.value = renamingFolder.value
  error.value = ''
  try {
    await mail.renameFolder(renamingFolder.value, newVal)
    renamingFolder.value = null
    renameValue.value = ''
  } catch (e) {
    error.value = `Could not rename folder: ${e.message}`
  } finally {
    busy.value = null
  }
}

// --- delete ------------------------------------------------------------------

function confirmDelete(name) {
  deleteTarget.value = name
}

async function doDelete() {
  busy.value = deleteTarget.value
  error.value = ''
  try {
    await mail.deleteFolder(deleteTarget.value)
    deleteTarget.value = null
  } catch (e) {
    error.value = `Could not delete folder: ${e.message}`
  } finally {
    busy.value = null
  }
}

// --- keyboard ----------------------------------------------------------------

function onKeydown(e) {
  if (e.key !== 'Escape') return
  if (deleteTarget.value) return // ConfirmDialog handles this via capture
  if (renamingFolder.value) { cancelRename(); return }
  if (visible.value) close()
}
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

defineExpose({ open, close })
</script>
