<script setup lang="ts">
import type { SidebarConfig } from './config'
import { computed } from 'vue'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '../../ui/sheet'
import { ScrollArea } from '../../ui/scroll-area'
import { useMenuUtils } from '../composables'
import LayoutSidebarItem from './LayoutSidebarItem.vue'
import LayoutSidebarSubMenu from './LayoutSidebarSubMenu.vue'

/**
 * LayoutSidebarMobile - 绉诲姩绔晶杈规爮缁勪欢
 * 浣跨敤 Sheet 缁勪欢瀹炵幇鎶藉眽寮忎晶杈规爮
 */

interface Props {
  /** 渚ф爮閰嶇疆 */
  config: SidebarConfig
  /** 灞曞紑鐨勫瓙鑿滃崟 keys */
  expandedKeys: Set<string>
  /** 鏄惁鎵撳紑 */
  open?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  /** 鏇存柊鎵撳紑鐘舵€?*/
  (e: 'update:open', value: boolean): void
  /** 鍒囨崲瀛愯彍鍗?*/
  (e: 'toggleSubMenu', key: string): void
  /** 瀵艰埅 */
  (e: 'navigate', path: string): void
}>()

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
  emit('update:open', false)
}

/**
 * 鍒囨崲瀛愯彍鍗? */
function handleToggleSubMenu(key: string): void {
  emit('toggleSubMenu', key)
}

/**
 * 鍒ゆ柇鏄惁鏈夊瓙鑿滃崟
 */
function hasChildren(item: { children?: unknown[] }): boolean {
  return !!item.children && item.children.length > 0
}

/**
 * 鎵撳紑鐘舵€? */
const openModel = computed({
  get: () => props.open ?? false,
  set: (value) => emit('update:open', value),
})
</script>

<template>
  <Sheet v-model:open="openModel">
    <SheetContent side="left" class="w-[280px] p-0">
      <!-- 澶撮儴 -->
      <SheetHeader class="border-b border-border/30 p-4">
        <div class="flex items-center gap-3">
          <slot name="logo">
            <div class="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
              <span class="text-sm font-bold text-primary">T</span>
            </div>
          </slot>
          <div class="flex flex-col min-w-0">
            <SheetTitle class="text-sm font-bold tracking-tight truncate">
              ApiPig Gateway
            </SheetTitle>
            <span class="text-[10px] text-muted-foreground truncate">管理系统</span>
          </div>
        </div>
      </SheetHeader>

      <!-- 鑿滃崟鍒楄〃 -->
      <ScrollArea class="flex-1 h-[calc(100vh-140px)]">
        <nav class="p-3 space-y-1">
          <template v-for="item in config.menus" :key="item.key">
            <LayoutSidebarSubMenu
              v-if="hasChildren(item)"
              :item="item"
              :collapsed="false"
              :active="isActive(item.path)"
              :expanded="isExpanded(item.key)"
              @toggle="handleToggleSubMenu(item.key)"
              @navigate="handleNavigate"
            />
            <LayoutSidebarItem
              v-else
              :item="item"
              :title="item.title"
              :collapsed="false"
              :active="isActive(item.path)"
              @navigate="handleNavigate"
            />
          </template>
        </nav>
      </ScrollArea>

      <!-- 搴曢儴 -->
      <div class="border-t border-border/30 p-4">
        <slot name="footer">
          <div class="flex items-center gap-3">
            <div class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
              <span class="text-sm font-semibold text-primary">U</span>
            </div>
            <div class="flex flex-col min-w-0 flex-1">
              <span class="text-sm font-medium truncate">用户</span>
              <span class="text-[11px] text-muted-foreground truncate">user@example.com</span>
            </div>
          </div>
        </slot>
      </div>
    </SheetContent>
  </Sheet>
</template>
