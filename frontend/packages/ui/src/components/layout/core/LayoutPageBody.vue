<script setup lang="ts">
import type { LayoutPageBodyProps } from '../types'
import { cn } from '@tabtab/utils'

const props = defineProps<LayoutPageBodyProps>()
</script>

<template>
  <div data-slot="layout-page-body" :class="cn('flex-1 overflow-auto', props.class)">
    <slot />
  </div>
</template>
