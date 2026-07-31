import type { MenuItem } from '../types'

type RouteMetaWithMenu = {
  title?: string
  titleKey?: string
  icon?: MenuItem['icon']
  localIcon?: string
  hideInMenu?: boolean
  hideInTab?: boolean
  href?: string
  order?: number
  badge?: string | number
  keepAlive?: boolean
  affixTab?: boolean
  affixTabOrder?: number
  activeMenu?: string
  permissions?: string[]
}

type RouteRecordLike = {
  name?: string | symbol | null
  path: string
  meta?: RouteMetaWithMenu
}

/**
 * 从已解析的路由记录中提取菜单项
 * @param routes - 已解析的路由记录数组（来自 router.getRoutes()）
 * @param filterHidden - 是否过滤隐藏项，默认 true
 * @returns 菜单项数组
 */
export function resolvedRoutesToMenuItems(
  routes: RouteRecordLike[],
  filterHidden = true,
): MenuItem[] {
  const itemMap = new Map<string, MenuItem>()
  const rootItems: MenuItem[] = []

  for (const route of routes) {
    const meta = route.meta
    const name = meta?.title ?? route.name?.toString() ?? route.path
    const id = route.name?.toString() ?? route.path
    const i18nKey = meta?.titleKey
    const hideInMenu = meta?.hideInMenu || route.path.includes(':')

    const menuItem: MenuItem = {
      id,
      name,
      title: name,
      i18nKey: i18nKey ?? name,
      path: route.path,
      icon: meta?.icon,
      localIcon: meta?.localIcon,
      hideInMenu,
      hideInTab: meta?.hideInTab,
      href: meta?.href,
      order: meta?.order ?? 0,
      badge: meta?.badge,
      keepAlive: meta?.keepAlive,
      affixTab: meta?.affixTab,
      affixTabOrder: meta?.affixTabOrder,
      activeMenu: meta?.activeMenu,
      permissions: meta?.permissions,
    }

    itemMap.set(route.path, menuItem)
  }

  for (const [, item] of itemMap) {
    const parentPaths = getParentPathCandidates(item.path)

    let parent: MenuItem | undefined

    for (const parentPath of parentPaths) {
      const directParent = itemMap.get(parentPath)
      if (directParent) {
        parent = directParent
        break
      }
    }

    if (!parent && parentPaths.length > 0) {
      const parentName = parentPaths[0].slice(1)
      for (const [, parentItem] of itemMap) {
        if (parentItem.name.toLowerCase().includes(parentName.toLowerCase())) {
          parent = parentItem
          break
        }
      }
    }

    if (parent) {
      if (parent.hideInMenu) {
        rootItems.push(item)
      } else {
        if (!parent.children) {
          parent.children = []
        }
        parent.children.push(item)
      }
    } else if (item.path.startsWith('/') && item.path.split('/').length === 2) {
      rootItems.push(item)
    } else if (item.path === '/') {
      if (item.children) {
        const visibleChildren = item.children.filter((child) => !child.hideInMenu)
        rootItems.push(...visibleChildren)
      }
    }
  }

  for (const item of itemMap.values()) {
    if (item.children) {
      item.children = sortMenuItems(item.children)
    }
  }

  const result = sortMenuItems(rootItems)

  if (filterHidden) {
    return filterHiddenItems(result)
  }

  return result
}

/**
 * 获取父路径
 * @param path - 当前路径
 * @returns 父路径
 */
function getParentPath(path: string): string {
  if (path === '/' || path === '') return '/'
  const lastSlash = path.lastIndexOf('/')
  if (lastSlash === 0) return '/'
  return path.substring(0, lastSlash)
}

function getParentPathCandidates(path: string): string[] {
  const candidates: string[] = []
  let current = getParentPath(path)

  while (current !== '/' && current !== path) {
    candidates.push(current)
    const next = getParentPath(current)
    if (next === current) {
      break
    }
    current = next
  }

  return candidates
}

/**
 * 过滤隐藏的菜单项
 * @param items - 菜单项数组
 * @returns 过滤后的菜单项数组
 */
function filterHiddenItems(items: MenuItem[]): MenuItem[] {
  return items
    .filter((item) => !item.hideInMenu)
    .map((item) => ({
      ...item,
      children: item.children ? filterHiddenItems(item.children) : undefined,
    }))
}

/**
 * 对菜单项按 order 排序
 */
function sortMenuItems(items: MenuItem[]): MenuItem[] {
  return [...items].sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
}

/**
 * 获取扁平化的菜单路径映射
 */
export function getFlattenedMenuMap(items: MenuItem[]): Map<string, MenuItem> {
  const map = new Map<string, MenuItem>()

  function flatten(menuItems: MenuItem[]) {
    for (const item of menuItems) {
      map.set(item.path, item)
      if (item.children) {
        flatten(item.children)
      }
    }
  }

  flatten(items)
  return map
}

/**
 * 根据当前路径查找活动的菜单项
 */
export function findActiveMenuItem(
  items: MenuItem[],
  currentPath: string,
): { item: MenuItem; parents: MenuItem[] } | null {
  function find(
    menuItems: MenuItem[],
    path: string,
    parents: MenuItem[],
  ): { item: MenuItem; parents: MenuItem[] } | null {
    for (const item of menuItems) {
      if (item.path === path) {
        return { item, parents }
      }
      if (item.children) {
        const found = find(item.children, path, [...parents, item])
        if (found) return found
      }
    }
    return null
  }

  return find(items, currentPath, [])
}

/**
 * 根据权限过滤菜单
 */
export function filterMenuByPermission(items: MenuItem[], permissions: string[]): MenuItem[] {
  return items
    .filter((item) => {
      if (!item.permissions?.length) return true
      return item.permissions.some((p) => permissions.includes(p))
    })
    .map((item) => ({
      ...item,
      children: item.children ? filterMenuByPermission(item.children, permissions) : undefined,
    }))
    .filter((item) => !item.hideInMenu)
}

/**
 * 获取需要固定在标签页的菜单项
 */
export function getAffixTabs(items: MenuItem[]): MenuItem[] {
  const affixTabs: MenuItem[] = []

  function find(menuItems: MenuItem[]) {
    for (const item of menuItems) {
      if (item.affixTab) {
        affixTabs.push(item)
      }
      if (item.children) {
        find(item.children)
      }
    }
  }

  find(items)
  return affixTabs.sort((a, b) => (a.affixTabOrder ?? 0) - (b.affixTabOrder ?? 0))
}
