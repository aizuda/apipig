<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Check, ChevronDown, Search } from '@lucide/vue'
import { Button, Input, Popover, PopoverContent, PopoverTrigger } from '@tabtab/ui'
import ProviderBrandIcon from './ProviderBrandIcon.vue'
import { getProviderIcon, providerIcons, resolveProviderIcon } from './provider-icons'

const props = defineProps<{
  modelValue?: string
  fallback?: string
  secondaryFallback?: string
  disabled?: boolean
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const open = ref(false)
const keyword = ref('')
const selectedIcon = computed(() => getProviderIcon(props.modelValue))
const automaticIcon = computed(() => resolveProviderIcon(props.fallback, props.secondaryFallback))
const displayedIcon = computed(() => selectedIcon.value || automaticIcon.value)
const triggerLabel = computed(() => {
  if (selectedIcon.value) return selectedIcon.value.name
  if (automaticIcon.value) return `自动匹配 · ${automaticIcon.value.name}`
  return '选择供应商图标'
})
const filteredIcons = computed(() => {
  const query = keyword.value.trim().toLocaleLowerCase()
  if (!query) return providerIcons
  return providerIcons.filter((icon) =>
    [icon.id, icon.name, ...icon.keywords].some((value) =>
      value.toLocaleLowerCase().includes(query),
    ),
  )
})

function selectIcon(iconID: string) {
  emit('update:modelValue', iconID)
  open.value = false
}

watch(open, (value) => {
  if (!value) keyword.value = ''
})
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        type="button"
        variant="outline"
        class="h-10 w-full justify-between px-3 font-normal"
        :disabled="disabled"
      >
        <span class="flex min-w-0 items-center gap-2">
          <ProviderBrandIcon :icon="displayedIcon?.id" size="sm" />
          <span class="truncate">{{ triggerLabel }}</span>
        </span>
        <ChevronDown class="size-4 shrink-0 text-muted-foreground" />
      </Button>
    </PopoverTrigger>
    <PopoverContent align="start" class="w-[min(440px,calc(100vw-2rem))] p-2" :side-offset="6">
      <div class="relative mb-2">
        <Search
          class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
        />
        <Input v-model="keyword" class="h-9 pl-9" placeholder="搜索供应商图标" />
      </div>
      <div class="grid max-h-72 grid-cols-4 gap-1 overflow-y-auto pr-1 sm:grid-cols-5">
        <button
          v-for="icon in filteredIcons"
          :key="icon.id"
          type="button"
          class="relative flex h-[76px] min-w-0 flex-col items-center justify-center gap-1.5 rounded-md border border-transparent px-1 text-center transition-colors hover:border-border hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          :class="{ 'border-primary/50 bg-primary/5': modelValue === icon.id }"
          :title="icon.name"
          @click="selectIcon(icon.id)"
        >
          <ProviderBrandIcon :icon="icon.id" size="md" />
          <span class="w-full truncate text-[11px] leading-4">{{ icon.name }}</span>
          <Check
            v-if="modelValue === icon.id"
            class="absolute right-1 top-1 size-3.5 text-primary"
          />
        </button>
      </div>
      <div
        v-if="filteredIcons.length === 0"
        class="flex h-24 items-center justify-center text-sm text-muted-foreground"
      >
        没有匹配的供应商图标
      </div>
    </PopoverContent>
  </Popover>
</template>
