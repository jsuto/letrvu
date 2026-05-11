<template>
  <div v-if="visible" class="overlay" @click.self="close">
    <div class="modal">
      <div class="modal-header">
        <span>Manage folders</span>
        <button @click="close" class="close">×</button>
      </div>
      <div class="modal-body">

        <!-- Create new folder -->
        <div class="create-row">
          <select v-model="newParent" class="parent-select" title="Parent folder">
            <option value="">— Top level —</option>
            <option v-for="f in mail.folders" :key="f.name" :value="f.name">{{ f.name }}</option>
          </select>
          <input
            v-model="newName"
            type="text"
            placeholder="New folder name…"
            class="create-input"
            @keydown.enter="create"
          />
          <button @click="create" :disabled="!newName.trim() || !!busy" class="create-btn">Create</button>
        </div>

        <p v-if="error" class="error">{{ error }}</p>

        <ul class="folder-list">
          <li v-for="folder in mail.folders" :key="folder.name" class="folder-row">

            <!-- Normal view -->
            <template v-if="renamingFolder !== folder.name">
              <span
                class="folder-name"
                :style="{ paddingLeft: folderDepth(folder) * 14 + 'px' }"
              >{{ folderBasename(folder) }}</span>
              <div class="row-actions">
                <button
                  class="toggle-btn"
                  :class="{ subscribed: folder.subscribed }"
                  :disabled="busy === folder.name"
                  @click="toggle(folder)"
                  :title="folder.subscribed ? 'Unsubscribe (hide from sidebar)' : 'Subscribe (show in sidebar)'"
                >{{ busy === folder.name ? '…' : folder.subscribed ? 'Subscribed' : 'Subscribe' }}</button>
                <button
                  class="icon-btn"
                  title="Rename"
                  :disabled="!!busy"
                  @click="startRename(folder.name)"
                >✎</button>
                <button
                  class="icon-btn danger"
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
                class="rename-input"
                @keydown.enter="commitRename"
                @keydown.escape="cancelRename"
                ref="renameInputEl"
              />
              <div class="row-actions">
                <button class="icon-btn" title="Save" :disabled="!renameValue.trim()" @click="commitRename">✓</button>
                <button class="icon-btn" title="Cancel" @click="cancelRename">✕</button>
              </div>
            </template>

          </li>
        </ul>
      </div>
    </div>
  </div>

  <!-- Delete confirmation -->
  <div v-if="deleteTarget" class="overlay confirm-overlay" @click.self="deleteTarget = null">
    <div class="confirm-modal">
      <p>Delete <strong>{{ deleteTarget }}</strong> and all its messages?</p>
      <p class="confirm-warn">This cannot be undone.</p>
      <div class="confirm-actions">
        <button @click="deleteTarget = null" class="cancel-btn">Cancel</button>
        <button @click="doDelete" :disabled="!!busy" class="delete-btn">Delete</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, onMounted, onUnmounted } from 'vue'
import { useMailStore } from '../stores/mail'

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
  if (deleteTarget.value) { deleteTarget.value = null; return }
  if (renamingFolder.value) { cancelRename(); return }
  if (visible.value) close()
}
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

defineExpose({ open, close })
</script>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal {
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  width: 440px;
  max-height: 75vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.12);
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 0.5px solid var(--color-border);
  font-size: 13px;
  font-weight: 500;
  flex-shrink: 0;
}
.close { background: none; border: none; font-size: 18px; cursor: pointer; color: var(--color-text-muted); }
.modal-body {
  overflow-y: auto;
  padding: 12px 16px;
  flex: 1;
}
.create-row {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.parent-select {
  padding: 7px 8px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 12px;
  font-family: inherit;
  background: var(--color-bg);
  color: var(--color-text);
  outline: none;
  max-width: 140px;
}
.parent-select:focus { border-color: var(--color-teal); }
.create-input {
  flex: 1;
  padding: 7px 10px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  font-size: 13px;
  font-family: inherit;
  background: var(--color-bg);
  color: var(--color-text);
  outline: none;
}
.create-input:focus { border-color: var(--color-teal); }
.create-btn {
  padding: 7px 14px;
  background: var(--color-teal);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
}
.create-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.folder-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.folder-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 7px 0;
  border-bottom: 0.5px solid var(--color-border);
  gap: 8px;
}
.folder-row:last-child { border-bottom: none; }
.folder-name { font-size: 13px; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.row-actions { display: flex; gap: 4px; align-items: center; flex-shrink: 0; }
.toggle-btn {
  padding: 4px 12px;
  border-radius: 5px;
  font-size: 12px;
  cursor: pointer;
  border: 0.5px solid var(--color-border);
  background: transparent;
  color: var(--color-text-muted);
  min-width: 90px;
}
.toggle-btn.subscribed {
  background: var(--color-teal-light);
  border-color: var(--color-teal);
  color: var(--color-teal);
  font-weight: 500;
}
.toggle-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.icon-btn {
  padding: 4px 7px;
  border: 0.5px solid var(--color-border);
  border-radius: 5px;
  background: transparent;
  font-size: 13px;
  cursor: pointer;
  color: var(--color-text-muted);
}
.icon-btn:hover { background: var(--color-bg); color: var(--color-text); }
.icon-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.icon-btn.danger:hover { color: #c0392b; border-color: #f5c6c6; }
.rename-input {
  flex: 1;
  padding: 5px 8px;
  border: 0.5px solid var(--color-teal);
  border-radius: 6px;
  font-size: 13px;
  font-family: inherit;
  background: var(--color-bg);
  color: var(--color-text);
  outline: none;
}
.error { font-size: 12px; color: #c0392b; margin-bottom: 10px; }

/* Delete confirmation */
.confirm-overlay { z-index: 110; }
.confirm-modal {
  background: var(--color-surface);
  border: 0.5px solid var(--color-border);
  border-radius: 10px;
  padding: 24px;
  width: 340px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.18);
}
.confirm-modal p { font-size: 14px; margin: 0 0 8px; }
.confirm-warn { font-size: 12px; color: #c0392b; }
.confirm-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; }
.cancel-btn {
  padding: 7px 16px;
  border: 0.5px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  font-size: 13px;
  cursor: pointer;
}
.cancel-btn:hover { background: var(--color-bg); }
.delete-btn {
  padding: 7px 16px;
  background: #c0392b;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
}
.delete-btn:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
