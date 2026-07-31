<script setup lang="ts">
import { RefreshCw } from '@lucide/vue'
import { Button } from '@tabtab/ui'

withDefaults(
  defineProps<{
    title: string
    description?: string
    loading?: boolean
    showRefresh?: boolean
    refreshText?: string
  }>(),
  {
    description: '',
    loading: false,
    showRefresh: true,
    refreshText: '刷新',
  },
)

const emit = defineEmits<{
  refresh: []
}>()
</script>

<template>
  <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
    <div class="min-w-0">
      <h1 class="text-2xl font-semibold tracking-tight">{{ title }}</h1>
      <p v-if="description" class="mt-0.5 text-sm leading-5 text-muted-foreground">
        {{ description }}
      </p>
    </div>
    <div class="flex w-full items-center gap-2 sm:w-auto">
      <slot name="actions" />
      <Button
        v-if="showRefresh"
        variant="outline"
        size="sm"
        class="w-full shrink-0 sm:w-auto"
        :disabled="loading"
        @click="emit('refresh')"
      >
        <RefreshCw class="mr-1.5 h-3.5 w-3.5" :class="{ 'animate-spin': loading }" />
        {{ refreshText }}
      </Button>
    </div>
  </div>
</template>
