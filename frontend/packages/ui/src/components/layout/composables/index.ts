export {
  useLayoutSidebar,
  defaultSidebarConfig,
  type SidebarConfig,
  type LayoutSidebarMenuItem,
} from './useSidebar'
export {
  useMenuUtils,
  formatBadge,
  getButtonVariant,
  flattenMenus,
  findMenuByPath,
  pxToPercent,
  isComponent,
  type UseMenuUtilsOptions,
  type MatchMode,
} from './useMenuUtils'

/**
 * 百分比转像素
 */
export function percentToPx(percent: number, containerWidth: number): number {
  return (percent / 100) * containerWidth
}
