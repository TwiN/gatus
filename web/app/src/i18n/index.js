import { createI18n } from 'vue-i18n'

import en from './locales/en.json'
import zhCN from './locales/zh-CN.json'

export const LOCALE_STORAGE_KEY = 'gatus:locale'

const SUPPORTED_LOCALES = {
  en: 'en',
  'zh-CN': 'zh-CN'
}

const normalizeLocale = (raw) => {
  if (!raw) return 'en'
  const lowered = String(raw).toLowerCase()
  if (lowered.startsWith('zh')) return 'zh-CN'
  return 'en'
}

const detectLocale = () => {
  // 1) User-selected locale from localStorage
  const stored = typeof window !== 'undefined' ? window.localStorage.getItem(LOCALE_STORAGE_KEY) : null
  if (stored && SUPPORTED_LOCALES[stored]) return stored

  // 2) Browser language matching (navigator.languages first, then navigator.language)
  const browserCandidates = []
  if (typeof navigator !== 'undefined') {
    if (Array.isArray(navigator.languages)) {
      browserCandidates.push(...navigator.languages)
    }
    if (navigator.language) {
      browserCandidates.push(navigator.language)
    }
  }
  for (const candidate of browserCandidates) {
    const normalized = normalizeLocale(candidate)
    if (SUPPORTED_LOCALES[normalized]) return normalized
  }

  // 3) Default fallback
  return 'en'
}

export const i18n = createI18n({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'en',
  messages: {
    en,
    'zh-CN': zhCN
  }
})

// Expose for non-vue modules (e.g. time utils).
if (typeof window !== 'undefined') {
  window.__gatusLocale = i18n.global.locale.value
}

export const setLocale = (nextLocale) => {
  const normalized = SUPPORTED_LOCALES[nextLocale] ? nextLocale : normalizeLocale(nextLocale)
  i18n.global.locale.value = normalized
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, normalized)
    window.__gatusLocale = normalized
  }
}

export default i18n

