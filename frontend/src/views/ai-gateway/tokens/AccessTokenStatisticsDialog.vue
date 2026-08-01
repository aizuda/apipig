<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Activity, ChartBarStacked, KeyRound, RefreshCw, Search, Tags } from '@lucide/vue'
import { Badge, Button, Dialog, DialogFixedContent, Input, Skeleton, toast } from '@tabtab/ui'
import {
  aiGatewayApi,
  type AccessTokenStatisticRecord,
  type AccessTokenStatistics,
  type AccessTokenStatisticsParams,
  type AccessTokenTag,
} from '@/api/ai-gateway'
import AppPagination from '@/components/AppPagination.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

type RangePreset = '7' | '30' | 'all' | 'custom'

const emptyStatistics = (): AccessTokenStatistics => ({
  startAt: 0,
  endAt: 0,
  total: 0,
  page: 1,
  pageSize: 10,
  tokenCount: 0,
  activeTokenCount: 0,
  callCount: 0,
  successCount: 0,
  failureCount: 0,
  totalTokens: 0,
  modelStatistics: [],
  items: [],
})

const loading = ref(false)
const errorMessage = ref('')
const rangePreset = ref<RangePreset>('7')
const customStart = ref('')
const customEnd = ref('')
const searchQuery = ref('')
const selectedTagId = ref('all')
const tokenTags = ref<AccessTokenTag[]>([])
const tagLoading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)
const statistics = ref<AccessTokenStatistics>(emptyStatistics())
let loadSequence = 0
let searchTimer: ReturnType<typeof setTimeout> | undefined
let suppressSearchReload = false

const maxTokens = computed(() =>
  Math.max(1, ...statistics.value.items.map((item) => item.totalTokens)),
)
const successRate = computed(() =>
  statistics.value.callCount > 0
    ? (statistics.value.successCount / statistics.value.callCount) * 100
    : 0,
)
const promptTokens = computed(() =>
  statistics.value.items.reduce((total, item) => total + item.promptTokens, 0),
)
const completionTokens = computed(() =>
  statistics.value.items.reduce((total, item) => total + item.completionTokens, 0),
)
const cacheTokens = computed(() =>
  statistics.value.items.reduce(
    (total, item) => total + item.cacheReadTokens + item.cacheWriteTokens,
    0,
  ),
)
const availableTags = computed(() =>
  tokenTags.value.filter((tag): tag is AccessTokenTag & { id: string } => Boolean(tag.id)),
)

function rangeParams(): AccessTokenStatisticsParams {
  if (rangePreset.value === 'all') return {}
  if (rangePreset.value === 'custom') {
    const startAt = new Date(customStart.value).getTime()
    const endAt = new Date(customEnd.value).getTime()
    if (!Number.isFinite(startAt) || !Number.isFinite(endAt)) {
      throw new Error('请选择完整的开始时间和结束时间')
    }
    if (endAt < startAt) throw new Error('结束时间不能早于开始时间')
    return { startAt, endAt }
  }
  const days = Number(rangePreset.value)
  const end = new Date()
  end.setHours(23, 59, 59, 999)
  const start = new Date(end)
  start.setDate(start.getDate() - days + 1)
  start.setHours(0, 0, 0, 0)
  return { startAt: start.getTime(), endAt: end.getTime() }
}

function toDatetimeLocal(value: Date) {
  const localTime = new Date(value.getTime() - value.getTimezoneOffset() * 60_000)
  return localTime.toISOString().slice(0, 16)
}

function initializeCustomRange() {
  if (customStart.value && customEnd.value) return
  const end = new Date()
  const start = new Date(end)
  start.setDate(start.getDate() - 6)
  start.setHours(0, 0, 0, 0)
  customStart.value = toDatetimeLocal(start)
  customEnd.value = toDatetimeLocal(end)
}

async function loadStatistics() {
  const sequence = ++loadSequence
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await aiGatewayApi.tokenStatistics({
      ...rangeParams(),
      page: currentPage.value,
      pageSize: pageSize.value,
      keyword: searchQuery.value.trim() || undefined,
      tagId: selectedTagId.value === 'all' ? undefined : selectedTagId.value,
    })
    if (sequence !== loadSequence) return
    statistics.value = { ...result, items: result.items || [] }
    total.value = result.total || 0
    currentPage.value = result.page || currentPage.value
    pageSize.value = result.pageSize || pageSize.value
  } catch (error) {
    if (sequence !== loadSequence) return
    errorMessage.value = error instanceof Error ? error.message : 'API 密钥统计加载失败'
    toast.error(errorMessage.value)
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

async function loadTagOptions() {
  tagLoading.value = true
  try {
    tokenTags.value = await aiGatewayApi.tokenTagList()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'API 密钥标签加载失败')
  } finally {
    tagLoading.value = false
  }
}

function setRange(value: RangePreset) {
  if (rangePreset.value === value && statistics.value.items.length) return
  rangePreset.value = value
  currentPage.value = 1
  if (value === 'custom') initializeCustomRange()
  void loadStatistics()
}

function applyCustomRange() {
  currentPage.value = 1
  void loadStatistics()
}

function changePage(page: number) {
  const totalPages = Math.max(1, Math.ceil(total.value / pageSize.value))
  if (page < 1 || page > totalPages || page === currentPage.value || loading.value) return
  currentPage.value = page
  void loadStatistics()
}

function changePageSize(size: number) {
  if (loading.value || size === pageSize.value) return
  pageSize.value = size
  currentPage.value = 1
  void loadStatistics()
}

function changeTag() {
  currentPage.value = 1
  void loadStatistics()
}

function segmentWidth(value: number) {
  return `${Math.max(0, (value / maxTokens.value) * 100)}%`
}

function formatTokens(value?: number) {
  return (value || 0).toLocaleString('zh-CN')
}

function formatCompact(value?: number) {
  return new Intl.NumberFormat('zh-CN', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value || 0)
}

function formatTime(value?: number) {
  if (!value) return '从未使用'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

function itemDetail(item: AccessTokenStatisticRecord) {
  return [
    `输入 ${formatTokens(item.promptTokens)}`,
    `输出 ${formatTokens(item.completionTokens)}`,
    `缓存读取 ${formatTokens(item.cacheReadTokens)}`,
    `缓存写入 ${formatTokens(item.cacheWriteTokens)}`,
    `推理 ${formatTokens(item.reasoningTokens)}`,
  ].join(' · ')
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      suppressSearchReload = true
      searchQuery.value = ''
      suppressSearchReload = false
      selectedTagId.value = 'all'
      currentPage.value = 1
      void loadTagOptions()
      void loadStatistics()
    } else {
      loadSequence += 1
    }
  },
)

watch(
  searchQuery,
  () => {
    if (suppressSearchReload || !props.open) return
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      currentPage.value = 1
      void loadStatistics()
    }, 300)
  },
  { flush: 'sync' },
)

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
  loadSequence += 1
})
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogFixedContent
      title="API 密钥统计分析"
      description="按 API 密钥查看调用情况与 Token 用量分布。"
      class="left-0 top-0 h-dvh max-h-dvh w-screen max-w-none translate-x-0 translate-y-0 rounded-none border-0 sm:left-1/2 sm:top-1/2 sm:h-[calc(100dvh-2rem)] sm:max-w-[calc(100vw-2rem)] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-lg sm:border"
      body-class="overflow-hidden p-0"
      @pointer-down-outside.prevent
    >
      <template #header>
        <div class="mt-4 space-y-3">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div
              class="inline-flex w-fit rounded-md border bg-muted/30 p-0.5"
              aria-label="统计范围"
            >
              <Button
                v-for="option in [
                  { value: '7', label: '近 7 天' },
                  { value: '30', label: '近 30 天' },
                  { value: 'all', label: '全部时间' },
                  { value: 'custom', label: '自定义' },
                ] as const"
                :key="option.value"
                size="sm"
                :variant="rangePreset === option.value ? 'secondary' : 'ghost'"
                class="h-7 rounded-sm px-3"
                :disabled="loading"
                @click="setRange(option.value)"
              >
                {{ option.label }}
              </Button>
            </div>
            <div
              class="grid min-w-0 grid-cols-[minmax(0,1fr)_32px] items-center gap-2 sm:grid-cols-[180px_minmax(0,1fr)_32px]"
            >
              <div class="relative col-span-2 min-w-0 sm:col-span-1">
                <Tags
                  class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
                />
                <select
                  v-model="selectedTagId"
                  class="h-8 w-full rounded-md border bg-background pl-9 pr-2 text-sm text-foreground"
                  :disabled="loading || tagLoading"
                  aria-label="按标签筛选"
                  @change="changeTag"
                >
                  <option value="all">全部标签</option>
                  <option v-for="tag in availableTags" :key="tag.id" :value="tag.id">
                    {{ tag.name }}
                  </option>
                </select>
              </div>
              <div class="relative min-w-0 flex-1 lg:w-72 lg:flex-none">
                <Search
                  class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
                />
                <Input v-model="searchQuery" class="h-8 pl-9" placeholder="搜索 API 密钥" />
              </div>
              <Button
                variant="outline"
                size="icon"
                class="h-8 w-8 shrink-0"
                :disabled="loading"
                title="刷新统计"
                aria-label="刷新统计"
                @click="loadStatistics"
              >
                <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
              </Button>
            </div>
          </div>
          <div
            v-if="rangePreset === 'custom'"
            class="grid gap-2 sm:grid-cols-[auto_minmax(0,1fr)_auto_minmax(0,1fr)_auto] sm:items-center"
          >
            <span class="text-xs text-muted-foreground">开始时间</span>
            <Input v-model="customStart" type="datetime-local" class="h-8" :disabled="loading" />
            <span class="text-xs text-muted-foreground">结束时间</span>
            <Input v-model="customEnd" type="datetime-local" class="h-8" :disabled="loading" />
            <Button size="sm" class="h-8" :disabled="loading" @click="applyCustomRange"
              >查询</Button
            >
          </div>
        </div>
      </template>

      <div class="flex h-full min-h-0 flex-col">
        <div class="grid shrink-0 grid-cols-2 border-b lg:grid-cols-4">
          <div class="border-b px-4 py-3 lg:border-b-0 lg:border-r">
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <KeyRound class="h-3.5 w-3.5" />API 密钥
            </div>
            <p class="mt-1 text-xl font-semibold tabular-nums">
              {{ statistics.activeTokenCount
              }}<span class="text-sm font-normal text-muted-foreground">
                / {{ statistics.tokenCount }}</span
              >
            </p>
          </div>
          <div class="border-b border-l px-4 py-3 lg:border-b-0 lg:border-l-0 lg:border-r">
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <Activity class="h-3.5 w-3.5" />调用次数
            </div>
            <p class="mt-1 text-xl font-semibold tabular-nums">
              {{ formatTokens(statistics.callCount) }}
            </p>
          </div>
          <div class="px-4 py-3 lg:border-r">
            <div class="text-xs text-muted-foreground">成功率</div>
            <p class="mt-1 text-xl font-semibold tabular-nums">{{ successRate.toFixed(1) }}%</p>
          </div>
          <div class="border-l px-4 py-3 lg:border-l-0">
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <ChartBarStacked class="h-3.5 w-3.5" />总 Token
            </div>
            <p class="mt-1 text-xl font-semibold tabular-nums">
              {{ formatTokens(statistics.totalTokens) }}
            </p>
          </div>
        </div>

        <div class="min-h-0 flex-1 overflow-auto px-4 py-4 sm:px-6">
          <div
            v-if="errorMessage"
            class="flex min-h-40 flex-col items-center justify-center gap-3 rounded-md border border-destructive/40 bg-destructive/5 px-4 text-sm text-destructive"
          >
            <span>{{ errorMessage }}</span>
            <Button variant="outline" size="sm" @click="loadStatistics">重新加载</Button>
          </div>

          <div v-else-if="loading && !statistics.items.length" class="space-y-3">
            <Skeleton v-for="index in 6" :key="index" class="h-16 w-full" />
          </div>

          <template v-else>
            <div
              class="mb-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-muted-foreground"
            >
              <span class="font-medium text-foreground">本页 Token 用量</span>
              <span class="flex items-center gap-1.5"
                ><i class="size-2.5 rounded-sm bg-indigo-500"></i>输入
                {{ formatCompact(promptTokens) }}</span
              >
              <span class="flex items-center gap-1.5"
                ><i class="size-2.5 rounded-sm bg-emerald-500"></i>输出
                {{ formatCompact(completionTokens) }}</span
              >
              <span class="flex items-center gap-1.5"
                ><i class="size-2.5 rounded-sm bg-amber-500"></i>缓存
                {{ formatCompact(cacheTokens) }}</span
              >
              <span class="w-full sm:ml-auto sm:w-auto">按总 Token 降序</span>
            </div>

            <div v-if="statistics.items.length" class="border-y">
              <div
                v-for="item in statistics.items"
                :key="item.tokenId"
                class="grid min-h-20 min-w-0 gap-3 overflow-hidden border-b px-1 py-3 last:border-b-0 md:grid-cols-[minmax(150px,240px)_minmax(260px,1fr)_110px] md:items-center"
              >
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="truncate text-sm font-medium" :title="item.tokenName">{{
                      item.tokenName
                    }}</span>
                    <Badge
                      :variant="item.status === 1 ? 'secondary' : 'outline'"
                      class="shrink-0 font-normal"
                    >
                      {{ item.status === 1 ? '启用' : '停用' }}
                    </Badge>
                  </div>
                  <p
                    class="mt-1 truncate text-xs text-muted-foreground"
                    :title="formatTime(item.lastUsedAt)"
                  >
                    {{ formatTokens(item.callCount) }} 次调用 · {{ formatTime(item.lastUsedAt) }}
                  </p>
                </div>

                <div class="min-w-0" :title="itemDetail(item)">
                  <div class="flex h-5 w-full overflow-hidden rounded-sm bg-muted">
                    <div
                      class="h-full bg-indigo-500"
                      :style="{ width: segmentWidth(item.promptTokens) }"
                    ></div>
                    <div
                      class="h-full bg-emerald-500"
                      :style="{ width: segmentWidth(item.completionTokens) }"
                    ></div>
                    <div
                      class="h-full bg-amber-500"
                      :style="{ width: segmentWidth(item.cacheReadTokens) }"
                    ></div>
                    <div
                      class="h-full bg-rose-500"
                      :style="{ width: segmentWidth(item.cacheWriteTokens) }"
                    ></div>
                  </div>
                  <div
                    class="mt-1.5 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-muted-foreground"
                  >
                    <span>输入 {{ formatCompact(item.promptTokens) }}</span>
                    <span>输出 {{ formatCompact(item.completionTokens) }}</span>
                    <span
                      >缓存读/写 {{ formatCompact(item.cacheReadTokens) }} /
                      {{ formatCompact(item.cacheWriteTokens) }}</span
                    >
                    <span>成功/失败 {{ item.successCount }} / {{ item.failureCount }}</span>
                  </div>
                </div>

                <div class="text-left md:text-right">
                  <p class="font-semibold tabular-nums">{{ formatTokens(item.totalTokens) }}</p>
                  <p class="text-[11px] text-muted-foreground">Token</p>
                </div>
              </div>
            </div>

            <div
              v-else
              class="flex min-h-48 items-center justify-center rounded-md border border-dashed text-sm text-muted-foreground"
            >
              {{ searchQuery ? '没有匹配的 API 密钥' : '暂无 API 密钥统计数据' }}
            </div>
          </template>
        </div>
        <AppPagination
          v-if="!errorMessage"
          class="shrink-0"
          :total="total"
          :page="currentPage"
          :page-size="pageSize"
          :loading="loading"
          @change-page="changePage"
          @change-page-size="changePageSize"
        />
      </div>
    </DialogFixedContent>
  </Dialog>
</template>
