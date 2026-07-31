/**
 * Store 统一导出
 */

import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

/**
 * 创建 Pinia 实例并配置持久化插件
 */
export function createStore() {
  const pinia = createPinia()
  pinia.use(piniaPluginPersistedstate)
  return pinia
}

export * from './locale'
export * from './theme'
export * from './user'
export * from './tabs'
export * from './notification'
export * from './menu'
