<script setup lang="ts">
interface Props {
  /** Logo size */
  size?: 'sm' | 'md' | 'lg'
  /** Show text beside logo */
  showText?: boolean
  /** Layout direction */
  direction?: 'horizontal' | 'vertical'
  /** Logo text (shown when showText is true) */
  text?: string
  /** Subtitle text */
  subtitle?: string
}

withDefaults(defineProps<Props>(), {
  size: 'md',
  showText: false,
  direction: 'horizontal',
  text: 'ApiPig',
  subtitle: 'Admin',
})

const sizeClasses = {
  sm: {
    container: 'h-9 w-9',
    scale: 0.9,
  },
  md: {
    container: 'h-10 w-10',
    scale: 1,
  },
  lg: {
    container: 'h-12 w-12',
    scale: 1.2,
  },
}
</script>

<template>
  <div :class="['flex items-center', direction === 'horizontal' ? 'gap-2' : 'flex-col gap-1']">
    <div
      :class="[
        'flex items-center justify-center p-1 text-primary',
        'transition-all duration-300 ease-out',
        'hover:scale-105',
        'cursor-pointer',
        sizeClasses[size].container,
      ]"
    >
      <span
        class="block h-full w-full bg-current transition-colors"
        :style="{
          transform: `scale(${sizeClasses[size].scale})`,
          maskImage: `url('/logo.svg')`,
          maskPosition: 'center',
          maskRepeat: 'no-repeat',
          maskSize: 'contain',
          WebkitMaskImage: `url('/logo.svg')`,
          WebkitMaskPosition: 'center',
          WebkitMaskRepeat: 'no-repeat',
          WebkitMaskSize: 'contain',
        }"
      />
    </div>
    <div
      v-if="showText"
      :class="['flex flex-col min-w-0', direction === 'vertical' && 'items-center text-center']"
    >
      <span class="text-sm font-semibold tracking-tight">{{ text }}</span>
      <span class="text-xs text-muted-foreground">{{ subtitle }}</span>
    </div>
  </div>
</template>
