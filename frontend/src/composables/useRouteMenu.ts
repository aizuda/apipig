import { computed, type ComputedRef } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { resolvedRoutesToMenuItems, findActiveMenuItem, getAffixTabs } from '@tabtab/ui'
import type { MenuItem } from '@tabtab/ui'
import { useUserStore } from '@/stores/user'
import { useMenuStore } from '@/stores/menu'

/**
 * 翻译菜单项
 * @param items 菜单项数组
 * @returns 翻译后的菜单项数组
 */
function translateMenuItems(items: MenuItem[], t: (key: string) => string): MenuItem[] {
  return items.map((item) => ({
    ...item,
    name: item.i18nKey ? t(item.i18nKey) : item.name,
    title: item.i18nKey ? t(item.i18nKey) : item.title || item.name,
    children: item.children ? translateMenuItems(item.children, t) : undefined,
  }))
}

function findMenuItem(items: MenuItem[], id: string): MenuItem | undefined {
  for (const item of items) {
    if (item.id === id) return item
    const child = item.children ? findMenuItem(item.children, id) : undefined
    if (child) return child
  }
  return undefined
}

function filterVisibleMenuItems(items: MenuItem[]): MenuItem[] {
  return items
    .filter((item) => !item.hideInMenu)
    .map((item) => ({
      ...item,
      children: item.children ? filterVisibleMenuItems(item.children) : undefined,
    }))
}

/**
 * 路由菜单 Composable 返回值
 */
export interface UseRouteMenuReturn {
  /** 所有菜单项（从路由生成） */
  menuItems: ComputedRef<MenuItem[]>
  /** 可见菜单项（过滤隐藏项，用于顶部导航和侧边栏） */
  visibleMenuItems: ComputedRef<MenuItem[]>
  /** 当前活动菜单项 */
  activeMenuItem: ComputedRef<{ item: MenuItem; parents: MenuItem[] } | null>
  /** 当前顶部导航 ID（用于 mixed 和 mixed-double 模式） */
  currentTopNavId: ComputedRef<string>
  /** 固定标签页 */
  affixTabs: ComputedRef<MenuItem[]>
}

/**
 * 从路由配置自动生成菜单数据的 Composable
 * @returns 菜单数据和活动状态
 */
export function useRouteMenu(): UseRouteMenuReturn {
  const router = useRouter()
  const route = useRoute()
  const userStore = useUserStore()
  const menuStore = useMenuStore()
  const { t } = useI18n()

  /**
   * 原始菜单项（从路由生成）
   * 使用 router.getRoutes() 获取已解析的路由，确保路径是完整的绝对路径
   * 菜单项会通过 i18n 翻译为当前语言
   */
  const rawMenuItems = computed<MenuItem[]>(() => {
    const resolvedRoutes = router.getRoutes()
    const items = resolvedRoutesToMenuItems(resolvedRoutes)
    return translateMenuItems(items, t)
  })

  /**
   * 根据权限过滤后的菜单项
   */
  const menuItems = computed<MenuItem[]>(() => {
    if (userStore.isAPITokenSession) {
      const callLogs = findMenuItem(rawMenuItems.value, 'AiGatewayLogs')
      return callLogs ? [callLogs] : []
    }
    return menuStore.menuItems
  })

  /**
   * 可见菜单项（过滤隐藏项）
   * 用于顶部导航和侧边栏菜单
   */
  const visibleMenuItems = computed<MenuItem[]>(() => {
    return filterVisibleMenuItems(menuItems.value)
  })

  /**
   * 当前活动菜单项
   */
  const activeMenuItem = computed<{ item: MenuItem; parents: MenuItem[] } | null>(() => {
    return findActiveMenuItem(menuItems.value, route.path)
  })

  /**
   * 当前顶部导航 ID（用于 mixed 和 mixed-double 模式）
   */
  const currentTopNavId = computed<string>(() => {
    if (activeMenuItem.value) {
      const parents = activeMenuItem.value.parents
      if (parents.length > 0) {
        return parents[0].id
      }
      return activeMenuItem.value.item.id
    }
    return ''
  })

  /**
   * 固定标签页
   */
  const affixTabs = computed<MenuItem[]>(() => {
    return getAffixTabs(rawMenuItems.value)
  })

  return {
    menuItems,
    visibleMenuItems,
    activeMenuItem,
    currentTopNavId,
    affixTabs,
  }
}

/**
 * 处理外链点击
 * @param item - 菜单项
 * @returns 是否为外链
 */
export function handleExternalLink(item: MenuItem): boolean {
  if (item.href) {
    window.open(item.href, '_blank', 'noopener,noreferrer')
    return true
  }
  return false
}
