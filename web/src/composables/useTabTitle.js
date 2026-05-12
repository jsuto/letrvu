import { watch, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useMailStore } from '../stores/mail'

const BASE_TITLE = 'letrvu'

export function useTabTitle() {
  const mail = useMailStore()
  const { folders } = storeToRefs(mail)

  function update(folderList) {
    const inbox = folderList.find(f => f.name.toLowerCase() === 'inbox')
    const unseen = inbox?.unseen ?? 0
    document.title = unseen > 0 ? `(${unseen}) ${BASE_TITLE}` : BASE_TITLE
  }

  const stop = watch(folders, update, { immediate: true, deep: true })

  onUnmounted(() => {
    stop()
    document.title = BASE_TITLE
  })
}
