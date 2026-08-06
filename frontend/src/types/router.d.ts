import type { Component } from 'vue'
import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    /** i18n 翻译 key（用于菜单和页面标题翻译） */
    titleKey?: string
    /** 页面标题，显示在菜单和标签页 */
    title?: string
    /** 菜单图标 */
    icon?: Component | string
    /** 本地 SVG 图标名（assets/svg-icon 文件夹的 svg 文件名） */
    localIcon?: string
    /** 是否在菜单中隐藏 */
    hideInMenu?: boolean
    /** 是否在标签页中隐藏 */
    hideInTab?: boolean
    /** 是否需要登录权限 */
    requiresAuth?: boolean
    /** 权限标识（哪些角色可以访问） */
    permissions?: string[]
    /** 是否缓存页面 */
    keepAlive?: boolean
    /** 是否固定标签页 */
    affixTab?: boolean
    /** 固定标签页排序 */
    affixTabOrder?: number
    /** 当前路由需要选中的菜单项（用于跳转至不在菜单显示的路由且需要高亮某个菜单） */
    activeMenu?: string
    /** 菜单排序 */
    order?: number
    /** 外链地址 */
    href?: string
    /** 是否支持多个 tab 页签（相同 name 的路由不会被替换） */
    multiTab?: boolean
    /** 徽标 */
    badge?: string | number
    /** 是否隐藏子菜单 */
    hideChildrenInMenu?: boolean
    /** 当前路由的父级菜单 key */
    parentKey?: string
    /** AI 网关当前展示的管理模块 */
    /** iframe 地址 */
    iframeSrc?: string
    /** Whether the layout content area provides vertical scrolling. */
    contentScrollable?: boolean
    /** 忽略权限（开发模式） */
    ignoreAuth?: boolean
  }
}
