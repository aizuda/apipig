/**
 * Toast 类型定义
 * 定义 Toast 通知系统的所有类型、接口和常量
 */

import type { Component } from 'vue'

/** Toast 类型 */
export type ToastType = 'default' | 'success' | 'error' | 'warning' | 'info' | 'loading'

/** Toast 位置 */
export type ToastPosition =
  | 'top-left'
  | 'top-center'
  | 'top-right'
  | 'bottom-left'
  | 'bottom-center'
  | 'bottom-right'

/** Toast 选项 */
export interface ToastOptions {
  /** Toast 唯一标识 */
  id?: string
  /** Toast 类型 */
  type?: ToastType
  /** 标题/主要内容 */
  title?: string
  /** 详细描述 */
  description?: string
  /** 显示位置 */
  position?: ToastPosition
  /** 自动关闭时间（毫秒），0 表示不自动关闭 */
  duration?: number
  /** 是否显示关闭按钮 */
  closeButton?: boolean
  /** 自定义 CSS 类名 */
  class?: string
  /** 自定义图标组件 */
  icon?: Component
  /** 是否重要通知（不自动关闭） */
  important?: boolean
  /** 点击回调 */
  onClick?: () => void
  /** 关闭回调 */
  onDismiss?: () => void
  /** 自动关闭回调 */
  onAutoClose?: () => void
}

/** Toast 数据对象 */
export interface ToastData {
  /** 唯一标识 */
  id: string
  /** Toast 类型 */
  type: ToastType
  /** 标题/主要内容 */
  title: string
  /** 详细描述 */
  description?: string
  /** 显示位置 */
  position: ToastPosition
  /** 自动关闭时间 */
  duration: number
  /** 是否显示关闭按钮 */
  closeButton: boolean
  /** 自定义 CSS 类名 */
  class?: string
  /** 自定义图标组件 */
  icon?: Component
  /** 是否重要通知 */
  important: boolean
  /** 创建时间 */
  createdAt: number
  /** 是否可见 */
  visible: boolean
  /** 点击回调 */
  onClick?: () => void
  /** 关闭回调 */
  onDismiss?: () => void
  /** 自动关闭回调 */
  onAutoClose?: () => void
}

/** Toaster 组件 Props */
export interface ToasterProps {
  /** Toast 显示位置 */
  position?: ToastPosition
  /** 默认自动关闭时间 */
  duration?: number
  /** 最大可见 Toast 数量 */
  visibleToasts?: number
  /** 是否显示关闭按钮 */
  closeButton?: boolean
  /** 自定义 CSS 类名 */
  class?: string
  /** 是否展开所有 Toast */
  expand?: boolean
  /** Toast 之间间距 */
  gap?: number
  /** 偏移量 */
  offset?: string | number
}

/** Promise Toast 选项 */
export interface PromiseToastOptions<T = unknown> {
  /** 加载中消息 */
  loading: string
  /** 成功消息 */
  success: string | ((data: T) => string)
  /** 错误消息 */
  error: string | ((error: unknown) => string)
  /** 其他选项 */
  options?: Omit<ToastOptions, 'type'>
}

/** 默认配置 */
export const DEFAULT_TOAST_DURATION = 4000
export const DEFAULT_TOAST_POSITION: ToastPosition = 'bottom-right'
export const DEFAULT_VISIBLE_TOASTS = 3
export const DEFAULT_GAP = 12
