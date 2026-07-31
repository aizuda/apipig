<script setup lang="ts">
import { Avatar, AvatarFallback, AvatarImage } from '../../ui/avatar'
import { computed } from 'vue'

interface Props {
  /** Avatar size */
  size?: 'sm' | 'md' | 'lg'
  /** User name */
  name?: string
  /** Avatar image URL */
  src?: string
  /** Avatar fallback text */
  fallback?: string
  /** Show online status indicator */
  showOnline?: boolean
  /** Is user online */
  online?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  name: 'Admin',
  fallback: '',
  showOnline: true,
  online: true,
})

const sizeClasses = {
  sm: {
    avatar: 'h-6 w-6',
    fallback: 'text-xs',
    indicator: 'h-2 w-2',
  },
  md: {
    avatar: 'h-7 w-7',
    fallback: 'text-xs',
    indicator: 'h-2.5 w-2.5',
  },
  lg: {
    avatar: 'h-8 w-8',
    fallback: 'text-sm',
    indicator: 'h-3 w-3',
  },
}

/**
 * Avatar class styles
 */
const avatarClass = computed(() => [
  sizeClasses[props.size].avatar,
  'ring-2 ring-border/50 transition-all duration-200',
  'group-hover:ring-primary/40 group-hover:scale-105',
])

/**
 * Fallback class styles
 */
const fallbackClass = computed(() => [
  sizeClasses[props.size].fallback,
  'font-medium bg-gradient-to-br from-primary/80 to-primary text-primary-foreground',
])

/**
 * Online indicator class styles
 */
const indicatorClass = computed(() => [
  'absolute rounded-full ring-2 ring-background shadow-sm',
  sizeClasses[props.size].indicator,
  props.online ? 'bg-emerald-500 shadow-emerald-500/50' : 'bg-gray-400 shadow-gray-400/50',
  props.size === 'sm' ? '-bottom-0.5 -right-0.5' : '-bottom-0.5 -right-0.5',
])

/**
 * Get user name initials
 */
const initials = computed(() => {
  if (props.fallback) return props.fallback
  return props.name?.charAt(0)?.toUpperCase() || 'A'
})
</script>

<template>
  <div class="relative inline-flex">
    <Avatar :class="avatarClass">
      <AvatarImage :src="src ?? ''" :alt="name ?? 'User'" />
      <AvatarFallback :class="fallbackClass">
        {{ initials }}
      </AvatarFallback>
    </Avatar>
    <span v-if="showOnline" :class="indicatorClass" />
  </div>
</template>
