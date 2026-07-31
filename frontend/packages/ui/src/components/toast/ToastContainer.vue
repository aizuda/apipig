/** * ToastContainer 组件 * Toast 通知容器，管理多个 Toast 项的显示、位置和动画 */

<script lang="ts" setup>
import { computed } from 'vue'
import { cn } from '@tabtab/utils'
import { useToast } from './useToast'
import ToastItem from './ToastItem.vue'
import type { ToastPosition, ToasterProps } from './types'
import {
  DEFAULT_TOAST_DURATION,
  DEFAULT_TOAST_POSITION,
  DEFAULT_VISIBLE_TOASTS,
  DEFAULT_GAP,
} from './types'

const props = withDefaults(defineProps<ToasterProps>(), {
  position: DEFAULT_TOAST_POSITION,
  duration: DEFAULT_TOAST_DURATION,
  visibleToasts: DEFAULT_VISIBLE_TOASTS,
  closeButton: true,
  expand: false,
  gap: DEFAULT_GAP,
  offset: '16px',
})

const { toasts, dismiss } = useToast()

/**
 * 根据位置获取 Toast 列表
 */
const getToastsByPosition = (position: ToastPosition) => {
  return toasts.value.filter((t) => t.position === position).slice(-props.visibleToasts)
}

/**
 * 所有 Toast 位置
 */
const positions: ToastPosition[] = [
  'top-left',
  'top-center',
  'top-right',
  'bottom-left',
  'bottom-center',
  'bottom-right',
]

/**
 * 获取位置的样式类名
 */
const getPositionClasses = (position: ToastPosition) => {
  const baseClasses = 'fixed z-[100] flex flex-col pointer-events-none'

  switch (position) {
    case 'top-left':
      return cn(baseClasses, 'top-0 left-0 items-start')
    case 'top-center':
      return cn(baseClasses, 'top-0 left-1/2 -translate-x-1/2 items-center')
    case 'top-right':
      return cn(baseClasses, 'top-0 right-0 items-end')
    case 'bottom-left':
      return cn(baseClasses, 'bottom-0 left-0 items-start flex-col-reverse')
    case 'bottom-center':
      return cn(baseClasses, 'bottom-0 left-1/2 -translate-x-1/2 items-center flex-col-reverse')
    case 'bottom-right':
      return cn(baseClasses, 'bottom-0 right-0 items-end flex-col-reverse')
    default:
      return baseClasses
  }
}

/**
 * 获取 Toast 容器的样式
 */
const getContainerStyle = (position: ToastPosition) => {
  const style: Record<string, string> = {}

  const offsetValue = typeof props.offset === 'number' ? `${props.offset}px` : props.offset

  if (position.startsWith('top')) {
    style.top = offsetValue
  } else {
    style.bottom = offsetValue
  }

  if (position.endsWith('left')) {
    style.left = offsetValue
  } else if (position.endsWith('right')) {
    style.right = offsetValue
  }

  style.gap = `${props.gap}px`

  return style
}

/**
 * 处理关闭事件
 */
const handleDismiss = (id: string) => {
  dismiss(id)
}

/**
 * 计算容器类名
 */
const containerClasses = computed(() => {
  return cn('toaster', props.class)
})
</script>

<template>
  <Teleport to="body">
    <div :class="containerClasses">
      <div
        v-for="position in positions"
        :key="position"
        :class="getPositionClasses(position)"
        :style="getContainerStyle(position)"
      >
        <TransitionGroup
          name="toast"
          tag="div"
          class="flex flex-col w-full px-4"
          :class="position.includes('bottom') ? 'flex-col-reverse' : 'flex-col'"
          :style="{ gap: `${gap}px` }"
        >
          <ToastItem
            v-for="toast in getToastsByPosition(position)"
            :key="toast.id"
            :toast="toast"
            class="pointer-events-auto"
            @dismiss="handleDismiss"
          />
        </TransitionGroup>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-move,
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.toast-enter-from {
  opacity: 0;
  transform: translateY(-100%) scale(0.95);
}

.toast-leave-to {
  opacity: 0;
  transform: scale(0.95);
}

.toast-leave-active {
  position: absolute;
  width: 100%;
}

[data-position^='bottom'] .toast-enter-from {
  transform: translateY(100%) scale(0.95);
}

[data-position^='top'] .toast-enter-from {
  transform: translateY(-100%) scale(0.95);
}
</style>
