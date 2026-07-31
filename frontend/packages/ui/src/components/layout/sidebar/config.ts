import type { Component } from 'vue'

/**
 * 侧边栏菜单项
 */
export interface LayoutSidebarMenuItem {
  /** 唯一标识 */
  key: string
  /** 菜单标题 */
  title: string
  /** 路由路径 */
  path: string
  /** 图标名称或组件 */
  icon?: Component | string
  /** 子菜单 */
  children?: LayoutSidebarMenuItem[]
  /** 是否禁用 */
  disabled?: boolean
  /** 是否默认展开 */
  defaultExpanded?: boolean
  /** 徽标数量 */
  badge?: number | string
}

/**
 * 侧边栏配置
 */
export interface SidebarConfig {
  /** 菜单列表 */
  menus: LayoutSidebarMenuItem[]
  /** 折叠状态宽度（px） */
  collapsedWidth: number
  /** 展开状态最小宽度（px） */
  minWidth: number
  /** 展开状态最大宽度（px） */
  maxWidth: number
  /** 默认展开宽度（px） */
  defaultWidth: number
}

/**
 * 默认侧栏配置
 */
export const defaultSidebarConfig: SidebarConfig = {
  collapsedWidth: 64,
  minWidth: 184,
  maxWidth: 320,
  defaultWidth: 232,
  menus: [],
}
