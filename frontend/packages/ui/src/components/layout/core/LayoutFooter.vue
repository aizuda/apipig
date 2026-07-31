<script setup lang="ts">
import type { LayoutFooterProps } from '../types'
import { cn } from '@tabtab/utils'

const props = withDefaults(defineProps<LayoutFooterProps>(), {
  fixed: false,
  bordered: true,
})
</script>

<template>
  <footer
    data-slot="layout-footer"
    :class="
      cn(
        'bg-background flex h-(--layout-footer-height) w-full items-center justify-center px-4 text-sm text-muted-foreground',
        props.fixed && 'sticky bottom-0',
        props.bordered && 'border-t',
        props.class,
      )
    "
  >
    <slot />
  </footer>
</template>
