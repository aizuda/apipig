<script setup lang="ts">
import { computed, type Component } from 'vue'
import { CircleHelp } from '@lucide/vue'
import { resolveMenuIcon } from './menu-icons'

const props = withDefaults(
  defineProps<{
    icon?: string
    fallback?: Component
  }>(),
  {
    icon: '',
    fallback: undefined,
  },
)

const iconComponent = computed(() => resolveMenuIcon(props.icon) || props.fallback || CircleHelp)
</script>

<template>
  <component :is="iconComponent" aria-hidden="true" />
</template>
