<script setup lang="ts">
import type { MenuItem, LayoutSearchProps } from '../types'
import { computed, ref, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMagicKeys, whenever, useEventListener } from '@vueuse/core'
import { Search } from '@lucide/vue'
import {
  CommandDialog,
  CommandInput,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandSeparator,
} from '../../ui/command'
import { Kbd } from '../../ui/kbd'
import { ScrollArea } from '../../ui/scroll-area'

const props = withDefaults(defineProps<LayoutSearchProps>(), {
  placeholder: 'Search menu...',
  description: 'Search for menu or features',
  shortcutKey: 'k',
  shortcutModifiers: () => ['ctrl', 'meta'],
  menuItems: () => [],
  recentItems: () => [],
  maxRecentItems: 5,
  showRecent: true,
  groupTitles: () => ({
    recent: 'Recent Access',
    navigation: 'Navigation',
    actions: 'Actions',
  }),
  emptyText: 'No results found',
})

const emit = defineEmits<{
  select: [item: MenuItem]
  open: []
  close: []
}>()

const router = useRouter()
const open = ref(false)
const isExpanded = ref(false)
const searchValue = ref('')

type SearchMenuItem = MenuItem & { parentTitle?: string }

const { ctrl_k, meta_k } = useMagicKeys({
  passive: false,
  onEventFired(e) {
    if (e.key === 'k' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
    }
  },
})

const stopCtrlK = whenever(ctrl_k, openSearchDialog)
const stopMetaK = whenever(meta_k, openSearchDialog)

const isMac =
  typeof navigator !== 'undefined' &&
  /Mac|iPhone|iPad|iPod/.test(navigator.platform || navigator.userAgent)

const shortcutKeys = computed(() => {
  const tokens: string[] = []
  const modifiers = props.shortcutModifiers ?? []

  if (isMac && modifiers.includes('meta')) {
    tokens.push('⌘')
  } else if (modifiers.includes('ctrl')) {
    tokens.push('Ctrl')
  } else if (modifiers.includes('meta')) {
    tokens.push('Meta')
  }

  if (props.shortcutKey) {
    tokens.push(
      props.shortcutKey.length === 1 ? props.shortcutKey.toUpperCase() : props.shortcutKey,
    )
  }

  return tokens
})

/**
 * 澶勭悊澶辩劍浜嬩欢
 */
function handleBlur() {
  setTimeout(() => {
    if (!searchValue.value) {
      isExpanded.value = false
    }
  }, 150)
}

/**
 * 鎵撳紑鎼滅储寮规
 */
function openSearchDialog() {
  open.value = true
  emit('open')
}

/**
 * 鏀剁缉杈撳叆妗? */
function collapse() {
  isExpanded.value = false
  searchValue.value = ''
}

/**
 * 澶勭悊閿洏浜嬩欢
 */
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (open.value) {
      open.value = false
    } else if (isExpanded.value) {
      collapse()
    }
  }
}

/**
 * 浣跨敤 useEventListener 鏇夸唬鎵嬪姩浜嬩欢鐩戝惉
 * 鑷姩鍦ㄧ粍浠跺嵏杞芥椂娓呯悊
 */
useEventListener(document, 'keydown', handleKeydown)

function handleOpenChange(value: boolean) {
  open.value = value
  if (value) {
    emit('open')
  } else {
    emit('close')
  }
}

/**
 * 鎵佸钩鍖栬彍鍗曢」锛岀敤浜庢悳绱? */
const flattenedMenuItems = computed<SearchMenuItem[]>(() => {
  const result: SearchMenuItem[] = []

  function flatten(menuItems: MenuItem[], parentTitle?: string) {
    for (const item of menuItems) {
      if (!item.hideInMenu) {
        result.push({ ...item, parentTitle })
      }
      if (item.children?.length) {
        flatten(item.children, item.name)
      }
    }
  }

  flatten(props.menuItems)
  return result
})

/**
 * 澶勭悊閫夋嫨鑿滃崟椤? */
function handleSelect(item: MenuItem) {
  emit('select', item)

  if (item.href) {
    window.open(item.href, '_blank')
  } else if (item.path) {
    router.push(item.path)
  }

  open.value = false
}

/**
 * 鑾峰彇鑿滃崟椤瑰浘鏍? */
function getItemIcon(item: MenuItem) {
  return item.icon
}

/**
 * 缁勪欢鍗歌浇鏃舵竻鐞?whenever 鐩戝惉鍣? */
onUnmounted(() => {
  stopCtrlK()
  stopMetaK()
})

defineExpose({
  open: openSearchDialog,
  close: () => {
    open.value = false
    emit('close')
  },
  toggle: openSearchDialog,
  collapse,
})
</script>

<template>
  <div data-slot="layout-search" class="relative" @click="openSearchDialog">
    <div
      :class="[
        'flex items-center h-9 rounded-lg border border-border/40 bg-background/60 backdrop-blur-md supports-[backdrop-filter]:bg-background/50',
        'transition-all duration-300 ease-out',
        'hover:border-border/60 hover:bg-accent/30 dark:hover:bg-accent/20',
        'focus-within:ring-2 focus-within:ring-ring/40 focus-within:border-ring/50',
        isExpanded ? 'w-9 px-2.5 sm:w-60 sm:px-3' : 'w-9 px-2.5 sm:w-44 sm:px-3',
      ]"
    >
      <Search class="h-4 w-4 shrink-0 text-muted-foreground" />

      <input
        v-model="searchValue"
        type="text"
        :placeholder="isExpanded ? placeholder : placeholder"
        class="hidden min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground sm:ml-2 sm:block"
        @click.stop
        @focus="isExpanded = true"
        @blur="handleBlur"
        @keydown.enter="openSearchDialog"
        @keydown.escape="collapse"
      />

      <span
        v-if="shortcutKeys.length"
        class="ml-2 hidden shrink-0 items-center gap-1 xl:inline-flex"
      >
        <Kbd v-for="key in shortcutKeys" :key="key">
          {{ key }}
        </Kbd>
      </span>
    </div>
  </div>

  <CommandDialog
    v-model:open="open"
    :title="placeholder"
    :description="description"
    class="max-w-lg"
    @update:open="handleOpenChange"
  >
    <CommandInput :placeholder="placeholder" :default-value="searchValue" />
    <ScrollArea class="h-[320px]">
      <CommandEmpty class="py-6 text-center text-sm text-muted-foreground">{{
        emptyText
      }}</CommandEmpty>

      <CommandGroup
        v-if="showRecent && recentItems.length > 0"
        :heading="groupTitles.recent"
        class="p-2"
      >
        <CommandItem
          v-for="item in recentItems.slice(0, maxRecentItems)"
          :key="item.id"
          :value="item.name"
          class="cursor-pointer rounded-lg px-3 py-2 transition-colors data-[highlighted]:bg-accent/50"
          @select="handleSelect(item)"
        >
          <component
            :is="getItemIcon(item)"
            v-if="item.icon"
            class="h-4 w-4 mr-3 text-muted-foreground"
          />
          <span class="text-sm">{{ item.name }}</span>
        </CommandItem>
      </CommandGroup>

      <CommandSeparator
        v-if="showRecent && recentItems.length > 0 && flattenedMenuItems.length > 0"
        class="my-1"
      />

      <CommandGroup
        v-if="flattenedMenuItems.length > 0"
        :heading="groupTitles.navigation"
        class="p-2"
      >
        <CommandItem
          v-for="item in flattenedMenuItems"
          :key="item.id"
          :value="item.name"
          class="cursor-pointer rounded-lg px-3 py-2 transition-colors data-[highlighted]:bg-accent/50"
          @select="handleSelect(item)"
        >
          <component
            :is="getItemIcon(item)"
            v-if="item.icon"
            class="h-4 w-4 mr-3 text-muted-foreground"
          />
          <span class="text-sm">{{ item.name }}</span>
          <span v-if="item.parentTitle" class="ml-auto text-xs text-muted-foreground">
            {{ item.parentTitle }}
          </span>
        </CommandItem>
      </CommandGroup>

      <slot name="actions" :on-select="handleSelect" />
    </ScrollArea>
  </CommandDialog>
</template>
