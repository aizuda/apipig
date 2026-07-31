<script setup lang="ts">
import type { MenuItem } from '../types'
import { computed } from 'vue'
import { ChevronDown } from '@lucide/vue'
import { Button } from '../../ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from '../../ui/dropdown-menu'
import { isComponent } from '../composables'
import LayoutTopNavDropdownItem from './LayoutTopNavDropdownItem.vue'

/**
 * LayoutTopNavItem - 椤堕儴瀵艰埅鑿滃崟椤圭粍浠? * 鍙傝€冨崟鏍忎晶鏍?SidebarItem 鐨勬牱寮忛鏍? * 鏀寔锛? * - 鏄剧ず icon
 * - 鍗曠骇鑿滃崟鐩存帴璺宠浆
 * - 澶氱骇鑿滃崟浣跨敤 DropdownMenu 涓嬫媺灞曠ず
 * - 鏃犻檺灞傜骇宓屽
 */

interface Props {
  /** 鑿滃崟椤规暟鎹?*/
  item: MenuItem
  /** 褰撳墠婵€娲荤殑鑿滃崟椤?key */
  activeId?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  /** 瀵艰埅浜嬩欢 */
  (e: 'navigate', item: MenuItem): void
}>()

/**
 * 鏄惁鏈夊瓙鑿滃崟
 */
const hasChildren = computed(() => {
  return !!props.item.children && props.item.children.length > 0
})

/**
 * 妫€鏌ュ綋鍓嶉」鎴栧叾浠绘剰瀛愰」鏄惁鍖归厤 targetId
 * 閫掑綊妫€鏌ユ墍鏈夊眰绾? */
function checkSubtree(item: MenuItem, targetId: string): boolean {
  if (item.id === targetId) return true
  if (item.children) {
    return item.children.some((child) => checkSubtree(child, targetId))
  }
  return false
}

/**
 * 褰撳墠椤规槸鍚︽縺娲? */
const isActive = computed(() => {
  if (!props.activeId) return false
  return props.item.id === props.activeId
})

/**
 * 鏄惁鏈夊瓙椤规縺娲伙紙閫掑綊妫€鏌ユ墍鏈夊瓙灞傜骇锛? */
const hasActiveChild = computed(() => {
  if (!props.activeId || !props.item.children) return false
  return props.item.children.some((child) => checkSubtree(child, props.activeId!))
})

/**
 * 鏄惁鏄剧ず涓烘縺娲荤姸鎬侊紙鑷韩婵€娲绘垨鏈夊瓙椤规縺娲伙級
 */
const isActiveState = computed(() => {
  return isActive.value || hasActiveChild.value
})

/**
 * 澶勭悊鐐瑰嚮
 */
function handleClick() {
  emit('navigate', props.item)
}

/**
 * 澶勭悊瀛愯彍鍗曞鑸? */
function handleChildNavigate(item: MenuItem) {
  emit('navigate', item)
}
</script>

<template>
  <!-- 鏈夊瓙鑿滃崟锛氭樉绀轰笅鎷夎彍鍗?-->
  <DropdownMenu v-if="hasChildren">
    <DropdownMenuTrigger as-child>
      <Button
        variant="ghost"
        size="sm"
        class="relative transition-all duration-200 group"
        :class="[
          isActiveState
            ? 'bg-primary/10 text-primary font-medium hover:bg-primary/15 shadow-sm'
            : 'hover:bg-muted/50',
        ]"
      >
        <component
          :is="item.icon"
          v-if="item.icon && isComponent(item.icon)"
          class="h-4 w-4 mr-1.5"
          :class="[
            isActiveState ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground',
          ]"
        />
        <span
          :class="[
            isActiveState
              ? 'text-primary font-medium'
              : 'text-muted-foreground group-hover:text-foreground',
          ]"
        >
          {{ item.name }}
        </span>
        <ChevronDown
          class="h-3.5 w-3.5 ml-1 transition-transform duration-200"
          :class="[
            isActiveState ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground',
          ]"
        />
        <!-- 搴曢儴杈规鎸囩ず鍣紙椤堕儴瀵艰埅鐗硅壊锛?-->
        <span
          v-if="isActiveState"
          class="absolute bottom-0 left-1/2 -translate-x-1/2 w-10 h-[3px] bg-gradient-to-r from-primary/80 via-primary to-primary/80 rounded-full shadow-sm shadow-primary/20"
        />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="start" class="min-w-[180px]">
      <LayoutTopNavDropdownItem
        v-for="child in item.children"
        :key="child.id"
        :item="child"
        :active-id="props.activeId"
        @navigate="handleChildNavigate"
      />
    </DropdownMenuContent>
  </DropdownMenu>

  <!-- 鏃犲瓙鑿滃崟锛氭樉绀烘櫘閫氭寜閽?-->
  <Button
    v-else
    variant="ghost"
    size="sm"
    class="relative transition-all duration-200 group"
    :class="[
      isActiveState
        ? 'bg-primary/10 text-primary font-medium hover:bg-primary/15 shadow-sm'
        : 'hover:bg-muted/50',
    ]"
    @click="handleClick"
  >
    <component
      :is="item.icon"
      v-if="item.icon && isComponent(item.icon)"
      class="h-4 w-4 mr-1.5"
      :class="[
        isActiveState ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground',
      ]"
    />
    <span
      :class="[
        isActiveState
          ? 'text-primary font-medium'
          : 'text-muted-foreground group-hover:text-foreground',
      ]"
    >
      {{ item.name }}
    </span>
    <!-- 搴曢儴杈规鎸囩ず鍣紙椤堕儴瀵艰埅鐗硅壊锛?-->
    <span
      v-if="isActiveState"
      class="absolute bottom-0 left-1/2 -translate-x-1/2 w-10 h-[3px] bg-gradient-to-r from-primary/80 via-primary to-primary/80 rounded-full shadow-sm shadow-primary/20"
    />
  </Button>
</template>
