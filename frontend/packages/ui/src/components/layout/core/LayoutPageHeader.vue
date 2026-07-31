<script setup lang="ts">
import type { LayoutPageHeaderProps } from '../types'
import { cn } from '@tabtab/utils'

const props = defineProps<LayoutPageHeaderProps>()
</script>

<template>
  <div data-slot="layout-page-header" :class="cn('mb-6 flex flex-col gap-1', props.class)">
    <div class="flex items-center justify-between">
      <div class="flex flex-col gap-1">
        <h1 v-if="props.title" class="text-2xl font-semibold tracking-tight">
          {{ props.title }}
        </h1>
        <p v-if="props.description" class="text-sm text-muted-foreground">
          {{ props.description }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <slot name="actions" />
      </div>
    </div>
    <slot />
  </div>
</template>
