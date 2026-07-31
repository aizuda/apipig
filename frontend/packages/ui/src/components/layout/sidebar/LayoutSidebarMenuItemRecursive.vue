<script setup lang="ts">
import type { LayoutSidebarMenuItem } from './config'
import { computed, ref, watch } from 'vue'
import { ChevronDown, ChevronRight } from '@lucide/vue'
import { Button } from '../../ui/button'
import { Badge } from '../../ui/badge'
import { formatBadge } from '../composables'
import { isComponent } from '../composables'

/**
 * LayoutSidebarMenuItemRecursive - 閫掑綊鑿滃崟娓叉煋缁勪欢
 * 鏀寔鏃犻檺灞傜骇宓屽
 */

interface Props {
  /** 鑿滃崟椤规暟鎹?*/
  item: LayoutSidebarMenuItem
  /** 鏄惁鎶樺彔 */
  collapsed: boolean
  /** 褰撳墠灞傜骇 */
  level?: number
  /** 褰撳墠婵€娲荤殑鑿滃崟椤?key */
  activeId?: string
}

const props = withDefaults(defineProps<Props>(), {
  level: 0,
  activeId: undefined,
})

const emit = defineEmits<{
  /** 瀵艰埅浜嬩欢 */
  (e: 'navigate', path: string): void
}>()

/**
 * 鏄惁灞曞紑
 */
const isExpanded = ref(props.item.defaultExpanded ?? false)

/**
 * 妫€鏌ュ綋鍓嶉」鎴栧叾浠绘剰瀛愰」鏄惁鍖归厤 targetId
 * 閫掑綊妫€鏌ユ墍鏈夊眰绾э紝鏀寔鏃犻檺绾ц彍鍗? */
function checkSubtree(item: LayoutSidebarMenuItem, targetId: string): boolean {
  if (item.key === targetId) return true
  if (item.children) {
    return item.children.some((child) => checkSubtree(child, targetId))
  }
  return false
}

/**
 * 褰撳墠椤规槸鍚︽縺娲? */
const isActive = computed(() => {
  if (!props.activeId) return false
  return props.item.key === props.activeId
})

/**
 * 鏄惁鏈夊瓙椤规縺娲伙紙閫掑綊妫€鏌ユ墍鏈夊瓙灞傜骇锛? */
const hasActiveChild = computed(() => {
  if (!props.activeId || !props.item.children) return false
  return props.item.children.some((child) => checkSubtree(child, props.activeId!))
})

/**
 * 鏄惁鏈夊瓙鑿滃崟
 */
const hasChildren = computed(() => {
  return !!props.item.children && props.item.children.length > 0
})

watch(
  () => props.activeId,
  (newId) => {
    if (!newId || !hasChildren.value) return
    if (checkSubtree(props.item, newId)) {
      isExpanded.value = true
    }
  },
  { immediate: true },
)

/**
 * 鑾峰彇鎸夐挳鏍峰紡绫? */
const buttonClasses = computed(() => {
  const baseClasses = 'w-full justify-between h-9 px-3 group transition-all duration-200 rounded-lg'

  if (props.item.disabled) {
    return `${baseClasses} opacity-50 cursor-not-allowed`
  }

  if (isActive.value) {
    return `${baseClasses} bg-primary/10 text-primary font-medium shadow-[inset_3px_0_0_0_hsl(var(--primary))] hover:bg-primary/15`
  }
  if (hasActiveChild.value) {
    return `${baseClasses} bg-muted/30 text-foreground shadow-[inset_3px_0_0_0_hsl(var(--primary)/0.3)] hover:bg-muted/50`
  }
  return `${baseClasses} hover:bg-accent hover:text-accent-foreground`
})

/**
 * 鑾峰彇鍥炬爣鏍峰紡绫? */
const iconClasses = computed(() => {
  if (isActive.value) {
    return 'h-4 w-4 text-primary'
  }
  return 'h-4 w-4 text-muted-foreground group-hover:text-accent-foreground'
})

/**
 * 鑾峰彇灞曞紑/鏀惰捣鍥炬爣鏍峰紡绫? */
const expandIconClasses = computed(() => {
  if (isActive.value || hasActiveChild.value) {
    return 'h-3.5 w-3.5 text-primary'
  }
  return 'h-3.5 w-3.5 text-muted-foreground'
})

/**
 * 澶勭悊鐐瑰嚮
 */
function handleClick(): void {
  if (hasChildren.value && !props.collapsed) {
    isExpanded.value = !isExpanded.value
    return
  }

  emit('navigate', props.item.path)
}

/**
 * 澶勭悊瀛愯彍鍗曞鑸? */
function handleChildNavigate(path: string): void {
  emit('navigate', path)
}

/**
 * 鑿滃崟鏍囬
 */
const menuTitle = computed(() => props.item.title)
</script>

<template>
  <div class="relative">
    <!-- 鑿滃崟椤?-->
    <Button
      :variant="isActive || hasActiveChild ? 'default' : 'ghost'"
      role="menuitem"
      :aria-expanded="hasChildren ? isExpanded : undefined"
      :aria-haspopup="hasChildren ? true : undefined"
      :aria-current="isActive ? 'page' : undefined"
      :disabled="item.disabled"
      :class="buttonClasses"
      @click="handleClick"
    >
      <div class="flex items-center gap-2 flex-1 min-w-0">
        <!-- 鍥炬爣 -->
        <component
          :is="item.icon"
          v-if="item.icon && isComponent(item.icon)"
          :class="iconClasses"
          aria-hidden="true"
        />
        <span class="truncate text-sm">{{ menuTitle }}</span>
      </div>

      <div class="flex items-center gap-1.5 flex-shrink-0">
        <Badge
          v-if="item.badge"
          variant="destructive"
          class="h-4 px-1.5 text-[10px] font-medium"
          role="status"
        >
          {{ formatBadge(typeof item.badge === 'number' ? item.badge : parseInt(item.badge) || 0) }}
        </Badge>

        <!-- 灞曞紑/鏀惰捣鍥炬爣 -->
        <ChevronDown
          v-if="hasChildren && !collapsed && isExpanded"
          :class="expandIconClasses"
          aria-hidden="true"
        />
        <ChevronRight
          v-else-if="hasChildren && !collapsed"
          :class="expandIconClasses"
          aria-hidden="true"
        />
      </div>
    </Button>

    <!-- 瀛愯彍鍗?-->
    <Transition
      enter-active-class="transition-all duration-200 ease-out"
      enter-from-class="opacity-0 max-h-0"
      enter-to-class="opacity-100 max-h-[500px]"
      leave-active-class="transition-all duration-150 ease-in"
      leave-from-class="opacity-100 max-h-[500px]"
      leave-to-class="opacity-0 max-h-0"
    >
      <div
        v-if="hasChildren && isExpanded && !collapsed"
        role="menu"
        :aria-label="`${item.title} 子菜单`"
        class="ml-4 mt-1 space-y-0.5 overflow-hidden border-l border-border/50 pl-3"
      >
        <!-- 閫掑綊娓叉煋瀛愯彍鍗曢」锛屼紶閫?activeId 鏀寔鏃犻檺绾ц彍鍗?-->
        <LayoutSidebarMenuItemRecursive
          v-for="child in item.children"
          :key="child.key"
          :item="child"
          :collapsed="collapsed"
          :level="level + 1"
          :active-id="activeId"
          @navigate="handleChildNavigate"
        />
      </div>
    </Transition>
  </div>
</template>
