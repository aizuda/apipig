import type { ComputedRef, Ref } from 'vue'
import type { LayoutSidebarMenuItem } from '../sidebar/config'
import { useRoute } from 'vue-router'

export type MatchMode = 'exact' | 'startsWith' | 'smart'

export interface UseMenuUtilsOptions {
  expandedKeys?: Ref<Set<string>> | ComputedRef<Set<string>>
  matchMode?: MatchMode
}

/**
 * 菜单工具 Composable
 * @param options - 配置选项
 * @returns 菜单工具方法
 */
export function useMenuUtils(options: UseMenuUtilsOptions = {}) {
  const route = useRoute()
  const { expandedKeys, matchMode = 'smart' } = options

  /**
   * 获取当前路径
   */
  function getCurrentPath(): string | undefined {
    return route?.path
  }

  /**
   * 判断菜单项是否激活
   */
  function isActive(path: string): boolean {
    // 防御性检查：确保 path 存在且有效
    if (!path) {
      return false
    }

    const currentPath = getCurrentPath()
    if (!currentPath) {
      return false
    }

    switch (matchMode) {
      case 'exact':
        return currentPath === path

      case 'startsWith':
        if (path === '/') {
          return currentPath === '/'
        }
        return currentPath.startsWith(path)

      case 'smart':
      default:
        if (currentPath === path) {
          return true
        }
        if (path === '/') {
          return false
        }

        const pathWithSlash = path.endsWith('/') ? path : `${path}/`
        if (!currentPath.startsWith(pathWithSlash)) {
          return false
        }

        const remainingPath = currentPath.slice(pathWithSlash.length)
        return !remainingPath.includes('/')
    }
  }

  /**
   * 判断子菜单是否展开
   */
  function isExpanded(key: string): boolean {
    return expandedKeys?.value.has(key) ?? false
  }

  /**
   * 判断子菜单中是否有活动项
   */
  function hasActiveChild(
    children?: Array<{ path: string; children?: Array<{ path: string }> }>,
  ): boolean {
    if (!children) {
      return false
    }
    return children.some((child) => isActive(child.path))
  }

  /**
   * 根据路径获取需要展开的菜单 keys
   */
  function getExpandedKeysByPath(menus: LayoutSidebarMenuItem[]): Set<string> {
    const keys = new Set<string>()

    function traverse(items: LayoutSidebarMenuItem[], parentKeys: string[] = []): boolean {
      for (const item of items) {
        const currentKeys = [...parentKeys, item.key]

        if (isActive(item.path)) {
          parentKeys.forEach((key) => keys.add(key))
          return true
        }

        if (item.children) {
          const hasActive = traverse(item.children, currentKeys)
          if (hasActive) {
            keys.add(item.key)
            return true
          }
        }
      }
      return false
    }

    traverse(menus)
    return keys
  }

  /**
   * 获取 ARIA current 属性
   */
  function getAriaCurrent(path: string): 'page' | undefined {
    return isActive(path) ? 'page' : undefined
  }

  /**
   * 获取 ARIA expanded 属性
   */
  function getAriaExpanded(key: string): boolean | undefined {
    if (!expandedKeys) {
      return undefined
    }
    return isExpanded(key)
  }

  return {
    isActive,
    isExpanded,
    hasActiveChild,
    getExpandedKeysByPath,
    getAriaCurrent,
    getAriaExpanded,
  }
}

/**
 * 格式化徽标数字
 */
export function formatBadge(num: number): string {
  return num > 99 ? '99+' : String(num)
}

/**
 * 获取按钮变体
 */
export function getButtonVariant(active: boolean): 'default' | 'ghost' {
  return active ? 'default' : 'ghost'
}

/**
 * 获取图标样式类
 */
export function getIconClass(active: boolean): string {
  return active
    ? 'h-5 w-5 text-primary-foreground'
    : 'h-5 w-5 text-muted-foreground group-hover:text-foreground transition-colors'
}

/**
 * 扁平化菜单
 */
export function flattenMenus<T extends { children?: T[] }>(menus: T[]): T[] {
  return menus.reduce((acc: T[], item) => {
    acc.push(item)
    if (item.children) {
      acc.push(...flattenMenus(item.children))
    }
    return acc
  }, [])
}

/**
 * 根据路径查找菜单
 */
export function findMenuByPath<T extends { path: string; children?: T[] }>(
  menus: T[],
  path: string,
): T | undefined {
  for (const menu of menus) {
    if (menu.path === path) {
      return menu
    }
    if (menu.children) {
      const found = findMenuByPath(menu.children, path)
      if (found) {
        return found
      }
    }
  }
  return undefined
}

/**
 * 像素转百分比
 */
export function pxToPercent(px: number, windowWidth: number): number {
  return (px / windowWidth) * 100
}

/**
 * 判断图标是否为组件
 */
export function isComponent(icon: unknown): icon is { __name?: string; render?: Function } {
  return typeof icon !== 'string' && icon !== undefined && icon !== null
}
