/**
 * 语言包聚合导出
 * 支持懒加载，按需加载语言包
 */

import type { LocaleMessages, SupportedLocale } from '../types'

/**
 * 支持的语言列表
 */
export const supportedLocales = ['zh-CN', 'en-US'] as const

/**
 * 语言显示名称
 */
export const localeNames: Record<SupportedLocale, string> = {
  'zh-CN': '简体中文',
  'en-US': 'English',
}

/**
 * 语言包加载器映射
 * 使用动态导入实现懒加载
 */
const localeLoaders: Record<SupportedLocale, () => Promise<{ default: LocaleMessages }>> = {
  'zh-CN': () => import('./zh-CN'),
  'en-US': () => import('./en-US'),
}

/**
 * 加载指定语言包
 * @param locale 语言代码
 * @returns 语言包对象
 */
export async function loadLocaleMessages(locale: SupportedLocale): Promise<LocaleMessages> {
  const loader = localeLoaders[locale]
  if (!loader) {
    throw new Error(`Unsupported locale: ${locale}`)
  }
  const module = await loader()
  return module.default
}

/**
 * 预加载语言包（可选优化）
 * 可以在应用空闲时预加载其他语言
 * @param locale 语言代码
 */
export function preloadLocaleMessages(locale: SupportedLocale): void {
  const preload = () => {
    loadLocaleMessages(locale).catch(() => {
      // 预加载失败不影响主流程
    })
  }

  if (typeof window !== 'undefined' && 'requestIdleCallback' in window) {
    window.requestIdleCallback(preload, { timeout: 2000 })
  } else {
    setTimeout(preload, 1000)
  }
}

/**
 * 获取浏览器默认语言
 */
export function getBrowserLocale(): SupportedLocale {
  const browserLang = navigator.language

  // 检查是否支持浏览器的语言
  if (browserLang.startsWith('zh')) {
    return 'zh-CN'
  }

  // 默认返回英文
  return 'en-US'
}

/**
 * 验证语言是否支持
 */
export function isSupportedLocale(locale: string): locale is SupportedLocale {
  return supportedLocales.includes(locale as SupportedLocale)
}

/**
 * 导出类型
 */
export type { LocaleMessages, SupportedLocale } from '../types'

// 导出翻译模块
export { default as commonZhCN } from './zh-CN/common'
export { default as commonEnUS } from './en-US/common'
export { default as menuZhCN } from './zh-CN/menu'
export { default as menuEnUS } from './en-US/menu'
export { default as loginZhCN } from './zh-CN/login'
export { default as loginEnUS } from './en-US/login'
export { default as dashboardZhCN } from './zh-CN/dashboard'
export { default as dashboardEnUS } from './en-US/dashboard'
export { default as settingsZhCN } from './zh-CN/settings'
export { default as settingsEnUS } from './en-US/settings'
