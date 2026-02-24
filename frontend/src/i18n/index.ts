import { createI18n } from 'vue-i18n'
import en from './locales/en.json'
import zhCN from './locales/zh-CN.json'
import zhTW from './locales/zh-TW.json'
import ja from './locales/ja.json'

export type Locale = 'en' | 'zh-CN' | 'zh-TW' | 'ja'

export const SUPPORTED_LOCALES: { value: Locale; label: string }[] = [
  { value: 'en', label: 'English' },
  { value: 'zh-CN', label: '简体中文' },
  { value: 'zh-TW', label: '繁體中文' },
  { value: 'ja', label: '日本語' }
]

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages: {
    en,
    'zh-CN': zhCN,
    'zh-TW': zhTW,
    ja
  }
})

export function setLocale(locale: Locale) {
  i18n.global.locale.value = locale
}

export function getLocale(): Locale {
  return i18n.global.locale.value as Locale
}

export default i18n
