<script setup lang="ts">
import type { LayoutHeaderProps } from '../types'
import { cn } from '@tabtab/utils'
import { computed } from 'vue'
import { Separator } from '../../ui/separator'
import { useLayout } from '../utils'

const props = withDefaults(defineProps<LayoutHeaderProps>(), {
  fixed: true,
  bordered: true,
})

defineSlots<{
  /** Logo 插槽 */
  logo(): void
  /** 面包屑插槽 */
  breadcrumb(): void
  /** 导航插槽 */
  nav(): void
  /** 操作区插槽 */
  actions(): void
  /** 默认插槽 */
  default(): void
}>()

const { mode, variant, isMixedDouble } = useLayout()

const showTopNav = computed(
  () => mode.value === 'top-nav' || mode.value === 'mixed' || isMixedDouble.value,
)

const headerClass = computed(() => {
  const classes = [
    'bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/70 z-50 flex w-full min-w-0 items-center px-2 sm:px-4 relative',
  ]

  if (props.fixed) {
    classes.push('sticky top-0')
  }

  if (props.bordered) {
    classes.push('border-b border-border/30')
  }

  if (variant.value === 'compact') {
    classes.push('h-12')
  } else {
    classes.push('h-(--layout-header-height)')
  }

  if (mode.value === 'top-nav') {
    classes.push('justify-between')
  }

  return classes
})
</script>

<template>
  <header data-slot="layout-header" :data-mode="mode" :class="cn(headerClass, props.class)">
    <div class="absolute inset-0 pointer-events-none">
      <div
        class="absolute top-0 left-0 right-0 h-16 bg-gradient-to-b from-primary/3 to-transparent"
      />
    </div>
    <div
      class="absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-border/40 to-transparent pointer-events-none"
    />
    <div class="flex min-w-0 flex-1 items-center gap-1.5 sm:gap-2 relative z-10">
      <Separator
        v-if="$slots.logo && $slots.breadcrumb"
        orientation="vertical"
        class="mr-2 h-6 bg-gradient-to-b from-transparent via-border to-transparent"
      />
      <slot name="logo" />
      <Separator
        v-if="$slots.logo && $slots.nav"
        orientation="vertical"
        class="h-6 bg-gradient-to-b from-transparent via-border to-transparent"
      />
      <slot name="breadcrumb" />
      <nav
        v-if="showTopNav"
        class="flex min-w-0 items-center gap-0.5 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      >
        <slot name="nav" />
      </nav>
    </div>
    <div class="flex shrink-0 items-center gap-0.5 sm:gap-2 relative z-10">
      <slot name="actions" />
    </div>
    <slot />
  </header>
</template>
