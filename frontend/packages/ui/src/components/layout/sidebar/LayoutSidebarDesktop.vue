<script setup lang="ts">
import type { SidebarConfig, LayoutSidebarMenuItem } from './config'
import { computed, ref } from 'vue'
import { ChevronUp, LogOut, PanelLeft, PanelRight, Settings, User } from '@lucide/vue'
import { Avatar, AvatarFallback } from '../../ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../../ui/dropdown-menu'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '../../ui/resizable'
import { ScrollArea } from '../../ui/scroll-area'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../../ui/tooltip'
import { pxToPercent, useMenuUtils } from '../composables'
import LayoutSidebarItem from './LayoutSidebarItem.vue'
import LayoutSidebarSubMenu from './LayoutSidebarSubMenu.vue'

/**
 * LayoutSidebarDesktop - 妗岄潰绔晶杈规爮缁勪欢
 * 浣跨敤 ResizablePanelGroup 瀹炵幇鍙皟鏁村搴? */

interface Props {
  /** 渚ф爮閰嶇疆 */
  config: SidebarConfig
  /** 鏄惁鎶樺彔 */
  collapsed: boolean
  /** 褰撳墠灏哄锛堢櫨鍒嗘瘮锛?*/
  currentSize: number
  /** 鏄惁鎷栨嫿涓?*/
  isDragging: boolean
  /** 灞曞紑鐨勫瓙鑿滃崟 keys */
  expandedKeys: Set<string>
}

const props = defineProps<Props>()

const emit = defineEmits<{
  /** 璋冩暣灏哄 */
  (e: 'resize', size: number): void
  /** 鎷栨嫿鐘舵€佸彉鍖?*/
  (e: 'dragging', dragging: boolean): void
  /** 鍒囨崲瀛愯彍鍗?*/
  (e: 'toggleSubMenu', key: string): void
  /** 瀵艰埅 */
  (e: 'navigate', path: string): void
  /** 鍒囨崲鎶樺彔鐘舵€?*/
  (e: 'toggleCollapse'): void
}>()

/**
 * 闈㈡澘澶у皬锛堢櫨鍒嗘瘮锛? */
const panelSize = computed(() => pxToPercent(props.config.defaultWidth, window.innerWidth))

/**
 * 鏈€灏忓昂瀵革紙鐧惧垎姣旓級
 */
const minSizePercent = computed(() => pxToPercent(props.config.minWidth, window.innerWidth))

/**
 * 鏈€澶у昂瀵革紙鐧惧垎姣旓級
 */
const maxSizePercent = computed(() => pxToPercent(props.config.maxWidth, window.innerWidth))

/**
 * 浣跨敤鑿滃崟宸ュ叿鍑芥暟
 */
const { isActive, isExpanded: checkExpanded } = useMenuUtils({
  expandedKeys: computed(() => props.expandedKeys),
})

/**
 * 鍒ゆ柇鏄惁灞曞紑
 */
function isExpanded(key: string): boolean {
  return checkExpanded(key)
}

/**
 * 澶勭悊瀵艰埅
 */
function handleNavigate(path: string): void {
  emit('navigate', path)
}

/**
 * 鍒囨崲瀛愯彍鍗? */
function handleToggleSubMenu(key: string): void {
  emit('toggleSubMenu', key)
}

/**
 * 鍒ゆ柇鏄惁鏈夊瓙鑿滃崟
 */
function hasChildren(item: LayoutSidebarMenuItem): boolean {
  return !!item.children && item.children.length > 0
}

/**
 * 澶勭悊鎶樺彔鍒囨崲
 */
function handleToggleCollapse(): void {
  emit('toggleCollapse')
}

/**
 * 鐢ㄦ埛鑿滃崟鎵撳紑鐘舵€? */
const isUserMenuOpen = ref(false)

/**
 * 鐢ㄦ埛濮撳悕棣栧瓧姣? */
const userInitials = computed(() => 'U')

/**
 * 澶勭悊瀵艰埅鍒颁釜浜鸿祫鏂? */
function handleGoToProfile(): void {
  emit('navigate', '/profile')
}

/**
 * 澶勭悊瀵艰埅鍒拌缃? */
function handleGoToSettings(): void {
  emit('navigate', '/settings')
}

/**
 * 澶勭悊閫€鍑虹櫥褰? */
function handleLogout(): void {
  emit('navigate', '/login')
}
</script>

<template>
  <ResizablePanelGroup direction="horizontal" class="h-full hidden lg:flex">
    <!-- 渚ф爮闈㈡澘 -->
    <ResizablePanel
      :min-size="minSizePercent"
      :max-size="maxSizePercent"
      :default-size="panelSize"
      class="flex flex-col border-r border-border/30 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/70 relative"
      :class="{ 'transition-none': isDragging }"
      :style="collapsed ? { flex: `0 0 ${config.collapsedWidth}px` } : {}"
      @resize="(size: number) => $emit('resize', size)"
    >
      <!-- 鑿滃崟鍖哄煙鑳屾櫙瑁呴グ -->
      <div v-if="!collapsed" class="absolute inset-0 pointer-events-none">
        <div
          class="absolute top-0 left-0 right-0 h-32 bg-gradient-to-b from-primary/3 to-transparent"
        />
        <div
          class="absolute bottom-0 left-0 right-0 h-32 bg-gradient-to-t from-muted/20 to-transparent"
        />
      </div>

      <!-- Logo 鍖哄煙 -->
      <div class="relative z-10 border-b border-border/30" :class="collapsed ? 'p-2' : 'p-3'">
        <div
          class="flex items-center transition-all duration-200"
          :class="collapsed ? 'justify-center' : 'gap-3'"
        >
          <slot name="logo">
            <div class="h-14 w-14 rounded-lg bg-primary/10 flex items-center justify-center">
              <span class="text-lg font-bold text-primary">T</span>
            </div>
          </slot>
          <div v-if="!collapsed" class="flex flex-col min-w-0">
            <span class="text-sm font-bold tracking-tight truncate">ApiPig</span>
            <span class="text-[10px] text-muted-foreground truncate">管理系统</span>
          </div>
        </div>

        <!-- 鎶樺彔鎸夐挳 -->
        <button
          v-if="!collapsed"
          class="mt-2 w-full h-8 flex items-center justify-center rounded-lg bg-muted/50 hover:bg-muted hover:text-primary transition-all duration-200 gap-2"
          @click="handleToggleCollapse"
        >
          <PanelLeft class="h-4 w-4" />
          <span class="text-xs text-muted-foreground">收起侧栏</span>
        </button>

        <TooltipProvider v-else>
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
              <span>展开侧栏</span>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>

      <!-- 鑿滃崟鍒楄〃 -->
      <ScrollArea class="flex-1 h-0 relative z-10">
        <nav class="p-3 space-y-1 relative z-10">
          <template v-for="item in config.menus" :key="item.key">
            <!-- 灞曞紑鐘舵€?-->
            <template v-if="!collapsed">
              <LayoutSidebarSubMenu
                v-if="hasChildren(item)"
                :item="item"
                :collapsed="collapsed"
                :active="isActive(item.path)"
                :expanded="isExpanded(item.key)"
                @toggle="handleToggleSubMenu(item.key)"
                @navigate="handleNavigate"
              />
              <LayoutSidebarItem
                v-else
                :item="item"
                :title="item.title"
                :collapsed="collapsed"
                :active="isActive(item.path)"
                @navigate="handleNavigate"
              />
            </template>

            <!-- 鎶樺彔鐘舵€?-->
            <template v-else>
              <LayoutSidebarSubMenu
                v-if="hasChildren(item)"
                :item="item"
                :collapsed="collapsed"
                :active="isActive(item.path)"
                :expanded="isExpanded(item.key)"
                @toggle="handleToggleSubMenu(item.key)"
                @navigate="handleNavigate"
              />
              <LayoutSidebarItem
                v-else
                :item="item"
                :title="item.title"
                :collapsed="collapsed"
                :active="isActive(item.path)"
                @navigate="handleNavigate"
              />
            </template>
          </template>
        </nav>
      </ScrollArea>

      <!-- 搴曢儴鍖哄煙 -->
      <TooltipProvider>
        <div class="relative flex-shrink-0 min-h-[70px]">
          <div
            class="absolute top-0 left-4 right-4 h-px bg-gradient-to-r from-transparent via-border to-transparent"
          />

          <div class="border-t border-border/30 bg-muted/40 backdrop-blur-md pt-1">
            <slot name="footer">
              <!-- 灞曞紑鐘舵€?-->
              <div v-if="!collapsed" class="px-3 py-2">
                <DropdownMenu v-model:open="isUserMenuOpen">
                  <DropdownMenuTrigger as-child>
                    <button
                      class="group flex items-center gap-2.5 min-w-0 w-full rounded-xl p-2 hover:bg-muted/60 transition-all duration-200"
                    >
                      <div class="relative flex-shrink-0">
                        <Avatar
                          class="h-9 w-9 ring-2 ring-primary/20 transition-all duration-200 group-hover:ring-primary/40 group-hover:shadow-md"
                        >
                          <AvatarFallback
                            class="text-sm font-semibold bg-gradient-to-br from-primary to-primary/70 text-primary-foreground"
                          >
                            {{ userInitials }}
                          </AvatarFallback>
                        </Avatar>
                        <span
                          class="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full bg-emerald-500 ring-2 ring-background"
                        />
                      </div>
                      <div class="flex flex-col min-w-0 flex-1 text-left">
                        <span
                          class="text-sm font-medium truncate group-hover:text-primary transition-colors duration-200"
                        >
                          用户
                        </span>
                        <span class="text-[11px] text-muted-foreground truncate">
                          user@example.com
                        </span>
                      </div>
                      <ChevronUp
                        class="h-4 w-4 text-muted-foreground flex-shrink-0 transition-transform duration-200 group-data-[state=open]:rotate-180"
                      />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start" class="w-56" :side-offset="8">
                    <div class="px-2 py-1.5">
                      <p class="text-xs font-medium text-muted-foreground">已登录</p>
                      <p class="text-sm font-semibold truncate">user@example.com</p>
                    </div>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem class="gap-2 cursor-pointer" @click="handleGoToProfile">
                      <User class="h-4 w-4" />
                      <span>个人资料</span>
                    </DropdownMenuItem>
                    <DropdownMenuItem class="gap-2 cursor-pointer" @click="handleGoToSettings">
                      <Settings class="h-4 w-4" />
                      <span>设置</span>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      class="gap-2 cursor-pointer text-destructive focus:text-destructive"
                      @click="handleLogout"
                    >
                      <LogOut class="h-4 w-4" />
                      <span>退出登录</span>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>

              <!-- 鎶樺彔鐘舵€侊細鐢ㄦ埛澶村儚 -->
              <div v-else class="py-3 px-2 flex flex-col items-center">
                <DropdownMenu v-model:open="isUserMenuOpen">
                  <DropdownMenuTrigger as-child>
                    <button class="group relative">
                      <Avatar
                        class="h-10 w-10 ring-2 ring-primary/20 transition-all duration-200 group-hover:ring-primary/50 group-hover:shadow-lg"
                      >
                        <AvatarFallback
                          class="text-sm font-semibold bg-gradient-to-br from-primary to-primary/80 text-primary-foreground"
                        >
                          {{ userInitials }}
                        </AvatarFallback>
                      </Avatar>
                      <span
                        class="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full bg-emerald-500 ring-2 ring-background"
                      />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start" side="right" class="w-56" :side-offset="8">
                    <div class="px-2 py-1.5">
                      <p class="text-xs font-medium text-muted-foreground">已登录</p>
                      <p class="text-sm font-semibold truncate">user@example.com</p>
                    </div>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem class="gap-2 cursor-pointer" @click="handleGoToProfile">
                      <User class="h-4 w-4" />
                      <span>涓汉璧勬枡</span>
                    </DropdownMenuItem>
                    <DropdownMenuItem class="gap-2 cursor-pointer" @click="handleGoToSettings">
                      <Settings class="h-4 w-4" />
                      <span>璁剧疆</span>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      class="gap-2 cursor-pointer text-destructive focus:text-destructive"
                      @click="handleLogout"
                    >
                      <LogOut class="h-4 w-4" />
                      <span>退出登录</span>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </slot>
          </div>
        </div>
      </TooltipProvider>
    </ResizablePanel>

    <!-- 鎷栨嫿鎵嬫焺 -->
    <ResizableHandle
      v-if="!collapsed"
      with-handle
      class="w-1.5 bg-transparent hover:bg-primary/20 transition-colors relative after:absolute after:inset-y-4 after:left-1/2 after:-translate-x-1/2 after:w-1 after:h-8 after:rounded-full after:bg-border/60 hover:after:bg-primary/50"
      @dragging="(dragging: boolean) => $emit('dragging', dragging)"
    />

    <!-- 鍐呭闈㈡澘 -->
    <ResizablePanel :min-size="50">
      <main class="h-full min-w-0 bg-background">
        <slot />
      </main>
    </ResizablePanel>
  </ResizablePanelGroup>
</template>
