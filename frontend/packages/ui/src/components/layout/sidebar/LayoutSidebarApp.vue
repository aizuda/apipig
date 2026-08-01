<script setup lang="ts">
import type { SidebarConfig, LayoutSidebarMenuItem } from './config'
import { computed, ref, watch, onUnmounted } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import { useLayoutSidebar } from '../composables'
import { defaultSidebarConfig } from './config'
import LayoutSidebarItem from './LayoutSidebarItem.vue'
import LayoutSidebarSubMenu from './LayoutSidebarSubMenu.vue'
import LayoutSidebarMobile from './LayoutSidebarMobile.vue'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../../ui/tooltip'
import { ScrollArea } from '../../ui/scroll-area'
import { PanelLeftClose, PanelRight } from '@lucide/vue'

/**
 * LayoutSidebarApp - 渚ц竟鏍忓叆鍙ｇ粍浠? * 鍝嶅簲寮忓垏鎹㈡闈㈢/绉诲姩绔? */

interface Props {
  /** 鑿滃崟鍒楄〃 */
  menus?: LayoutSidebarMenuItem[]
  /** 鏄惁鎶樺彔 */
  collapsed?: boolean
  /** 渚ц竟鏍忓搴?*/
  width?: number
  /** 鑷畾涔夌被鍚?*/
  class?: HTMLAttributes['class']
  /** 褰撳墠婵€娲荤殑鑿滃崟椤?key */
  activeId?: string
}

import type { HTMLAttributes } from 'vue'

const props = withDefaults(defineProps<Props>(), {
  menus: () => [],
  collapsed: undefined,
  width: undefined,
  activeId: undefined,
})

const emit = defineEmits<{
  /** 鏇存柊鎶樺彔鐘舵€?*/
  (e: 'update:collapsed', value: boolean): void
  /** 鍒囨崲鎶樺彔 */
  (e: 'toggleCollapse'): void
  /** 瀵艰埅 */
  (e: 'navigate', path: string): void
}>()

defineSlots<{
  /** Logo 鎻掓Ы */
  logo(): void
  /** 渚ф爮鏍囬鎻掓Ы */
  'sidebar-title'(): void
  /** 鎶樺彔鎸夐挳鎻掓Ы */
  'collapse-button'(): void
  /** 灞曞紑鎸夐挳鎻掓Ы */
  'expand-button'(): void
  /** 搴曢儴鎻掓Ы */
  footer(props: { collapsed: boolean }): void
}>()

/**
 * 鏄惁涓烘闈㈢
 */
const isDesktop = useMediaQuery('(min-width: 1024px)')

/**
 * 鍔ㄦ€佷晶鏍忛厤缃? */
const dynamicSidebarConfig = computed<SidebarConfig>(() => ({
  ...defaultSidebarConfig,
  menus: props.menus,
  defaultWidth: props.width ?? defaultSidebarConfig.defaultWidth,
}))

/**
 * 浣跨敤渚ф爮閫昏緫
 */
const sidebarState = useLayoutSidebar(dynamicSidebarConfig.value)

/**
 * 鎶樺彔鐘舵€?- 浼樺厛浣跨敤澶栭儴浼犲叆鐨勫€? */
const collapsed = computed({
  get: () => props.collapsed ?? sidebarState.collapsed.value,
  set: (value) => {
    sidebarState.collapsed.value = value
    emit('update:collapsed', value)
  },
})

/**
 * 澶勭悊鎶樺彔鍒囨崲
 */
function handleToggleCollapse(): void {
  collapsed.value = !collapsed.value
  emit('toggleCollapse')
}

/**
 * 澶勭悊瀵艰埅
 */
function handleNavigate(path: string): void {
  emit('navigate', path)
}

/**
 * 棰勮绠楁墍鏈夎彍鍗曢」鐨勬縺娲荤姸鎬? * 鍙傝€?DoubleSidebarMenu 鐨勫疄鐜版柟寮? */
const activeStates = computed(() => {
  const states = new Map<string, boolean>()

  function checkSubtree(item: LayoutSidebarMenuItem, targetId?: string): boolean {
    if (!targetId) return false
    if (item.key === targetId) return true
    if (item.children) {
      return item.children.some((child) => checkSubtree(child, targetId))
    }
    return false
  }

  for (const item of props.menus) {
    states.set(item.key, checkSubtree(item, props.activeId))
  }

  return states
})

function containsActiveId(item: LayoutSidebarMenuItem, targetId?: string): boolean {
  if (!targetId) return false
  if (item.key === targetId) return true
  if (item.children?.length) {
    return item.children.some((child) => containsActiveId(child, targetId))
  }
  return false
}

/**
 * 鑾峰彇鑿滃崟椤圭殑婵€娲荤姸鎬? */
function isMenuActive(item: LayoutSidebarMenuItem): boolean {
  return activeStates.value.get(item.key) ?? false
}

/**
 * 鍒ゆ柇鑿滃崟鏄惁鏈夊瓙椤瑰浜庢縺娲荤姸鎬? */
function hasActiveChild(item: LayoutSidebarMenuItem): boolean {
  if (!item.children?.length) return false
  return item.children.some((child) => activeStates.value.get(child.key) ?? false)
}

/**
 * 鐢ㄦ埛鎵嬪姩灞曞紑鐨勮彍鍗?keys
 * 鍙傝€?DoubleSidebar 浣跨敤 ref 绠＄悊灞曞紑鐘舵€? */
const expandedKeys = ref<Set<string>>(new Set())

/**
 * 鍒囨崲灞曞紑鐘舵€? */
function toggleSubMenu(key: string) {
  const next = new Set(expandedKeys.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  expandedKeys.value = next
}

/**
 * 鍒ゆ柇瀛愯彍鍗曟槸鍚﹀簲璇ュ睍寮€
 * 濡傛灉褰撳墠婵€娲婚」鏄鑿滃崟鐨勫瓙椤癸紝鍒欏己鍒跺睍寮€
 * 鍚﹀垯浣跨敤鐢ㄦ埛鎵嬪姩灞曞紑鐨勭姸鎬? */
function isMenuExpanded(item: LayoutSidebarMenuItem): boolean {
  if (!item.children?.length) return false

  // 鍚﹀垯锛屼娇鐢ㄧ敤鎴锋墜鍔ㄥ睍寮€鐨勭姸鎬?
  return expandedKeys.value.has(item.key)
}

/**
 * watch 鐩戝惉 activeId 鍙樺寲锛岃嚜鍔ㄥ睍寮€鍖呭惈婵€娲婚」鐨勭埗鑿滃崟
 * 鍙傝€?DoubleSidebar 鐨勫疄鐜? */
watch(
  () => props.activeId,
  (newId) => {
    if (!newId) return
    const nextExpanded = new Set(expandedKeys.value)
    for (const item of props.menus) {
      if (containsActiveId(item, newId)) {
        nextExpanded.add(item.key)
      }
    }
    expandedKeys.value = nextExpanded
  },
  { immediate: true },
)

/**
 * 绉诲姩绔晶杈规爮鎵撳紑鐘舵€? */
const mobileOpen = computed({
  get: () => !isDesktop.value && !collapsed.value,
  set: (value) => {
    if (!isDesktop.value) {
      collapsed.value = !value
    }
  },
})

/**
 * 渚ф爮瀹藉害鏍峰紡
 */
const sidebarStyle = computed(() => {
  if (collapsed.value) {
    return { width: `${dynamicSidebarConfig.value.collapsedWidth}px` }
  }
  return { width: `${sidebarState.currentWidth.value}px` }
})

/**
 * 鏄惁姝ｅ湪鎷栨嫿
 */
const isDragging = ref(false)

/**
 * 鎷栨嫿寮€濮嬩綅缃? */
let dragStartX = 0
let dragStartWidth = 0

/**
 * RAF ID 鐢ㄤ簬鍙栨秷鏈墽琛岀殑甯? */
let rafId: number | null = null

/**
 * 寰呮洿鏂扮殑瀹藉害鍊? */
let pendingWidth: number | null = null

/**
 * 澶勭悊鎷栨嫿寮€濮? */
function handleDragStart(event: MouseEvent): void {
  if (collapsed.value) return

  isDragging.value = true
  sidebarState.isDragging.value = true
  dragStartX = event.clientX
  dragStartWidth = sidebarState.currentWidth.value

  document.addEventListener('mousemove', handleDragMove)
  document.addEventListener('mouseup', handleDragEnd)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

/**
 * 澶勭悊鎷栨嫿绉诲姩 - 浣跨敤 RAF 鑺傛祦
 */
function handleDragMove(event: MouseEvent): void {
  if (!isDragging.value) return

  const deltaX = event.clientX - dragStartX
  const newWidth = dragStartWidth + deltaX

  const minWidth = dynamicSidebarConfig.value.minWidth
  const maxWidth = dynamicSidebarConfig.value.maxWidth

  pendingWidth = Math.max(minWidth, Math.min(maxWidth, newWidth))

  // 如果没有待执行的 RAF，则调度一个
  if (rafId === null) {
    rafId = requestAnimationFrame(flushUpdate)
  }
}

/**
 * 鍒锋柊鏇存柊 - 鍦?RAF 涓墽琛? */
function flushUpdate(): void {
  rafId = null

  if (pendingWidth !== null) {
    sidebarState.size.value = (pendingWidth / window.innerWidth) * 100
    pendingWidth = null
  }
}

/**
 * 澶勭悊鎷栨嫿缁撴潫
 */
function handleDragEnd(): void {
  isDragging.value = false
  sidebarState.isDragging.value = false

  // 鍙栨秷鏈墽琛岀殑 RAF
  if (rafId !== null) {
    cancelAnimationFrame(rafId)
    rafId = null
  }

  document.removeEventListener('mousemove', handleDragMove)
  document.removeEventListener('mouseup', handleDragEnd)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

/**
 * 娓呯悊鎷栨嫿浜嬩欢鐩戝惉鍣? */
function cleanupDragListeners(): void {
  if (isDragging.value) {
    // 鍙栨秷鏈墽琛岀殑 RAF
    if (rafId !== null) {
      cancelAnimationFrame(rafId)
      rafId = null
    }
    document.removeEventListener('mousemove', handleDragMove)
    document.removeEventListener('mouseup', handleDragEnd)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }
}

/**
 * 鐩戝惉 props.width 鍙樺寲锛屽悓姝ュ埌 sidebarState
 * 娉ㄦ剰锛氫笉浣跨敤 immediate: true锛屽洜涓虹姸鎬佸凡鍦?useSidebar 涓悓姝ユ仮澶? */
watch(
  () => props.width,
  (newWidth, oldWidth) => {
    if (newWidth !== undefined && !isDragging.value && oldWidth !== undefined) {
      const percentSize = (newWidth / window.innerWidth) * 100
      sidebarState.size.value = percentSize
    }
  },
)

/**
 * 缁勪欢鍗歌浇鏃舵竻鐞嗘嫋鎷戒簨浠剁洃鍚櫒
 */
onUnmounted(() => {
  cleanupDragListeners()
})
</script>

<template>
  <!-- 鏍瑰鍣?- 鍖呰妗岄潰绔拰绉诲姩绔晶鏍忎綔涓哄崟涓€鏍硅妭鐐?-->
  <div>
    <!-- 妗岄潰绔晶鏍?-->
    <aside
      v-if="isDesktop"
      class="h-full flex-shrink-0 border-r border-border/30 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/70 relative flex flex-col transition-all duration-200"
      :class="[props.class, { 'select-none': isDragging }]"
      :style="sidebarStyle"
    >
      <!-- 鑿滃崟鍖哄煙鑳屾櫙瑁呴グ -->
      <div v-if="!collapsed" class="absolute inset-0 pointer-events-none">
        <div
          class="absolute top-0 left-0 right-0 h-32 bg-gradient-to-b from-primary/3 to-transparent"
        />
        <div
          class="absolute bottom-0 left-0 right-0 h-32 bg-gradient-to-t from-muted/10 to-transparent"
        />
      </div>

      <!-- Logo 鍖哄煙 -->
      <div class="relative z-10 border-b border-border/30" :class="collapsed ? 'p-2' : 'p-3'">
        <div
          class="flex items-center transition-all duration-200"
          :class="collapsed ? 'justify-center' : 'gap-3'"
        >
          <slot name="logo">
            <div
              class="flex items-center justify-center rounded-lg transition-colors duration-150"
              :class="
                collapsed
                  ? 'h-12 w-12 bg-primary text-primary-foreground'
                  : 'h-10 w-10 bg-primary/10'
              "
              :style="{
                filter: 'drop-shadow(0 1px 2px rgba(0, 0, 0, 0.15))',
                transform: 'translateZ(0)',
              }"
            >
              <svg
                class="block"
                :width="collapsed ? 28 : 24"
                :height="collapsed ? 28 : 24"
                viewBox="0 0 48 48"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
                :style="{ transform: 'translateZ(0)' }"
              >
                <defs>
                  <linearGradient id="logoGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" :stop-color="'var(--primary)'" stop-opacity="1" />
                    <stop offset="100%" :stop-color="'var(--primary)'" stop-opacity="0.85" />
                  </linearGradient>
                </defs>
                <rect x="4" y="4" width="18" height="18" rx="4" fill="url(#logoGradient)" />
                <g class="text-foreground">
                  <rect
                    x="26"
                    y="4"
                    width="18"
                    height="18"
                    rx="4"
                    fill="currentColor"
                    opacity="0.9"
                  />
                </g>
                <rect
                  x="4"
                  y="26"
                  width="18"
                  height="18"
                  rx="4"
                  fill="url(#logoGradient)"
                  opacity="0.7"
                />
                <rect
                  x="26"
                  y="26"
                  width="18"
                  height="18"
                  rx="4"
                  fill="url(#logoGradient)"
                  opacity="0.5"
                />
              </svg>
            </div>
          </slot>
          <slot name="sidebar-title">
            <div v-if="!collapsed" class="flex flex-col min-w-0">
              <span class="text-sm font-bold tracking-tight truncate">ApiPig Gateway</span>
              <span class="text-[10px] text-muted-foreground truncate">管理系统</span>
            </div>
          </slot>
        </div>

        <!-- 鎶樺彔鎸夐挳 - Logo 涓嬫柟 -->
        <!-- 灞曞紑鐘舵€侊細鐩存帴娓叉煋鎸夐挳锛屾棤 Tooltip 鍖呰９ -->
        <button
          v-if="!collapsed"
          class="mt-2 w-full h-8 flex items-center justify-center rounded-lg bg-muted/50 hover:bg-muted hover:text-primary transition-all duration-200 gap-2"
          @click="handleToggleCollapse"
        >
          <PanelLeftClose class="h-4 w-4" />
          <slot name="collapse-button">
            <span class="text-xs text-muted-foreground">收起侧栏</span>
          </slot>
        </button>

        <!-- 鎶樺彔鐘舵€侊細浣跨敤 Tooltip 鎻愪緵鎮仠鎻愮ず -->
        <TooltipProvider v-else :delay-duration="200">
          <Tooltip>
            <TooltipTrigger as-child>
              <button
                class="mt-2 w-full h-8 flex items-center justify-center rounded-lg bg-muted/50 hover:bg-muted hover:text-primary transition-all duration-200 px-0"
                @click="handleToggleCollapse"
              >
                <PanelRight class="h-4 w-4" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="right">
              <slot name="expand-button">
                <span>展开侧栏</span>
              </slot>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>

      <!-- 鑿滃崟鍒楄〃 -->
      <ScrollArea class="flex-1 min-h-0 relative z-10">
        <nav class="p-3 space-y-1">
          <template v-for="item in dynamicSidebarConfig.menus" :key="item.key">
            <LayoutSidebarSubMenu
              v-if="item.children && item.children.length > 0"
              :item="item"
              :collapsed="collapsed"
              :active="isMenuActive(item)"
              :has-active-child="hasActiveChild(item)"
              :expanded="isMenuExpanded(item)"
              :active-id="props.activeId"
              @toggle="toggleSubMenu(item.key)"
              @navigate="handleNavigate"
            />
            <LayoutSidebarItem
              v-else
              :item="item"
              :title="item.title"
              :collapsed="collapsed"
              :active="isMenuActive(item)"
              @navigate="handleNavigate"
            />
          </template>
        </nav>
      </ScrollArea>

      <!-- 搴曢儴鍖哄煙 -->
      <div class="border-t border-border/30 flex-shrink-0 min-h-[70px]">
        <slot name="footer" :collapsed="collapsed" />
      </div>

      <!-- 鎷栨嫿鎵嬫焺 -->
      <div
        v-if="!collapsed"
        class="absolute top-0 right-0 bottom-0 w-2 cursor-col-resize group hover:bg-primary/20 transition-colors z-20"
        :class="{ 'bg-primary/30': isDragging }"
        @mousedown.prevent="handleDragStart"
      >
        <div
          class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-0.5 h-8 rounded-full bg-border/60 group-hover:bg-primary/50 transition-colors pointer-events-none"
          :class="{ 'bg-primary/60': isDragging }"
        />
      </div>
    </aside>

    <!-- 绉诲姩绔晶鏍?-->
    <template v-else>
      <LayoutSidebarMobile
        :config="dynamicSidebarConfig"
        :expanded-keys="expandedKeys"
        :open="mobileOpen"
        @update:open="mobileOpen = $event"
        @toggle-sub-menu="toggleSubMenu"
        @navigate="handleNavigate"
      >
        <template #logo>
          <slot name="logo" />
        </template>
        <template #footer>
          <slot name="footer" :collapsed="false" />
        </template>
      </LayoutSidebarMobile>
    </template>
  </div>
</template>
