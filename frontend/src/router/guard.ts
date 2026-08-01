import type { Router } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useMenuStore } from '@/stores/menu'
import { i18n } from '@/i18n'

const t = i18n.global.t

/**
 * 设置路由守卫
 * @param router - Vue Router 实例
 */
export function setupRouterGuard(router: Router) {
  router.beforeEach(async (to) => {
    const userStore = useUserStore()
    const menuStore = useMenuStore()
    const isAuthenticated = userStore.isAuthenticated

    if (!isAuthenticated) {
      menuStore.resetMenus()
    } else if (userStore.isAPITokenSession) {
      menuStore.resetMenus()
    } else {
      await menuStore.loadMenus()
    }

    if (to.meta.requiresAuth !== false && !isAuthenticated) {
      return {
        name: 'Login',
        query: { redirect: to.fullPath },
      }
    }

    if (to.name === 'Login' && isAuthenticated) {
      return { name: userStore.isAPITokenSession ? 'APITokenUsage' : 'Dashboard' }
    }

    if (isAuthenticated && userStore.isAPITokenSession && to.name !== 'APITokenUsage') {
      return { name: 'APITokenUsage' }
    }

    if (isAuthenticated && !userStore.isAPITokenSession && to.name === 'APITokenUsage') {
      return { name: 'Dashboard' }
    }

    if (to.meta.permissions?.length) {
      const userPermissions = userStore.permissions || []
      const hasPermission = to.meta.permissions.some((permission) =>
        userPermissions.includes(permission),
      )
      if (!hasPermission) {
        return { name: '403' }
      }
    }

    const titleKey = to.meta.titleKey as string
    const title = titleKey ? t(titleKey) : (to.meta.title as string)
    document.title = title ? `${title} | ApiPig Gateway` : 'ApiPig Gateway'
  })

  router.afterEach(() => {
    window.scrollTo(0, 0)
  })
}
