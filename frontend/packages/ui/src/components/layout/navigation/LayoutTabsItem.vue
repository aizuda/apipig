<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@tabtab/utils'
import { computed } from 'vue'
import { X, Pin, GripVertical, RefreshCw, Loader2 } from '@lucide/vue'
import { isComponent } from '../composables'
import type { TabItem } from '../types'

export type { TabItem }

interface Props {
  tab: TabItem
  active?: boolean
  class?: HTMLAttributes['class']
}

const props = withDefaults(defineProps<Props>(), {
  active: false,
})

const emit = defineEmits<{
  close: [key: string]
  click: [key: string]
  dragstart: [e: DragEvent, key: string]
  dragover: [e: DragEvent, key: string]
  dragend: [e: DragEvent]
}>()

/**
 * 鏄惁鏄剧ず鍏抽棴鎸夐挳
 */
const showClose = computed(() => !props.tab.affix)

/**
 * 澶勭悊鐐瑰嚮浜嬩欢
 */
function handleClick() {
  emit('click', props.tab.key)
}

/**
 * 澶勭悊鍏抽棴浜嬩欢
 */
function handleClose(e: MouseEvent) {
  e.stopPropagation()
  emit('close', props.tab.key)
}

/**
 * 澶勭悊鎷栨嫿寮€濮? */
function handleDragStart(e: DragEvent) {
  emit('dragstart', e, props.tab.key)
}

/**
 * 澶勭悊鎷栨嫿鎮仠
 */
function handleDragOver(e: DragEvent) {
  emit('dragover', e, props.tab.key)
}

/**
 * 澶勭悊鎷栨嫿缁撴潫
 */
function handleDragEnd(e: DragEvent) {
  emit('dragend', e)
}
</script>

<template>
  <div
    data-slot="layout-tabs-item"
    :data-tab-key="tab.key"
    :data-active="active"
    draggable="true"
    :class="
      cn(
        'group relative flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md cursor-pointer transition-colors duration-200 select-none whitespace-nowrap border',
        active
          ? ['bg-primary text-primary-foreground border-primary']
          : [
              'bg-muted/50 text-muted-foreground border-transparent',
              'hover:bg-muted/50 hover:text-foreground hover:border-border',
            ],
        tab.isRefreshing && 'animate-pulse',
        props.class,
      )
    "
    @click="handleClick"
    @dragstart="handleDragStart"
    @dragover="handleDragOver"
    @dragend="handleDragEnd"
  >
    <GripVertical
      class="h-3 w-3 opacity-0 group-hover:opacity-50 cursor-grab active:cursor-grabbing transition-opacity"
      :class="active ? 'text-primary/70' : 'text-muted-foreground'"
    />
    <Loader2 v-if="tab.isLoading" class="h-3.5 w-3.5 shrink-0 animate-spin" />
    <RefreshCw v-else-if="tab.isRefreshing" class="h-3.5 w-3.5 shrink-0 animate-spin" />
    <component
      :is="tab.icon"
      v-else-if="tab.icon && isComponent(tab.icon)"
      class="h-3.5 w-3.5 shrink-0"
    />
    <span class="max-w-[120px] truncate">{{ tab.title }}</span>
    <Pin
      v-if="tab.affix"
      class="h-3 w-3 shrink-0"
      :class="active ? 'text-primary/70' : 'text-muted-foreground'"
    />
    <button
      v-if="showClose"
      class="ml-0.5 rounded-sm opacity-0 group-hover:opacity-100 transition-all p-0.5 min-w-[20px] min-h-[20px] flex items-center justify-center"
      :class="active ? 'hover:bg-primary/20' : 'hover:bg-foreground/10'"
      @click="handleClose"
    >
      <X class="h-3 w-3" />
    </button>
    <div
      v-if="active"
      class="absolute bottom-0 left-1/2 -translate-x-1/2 w-4 h-0.5 rounded-full bg-primary/50"
    />
  </div>
</template>

<style scoped>
.tab-item {
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.tab-item:hover {
  transform: translateY(-1px);
}

.tab-item:active {
  transform: translateY(0);
}

.tab-item[data-active='true'] {
  animation: tab-activate 0.2s ease-out;
}

@keyframes tab-activate {
  0% {
    transform: scale(0.95);
    opacity: 0.8;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .tab-item {
    transition: none;
  }

  .tab-item[data-active='true'] {
    animation: none;
  }
}
</style>
