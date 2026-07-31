<script setup lang="ts">
import type { MenuItem } from '../types'
import { cn } from '@tabtab/utils'
import { computed, watch, onUnmounted, type WatchStopHandle } from 'vue'
import { ScrollArea } from '../../ui/scroll-area'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../../ui/tooltip'
import { useLayout } from '../utils'
import { isComponent } from '../composables'
import LayoutDoubleSidebarMenu from './LayoutDoubleSidebarMenu.vue'

export interface LayoutDoubleSidebarProps {
  items: MenuItem[]
  activeId?: string
  collapsed?: boolean
  fixed?: boolean
  class?: HTMLAttributes['class']
}

import type { HTMLAttributes } from 'vue'

const props = withDefaults(defineProps<LayoutDoubleSidebarProps>(), {
  collapsed: false,
  fixed: true,
})

const emit = defineEmits<{
  (e: 'select', item: MenuItem, child?: MenuItem): void
  (e: 'update:collapsed', value: boolean): void
}>()

const {
  isMixedDouble,
  doubleSidebarExpandedId,
  setDoubleSidebarExpandedId,
  setDoubleSidebarHasExpandedChildren,
} = useLayout()

/**
 * 递归检查子菜单中是否包含目标 ID
 * @param items 菜单项数组
 * @param targetId 目标菜单 ID
 * @returns 是否找到目标 ID
 */
function findInChildren(items: MenuItem[], targetId: string): boolean {
  for (const item of items) {
    if (item.id === targetId) {
      return true
    }
    if (item.children?.length && findInChildren(item.children, targetId)) {
      return true
    }
  }
  return false
}

/**
 * 递归查找包含目标 ID 的根级父级菜单（一级菜单）
 * @param items 菜单项数组（一级菜单）
 * @param targetId 目标菜单 ID
 * @returns 根级父级菜单 ID，如果未找到则返回 undefined
 */
function findRootParentId(items: MenuItem[], targetId: string): string | undefined {
  for (const item of items) {
    // 如果直接匹配，说明是一级菜单本身
    if (item.id === targetId) {
      return item.id
    }
    // 递归查找子菜单
    if (item.children?.length && findInChildren(item.children, targetId)) {
      // 找到了，返回当前一级菜单的 ID
      return item.id
    }
  }
  return undefined
}

const activeParent = computed(() => {
  if (!props.items || !Array.isArray(props.items) || !props.activeId) {
    return null
  }

  // 查找根级父级（一级菜单）
  const rootParentId = findRootParentId(props.items, props.activeId)
  return rootParentId ?? null
})

const hasExpandedChildren = computed(() => {
  if (!doubleSidebarExpandedId.value || !props.items?.length) return false
  const item = props.items.find((i) => i?.id === doubleSidebarExpandedId.value)
  return !!item?.children?.length
})

/**
 * watch 停止函数集合
 */
const watchStops: WatchStopHandle[] = []

watchStops.push(
  watch(
    [() => props.activeId, () => props.items],
    () => {
      if (!props.items?.length) return

      if (activeParent.value) {
        const parentItem = props.items.find((i) => i?.id === activeParent.value)

        if (parentItem?.children?.length) {
          // 有子菜单，展开当前一级菜单
          if (doubleSidebarExpandedId.value !== activeParent.value) {
            setDoubleSidebarExpandedId(activeParent.value)
          }
        } else {
          // 无子菜单，清除展开状态
          setDoubleSidebarExpandedId(null)
        }
      }
    },
    { immediate: true, deep: true },
  ),
)

/**
 * 同步 hasExpandedChildren 状态到 Layout Context
 */
watchStops.push(
  watch(
    hasExpandedChildren,
    (value) => {
      setDoubleSidebarHasExpandedChildren(value)
    },
    { immediate: true },
  ),
)

function handleIconClick(item: MenuItem) {
  if (item.children?.length) {
    setDoubleSidebarExpandedId(doubleSidebarExpandedId.value === item.id ? null : item.id)
  } else {
    setDoubleSidebarExpandedId(null)
    emit('select', item)
  }
}

function handleChildClick(parent: MenuItem, child: MenuItem) {
  emit('select', parent, child)
}

const containerClass = computed(() => {
  const classes = ['flex shrink-0']
  if (props.fixed && !isMixedDouble.value) {
    classes.push('fixed left-0 top-0 h-svh z-20')
  } else if (isMixedDouble.value) {
    classes.push('sticky top-0 h-full self-stretch')
  } else {
    classes.push('h-full')
  }
  return classes
})

/**
 * 组件卸载时清理所有 watch 监听器
 */
onUnmounted(() => {
  watchStops.forEach((stop) => stop())
  watchStops.length = 0
})

defineExpose({
  expandedId: doubleSidebarExpandedId,
  setExpandedId: setDoubleSidebarExpandedId,
})
</script>

<template>
  <div data-slot="double-sidebar" :class="cn(containerClass, props.class)">
    <div
      class="flex flex-col items-center border-r border-border/30 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/70 h-full transition-all duration-200 relative"
      style="width: var(--layout-double-sidebar-icon-width)"
    >
      <div class="absolute inset-0 pointer-events-none">
        <div
          class="absolute top-0 left-0 right-0 h-32 bg-gradient-to-b from-primary/3 to-transparent"
        />
        <div
          class="absolute bottom-0 left-0 right-0 h-32 bg-gradient-to-t from-muted/10 to-transparent"
        />
      </div>

      <div class="relative z-10 py-3 w-full flex justify-center">
        <slot name="logo">
          <div class="h-10 w-10 rounded-lg bg-primary flex items-center justify-center shadow-sm">
            <svg
              width="24"
              height="24"
              viewBox="0 0 48 48"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <rect
                x="4"
                y="4"
                width="18"
                height="18"
                rx="4"
                fill="currentColor"
                class="text-primary-foreground"
              />
              <rect
                x="26"
                y="4"
                width="18"
                height="18"
                rx="4"
                fill="currentColor"
                class="text-primary-foreground"
                opacity="0.9"
              />
              <rect
                x="4"
                y="26"
                width="18"
                height="18"
                rx="4"
                fill="currentColor"
                class="text-primary-foreground"
                opacity="0.7"
              />
              <rect
                x="26"
                y="26"
                width="18"
                height="18"
                rx="4"
                fill="currentColor"
                class="text-primary-foreground"
                opacity="0.5"
              />
            </svg>
          </div>
        </slot>
      </div>

      <ScrollArea class="flex-1 h-0 relative z-10">
        <div class="flex flex-col items-center gap-1 px-1 pt-3">
          <TooltipProvider v-for="item in items" :key="item.id" :delay-duration="200">
            <Tooltip>
              <TooltipTrigger as-child>
                <button
                  :class="
                    cn(
                      'relative flex h-10 w-10 items-center justify-center rounded-lg transition-all duration-200',
                      doubleSidebarExpandedId === item.id || activeParent === item.id
                        ? 'bg-primary text-primary-foreground'
                        : 'hover:bg-primary/10 hover:text-primary text-sidebar-foreground/70',
                    )
                  "
                  @click="handleIconClick(item)"
                >
                  <slot :name="`icon-${item.id}`" :item="item">
                    <component :is="item.icon" v-if="isComponent(item.icon)" class="h-5 w-5" />
                    <span v-else class="text-lg">{{ item.icon }}</span>
                  </slot>
                  <span
                    v-if="item.children?.length"
                    class="absolute -bottom-1 -right-1 h-4 min-w-4 px-1 text-[10px] font-medium flex items-center justify-center rounded-full border z-10"
                    :class="
                      doubleSidebarExpandedId === item.id || activeParent === item.id
                        ? 'bg-primary-foreground text-primary border-primary-foreground/50'
                        : 'bg-primary/10 text-primary border-primary/20'
                    "
                  >
                    {{ item.children.length }}
                  </span>
                </button>
              </TooltipTrigger>
              <TooltipContent side="right">
                <span>{{ item.name }}</span>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </ScrollArea>
      <div class="flex-shrink-0 min-h-[60px]">
        <slot name="footer" />
      </div>
    </div>

    <div
      class="flex flex-col border-r border-border/30 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/70 transition-all duration-200 h-full overflow-hidden relative"
      :style="{ width: hasExpandedChildren ? 'var(--layout-double-sidebar-menu-width)' : '0px' }"
    >
      <div v-if="hasExpandedChildren" class="absolute inset-0 pointer-events-none">
        <div
          class="absolute top-0 left-0 right-0 h-32 bg-gradient-to-b from-primary/3 to-transparent"
        />
        <div
          class="absolute bottom-0 left-0 right-0 h-32 bg-gradient-to-t from-muted/10 to-transparent"
        />
      </div>

      <div v-if="hasExpandedChildren" class="flex h-full flex-col relative z-10">
        <div class="flex h-14 items-center border-b border-border/30 px-4 min-w-0">
          <span class="text-sm font-medium truncate">
            {{ items.find((i) => i.id === doubleSidebarExpandedId)?.name }}
          </span>
        </div>
        <ScrollArea class="flex-1 h-0">
          <div class="p-2">
            <LayoutDoubleSidebarMenu
              :items="items.find((i) => i.id === doubleSidebarExpandedId)?.children || []"
              :active-id="activeId"
              @select="
                (item) =>
                  handleChildClick(items.find((i) => i.id === doubleSidebarExpandedId)!, item)
              "
            />
          </div>
        </ScrollArea>
      </div>
    </div>
  </div>
</template>
