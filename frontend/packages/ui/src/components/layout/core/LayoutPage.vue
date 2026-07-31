<script setup lang="ts">
import type { LayoutPageProps } from '../types'
import { cn } from '@tabtab/utils'

const props = defineProps<LayoutPageProps>()
</script>

<template>
  <div
    data-slot="layout-page"
    :class="cn('flex flex-1 min-h-0 min-w-0 flex-col p-3 sm:p-4 lg:p-6', props.class)"
  >
    <slot />
  </div>
</template>
