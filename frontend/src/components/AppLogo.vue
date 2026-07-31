<script setup lang="ts">
interface Props {
  size?: 'sm' | 'md' | 'lg'
}

withDefaults(defineProps<Props>(), {
  size: 'md',
})

const sizeClasses = {
  sm: 'h-9 w-9',
  md: 'h-10 w-10',
  lg: 'h-12 w-12',
}
</script>

<template>
  <span
    :class="[
      'inline-flex shrink-0 items-center justify-center text-primary transition-transform duration-300 hover:scale-105',
      sizeClasses[size],
    ]"
    aria-label="ApiPig"
    role="img"
  >
    <span
      class="block h-full w-full bg-primary transition-colors"
      :style="{
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
  </span>
</template>
