<script setup lang="ts">
import type { LayoutProps } from '../types'
import { cn } from '@tabtab/utils'
import { computed, ref, toRef } from 'vue'
import { useVModel } from '@vueuse/core'
import { provideLayoutContext, createLayoutComputed } from '../utils'

const props = withDefaults(defineProps<LayoutProps>(), {
  mode: 'sidebar',
  variant: 'fixed',
  collapsed: false,
})

const emit = defineEmits<{
  'update:collapsed': [value: boolean]
}>()

const collapsed = useVModel(props, 'collapsed', emit, {
  defaultValue: false,
  passive: true,
})

const mode = toRef(props, 'mode')
const variant = toRef(props, 'variant')

const modeComputed = computed(() => props.mode)
const { hasSidebar, hasDoubleSidebar, isMixedDouble } = createLayoutComputed(modeComputed)

const doubleSidebarExpandedId = ref<string | null>(null)

/**
 * 侧栏完全隐藏状态（由顶栏 SidebarTrigger 控制）
 */
const hidden = ref(false)

/**
 * 双栏侧栏的二级菜单是否真正展开（有子菜单）
 */
const doubleSidebarHasExpandedChildren = ref(false)

function setCollapsed(value: boolean) {
  collapsed.value = value
}

function toggleCollapsed() {
  setCollapsed(!collapsed.value)
}

function setHidden(value: boolean) {
  hidden.value = value
}

function toggleHidden() {
  setHidden(!hidden.value)
}

function setDoubleSidebarExpandedId(value: string | null) {
  doubleSidebarExpandedId.value = value
}

function setDoubleSidebarHasExpandedChildren(value: boolean) {
  doubleSidebarHasExpandedChildren.value = value
}

provideLayoutContext({
  mode: modeComputed,
  variant: computed(() => props.variant),
  collapsed,
  setCollapsed,
  toggleCollapsed,
  hidden,
  setHidden,
  toggleHidden,
  hasSidebar,
  hasDoubleSidebar,
  isMixedDouble,
  doubleSidebarExpandedId,
  setDoubleSidebarExpandedId,
  doubleSidebarHasExpandedChildren,
  setDoubleSidebarHasExpandedChildren,
})

const layoutClass = computed(() => {
  const classes = ['bg-background text-foreground flex w-full']

  switch (props.mode) {
    case 'fullscreen':
      classes.push('h-svh')
      break
    case 'centered':
      classes.push('min-h-svh items-center justify-center')
      break
    case 'top-nav':
      classes.push('h-svh flex-col overflow-hidden')
      break
    case 'mixed':
    case 'mixed-double':
      classes.push('h-svh flex-col overflow-hidden')
      break
    case 'double-sidebar':
    case 'sidebar':
    default:
      classes.push('h-svh flex-row overflow-hidden')
      break
  }

  return classes
})

defineExpose({
  collapsed,
  toggleCollapsed,
  setCollapsed,
  hidden,
  toggleHidden,
  setHidden,
  doubleSidebarExpandedId,
  setDoubleSidebarExpandedId,
  doubleSidebarHasExpandedChildren,
  setDoubleSidebarHasExpandedChildren,
})
</script>

<template>
  <div
    data-slot="layout"
    :data-mode="mode"
    :data-variant="variant"
    :data-collapsed="collapsed"
    :data-hidden="hidden"
    :class="cn(layoutClass, props.class)"
  >
    <slot />
  </div>
</template>
