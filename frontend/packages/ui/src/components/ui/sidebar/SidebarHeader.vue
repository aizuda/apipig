<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@tabtab/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <div
    data-slot="sidebar-header"
    data-sidebar="header"
    :class="cn('flex flex-col gap-2 overflow-hidden', props.class)"
  >
    <slot />
  </div>
</template>
