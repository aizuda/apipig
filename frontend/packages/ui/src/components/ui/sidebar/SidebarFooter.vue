<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@tabtab/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <div
    data-slot="sidebar-footer"
    data-sidebar="footer"
    :class="cn('flex flex-col gap-2 overflow-hidden', props.class)"
  >
    <slot />
  </div>
</template>
