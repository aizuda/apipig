/**
 * Toast Composable
 * 提供全局 Toast 通知功能
 */

import { computed, reactive } from 'vue'
import type { ToastData, ToastOptions, ToastPosition, PromiseToastOptions } from './types'
import { DEFAULT_TOAST_DURATION, DEFAULT_TOAST_POSITION } from './types'

/**
 * 生成唯一 ID
 */
function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`
}

/** 全局 Toast 状态 */
const toasts = reactive<ToastData[]>([])

/** 自动关闭定时器映射 */
const dismissTimers = new Map<string, ReturnType<typeof setTimeout>>()

/**
 * 创建 Toast 数据对象
 */
function createToastData(options: ToastOptions & { title: string }): ToastData {
  return {
    id: options.id || generateId(),
    title: options.title,
    description: options.description,
    type: options.type || 'default',
    position: options.position || DEFAULT_TOAST_POSITION,
    duration: options.duration ?? DEFAULT_TOAST_DURATION,
    closeButton: options.closeButton ?? true,
    class: options.class,
    icon: options.icon,
    important: options.important || false,
    createdAt: Date.now(),
    visible: true,
    onClick: options.onClick,
    onDismiss: options.onDismiss,
    onAutoClose: options.onAutoClose,
  }
}

/**
 * 设置自动关闭定时器
 */
function setDismissTimer(id: string, duration: number) {
  const existingTimer = dismissTimers.get(id)
  if (existingTimer) {
    clearTimeout(existingTimer)
    dismissTimers.delete(id)
  }

  if (duration === 0) return

  const timer = setTimeout(() => {
    dismiss(id)
  }, duration)

  dismissTimers.set(id, timer)
}

/**
 * 添加 Toast
 */
function add(toast: Omit<ToastData, 'createdAt' | 'visible'> & { id?: string }): string {
  const newToast: ToastData = {
    ...toast,
    id: toast.id || generateId(),
    createdAt: Date.now(),
    visible: true,
  }

  toasts.push(newToast)

  if (!newToast.important && newToast.duration > 0) {
    setDismissTimer(newToast.id, newToast.duration)
  }

  return newToast.id
}

/**
 * 更新 Toast
 */
function update(id: string, updates: Partial<ToastData>) {
  const index = toasts.findIndex((t) => t.id === id)
  if (index === -1) return

  Object.assign(toasts[index], updates)

  if (updates.duration !== undefined && !toasts[index].important) {
    setDismissTimer(id, updates.duration)
  }
}

/**
 * 移除 Toast
 */
function dismiss(id: string) {
  const index = toasts.findIndex((t) => t.id === id)
  if (index === -1) return

  const toast = toasts[index]

  toast.visible = false

  const timer = dismissTimers.get(id)
  if (timer) {
    clearTimeout(timer)
    dismissTimers.delete(id)
  }

  toast.onDismiss?.()

  setTimeout(() => {
    const currentIndex = toasts.findIndex((t) => t.id === id)
    if (currentIndex !== -1) {
      toasts.splice(currentIndex, 1)
    }
  }, 300)
}

/**
 * 移除所有 Toast
 */
function dismissAll() {
  dismissTimers.forEach((timer) => clearTimeout(timer))
  dismissTimers.clear()

  toasts.forEach((toast) => {
    toast.visible = false
    toast.onDismiss?.()
  })

  setTimeout(() => {
    toasts.splice(0, toasts.length)
  }, 300)
}

/**
 * 根据位置获取 Toast
 */
function getToastsByPosition(position: ToastPosition): ToastData[] {
  return toasts.filter((t) => t.position === position)
}

/**
 * 基础 Toast 函数
 */
function toast(message: string, options?: Omit<ToastOptions, 'title'>): string {
  return add(createToastData({ title: message, ...options }))
}

/**
 * 成功 Toast
 */
toast.success = (message: string, options?: Omit<ToastOptions, 'title' | 'type'>): string => {
  return add(createToastData({ title: message, type: 'success', ...options }))
}

/**
 * 错误 Toast
 */
toast.error = (message: string, options?: Omit<ToastOptions, 'title' | 'type'>): string => {
  return add(createToastData({ title: message, type: 'error', ...options }))
}

/**
 * 警告 Toast
 */
toast.warning = (message: string, options?: Omit<ToastOptions, 'title' | 'type'>): string => {
  return add(createToastData({ title: message, type: 'warning', ...options }))
}

/**
 * 信息 Toast
 */
toast.info = (message: string, options?: Omit<ToastOptions, 'title' | 'type'>): string => {
  return add(createToastData({ title: message, type: 'info', ...options }))
}

/**
 * 加载中 Toast
 */
toast.loading = (message: string, options?: Omit<ToastOptions, 'title' | 'type'>): string => {
  return add(
    createToastData({
      title: message,
      type: 'loading',
      duration: 0,
      ...options,
    }),
  )
}

/**
 * Promise Toast
 */
toast.promise = <T>(promise: Promise<T>, options: PromiseToastOptions<T>): Promise<T> => {
  const { loading, success, error, options: toastOptions } = options

  const id = toast.loading(loading, toastOptions)

  return promise
    .then((data) => {
      const successMessage = typeof success === 'function' ? success(data) : success
      update(id, {
        type: 'success',
        title: successMessage,
        duration: DEFAULT_TOAST_DURATION,
      })
      return data
    })
    .catch((err) => {
      const errorMessage = typeof error === 'function' ? error(err) : error
      update(id, {
        type: 'error',
        title: errorMessage,
        duration: DEFAULT_TOAST_DURATION,
      })
      throw err
    })
}

/**
 * 自定义 Toast
 */
toast.custom = (id: string, options: Partial<ToastOptions>) => {
  update(id, options)
}

/**
 * 移除指定 Toast
 */
toast.dismiss = dismiss

/**
 * 移除所有 Toast
 */
toast.dismissAll = dismissAll

/**
 * 使用 Toast Composable
 */
export function useToast() {
  return {
    toasts: computed(() => toasts),
    add,
    update,
    dismiss,
    dismissAll,
    getToastsByPosition,
  }
}

export { toast }
