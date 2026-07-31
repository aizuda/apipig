<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@tabtab/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <div
    data-slot="sidebar-content"
    data-sidebar="content"
    :class="
      cn(
        'flex min-h-0 flex-1 flex-col gap-2 overflow-auto scrollbar-hide',
        'group-data-[collapsible=icon]:overflow-hidden group-data-[collapsible=icon]:px-1',
        'transition-[padding] duration-200',
        '[&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]',
        props.class,
      )
    "
  >
    <slot />
  </div>
</template>
