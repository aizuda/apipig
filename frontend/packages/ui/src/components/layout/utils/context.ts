import type { ComputedRef, Ref } from 'vue'
import type { LayoutMode, LayoutVariant } from '../types'
import { createContext } from 'reka-ui'
import { computed } from 'vue'

export interface LayoutContextValue {
  mode: ComputedRef<LayoutMode>
  variant: ComputedRef<LayoutVariant>
  /** 侧栏是否折叠成 icon（64px） */
  collapsed: Ref<boolean>
  setCollapsed: (value: boolean) => void
  toggleCollapsed: () => void
  /** 侧栏是否完全隐藏 */
  hidden: Ref<boolean>
  setHidden: (value: boolean) => void
  toggleHidden: () => void
  hasSidebar: ComputedRef<boolean>
  hasDoubleSidebar: ComputedRef<boolean>
  isMixedDouble: ComputedRef<boolean>
  doubleSidebarExpandedId: Ref<string | null>
  setDoubleSidebarExpandedId: (value: string | null) => void
  /** 双栏侧栏的二级菜单是否真正展开（有子菜单） */
  doubleSidebarHasExpandedChildren: Ref<boolean>
  setDoubleSidebarHasExpandedChildren: (value: boolean) => void
}

export const [useLayout, provideLayoutContext] = createContext<LayoutContextValue>('Layout')

/**
 * 创建布局相关的计算属性
 */
export function createLayoutComputed(mode: ComputedRef<LayoutMode>) {
  const hasSidebar = computed(() => mode.value === 'sidebar' || mode.value === 'mixed')

  const hasDoubleSidebar = computed(
    () => mode.value === 'double-sidebar' || mode.value === 'mixed-double',
  )

  const isMixedDouble = computed(() => mode.value === 'mixed-double')

  return {
    hasSidebar,
    hasDoubleSidebar,
    isMixedDouble,
  }
}
