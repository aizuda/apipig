<script setup lang="ts">
import type { LayoutSidebarMenuItem } from './config'
import { computed } from 'vue'
import { Button } from '../../ui/button'
import { Badge } from '../../ui/badge'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../../ui/tooltip'
import { formatBadge, getButtonVariant } from '../composables'
import { isComponent } from '../composables'

/**
 * LayoutSidebarItem - 单级菜单项组件
 * 支持折叠/展开两种状态
 */

interface Props {
  /** 菜单项数据 */
  item: LayoutSidebarMenuItem
  /** 是否折叠 */
  collapsed: boolean
  /** 是否激活 */
  active: boolean
  /** 菜单标题（已翻译的文本） */
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
})

const emit = defineEmits<{
  /** 导航事件 */
  (e: 'navigate', path: string): void
}>()

/**
 * 处理点击
 */
function handleClick(): void {
  emit('navigate', props.item.path)
}

/**
 * 按钮变体
 */
const variant = computed(() => getButtonVariant(props.active))

/**
 * 菜单标题（优先使用 props.title，否则使用 item.title）
 */
const menuTitle = computed(() => {
  return props.title || props.item.title
})

/**
 * ARIA 标签（折叠状态下使用）
 */
const ariaLabel = computed(() => {
  if (!props.collapsed) {
    return undefined
  }
  return props.item.badge ? `${menuTitle.value} (${props.item.badge} 条通知)` : menuTitle.value
})
</script>

<template>
  <!-- 折叠状态：显示 Tooltip -->
  <TooltipProvider v-if="collapsed">
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          :variant="variant"
          size="icon"
          role="menuitem"
          :aria-label="ariaLabel"
          :aria-current="active ? 'page' : undefined"
          class="relative h-10 w-10 transition-colors duration-200 rounded-lg"
          :class="[
            active
              ? 'bg-primary text-primary-foreground'
              : 'hover:bg-primary/10 hover:text-primary',
          ]"
          @click="handleClick"
        >
          <component
            :is="item.icon"
            v-if="item.icon && isComponent(item.icon)"
            class="h-5 w-5"
            aria-hidden="true"
          />

          <Badge
            v-if="item.badge"
            variant="destructive"
            class="absolute -top-1 -left-1 h-4 min-w-4 !px-1 text-[10px]"
            role="status"
            :aria-label="`${item.badge} 条通知`"
          >
            {{
              formatBadge(typeof item.badge === 'number' ? item.badge : parseInt(item.badge) || 0)
            }}
          </Badge>
        </Button>
      </TooltipTrigger>

      <TooltipContent side="right" :side-offset="10">
        <div class="flex items-center gap-2">
          <span>{{ menuTitle }}</span>
          <Badge v-if="item.badge" variant="destructive" class="h-4 px-1 text-[10px]">
            {{ item.badge }}
          </Badge>
        </div>
      </TooltipContent>
    </Tooltip>
  </TooltipProvider>

  <!-- 展开状态：显示完整按钮 -->
  <Button
    v-else
    :variant="variant"
    role="menuitem"
    :aria-current="active ? 'page' : undefined"
    class="w-full justify-start gap-2 h-9 px-3 group transition-all duration-200 rounded-lg"
    :class="[
      active
        ? 'bg-primary/10 text-primary font-medium shadow-[inset_3px_0_0_0_hsl(var(--primary))] hover:bg-primary/15'
        : 'hover:bg-muted/50 hover:text-foreground',
    ]"
    @click="handleClick"
  >
    <component
      :is="item.icon"
      v-if="item.icon && isComponent(item.icon)"
      class="h-4 w-4"
      :class="[
        active ? 'text-primary' : 'text-muted-foreground group-hover:text-accent-foreground',
      ]"
      aria-hidden="true"
    />

    <span class="flex-1 text-left truncate text-sm">{{ menuTitle }}</span>

    <Badge
      v-if="item.badge"
      variant="destructive"
      class="h-4 px-1.5 text-[10px] font-medium"
      role="status"
    >
      {{ item.badge }}
    </Badge>
  </Button>
</template>
