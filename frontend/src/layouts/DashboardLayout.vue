<script setup lang="ts">
import {
  Layout,
  LayoutHeader,
  LayoutContent,
  LayoutPage,
  LayoutDoubleSidebar,
  LayoutSidebarApp,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  LayoutTopNavItem,
  isComponent,
  type LayoutSidebarMenuItem,
  type MenuItem,
  LayoutSidebarUserAvatar,
  LayoutSidebarFooter,
  LayoutDoubleSidebarFooter,
  LayoutTabs,
  LayoutSidebarTrigger,
} from '@tabtab/ui'
import { computed, onMounted, onUnmounted, ref, watch, type WatchStopHandle, KeepAlive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'
import { useTabsStore } from '@/stores/tabs'
import { useRouteMenu, handleExternalLink } from '@/composables/useRouteMenu'
import HeaderActions from '@/components/HeaderActions.vue'
import AppLogo from '@/components/AppLogo.vue'

/**
 * DashboardLayout - 仪表盘主布局组件
 * 支持多种布局模式：sidebar, top-nav, mixed, double-sidebar, mixed-double
 */
const router = useRouter()
const route = useRoute()
const { t, locale } = useI18n()

// 监听语言变化，更新标签页标题
watch(locale, () => {
  tabsStore.updateAllTabsTitle((tab) => {
    if (tab.titleKey) {
      return t(tab.titleKey)
    }
    return tab.title
  })
  tabsStore.updateAllTabsMenuText()
})

const userStore = useUserStore()
const themeStore = useThemeStore()
const tabsStore = useTabsStore()

const { menuItems, visibleMenuItems, activeMenuItem, currentTopNavId } = useRouteMenu()

const activeTopNavId = ref('')
const activeDoubleSidebarId = ref('')

/**
 * watch 停止函数集合
 */
const watchStops: WatchStopHandle[] = []

watchStops.push(
  watch(
    currentTopNavId,
    (newId) => {
      if (newId) {
        activeTopNavId.value = newId
      }
    },
    { immediate: true },
  ),
)

watchStops.push(
  watch(
    () => route.name,
    (name) => {
      if (name) {
        activeDoubleSidebarId.value = name.toString()
      }
    },
    { immediate: true },
  ),
)

watchStops.push(
  watch(
    () => route.path,
    () => {
      if (activeMenuItem.value?.item) {
        activeDoubleSidebarId.value = activeMenuItem.value.item.id
      }
    },
    { immediate: true },
  ),
)

/**
 * 将 MenuItem 转换为 LayoutSidebarMenuItem
 */
function convertToSidebarMenuItem(item: MenuItem): LayoutSidebarMenuItem {
  return {
    key: item.id,
    title: item.name,
    path: item.path,
    icon: item.icon,
    badge: typeof item.badge === 'number' ? item.badge : undefined,
    children: item.children?.map(convertToSidebarMenuItem),
  }
}

/**
 * 获取当前顶部导航下的子菜单项（用于 mixed-double 模式）
 * 如果一级菜单有子菜单，显示子菜单
 * 如果一级菜单没有子菜单，显示该一级菜单本身
 */
const currentSidebarMenuItems = computed<MenuItem[]>(() => {
  const topNav = visibleMenuItems.value.find((item) => item.id === activeTopNavId.value)
  if (!topNav) {
    return []
  }
  if (topNav.children?.length) {
    return topNav.children
  }
  return [topNav]
})

/**
 * 获取混合布局模式下的侧栏菜单项
 */
const mixedSidebarMenuItems = computed<MenuItem[]>(() => {
  const currentTopNav = visibleMenuItems.value.find((item) => item.id === activeTopNavId.value)

  if (!currentTopNav) {
    return []
  }

  if (currentTopNav.children?.length) {
    return currentTopNav.children
  }

  return [currentTopNav]
})

/**
 * 获取双栏侧边栏菜单项
 */
const sidebarMenuItems = computed<MenuItem[]>(() => {
  if (isDoubleSidebarFixed.value) {
    return menuItems.value
  }
  return currentSidebarMenuItems.value
})

/**
 * 获取单栏侧边栏菜单项（LayoutSidebarMenuItem 格式）
 * sidebar 模式：显示所有菜单
 * mixed 模式：显示当前选中一级菜单的子菜单（或一级菜单本身如果没有子菜单）
 */
const sidebarMenus = computed<LayoutSidebarMenuItem[]>(() => {
  const items =
    themeStore.layoutMode === 'sidebar' ? visibleMenuItems.value : mixedSidebarMenuItems.value
  return items.map(convertToSidebarMenuItem)
})

/**
 * 当前激活的单栏侧边栏菜单项 key
 * 类似 activeDoubleSidebarId，用于 sidebar 模式
 */
const activeSidebarId = computed(() => {
  if (activeMenuItem.value?.item) {
    return activeMenuItem.value.item.id
  }
  return route.name?.toString() || ''
})

/**
 * 处理用户登出
 */
function handleLogout() {
  userStore.logout()
  router.push({ name: 'Login' })
}

function handleAccountSettings() {
  router.push({ name: 'SettingsAccount' })
}

/**
 * 处理侧边栏导航
 */
function handleSidebarNavigate(path: string) {
  router.push(path)
}

/**
 * 处理顶部导航点击
 */
function handleTopNavClick(item: MenuItem) {
  activeTopNavId.value = item.id
  if (handleExternalLink(item)) {
    return
  }
  if (item.children?.length) {
    const firstChild = item.children[0]
    if (firstChild) {
      activeDoubleSidebarId.value = firstChild.id
      if (!handleExternalLink(firstChild)) {
        router.push(firstChild.path)
      }
    }
  } else {
    router.push(item.path)
  }
}

/**
 * 处理顶部导航菜单项导航（用于 TopNavItem 组件）
 */
function handleTopNavNavigate(item: MenuItem) {
  if (handleExternalLink(item)) {
    return
  }
  const parentId = findParentId(item.id, visibleMenuItems.value)
  if (parentId) {
    activeTopNavId.value = parentId
  }
  activeDoubleSidebarId.value = item.id
  router.push(item.path)
}

/**
 * 查找菜单项的父级 ID
 */
function findParentId(targetId: string, items: MenuItem[], parentId?: string): string | undefined {
  for (const item of items) {
    if (item.id === targetId) {
      return parentId
    }
    if (item.children) {
      const found = findParentId(targetId, item.children, item.id)
      if (found !== undefined) {
        return found || item.id
      }
    }
  }
  return undefined
}

/**
 * 处理双栏侧边栏选择
 */
function handleDoubleSidebarSelect(parent: MenuItem, child?: MenuItem) {
  if (child?.path) {
    activeDoubleSidebarId.value = child.id
    router.push(child.path)
  } else if (parent?.path) {
    activeDoubleSidebarId.value = parent.id
    router.push(parent.path)
  }
}

const showTopNav = computed(
  () =>
    themeStore.layoutMode === 'top-nav' ||
    themeStore.layoutMode === 'mixed' ||
    themeStore.layoutMode === 'mixed-double',
)

const isDoubleSidebarFixed = computed(() => themeStore.layoutMode === 'double-sidebar')

const isMixedDouble = computed(() => themeStore.layoutMode === 'mixed-double')

const isMixed = computed(() => themeStore.layoutMode === 'mixed')

/**
 * 判断当前一级菜单是否有内容可显示
 * 只要有当前选中的一级菜单就显示侧栏
 */
const hasSidebarContent = computed(() => {
  const topNav = visibleMenuItems.value.find((item) => item.id === activeTopNavId.value)
  return !!topNav
})

const showHeader = computed(
  () => themeStore.layoutMode !== 'fullscreen' && themeStore.layoutMode !== 'centered',
)

const contentScrollable = computed(() => route.meta.contentScrollable !== false)

const hasSidebar = computed(
  () => themeStore.layoutMode === 'sidebar' || themeStore.layoutMode === 'mixed',
)

const hasDoubleSidebar = computed(
  () => themeStore.layoutMode === 'double-sidebar' || themeStore.layoutMode === 'mixed-double',
)

/**
 * 侧边栏折叠状态（使用 ref 以支持双向绑定）
 */
const sidebarCollapsed = ref(themeStore.sidebarCollapsed)
const sidebarHidden = ref(false)

/**
 * 同步 themeStore 的折叠状态到本地 ref
 */
watchStops.push(
  watch(
    () => themeStore.sidebarCollapsed,
    (value) => {
      sidebarCollapsed.value = value
    },
  ),
)

/**
 * 同步本地折叠状态到 themeStore
 */
watchStops.push(
  watch(sidebarCollapsed, (value) => {
    themeStore.setSidebarCollapsed(value)
  }),
)

onMounted(() => {
  tabsStore.addTab(route)
})

/**
 * 监听路由变化，自动添加标签
 */
watchStops.push(
  watch(
    () => route.fullPath,
    () => {
      tabsStore.addTab(route)
    },
  ),
)

/**
 * 处理标签点击切换
 */
function handleTabClick(key: string) {
  const tab = tabsStore.tabs.find((t) => t.key === key)
  if (tab && tab.fullPath !== route.fullPath) {
    router.push(tab.fullPath)
  }
}

/**
 * 处理标签关闭
 */
function handleTabClose(key: string) {
  const newActiveKey = tabsStore.removeTab(key)
  if (newActiveKey) {
    const tab = tabsStore.tabs.find((t) => t.key === newActiveKey)
    if (tab) {
      router.push(tab.fullPath)
    }
  }
}

/**
 * 处理关闭其他标签
 */
function handleCloseOther(key: string) {
  tabsStore.closeOtherTabs(key)
}

/**
 * 处理关闭左侧标签
 */
function handleCloseLeft(key: string) {
  tabsStore.closeLeftTabs(key)
}

/**
 * 处理关闭右侧标签
 */
function handleCloseRight(key: string) {
  tabsStore.closeRightTabs(key)
}

/**
 * 处理关闭所有标签
 */
function handleCloseAll() {
  const newActiveKey = tabsStore.closeAllTabs()
  if (newActiveKey) {
    const tab = tabsStore.tabs.find((t) => t.key === newActiveKey)
    if (tab) {
      router.push(tab.fullPath)
    }
  }
}

/**
 * 处理刷新标签
 */
function handleRefresh(key: string) {
  tabsStore.refreshTab(key)
}

/**
 * 处理切换固定状态
 */
function handleToggleAffix(key: string) {
  tabsStore.toggleTabAffix(key)
}

/**
 * 处理拖拽排序
 */
function handleReorder(fromKey: string, toKey: string) {
  tabsStore.reorderTabs(fromKey, toKey)
}

/**
 * 组件卸载时清理所有 watch 监听器
 */
onUnmounted(() => {
  watchStops.forEach((stop) => stop())
  watchStops.length = 0
})
</script>

<template>
  <Layout
    :mode="themeStore.layoutMode"
    :variant="themeStore.layoutVariant"
    v-model:collapsed="sidebarCollapsed"
    v-model:hidden="sidebarHidden"
    class="transition-colors duration-300"
  >
    <!-- mixed-double 模式 -->
    <template v-if="isMixedDouble">
      <LayoutHeader>
        <template #nav>
          <nav class="flex items-center gap-1" aria-label="主导航">
            <Button
              v-for="item in visibleMenuItems"
              :key="item.id"
              :variant="activeTopNavId === item.id ? 'secondary' : 'ghost'"
              size="sm"
              class="relative transition-all duration-200 group"
              @click="handleTopNavClick(item)"
            >
              <component
                :is="item.icon"
                v-if="item.icon && isComponent(item.icon)"
                class="h-4 w-4 mr-1.5"
                :class="
                  activeTopNavId === item.id
                    ? 'text-foreground'
                    : 'text-muted-foreground group-hover:text-foreground'
                "
              />
              <span
                :class="
                  activeTopNavId === item.id
                    ? 'text-foreground font-medium'
                    : 'text-muted-foreground group-hover:text-foreground'
                "
              >
                {{ item.name }}
              </span>
              <span
                v-if="activeTopNavId === item.id"
                class="absolute -bottom-1.5 left-1/2 -translate-x-1/2 w-4 h-0.5 bg-primary rounded-full shadow-sm shadow-primary/50"
              />
            </Button>
          </nav>
        </template>

        <template #actions>
          <HeaderActions
            :visible-menu-items="visibleMenuItems"
            :placeholder="t('common.searchMenu')"
            :description="t('common.searchDescription')"
            :empty-text="t('common.searchEmpty')"
            :group-titles="{
              recent: t('common.recentAccess'),
              navigation: t('common.navigation'),
              actions: t('common.actions'),
            }"
            :show-theme-settings="true"
          />
        </template>
      </LayoutHeader>

      <div class="flex flex-1 min-h-0 overflow-hidden h-full">
        <LayoutDoubleSidebar
          v-if="hasSidebarContent"
          :items="sidebarMenuItems"
          :active-id="activeDoubleSidebarId"
          :fixed="false"
          @select="handleDoubleSidebarSelect"
        >
          <template #logo>
            <div class="flex h-12 w-full items-center justify-center border-b border-border/30">
              <AppLogo size="sm" />
            </div>
          </template>
          <template #footer>
            <LayoutDoubleSidebarFooter
              :user-name="userStore.displayName"
              :user-email="userStore.displayEmail"
              :avatar-src="userStore.defaultAvatar"
              :avatar-fallback="userStore.avatarFallback"
              :label-logged-in="t('common.loggedIn')"
              :label-account-settings="t('common.accountSettings')"
              :label-logout="t('common.logout')"
              @logout="handleLogout"
              @settings="handleAccountSettings"
            />
          </template>
        </LayoutDoubleSidebar>

        <LayoutContent
          class="flex-1 min-w-0"
          :fixed-header="themeStore.tabsFixed"
          :scrollable="contentScrollable"
        >
          <template #header>
            <LayoutTabs
              v-if="themeStore.showTabs && tabsStore.tabs.length > 0"
              :tabs="tabsStore.tabs"
              :active-key="tabsStore.activeKey"
              :fixed="themeStore.tabsFixed"
              @update:active-key="handleTabClick"
              @close="handleTabClose"
              @close-other="handleCloseOther"
              @close-left="handleCloseLeft"
              @close-right="handleCloseRight"
              @close-all="handleCloseAll"
              @refresh="handleRefresh"
              @toggle-affix="handleToggleAffix"
              @reorder="handleReorder"
            />
          </template>
          <LayoutPage>
            <RouterView v-slot="{ Component, route: currentRoute }">
              <KeepAlive :include="tabsStore.cachedNames">
                <component :is="Component" :key="currentRoute.fullPath" />
              </KeepAlive>
            </RouterView>
          </LayoutPage>
        </LayoutContent>
      </div>
    </template>

    <!-- mixed 模式 -->
    <template v-else-if="isMixed">
      <div class="flex flex-col w-full min-h-0 flex-1 overflow-hidden h-full">
        <LayoutHeader>
          <template #nav>
            <nav class="flex items-center gap-1" aria-label="主导航">
              <Button
                v-for="item in visibleMenuItems"
                :key="item.id"
                :variant="activeTopNavId === item.id ? 'secondary' : 'ghost'"
                size="sm"
                class="relative transition-all duration-200 group"
                @click="handleTopNavClick(item)"
              >
                <component
                  :is="item.icon"
                  v-if="item.icon && isComponent(item.icon)"
                  class="h-4 w-4 mr-1.5"
                  :class="
                    activeTopNavId === item.id
                      ? 'text-foreground'
                      : 'text-muted-foreground group-hover:text-foreground'
                  "
                />
                <span
                  :class="
                    activeTopNavId === item.id
                      ? 'text-foreground font-medium'
                      : 'text-muted-foreground group-hover:text-foreground'
                  "
                >
                  {{ item.name }}
                </span>
                <span
                  v-if="activeTopNavId === item.id"
                  class="absolute -bottom-1.5 left-1/2 -translate-x-1/2 w-4 h-0.5 bg-primary rounded-full shadow-sm shadow-primary/50"
                />
              </Button>
            </nav>
          </template>

          <template #actions>
            <HeaderActions
              :visible-menu-items="visibleMenuItems"
              :placeholder="t('common.searchMenu')"
              :show-theme-settings="true"
            />
          </template>
        </LayoutHeader>

        <div class="flex flex-1 min-h-0 overflow-hidden h-full">
          <LayoutSidebarApp
            :menus="sidebarMenus"
            :collapsed="sidebarCollapsed"
            :width="themeStore.sidebarWidth"
            :active-id="activeSidebarId"
            class="h-full"
            @navigate="handleSidebarNavigate"
            @update:collapsed="sidebarCollapsed = $event"
          >
            <template #logo>
              <AppLogo size="sm" />
            </template>
            <template #sidebar-title>
              <div v-if="!sidebarCollapsed" class="flex flex-col min-w-0">
                <span class="text-sm font-bold tracking-tight truncate">{{
                  t('common.appName')
                }}</span>
                <span class="text-[10px] text-muted-foreground truncate">{{
                  t('common.admin')
                }}</span>
              </div>
            </template>
            <template #collapse-button>
              <span class="text-xs text-muted-foreground">{{ t('common.collapseSidebar') }}</span>
            </template>
            <template #expand-button>
              <span>{{ t('common.expandSidebar') }}</span>
            </template>
            <template #footer="{ collapsed }">
              <LayoutSidebarFooter
                :collapsed="collapsed"
                :user-name="userStore.displayName"
                :user-email="userStore.displayEmail"
                :avatar-src="userStore.defaultAvatar"
                :avatar-fallback="userStore.avatarFallback"
                :label-logged-in="t('common.loggedIn')"
                :label-account-settings="t('common.accountSettings')"
                :label-logout="t('common.logout')"
                @logout="handleLogout"
                @settings="handleAccountSettings"
              />
            </template>
          </LayoutSidebarApp>

          <LayoutContent
            class="flex-1 min-w-0"
            :fixed-header="themeStore.tabsFixed"
            :scrollable="contentScrollable"
          >
            <template #header>
              <LayoutTabs
                v-if="themeStore.showTabs && tabsStore.tabs.length > 0"
                :tabs="tabsStore.tabs"
                :active-key="tabsStore.activeKey"
                :fixed="themeStore.tabsFixed"
                @update:active-key="handleTabClick"
                @close="handleTabClose"
                @close-other="handleCloseOther"
                @close-left="handleCloseLeft"
                @close-right="handleCloseRight"
                @close-all="handleCloseAll"
                @refresh="handleRefresh"
                @toggle-affix="handleToggleAffix"
                @reorder="handleReorder"
              />
            </template>
            <LayoutPage>
              <RouterView v-slot="{ Component, route: currentRoute }">
                <KeepAlive :include="tabsStore.cachedNames">
                  <component :is="Component" :key="currentRoute.fullPath" />
                </KeepAlive>
              </RouterView>
            </LayoutPage>
          </LayoutContent>
        </div>
      </div>
    </template>

    <!-- sidebar 模式 -->
    <template v-else-if="hasSidebar">
      <LayoutSidebarApp
        v-show="!sidebarHidden"
        :menus="sidebarMenus"
        :collapsed="sidebarCollapsed"
        :width="themeStore.sidebarWidth"
        :active-id="activeSidebarId"
        @navigate="handleSidebarNavigate"
        @update:collapsed="sidebarCollapsed = $event"
      >
        <template #logo>
          <AppLogo size="sm" />
        </template>
        <template #sidebar-title>
          <div v-if="!sidebarCollapsed" class="flex flex-col min-w-0">
            <span class="text-sm font-bold tracking-tight truncate">{{ t('common.appName') }}</span>
            <span class="text-[10px] text-muted-foreground truncate">{{ t('common.admin') }}</span>
          </div>
        </template>
        <template #collapse-button>
          <span class="text-xs text-muted-foreground">{{ t('common.collapseSidebar') }}</span>
        </template>
        <template #expand-button>
          <span>{{ t('common.expandSidebar') }}</span>
        </template>
        <template #footer="{ collapsed }">
          <LayoutSidebarFooter
            :collapsed="collapsed"
            :user-name="userStore.displayName"
            :user-email="userStore.displayEmail"
            :avatar-src="userStore.defaultAvatar"
            :avatar-fallback="userStore.avatarFallback"
            :label-logged-in="t('common.loggedIn')"
            :label-account-settings="t('common.accountSettings')"
            :label-logout="t('common.logout')"
            @logout="handleLogout"
            @settings="handleAccountSettings"
          />
        </template>
      </LayoutSidebarApp>

      <LayoutContent
        :fixed-header="themeStore.tabsFixed"
        :scrollable="contentScrollable"
      >
        <template #header>
          <LayoutHeader v-if="showHeader">
            <template #breadcrumb>
              <LayoutSidebarTrigger
                :label-show="t('common.showSidebar')"
                :label-hide="t('common.hideSidebar')"
              />
              <Breadcrumb>
                <BreadcrumbList>
                  <BreadcrumbItem>
                    <BreadcrumbPage>{{ t('common.appName') }}</BreadcrumbPage>
                  </BreadcrumbItem>
                </BreadcrumbList>
              </Breadcrumb>
            </template>

            <template #actions>
              <HeaderActions
                :visible-menu-items="visibleMenuItems"
                :placeholder="t('common.searchMenu')"
                :show-theme-settings="true"
              />
            </template>
          </LayoutHeader>

          <LayoutTabs
            v-if="themeStore.showTabs && tabsStore.tabs.length > 0"
            :tabs="tabsStore.tabs"
            :active-key="tabsStore.activeKey"
            :fixed="themeStore.tabsFixed"
            @update:active-key="handleTabClick"
            @close="handleTabClose"
            @close-other="handleCloseOther"
            @close-left="handleCloseLeft"
            @close-right="handleCloseRight"
            @close-all="handleCloseAll"
            @refresh="handleRefresh"
            @toggle-affix="handleToggleAffix"
            @reorder="handleReorder"
          />
        </template>
        <LayoutPage>
          <RouterView v-slot="{ Component, route: currentRoute }">
            <KeepAlive :include="tabsStore.cachedNames">
              <component :is="Component" :key="currentRoute.fullPath" />
            </KeepAlive>
          </RouterView>
        </LayoutPage>
      </LayoutContent>
    </template>

    <!-- double-sidebar 模式 -->
    <template v-else-if="hasDoubleSidebar">
      <LayoutDoubleSidebar
        :items="sidebarMenuItems"
        :active-id="activeDoubleSidebarId"
        :fixed="true"
        @select="handleDoubleSidebarSelect"
      >
        <template #logo>
          <div class="flex h-12 w-full items-center justify-center border-b border-border/30">
            <AppLogo size="sm" />
          </div>
        </template>
        <template #footer>
          <LayoutDoubleSidebarFooter
            :user-name="userStore.displayName"
            :user-email="userStore.displayEmail"
            :avatar-src="userStore.defaultAvatar"
            :avatar-fallback="userStore.avatarFallback"
            :label-logged-in="t('common.loggedIn')"
            :label-account-settings="t('common.accountSettings')"
            :label-logout="t('common.logout')"
            @logout="handleLogout"
            @settings="handleAccountSettings"
          />
        </template>
      </LayoutDoubleSidebar>

      <LayoutContent
        :fixed-header="themeStore.tabsFixed"
        :scrollable="contentScrollable"
      >
        <template #header>
          <LayoutHeader v-if="showHeader">
            <template #breadcrumb>
              <Breadcrumb>
                <BreadcrumbList>
                  <BreadcrumbItem>
                    <BreadcrumbPage>ApiPig</BreadcrumbPage>
                  </BreadcrumbItem>
                </BreadcrumbList>
              </Breadcrumb>
            </template>

            <template #actions>
              <HeaderActions
                :visible-menu-items="visibleMenuItems"
                :placeholder="t('common.searchMenu')"
                :show-theme-settings="true"
              />
            </template>
          </LayoutHeader>

          <LayoutTabs
            v-if="themeStore.showTabs && tabsStore.tabs.length > 0"
            :tabs="tabsStore.tabs"
            :active-key="tabsStore.activeKey"
            :fixed="themeStore.tabsFixed"
            @update:active-key="handleTabClick"
            @close="handleTabClose"
            @close-other="handleCloseOther"
            @close-left="handleCloseLeft"
            @close-right="handleCloseRight"
            @close-all="handleCloseAll"
            @refresh="handleRefresh"
            @toggle-affix="handleToggleAffix"
            @reorder="handleReorder"
          />
        </template>
        <LayoutPage>
          <RouterView v-slot="{ Component, route: currentRoute }">
            <KeepAlive :include="tabsStore.cachedNames">
              <component :is="Component" :key="currentRoute.fullPath" />
            </KeepAlive>
          </RouterView>
        </LayoutPage>
      </LayoutContent>
    </template>

    <!-- top-nav 模式 -->
    <template v-else>
      <LayoutContent
        :fixed-header="themeStore.tabsFixed"
        :scrollable="contentScrollable"
      >
        <template #header>
          <LayoutHeader v-if="showHeader">
            <template #logo>
              <div class="flex items-center gap-2">
                <AppLogo size="sm" />
                <span class="font-semibold">{{ t('common.appName') }}</span>
              </div>
            </template>

            <template #nav>
              <nav v-if="showTopNav" class="flex items-center gap-1" aria-label="主导航">
                <LayoutTopNavItem
                  v-for="item in visibleMenuItems"
                  :key="item.id"
                  :item="item"
                  :active-id="activeSidebarId"
                  @navigate="handleTopNavNavigate"
                />
              </nav>
            </template>

            <template #actions>
              <div class="flex items-center gap-2">
                <HeaderActions
                  :visible-menu-items="visibleMenuItems"
                  :placeholder="t('common.searchMenu')"
                  :show-theme-settings="true"
                />
                <DropdownMenu v-if="themeStore.layoutMode === 'top-nav'">
                  <DropdownMenuTrigger as-child>
                    <Button
                      variant="ghost"
                      size="icon"
                      :aria-label="t('common.userMenu')"
                      class="relative group"
                    >
                      <LayoutSidebarUserAvatar
                        :src="userStore.defaultAvatar"
                        :name="userStore.displayName"
                        :fallback="userStore.avatarFallback"
                        size="md"
                      />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <div class="px-2 py-1.5 text-sm font-medium">
                      {{ userStore.displayName }}
                    </div>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem class="cursor-pointer" @click="handleAccountSettings">
                      <span>{{ t('menu.account') }}</span>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      class="cursor-pointer text-destructive focus:text-destructive"
                      @click="handleLogout"
                    >
                      <span>{{ t('common.logout') }}</span>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </template>
          </LayoutHeader>

          <LayoutTabs
            v-if="themeStore.showTabs && tabsStore.tabs.length > 0"
            :tabs="tabsStore.tabs"
            :active-key="tabsStore.activeKey"
            :fixed="themeStore.tabsFixed"
            @update:active-key="handleTabClick"
            @close="handleTabClose"
            @close-other="handleCloseOther"
            @close-left="handleCloseLeft"
            @close-right="handleCloseRight"
            @close-all="handleCloseAll"
            @refresh="handleRefresh"
            @toggle-affix="handleToggleAffix"
            @reorder="handleReorder"
          />
        </template>
        <LayoutPage>
          <RouterView v-slot="{ Component, route: currentRoute }">
            <KeepAlive :include="tabsStore.cachedNames">
              <component :is="Component" :key="currentRoute.fullPath" />
            </KeepAlive>
          </RouterView>
        </LayoutPage>
      </LayoutContent>
    </template>
  </Layout>
</template>
