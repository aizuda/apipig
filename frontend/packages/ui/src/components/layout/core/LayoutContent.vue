<script setup lang="ts">
import type { LayoutContentProps } from '../types'
import { cn } from '@tabtab/utils'
import { computed } from 'vue'
import { ScrollArea } from '../../ui/scroll-area'
import { SidebarInset } from '../../ui/sidebar'
import { useLayout } from '../utils'

const props = withDefaults(defineProps<LayoutContentProps>(), {
  scrollable: true,
  fixedHeader: true,
})

defineSlots<{
  /** 头部插槽 */
  header(): void
  /** 默认内容插槽 */
  default(): void
}>()

const {
  mode,
  variant,
  collapsed,
  hidden,
  hasSidebar,
  hasDoubleSidebar,
  doubleSidebarHasExpandedChildren,
} = useLayout()

const contentStyle = computed(() => {
  if (hasSidebar.value && !hidden.value) {
    return {
      marginLeft: collapsed.value
        ? 'var(--layout-sidebar-width-collapsed)'
        : 'var(--layout-sidebar-width)',
    }
  }
  if (hasDoubleSidebar.value && mode.value === 'double-sidebar' && !hidden.value) {
    const menuWidth = doubleSidebarHasExpandedChildren.value
      ? 'var(--layout-double-sidebar-menu-width)'
      : '0px'
    return {
      marginLeft: `calc(var(--layout-double-sidebar-icon-width) + ${menuWidth})`,
    }
  }
  return {}
})

const contentClass = computed(() => {
  const classes = ['flex flex-1 flex-col overflow-hidden']

  if (mode.value === 'fullscreen') {
    classes.push('h-full')
  }

  return classes
})

const contentContainerClass = computed(() => {
  const classes: string[] = []

  if (variant.value === 'fixed') {
    classes.push('max-w-screen-2xl', 'mx-auto', 'w-full')
  } else if (variant.value === 'streamer') {
    classes.push('max-w-none', 'w-full')
  }

  return classes
})
</script>

<template>
  <SidebarInset v-if="hasSidebar" :class="cn(props.class)">
    <div :class="cn('h-full flex flex-col', contentContainerClass)">
      <!-- 固定 header：在 ScrollArea 外部 -->
      <template v-if="fixedHeader">
        <slot name="header" />
        <ScrollArea v-if="scrollable" class="flex-1 min-h-0">
          <slot />
        </ScrollArea>
        <slot v-else />
      </template>
      <!-- 非固定 header：在 ScrollArea 内部 -->
      <template v-else>
        <ScrollArea v-if="scrollable" class="flex-1 min-h-0">
          <slot name="header" />
          <slot />
        </ScrollArea>
        <template v-else>
          <slot name="header" />
          <slot />
        </template>
      </template>
    </div>
  </SidebarInset>

  <div
    v-else
    data-slot="layout-content"
    :data-mode="mode"
    :class="cn(contentClass, props.class)"
    :style="contentStyle"
  >
    <div :class="cn('h-full flex flex-col', contentContainerClass)">
      <!-- 固定 header：在 ScrollArea 外部 -->
      <template v-if="fixedHeader">
        <slot name="header" />
        <ScrollArea v-if="scrollable" class="flex-1 min-h-0">
          <slot />
        </ScrollArea>
        <slot v-else />
      </template>
      <!-- 非固定 header：在 ScrollArea 内部 -->
      <template v-else>
        <ScrollArea v-if="scrollable" class="flex-1 min-h-0">
          <slot name="header" />
          <slot />
        </ScrollArea>
        <template v-else>
          <slot name="header" />
          <slot />
        </template>
      </template>
    </div>
  </div>
</template>
