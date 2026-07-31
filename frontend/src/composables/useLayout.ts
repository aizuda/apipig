import { computed, type ComputedRef, type Ref } from 'vue'
import type { MenuItem } from '@tabtab/ui'

export interface HeaderActionsOptions {
  visibleMenuItems: ComputedRef<MenuItem[]>
  t: (key: string) => string
}

export interface LayoutTabsOptions {
  themeStore: {
    showTabs: Ref<boolean>
    tabsFixed: Ref<boolean>
  }
  tabsStore: {
    tabs: Ref<any[]>
    activeKey: Ref<string>
  }
}

export function useHeaderActions({ visibleMenuItems, t }: HeaderActionsOptions) {
  const searchProps = computed(() => ({
    menuItems: visibleMenuItems.value,
    placeholder: t('common.searchMenu'),
    description: t('common.searchDescription'),
    emptyText: t('common.searchEmpty'),
    groupTitles: {
      recent: t('common.recentAccess'),
      navigation: t('common.navigation'),
      actions: t('common.actions'),
    },
  }))

  return {
    searchProps,
  }
}

export function useLayoutTabs(options: LayoutTabsOptions) {
  const showTabs = computed(
    () => options.themeStore.showTabs.value && options.tabsStore.tabs.value.length > 0,
  )

  const tabsProps = computed(() => ({
    tabs: options.tabsStore.tabs.value,
    activeKey: options.tabsStore.activeKey.value,
    fixed: options.themeStore.tabsFixed.value,
  }))

  return {
    showTabs,
    tabsProps,
  }
}
