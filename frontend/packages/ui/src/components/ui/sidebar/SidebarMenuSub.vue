<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@tabtab/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <ul
    data-slot="sidebar-menu-sub"
    data-sidebar="menu-sub"
    :class="
      cn(
        'mx-3.5 flex min-w-0 translate-x-px flex-col gap-1 border-l-2 border-primary/20 pl-3 pr-2 py-1 my-1',
        'bg-sidebar-accent/30 rounded-r-md',
        'group-data-[collapsible=icon]:hidden',
        props.class,
      )
    "
  >
    <slot />
  </ul>
</template>
