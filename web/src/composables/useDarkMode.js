import { ref, watchEffect } from 'vue'

// Module-level ref so all consumers share the same state.
const dark = ref(
  localStorage.getItem('theme') === 'dark' ||
  (!localStorage.getItem('theme') && window.matchMedia('(prefers-color-scheme: dark)').matches),
)

export function useDarkMode() {
  watchEffect(() => {
    document.documentElement.setAttribute('data-theme', dark.value ? 'dark' : 'light')
    localStorage.setItem('theme', dark.value ? 'dark' : 'light')
  })

  function toggle() {
    dark.value = !dark.value
  }

  return { dark, toggle }
}
