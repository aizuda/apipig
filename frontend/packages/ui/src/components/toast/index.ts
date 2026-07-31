/**
 * Toast 模块导出
 */

export { default as Toast } from './Toast.vue'
export { default as ToastContainer } from './ToastContainer.vue'
export { default as ToastItem } from './ToastItem.vue'
export { useToast, toast } from './useToast'

export type {
  ToastType,
  ToastPosition,
  ToastOptions,
  ToastData,
  ToasterProps,
  PromiseToastOptions,
} from './types'

export {
  DEFAULT_TOAST_DURATION,
  DEFAULT_TOAST_POSITION,
  DEFAULT_VISIBLE_TOASTS,
  DEFAULT_GAP,
} from './types'
