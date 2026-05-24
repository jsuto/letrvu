import { createI18n } from 'vue-i18n'
import en from './locales/en.json'

const savedLocale = localStorage.getItem('locale') || 'en'

const i18n = createI18n({
  legacy: false,
  locale: savedLocale,
  fallbackLocale: 'en',
  messages: { en },
})

export default i18n
export const SUPPORTED_LOCALES = [
  { code: 'en', label: 'English' },
]
