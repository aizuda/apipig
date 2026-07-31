import type { SidebarConfig, LayoutSidebarMenuItem } from '../sidebar/config'
import { computed, onUnmounted, ref, watch, type WatchStopHandle } from 'vue'
import { useWindowSize } from '@vueuse/core'
import { defaultSidebarConfig } from '../sidebar/config'
import { useMenuUtils } from './useMenuUtils'

const STORAGE_KEY = 'sidebar-state'

interface SidebarState {
  collapsed: boolean
  size: number
  expandedKeys: string[]
}

/**
 * 同步读取 localStorage 中的侧栏状态
 * 在组件初始化时立即调用，避免渲染后跳变
 */
function getSavedState(): SidebarState | null {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      return JSON.parse(saved)
    }
  } catch {
    // 忽略解析错误
  }
  return null
}

/**
 * 侧边栏状态管理 Composable
 * @param config - 侧边栏配置
 * @returns 侧边栏状态和方法
 */
export function useLayoutSidebar(config: SidebarConfig = defaultSidebarConfig) {
  const { width: windowWidth } = useWindowSize()

  /**
   * watch 停止函数集合
   */
  const watchStops: WatchStopHandle[] = []

  /**
   * 像素转百分比
   */
  function pxToPercent(px: number): number {
    return (px / windowWidth.value) * 100
  }

  /**
   * 百分比转像素
   */
  function percentToPx(percent: number): number {
    return (percent / 100) * windowWidth.value
  }

  /**
   * 同步获取保存的状态，避免渲染后跳变
   */
  const savedState = getSavedState()

  /**
   * 折叠状态 - 优先使用保存的值
   */
  const collapsed = ref(savedState?.collapsed ?? false)

  /**
   * 当前尺寸（百分比）- 优先使用保存的值
   */
  const minSize = (config.minWidth / windowWidth.value) * 100
  const maxSize = (config.maxWidth / windowWidth.value) * 100
  const defaultSize = pxToPercent(config.defaultWidth)
  const size = ref(
    savedState?.size ? Math.max(minSize, Math.min(maxSize, savedState.size)) : defaultSize,
  )

  /**
   * 展开的子菜单 keys - 优先使用保存的值
   */
  const expandedKeys = ref<Set<string>>(new Set(savedState?.expandedKeys ?? []))

  /**
   * 是否正在拖拽
   */
  const isDragging = ref(false)

  /**
   * 菜单工具
   */
  const menuUtils = useMenuUtils({ expandedKeys })

  /**
   * 当前宽度（像素）
   */
  const currentWidth = computed(() =>
    collapsed.value ? config.collapsedWidth : percentToPx(size.value),
  )

  /**
   * 当前尺寸（百分比）
   */
  const currentSize = computed(() =>
    collapsed.value ? pxToPercent(config.collapsedWidth) : size.value,
  )

  /**
   * 自动保存状态到 localStorage
   * 拖拽期间暂停保存，避免频繁 I/O 阻塞主线程
   */
  const stateWatchStop = watch(
    [collapsed, size, expandedKeys],
    () => {
      // 拖拽期间不保存，避免频繁 I/O
      if (isDragging.value) return

      try {
        const state: SidebarState = {
          collapsed: collapsed.value,
          size: size.value,
          expandedKeys: Array.from(expandedKeys.value),
        }
        localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
      } catch {
        // 忽略存储错误
      }
    },
    { deep: true },
  )
  watchStops.push(stateWatchStop)

  /**
   * 切换折叠状态
   */
  function toggleCollapse(): void {
    collapsed.value = !collapsed.value
  }

  /**
   * 展开侧边栏
   */
  function expand(): void {
    if (collapsed.value) {
      collapsed.value = false
    }
  }

  /**
   * 折叠侧边栏
   */
  function collapse(): void {
    if (!collapsed.value) {
      collapsed.value = true
    }
  }

  /**
   * 设置尺寸
   */
  function setSize(newSize: number): void {
    const minSize = pxToPercent(config.minWidth)
    const maxSize = pxToPercent(config.maxWidth)
    size.value = Math.max(minSize, Math.min(maxSize, newSize))
  }

  /**
   * 开始拖拽
   */
  function startDrag(): void {
    isDragging.value = true
  }

  /**
   * 结束拖拽
   */
  function finalizeDrag(): void {
    isDragging.value = false
  }

  /**
   * 切换子菜单展开状态
   */
  function toggleSubMenu(key: string): void {
    const newSet = new Set(expandedKeys.value)
    if (newSet.has(key)) {
      newSet.delete(key)
    } else {
      newSet.add(key)
    }
    expandedKeys.value = newSet
  }

  /**
   * 展开子菜单
   */
  function expandSubMenu(key: string): void {
    if (!expandedKeys.value.has(key)) {
      expandedKeys.value = new Set([...expandedKeys.value, key])
    }
  }

  /**
   * 折叠子菜单
   */
  function collapseSubMenu(key: string): void {
    if (expandedKeys.value.has(key)) {
      const newSet = new Set(expandedKeys.value)
      newSet.delete(key)
      expandedKeys.value = newSet
    }
  }

  /**
   * 展开到当前激活菜单项
   */
  function expandToActive(): void {
    const keys = menuUtils.getExpandedKeysByPath(config.menus)
    expandedKeys.value = new Set([...expandedKeys.value, ...keys])
  }

  /**
   * 清理所有 watch 监听器
   */
  function cleanup(): void {
    watchStops.forEach((stop) => stop())
    watchStops.length = 0
  }

  /**
   * 组件卸载时自动清理
   */
  onUnmounted(() => {
    cleanup()
  })

  return {
    collapsed,
    width: currentWidth,
    size,
    currentWidth,
    currentSize,
    expandedKeys,
    isDragging,
    config,
    ...menuUtils,
    toggleCollapse,
    expand,
    collapse,
    setSize,
    startDrag,
    finalizeDrag,
    toggleSubMenu,
    expandSubMenu,
    collapseSubMenu,
    expandToActive,
    cleanup,
  }
}

export { defaultSidebarConfig }
export type { SidebarConfig, LayoutSidebarMenuItem }
