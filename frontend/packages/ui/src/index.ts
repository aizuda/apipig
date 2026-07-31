export * from './components'

export type {
  LayoutMode,
  LayoutVariant,
  LayoutProps,
  LayoutHeaderProps,
  LayoutSidebarProps,
  LayoutContentProps,
  LayoutFooterProps,
  LayoutPageProps,
  LayoutPageHeaderProps,
  LayoutPageBodyProps,
  MenuItem,
} from './components/layout/types'

export {
  resolvedRoutesToMenuItems,
  getFlattenedMenuMap,
  findActiveMenuItem,
  filterMenuByPermission,
  getAffixTabs,
} from './components/layout/utils/menu'

export {
  Layout,
  LayoutHeader,
  LayoutContent,
  LayoutFooter,
  LayoutPage,
  LayoutPageHeader,
  LayoutPageBody,
} from './components/layout/core'

export { LayoutSearch } from './components/layout/search'
export type { LayoutSearchProps } from './components/layout'

export {
  LayoutDoubleSidebar,
  LayoutDoubleSidebarMenu,
  LayoutDoubleSidebarFooter,
} from './components/layout/double-sidebar'

export {
  LayoutTopNavItem,
  LayoutTopNavDropdownItem,
  LayoutTabs,
  LayoutTabsItem,
  LayoutSidebarTrigger,
} from './components/layout/navigation'

export type { TabItem } from './components/layout'

export {
  LayoutSidebarApp,
  LayoutSidebarDesktop,
  LayoutSidebarMobile,
  LayoutSidebarItem,
  LayoutSidebarFooter,
  LayoutSidebarLogo,
  LayoutSidebarUserAvatar,
  defaultSidebarConfig,
  type SidebarConfig,
  type LayoutSidebarMenuItem,
} from './components/layout/sidebar'

export { SidebarResizable, LayoutSidebarMenuItemComponent } from './components/layout/sidebar'

export {
  useLayoutSidebar,
  useMenuUtils,
  pxToPercent,
  percentToPx,
  isComponent,
  formatBadge,
  getButtonVariant,
  type UseMenuUtilsOptions,
  type MatchMode,
} from './components/layout/composables'

export { useLayout, provideLayoutContext, createLayoutComputed } from './components/layout/utils'
export type { LayoutContextValue } from './components/layout/utils'

export { Toast, ToastContainer, ToastItem, useToast, toast } from './components/toast'
export type {
  ToastType,
  ToastPosition,
  ToastOptions,
  ToastData,
  ToasterProps,
  PromiseToastOptions,
} from './components/toast'
export {
  DEFAULT_TOAST_DURATION,
  DEFAULT_TOAST_POSITION,
  DEFAULT_VISIBLE_TOASTS,
  DEFAULT_GAP,
} from './components/toast'
