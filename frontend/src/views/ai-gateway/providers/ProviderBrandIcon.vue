<script setup lang="ts">
import { computed } from 'vue'
import { Bot } from '@lucide/vue'
import { resolveProviderIcon } from './provider-icons'

const props = withDefaults(
  defineProps<{
    icon?: string
    fallback?: string
    secondaryFallback?: string
    size?: 'sm' | 'md' | 'lg'
  }>(),
  { size: 'md' },
)

const definition = computed(() =>
  resolveProviderIcon(props.icon, props.fallback, props.secondaryFallback),
)
const sizeClass = computed(() => ({ sm: 'size-7', md: 'size-9', lg: 'size-11' })[props.size])
const imageClass = computed(() => ({ sm: 'size-4', md: 'size-5', lg: 'size-7' })[props.size])
</script>

<template>
  <span
    class="inline-flex shrink-0 items-center justify-center rounded-md border bg-white"
    :class="sizeClass"
    :title="definition?.name || '未选择图标'"
  >
    <img
      v-if="definition"
      :src="definition.source"
      alt=""
      class="object-contain"
      :class="imageClass"
    />
    <Bot v-else class="size-4 text-slate-500" />
  </span>
</template>
