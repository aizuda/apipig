<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Check, ChevronDown, LoaderCircle, Search } from '@lucide/vue'
import {
  Button,
  Input,
  Popover,
  PopoverContent,
  PopoverTrigger,
  toast,
} from '@tabtab/ui'
import { wechatBotApi, type WechatBot } from '@/api/ai-applications/wechat-bot'

const props = withDefaults(
  defineProps<{
    id?: string
    modelValue: string
    disabled?: boolean
    placeholder?: string
  }>(),
  { placeholder: '选择在线 Bot' },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  select: [bot: WechatBot]
}>()

const open = ref(false)
const name = ref('')
const bots = ref<WechatBot[]>([])
const loading = ref(false)
const selectedBot = ref<WechatBot | null>(null)
const triggerLabel = computed(() => {
  if (!props.modelValue) return props.placeholder
  return selectedBot.value?.name || '已选择的 Bot 当前不可用'
})
let searchTimer: number | undefined
let requestId = 0

async function loadBots() {
  const currentRequestId = ++requestId
  loading.value = true
  try {
    const result = await wechatBotApi.list({
      name: name.value.trim(),
      status: 'ONLINE',
      enabled: true,
    })
    if (currentRequestId !== requestId) return
    bots.value = result || []
    selectedBot.value =
      bots.value.find((bot) => bot.id === props.modelValue) ||
      (selectedBot.value?.id === props.modelValue ? selectedBot.value : null)
  } catch (error) {
    if (currentRequestId !== requestId) return
    toast.error(error instanceof Error ? error.message : '加载微信 Bot 失败')
  } finally {
    if (currentRequestId === requestId) loading.value = false
  }
}

function selectBot(bot: WechatBot) {
  selectedBot.value = bot
  emit('update:modelValue', bot.id)
  emit('select', bot)
  open.value = false
}

watch(open, (value) => {
  window.clearTimeout(searchTimer)
  if (value) void loadBots()
})

watch(name, () => {
  if (!open.value) return
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(loadBots, 250)
})

watch(
  () => props.modelValue,
  (value) => {
    if (!value) selectedBot.value = null
    else {
      selectedBot.value =
        bots.value.find((bot) => bot.id === value) ||
        (selectedBot.value?.id === value ? selectedBot.value : null)
    }
  },
)

onBeforeUnmount(() => window.clearTimeout(searchTimer))
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        :id="id"
        type="button"
        variant="outline"
        class="h-10 w-full justify-between px-3 font-normal"
        :disabled="disabled"
      >
        <span class="truncate">{{ triggerLabel }}</span>
        <LoaderCircle v-if="loading" class="size-4 shrink-0 animate-spin text-muted-foreground" />
        <ChevronDown v-else class="size-4 shrink-0 text-muted-foreground" />
      </Button>
    </PopoverTrigger>
    <PopoverContent
      align="start"
      class="w-[var(--reka-popover-trigger-width)] p-2"
      :side-offset="6"
    >
      <div class="relative mb-2">
        <Search
          class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
        />
        <Input v-model="name" class="h-9 pl-9" placeholder="按自定义名称搜索" />
      </div>
      <div class="max-h-60 space-y-1 overflow-y-auto">
        <button
          v-for="bot in bots"
          :key="bot.id"
          type="button"
          class="flex h-9 w-full items-center justify-between gap-2 rounded-sm px-2 text-left text-sm hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          :class="{ 'bg-accent': bot.id === modelValue }"
          @click="selectBot(bot)"
        >
          <span class="truncate">{{ bot.name }}</span>
          <Check v-if="bot.id === modelValue" class="size-4 shrink-0 text-primary" />
        </button>
        <div
          v-if="loading"
          class="flex h-20 items-center justify-center text-muted-foreground"
        >
          <LoaderCircle class="size-4 animate-spin" />
        </div>
        <div
          v-else-if="bots.length === 0"
          class="flex h-20 items-center justify-center text-sm text-muted-foreground"
        >
          未找到匹配的在线 Bot
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>
