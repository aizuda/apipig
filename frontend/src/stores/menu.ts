import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { MenuItem } from '@tabtab/ui'
import { systemApi, type ResourceMenuNode } from '@/api/system'
import { resolveMenuIcon } from '@/components/menu/menu-icons'
import { useUserStore } from '@/stores/user'

function convertResourceMenu(node: ResourceMenuNode): MenuItem {
  const title = node.meta?.title?.trim() || node.name?.trim() || node.path
  const path = node.path?.trim() || node.redirect?.trim() || '/'
  const isExternal = node.meta?.type === 3 || /^https?:\/\//i.test(path)

  return {
    id: node.name?.trim() || path,
    name: title,
    title,
    path,
    icon: resolveMenuIcon(node.meta?.icon),
    hideInMenu: Boolean(node.meta?.hidden),
    order: node.meta?.order ?? 0,
    activeMenu: node.meta?.parentRoute?.trim() || undefined,
    href: isExternal ? path : undefined,
    children: node.children?.map(convertResourceMenu),
  }
}

export const useMenuStore = defineStore('menu', () => {
  const menuItems = ref<MenuItem[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const errorMessage = ref('')
  let pendingRequest: Promise<void> | null = null
  let requestVersion = 0

  async function loadMenus(force = false) {
    if (pendingRequest) return pendingRequest
    if (loaded.value && !force) return

    const userStore = useUserStore()
    const currentRequestVersion = ++requestVersion
    loading.value = true
    errorMessage.value = ''
    pendingRequest = (async () => {
      try {
        const result = await systemApi.resourceMenu()
        if (currentRequestVersion !== requestVersion) return
        menuItems.value = (result.menus || []).map(convertResourceMenu)
        userStore.setPermissions(result.permissions || [])
      } catch (error) {
        if (currentRequestVersion !== requestVersion) return
        menuItems.value = []
        userStore.setPermissions([])
        errorMessage.value = error instanceof Error ? error.message : 'Menu loading failed'
      } finally {
        if (currentRequestVersion === requestVersion) {
          loaded.value = true
          loading.value = false
          pendingRequest = null
        }
      }
    })()

    return pendingRequest
  }

  function resetMenus() {
    requestVersion += 1
    menuItems.value = []
    loading.value = false
    loaded.value = false
    errorMessage.value = ''
    pendingRequest = null
    useUserStore().setPermissions([])
  }

  return {
    menuItems,
    loading,
    loaded,
    errorMessage,
    loadMenus,
    resetMenus,
  }
})
