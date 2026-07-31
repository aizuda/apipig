import type { Component, HTMLAttributes } from 'vue'

export type LayoutMode =
  | 'sidebar'
  | 'top-nav'
  | 'mixed'
  | 'double-sidebar'
  | 'mixed-double'
  | 'fullscreen'
  | 'centered'
export type LayoutVariant = 'fixed' | 'fluid' | 'compact' | 'streamer'

/**
 * 菜单项接口定义
 */
export interface MenuItem {
  id: string
  name: string
  title?: string
  i18nKey?: string
  path: string
  icon?: Component | string
  localIcon?: string
  hideInMenu?: boolean
  hideInTab?: boolean
  href?: string
  order?: number
  badge?: string | number
  keepAlive?: boolean
  affixTab?: boolean
  affixTabOrder?: number
  activeMenu?: string
  permissions?: string[]
  children?: MenuItem[]
}

export interface LayoutProps {
  mode?: LayoutMode
  variant?: LayoutVariant
  collapsed?: boolean
  class?: HTMLAttributes['class']
}

export interface LayoutHeaderProps {
  fixed?: boolean
  bordered?: boolean
  class?: HTMLAttributes['class']
  slots?: {
    logo?: true
    breadcrumb?: true
    nav?: true
    actions?: true
    default?: true
  }
}

export interface LayoutSidebarProps {
  side?: 'left' | 'right'
  collapsible?: 'offcanvas' | 'icon' | 'none'
  static?: boolean
  class?: HTMLAttributes['class']
}

export interface LayoutContentProps {
  scrollable?: boolean
  fixedHeader?: boolean
  class?: HTMLAttributes['class']
}

export interface LayoutFooterProps {
  fixed?: boolean
  bordered?: boolean
  class?: HTMLAttributes['class']
}

export interface LayoutPageProps {
  class?: HTMLAttributes['class']
}

export interface LayoutPageHeaderProps {
  title?: string
  description?: string
  class?: HTMLAttributes['class']
}

export interface LayoutPageBodyProps {
  class?: HTMLAttributes['class']
}

export interface LayoutSearchProps {
  placeholder?: string
  description?: string
  shortcutKey?: string
  shortcutModifiers?: string[]
  menuItems?: MenuItem[]
  recentItems?: MenuItem[]
  maxRecentItems?: number
  showRecent?: boolean
  groupTitles?: {
    recent?: string
    navigation?: string
    actions?: string
  }
  emptyText?: string
}

export interface TabItem {
  key: string
  name: string
  title: string
  path: string
  fullPath: string
  icon?: Component
  affix?: boolean
  keepAlive?: boolean
  isRefreshing?: boolean
  isLoading?: boolean
  menuText?: {
    refresh?: string
    pin?: string
    unpin?: string
    close?: string
    closeOthers?: string
    closeLeft?: string
    closeRight?: string
    closeAll?: string
    scrollToStart?: string
    scrollToEnd?: string
  }
}
