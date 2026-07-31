<script setup lang="ts">
import type { MenuItem } from '../types'
import { computed } from 'vue'
import { DropdownMenuItem } from '../../ui/dropdown-menu'
import DropdownMenuSub from '../../ui/dropdown-menu/DropdownMenuSub.vue'
import DropdownMenuSubContent from '../../ui/dropdown-menu/DropdownMenuSubContent.vue'
import DropdownMenuSubTrigger from '../../ui/dropdown-menu/DropdownMenuSubTrigger.vue'
import { isComponent } from '../composables'

/**
 * LayoutTopNavDropdownItem - 下拉菜单递归菜单项组件
 * 用于顶部导航下拉菜单内部，支持无限层级嵌套
 */

interface Props {
  /** 菜单项数据 */
  item: MenuItem
  /** 当前激活的菜单ID */
  activeId?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  /** 导航事件 */
  (e: 'navigate', item: MenuItem): void
}>()

/**
 * 递归检查子树中是否有激活项
 */
function checkSubtree(item: MenuItem, targetId: string): boolean {
  if (item.id === targetId) return true
  if (item.children) {
    return item.children.some((child) => checkSubtree(child, targetId))
  }
  return false
}

/**
 * 是否有子菜单
 */
const hasChildren = computed(() => {
  return !!props.item.children && props.item.children.length > 0
})

/**
 * 当前菜单项是否激活
 */
const isActive = computed(() => {
  if (!props.activeId) return false
  return props.item.id === props.activeId
})

/**
 * 子菜单中是否有激活项
 */
const hasActiveChild = computed(() => {
  if (!props.activeId || !props.item.children) return false
  return props.item.children.some((child) => checkSubtree(child, props.activeId!))
})

/**
 * 是否显示为激活状态
 */
const isActiveState = computed(() => {
  return isActive.value || hasActiveChild.value
})

/**
 * 处理点击
 */
function handleClick() {
  if (!hasChildren.value) {
    emit('navigate', props.item)
  }
}

/**
 * 处理子菜单导航
 */
function handleChildNavigate(item: MenuItem) {
  emit('navigate', item)
}
</script>

<template>
  <!-- 有子菜单：显示嵌套下拉菜单 -->
  <DropdownMenuSub v-if="hasChildren">
    <DropdownMenuSubTrigger
      :class="[
        'cursor-pointer relative transition-all duration-200 group rounded-md mx-1',
        isActiveState ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-muted/50',
      ]"
    >
      <!-- 左侧激活指示条 -->
      <span
        v-if="isActiveState"
        class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 bg-gradient-to-b from-primary/80 via-primary to-primary/80 rounded-r-full shadow-sm shadow-primary/20"
      />
      <component
        :is="item.icon"
        v-if="item.icon && isComponent(item.icon)"
        class="h-4 w-4 mr-2"
        :class="
          isActiveState ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground'
        "
      />
      <span :class="isActiveState ? '' : 'text-muted-foreground group-hover:text-foreground'">{{
        item.name
      }}</span>
    </DropdownMenuSubTrigger>
    <DropdownMenuSubContent align="start" class="min-w-[160px]">
      <LayoutTopNavDropdownItem
        v-for="child in item.children"
        :key="child.id"
        :item="child"
        :active-id="activeId"
        @navigate="handleChildNavigate"
      />
    </DropdownMenuSubContent>
  </DropdownMenuSub>

  <!-- 无子菜单：显示普通菜单项 -->
  <DropdownMenuItem
    v-else
    :class="[
      'cursor-pointer relative transition-all duration-200 group rounded-md mx-1',
      isActive ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-muted/50',
    ]"
    @click="handleClick"
  >
    <!-- 左侧激活指示条 -->
    <span
      v-if="isActive"
      class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 bg-gradient-to-b from-primary/80 via-primary to-primary/80 rounded-r-full shadow-sm shadow-primary/20"
    />
    <component
      :is="item.icon"
      v-if="item.icon && isComponent(item.icon)"
      class="h-4 w-4 mr-2"
      :class="isActive ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground'"
    />
    <span :class="isActive ? '' : 'text-muted-foreground group-hover:text-foreground'">{{
      item.name
    }}</span>
  </DropdownMenuItem>
</template>
