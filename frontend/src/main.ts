import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { i18n, initI18n } from './i18n'
import { createStore, useLocaleStore } from './stores'
import { useThemeStore } from './stores/theme'

import '@tabtab/styles'

/**
 * 初始化应用
 */
async function initApp() {
  const app = createApp(App)
  const pinia = createStore()

  app.use(pinia)
  app.use(i18n)

  // 初始化 i18n 语言包（必须在 router 注册前完成）
  await initI18n()

  // 初始化语言设置
  const localeStore = useLocaleStore()
  await localeStore.init()

  // 提前初始化主题 - 在挂载前应用，防止闪烁
  const themeStore = useThemeStore()
  themeStore.initTheme()

  // 注册路由（会触发初始导航，需要语言包已加载）
  app.use(router)

  app.mount('#app')
}

initApp().catch((error) => {
  console.error('Failed to initialize app:', error)
})
