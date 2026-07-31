<script setup lang="ts">
import type { MenuItem } from '../types'
import { cn } from '@tabtab/utils'
import { ref, computed } from 'vue'
import { ChevronRight } from '@lucide/vue'
import { isComponent } from '../composables'

export interface LayoutDoubleSidebarMenuProps {
  items: MenuItem[]
  activeId?: string
  level?: number
}

const props = withDefaults(defineProps<LayoutDoubleSidebarMenuProps>(), {
  level: 0,
})

const emit = defineEmits<{
  (e: 'select', item: MenuItem): void
}>()

const expandedIds = ref<string[]>([])

function toggleExpand(id: string) {
  const index = expandedIds.value.indexOf(id)
  if (index === -1) {
    expandedIds.value.push(id)
  } else {
    expandedIds.value.splice(index, 1)
  }
}

function handleSelect(item: MenuItem) {
  emit('select', item)
}

function handleParentClick(item: MenuItem) {
  toggleExpand(item.id)
}

const indentClass = computed(() => {
  return props.level > 0 ? 'mx-2' : ''
})

/**
 * 棰勮绠楁瘡涓彍鍗曢」鐨勬縺娲荤姸鎬侊紝閬垮厤妯℃澘涓噸澶嶈皟鐢ㄥ嚱鏁? */
const itemActiveStates = computed(() => {
  const states = new Map<string, boolean>()

  function checkSubtree(item: MenuItem, targetId?: string): boolean {
    if (!targetId) return false
    if (item.id === targetId) return true
    if (item.children) {
      return item.children.some((child) => checkSubtree(child, targetId))
    }
    return false
  }

  for (const item of props.items) {
    states.set(item.id, checkSubtree(item, props.activeId))
  }

  return states
})

/**
 * 鑾峰彇鑿滃崟椤圭殑婵€娲荤姸鎬侊紙浠庣紦瀛樹腑璇诲彇锛? */
function isItemActive(item: MenuItem): boolean {
  return itemActiveStates.value.get(item.id) ?? false
}
</script>

<template>
  <div class="flex flex-col gap-1">
    <template v-for="item in items" :key="item.id">
      <template v-if="item.children?.length">
        <button
          :class="
            cn(
              'flex min-h-9 w-full items-center justify-between rounded-md px-3 text-sm transition-all duration-200',
              'hover:bg-primary/5 hover:text-primary',
              'active:scale-[0.98]',
              indentClass,
              isItemActive(item)
                ? 'bg-primary/10 text-primary font-medium shadow-[inset_3px_0_0_0_hsl(var(--primary))]'
                : 'text-sidebar-foreground/70',
            )
          "
          @click="handleParentClick(item)"
        >
          <span class="flex items-center gap-2 min-w-0">
            <component
              :is="item.icon"
              v-if="isComponent(item.icon)"
              :class="
                cn(
                  'h-4 w-4 shrink-0 transition-colors duration-200',
                  isItemActive(item) ? 'text-primary' : 'text-sidebar-foreground/50',
                )
              "
            />
            <span v-else-if="item.icon" class="text-base shrink-0">{{ item.icon }}</span>
            <span class="truncate">{{ item.name }}</span>
          </span>
          <ChevronRight
            :class="
              cn(
                'h-4 w-4 shrink-0 transition-transform duration-200',
                isItemActive(item) ? 'text-primary' : 'text-sidebar-foreground/50',
                expandedIds.includes(item.id) && 'rotate-90',
              )
            "
          />
        </button>
        <div v-if="expandedIds.includes(item.id)" class="border-l border-primary/30 ml-3 mr-2 pl-2">
          <LayoutDoubleSidebarMenu
            :items="item.children"
            :active-id="activeId"
            :level="level + 1"
            @select="handleSelect"
          />
        </div>
      </template>
      <button
        v-else
        :class="
          cn(
            'flex min-h-9 w-full items-center rounded-md px-3 text-sm transition-all duration-200',
            'hover:bg-primary/5 hover:text-primary',
            'active:scale-[0.98]',
            indentClass,
            activeId === item.id
              ? 'bg-primary/10 text-primary font-medium shadow-[inset_3px_0_0_0_hsl(var(--primary))]'
              : 'text-sidebar-foreground/70',
          )
        "
        @click="handleSelect(item)"
      >
        <span class="flex items-center gap-2 min-w-0">
          <component
            :is="item.icon"
            v-if="isComponent(item.icon)"
            :class="
              cn(
                'h-4 w-4 shrink-0 transition-colors duration-200',
                activeId === item.id ? 'text-primary' : 'text-sidebar-foreground/50',
              )
            "
          />
          <span v-else-if="item.icon" class="text-base shrink-0">{{ item.icon }}</span>
          <span class="truncate">{{ item.name }}</span>
        </span>
      </button>
    </template>
  </div>
</template>
