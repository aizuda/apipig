import type { Component } from 'vue'
import type { RouteLocationNormalized } from 'vue-router'
import { defineStore } from 'pinia'
import { ref, computed, type WatchStopHandle } from 'vue'
import { useLocalStorage } from '@vueuse/core'
import { i18n } from '@/i18n'

/**
 * 标签项接口定义
 */
export interface TabItem {
  /** 唯一标识，使用 route.fullPath */
  key: string
  /** 路由名称 */
  name: string
  /** 标签标题 */
  title: string
  /** 标签标题的 i18n key */
  titleKey?: string
  /** 路由路径 */
  path: string
  /** 完整路径（包含查询参数） */
  fullPath: string
  /** 图标组件 */
  icon?: Component
  /** 是否固定 */
  affix?: boolean
  /** 是否缓存 */
  keepAlive?: boolean
  /** 是否正在刷新 */
  isRefreshing?: boolean
  /** 是否正在加载 */
  isLoading?: boolean
  /** 标签操作菜单翻译文本 */
  menuText?: {
    refresh?: string
    close?: string
    closeOthers?: string
    closeLeft?: string
    closeRight?: string
    closeAll?: string
    pin?: string
    unpin?: string
    scrollToStart?: string
    scrollToEnd?: string
  }
}

/**
 * 标签状态管理 Store
 * 管理当前打开的标签页列表，支持添加、关闭、固定、拖拽排序等操作
 */
export const useTabsStore = defineStore('tabs', () => {
  /**
   * 标签页列表
   */
  const tabs = useLocalStorage<TabItem[]>('tabs-list', [])

  /**
   * 当前激活的标签 key
   */
  const activeKey = ref<string>('')

  /**
   * 当前激活的标签
   */
  const activeTab = computed(() => tabs.value.find((tab) => tab.key === activeKey.value))

  /**
   * 固定标签列表
   */
  const affixTabs = computed(() => tabs.value.filter((tab) => tab.affix))

  /**
   * 非固定标签列表
   */
  const normalTabs = computed(() => tabs.value.filter((tab) => !tab.affix))

  /**
   * 可关闭的标签列表
   */
  const closableTabs = computed(() => tabs.value.filter((tab) => !tab.affix))

  /**
   * 需要缓存的标签名称列表（用于 KeepAlive）
   */
  const cachedNames = computed(() =>
    tabs.value.filter((tab) => tab.keepAlive && tab.name).map((tab) => tab.name),
  )

  /**
   * watch 停止函数集合
   */
  const watchStops: WatchStopHandle[] = []

  /**
   * 添加标签
   * @param route - 路由对象
   */
  function addTab(route: RouteLocationNormalized) {
    const { meta, name, path, fullPath } = route
    const t = i18n.global.t

    if (meta.hideInTab || !name) {
      return
    }

    const tabKey = fullPath || path
    const existingTab = tabs.value.find((tab) => tab.key === tabKey)

    const titleKey = meta.titleKey as string | undefined
    const translatedTitle = titleKey ? t(titleKey) : (meta.title as string) || name.toString()

    if (existingTab) {
      existingTab.title = translatedTitle
      existingTab.titleKey = titleKey
      existingTab.name = name as string
      existingTab.icon = meta.icon as Component | undefined
      existingTab.keepAlive = meta.keepAlive as boolean | undefined
      activeKey.value = tabKey
      return
    }

    const newTab: TabItem = {
      key: tabKey,
      name: name as string,
      title: translatedTitle,
      titleKey,
      path,
      fullPath,
      icon: meta.icon as Component | undefined,
      affix: meta.affixTab as boolean | undefined,
      keepAlive: meta.keepAlive as boolean | undefined,
      menuText: {
        refresh: t('common.tabs.refreshPage'),
        close: t('common.tabs.closeTab'),
        closeOthers: t('common.tabs.closeOthers'),
        closeLeft: t('common.tabs.closeLeft'),
        closeRight: t('common.tabs.closeRight'),
        closeAll: t('common.tabs.closeAll'),
        pin: t('common.tabs.pin'),
        unpin: t('common.tabs.unpin'),
        scrollToStart: t('common.tabs.scrollToStart'),
        scrollToEnd: t('common.tabs.scrollToEnd'),
      },
    }

    if (newTab.affix) {
      const affixIndex = tabs.value.findIndex((tab) => tab.affix)
      if (affixIndex === -1) {
        tabs.value.unshift(newTab)
      } else {
        const lastAffixIndex = tabs.value.reduce(
          (lastIdx, tab, idx) => (tab.affix ? idx : lastIdx),
          -1,
        )
        tabs.value.splice(lastAffixIndex + 1, 0, newTab)
      }
    } else {
      tabs.value.push(newTab)
    }

    activeKey.value = tabKey
  }

  /**
   * 移除标签
   * @param key - 标签 key
   * @returns 关闭后应该激活的标签 key，如果无法关闭则返回 null
   */
  function removeTab(key: string): string | null {
    const index = tabs.value.findIndex((tab) => tab.key === key)

    if (index === -1) {
      return null
    }

    const tab = tabs.value[index]

    if (tab.affix) {
      return null
    }

    tabs.value.splice(index, 1)

    if (activeKey.value === key) {
      const newActiveIndex = Math.min(index, tabs.value.length - 1)
      return tabs.value[newActiveIndex]?.key || null
    }

    return null
  }

  /**
   * 关闭其他标签（保留当前和固定标签）
   * @param key - 当前标签 key
   */
  function closeOtherTabs(key: string) {
    tabs.value = tabs.value.filter((tab) => tab.key === key || tab.affix)
  }

  /**
   * 关闭左侧标签
   * @param key - 当前标签 key
   */
  function closeLeftTabs(key: string) {
    const index = tabs.value.findIndex((tab) => tab.key === key)
    if (index === -1) return

    tabs.value = tabs.value.filter((tab, idx) => idx >= index || tab.affix)
  }

  /**
   * 关闭右侧标签
   * @param key - 当前标签 key
   */
  function closeRightTabs(key: string) {
    const index = tabs.value.findIndex((tab) => tab.key === key)
    if (index === -1) return

    tabs.value = tabs.value.filter((tab, idx) => idx <= index || tab.affix)
  }

  /**
   * 关闭所有标签（保留固定标签）
   * @returns 应该激活的固定标签 key
   */
  function closeAllTabs(): string | null {
    tabs.value = tabs.value.filter((tab) => tab.affix)
    return tabs.value[0]?.key || null
  }

  /**
   * 刷新标签页
   * @param key - 标签 key
   */
  function refreshTab(key: string) {
    const tab = tabs.value.find((t) => t.key === key)
    if (tab) {
      tab.isRefreshing = true
      setTimeout(() => {
        tab.isRefreshing = false
      }, 500)
      window.dispatchEvent(new CustomEvent('tab-refresh', { detail: { key, path: tab.path } }))
    }
  }

  /**
   * 设置标签页加载状态
   * @param key - 标签 key
   * @param loading - 是否加载中
   */
  function setTabLoading(key: string, loading: boolean) {
    const tab = tabs.value.find((t) => t.key === key)
    if (tab) {
      tab.isLoading = loading
    }
  }

  /**
   * 切换标签固定状态
   * @param key - 标签 key
   */
  function toggleTabAffix(key: string) {
    const tab = tabs.value.find((t) => t.key === key)
    if (tab) {
      tab.affix = !tab.affix

      if (tab.affix) {
        const currentIndex = tabs.value.indexOf(tab)
        tabs.value.splice(currentIndex, 1)

        const lastAffixIndex = tabs.value.reduce((lastIdx, t, idx) => (t.affix ? idx : lastIdx), -1)
        tabs.value.splice(lastAffixIndex + 1, 0, tab)
      }
    }
  }

  /**
   * 设置当前激活标签
   * @param key - 标签 key
   */
  function setActiveKey(key: string) {
    activeKey.value = key
  }

  /**
   * 激活标签页
   * @param key - 标签 key
   */
  function activateTab(key: string) {
    const tab = tabs.value.find((t) => t.key === key)
    if (tab) {
      activeKey.value = key
    }
  }

  /**
   * 更新标签页标题
   * @param key - 标签 key
   * @param title - 新标题
   */
  function updateTabTitle(key: string, title: string) {
    const tab = tabs.value.find((t) => t.key === key)
    if (tab) {
      tab.title = title
    }
  }

  /**
   * 更新所有标签页标题
   * @param getTitle - 获取标题的函数
   */
  function updateAllTabsTitle(getTitle: (tab: TabItem) => string) {
    tabs.value.forEach((tab) => {
      tab.title = getTitle(tab)
    })
  }

  /**
   * 更新所有标签页菜单翻译文本
   * 用于语言切换时更新菜单文本
   */
  function updateAllTabsMenuText() {
    const t = i18n.global.t
    tabs.value.forEach((tab) => {
      tab.menuText = {
        refresh: t('common.tabs.refreshPage'),
        close: t('common.tabs.closeTab'),
        closeOthers: t('common.tabs.closeOthers'),
        closeLeft: t('common.tabs.closeLeft'),
        closeRight: t('common.tabs.closeRight'),
        closeAll: t('common.tabs.closeAll'),
        pin: t('common.tabs.pin'),
        unpin: t('common.tabs.unpin'),
        scrollToStart: t('common.tabs.scrollToStart'),
        scrollToEnd: t('common.tabs.scrollToEnd'),
      }
    })
  }

  /**
   * 拖拽排序标签
   * @param fromKey - 源标签 key
   * @param toKey - 目标标签 key
   */
  function reorderTabs(fromKey: string, toKey: string) {
    const fromIndex = tabs.value.findIndex((t) => t.key === fromKey)
    const toIndex = tabs.value.findIndex((t) => t.key === toKey)

    if (fromIndex !== -1 && toIndex !== -1) {
      const [movedTab] = tabs.value.splice(fromIndex, 1)
      tabs.value.splice(toIndex, 0, movedTab)
    }
  }

  /**
   * 根据路由初始化固定标签
   * @param routes - 路由列表
   */
  function initAffixTabs(routes: RouteLocationNormalized[]) {
    const t = i18n.global.t

    routes.forEach((route) => {
      if (route.meta?.affixTab && !route.meta?.hideInTab && route.name) {
        const existingTab = tabs.value.find((tab) => tab.key === route.fullPath)
        if (!existingTab) {
          const titleKey = route.meta.titleKey as string | undefined
          const translatedTitle = titleKey
            ? t(titleKey)
            : (route.meta.title as string) || route.name.toString()

          const newTab: TabItem = {
            key: route.fullPath,
            name: route.name as string,
            title: translatedTitle,
            titleKey,
            path: route.path,
            fullPath: route.fullPath,
            icon: route.meta.icon as Component | undefined,
            affix: true,
            keepAlive: route.meta.keepAlive as boolean | undefined,
            menuText: {
              refresh: t('common.tabs.refreshPage'),
              close: t('common.tabs.closeTab'),
              closeOthers: t('common.tabs.closeOthers'),
              closeLeft: t('common.tabs.closeLeft'),
              closeRight: t('common.tabs.closeRight'),
              closeAll: t('common.tabs.closeAll'),
              pin: t('common.tabs.pin'),
              unpin: t('common.tabs.unpin'),
              scrollToStart: t('common.tabs.scrollToStart'),
              scrollToEnd: t('common.tabs.scrollToEnd'),
            },
          }
          tabs.value.unshift(newTab)
        }
      }
    })
  }

  /**
   * 重置标签列表
   */
  function resetTabs() {
    tabs.value = tabs.value.filter((tab) => tab.affix)
    activeKey.value = tabs.value[0]?.key || ''
  }

  /**
   * 清理 watch 监听器
   */
  function cleanup() {
    watchStops.forEach((stop) => stop())
    watchStops.length = 0
  }

  return {
    tabs,
    activeKey,
    activeTab,
    affixTabs,
    normalTabs,
    closableTabs,
    cachedNames,
    addTab,
    removeTab,
    closeOtherTabs,
    closeLeftTabs,
    closeRightTabs,
    closeAllTabs,
    refreshTab,
    setTabLoading,
    toggleTabAffix,
    setActiveKey,
    activateTab,
    updateTabTitle,
    updateAllTabsTitle,
    updateAllTabsMenuText,
    reorderTabs,
    initAffixTabs,
    resetTabs,
    cleanup,
  }
})
