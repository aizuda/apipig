/** * Toast 组件 * Toast 通知系统入口组件 */

<script lang="ts" setup>
import { cn } from '@tabtab/utils'
import ToastContainer from './ToastContainer.vue'
import type { ToasterProps } from './types'
import { DEFAULT_TOAST_DURATION, DEFAULT_TOAST_POSITION } from './types'
import { toast } from './useToast'

defineOptions({
  name: 'Toast',
})

const props = withDefaults(defineProps<ToasterProps>(), {
  position: DEFAULT_TOAST_POSITION,
  duration: DEFAULT_TOAST_DURATION,
  visibleToasts: 3,
  closeButton: true,
  expand: false,
  gap: 12,
  offset: '16px',
})

defineExpose({
  toast,
})
</script>

<template>
  <ToastContainer
    :class="cn('toaster', props.class)"
    :position="props.position"
    :duration="props.duration"
    :visible-toasts="props.visibleToasts"
    :close-button="props.closeButton"
    :expand="props.expand"
    :gap="props.gap"
    :offset="props.offset"
    v-bind="$attrs"
  />
</template>
