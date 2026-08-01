<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  Activity,
  ChartNoAxesCombined,
  ChartPie,
  CheckCircle2,
  Clock3,
  Eye,
  KeyRound,
  Layers3,
  LogOut,
  RefreshCw,
  Search,
  ShieldAlert,
  X,
} from '@lucide/vue'
import { Badge, Button, Input, Skeleton, toast } from '@tabtab/ui'
import { aiGatewayApi, type AccessTokenStatistics, type CallLog } from '@/api/ai-gateway'
import AppPagination from '@/components/AppPagination.vue'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'
import { useUserStore } from '@/stores/user'

defineOptions({ name: 'APITokenUsage' })

type RangePreset = '24h' | '7d' | '30d' | 'all' | 'custom'

const router = useRouter()
const userStore = useUserStore()
const loading = ref(false)
const logsLoading = ref(false)
const errorMessage = ref('')
const rangePreset = ref<RangePreset>('7d')
const customStart = ref('')
const customEnd = ref('')
const searchQuery = ref('')
const statusFilter = ref<'all' | 'success' | 'failed'>('all')
const modelSearchQuery = ref('')
const modelSort = ref<'tokens' | 'calls' | 'cost' | 'latency'>('tokens')
const selectedModel = ref('')
const logs = ref<CallLog[]>([])
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)
const statistics = ref<AccessTokenStatistics>({
  startAt: 0,
  endAt: 0,
  total: 0,
  page: 1,
  pageSize: 1,
  tokenCount: 0,
  activeTokenCount: 0,
  callCount: 0,
  successCount: 0,
  failureCount: 0,
  totalTokens: 0,
  modelStatistics: [],
  items: [],
})
let loadSequence = 0
let logSequence = 0
let searchTimer: ReturnType<typeof setTimeout> | undefined

const rangeOptions: { value: RangePreset; label: string }[] = [
  { value: '24h', label: '24 小时' },
  { value: '7d', label: '7 天' },
  { value: '30d', label: '30 天' },
  { value: 'all', label: '全部' },
  { value: 'custom', label: '自定义' },
]
const integerFormatter = new Intl.NumberFormat('zh-CN')
const compactFormatter = new Intl.NumberFormat('zh-CN', {
  notation: 'compact',
  maximumFractionDigits: 1,
})
const dateTimeFormatter = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
})

const usage = computed(() => statistics.value.items[0])
const successRate = computed(() => {
  if (!statistics.value.callCount) return 0
  return (statistics.value.successCount / statistics.value.callCount) * 100
})
const expiryText = computed(() =>
  userStore.expiresAt ? dateTimeFormatter.format(new Date(userStore.expiresAt)) : '-',
)
const tokenParts = computed(() => [
  { label: '输入', value: usage.value?.promptTokens || 0, color: 'bg-blue-500' },
  { label: '输出', value: usage.value?.completionTokens || 0, color: 'bg-emerald-500' },
  { label: '缓存读取', value: usage.value?.cacheReadTokens || 0, color: 'bg-amber-500' },
  { label: '缓存写入', value: usage.value?.cacheWriteTokens || 0, color: 'bg-rose-500' },
])
const maxTokenPart = computed(() => Math.max(1, ...tokenParts.value.map((part) => part.value)))
const modelColors = [
  '#6366f1',
  '#10b981',
  '#f59e0b',
  '#f43f5e',
  '#06b6d4',
  '#8b5cf6',
  '#84cc16',
  '#f97316',
]
const totalModelTokens = computed(() =>
  statistics.value.modelStatistics.reduce((total, item) => total + (item.totalTokens || 0), 0),
)
const modelRows = computed(() => {
  const rows = statistics.value.modelStatistics.map((item, index) => ({
    ...item,
    color: modelColors[index % modelColors.length],
    percent:
      totalModelTokens.value > 0 ? ((item.totalTokens || 0) / totalModelTokens.value) * 100 : 0,
  }))
  return rows.sort((left, right) => {
    if (modelSort.value === 'calls') {
      return right.callCount - left.callCount || right.totalTokens - left.totalTokens
    }
    if (modelSort.value === 'cost') {
      return right.cost - left.cost || right.totalTokens - left.totalTokens
    }
    if (modelSort.value === 'latency') {
      return right.avgLatencyMs - left.avgLatencyMs || right.totalTokens - left.totalTokens
    }
    return right.totalTokens - left.totalTokens || right.callCount - left.callCount
  })
})
type ModelUsageRow = (typeof modelRows.value)[number]
type ModelPieItem = ModelUsageRow & { chartPercent: number }
const tooltip = ref<{ item: ModelPieItem; x: number; y: number } | null>(null)
const tooltipStyle = computed(() =>
  tooltip.value ? { left: `${tooltip.value.x}px`, top: `${tooltip.value.y}px` } : {},
)
const visibleModelRows = computed(() => {
  const keyword = modelSearchQuery.value.trim().toLowerCase()
  if (!keyword) return modelRows.value
  return modelRows.value.filter((item) => item.model.toLowerCase().includes(keyword))
})
const visibleModelTokens = computed(() =>
  visibleModelRows.value.reduce((total, item) => total + (item.totalTokens || 0), 0),
)
const selectedVisibleModel = computed(() =>
  visibleModelRows.value.some((item) => item.model === selectedModel.value)
    ? selectedModel.value
    : visibleModelRows.value[0]?.model || '',
)
const activeModelItem = computed(
  () =>
    visibleModelRows.value.find((item) => item.model === selectedVisibleModel.value) ||
    modelRows.value[0],
)
const modelTokenParts = computed(() => {
  const item = activeModelItem.value
  if (!item) return []
  return [
    { label: '输入', value: item.promptTokens || 0, color: 'bg-blue-500' },
    { label: '输出', value: item.completionTokens || 0, color: 'bg-emerald-500' },
    { label: '缓存读取', value: item.cacheReadTokens || 0, color: 'bg-amber-500' },
    { label: '缓存写入', value: item.cacheWriteTokens || 0, color: 'bg-rose-500' },
  ]
})
const maxModelTokenPart = computed(() =>
  Math.max(1, ...modelTokenParts.value.map((part) => part.value)),
)
const modelPieSegments = computed(() => {
  const total = Math.max(1, visibleModelTokens.value)
  let startAngle = -Math.PI / 2
  return visibleModelRows.value.map((item) => {
    const sweep = ((item.totalTokens || 0) / total) * Math.PI * 2
    const endAngle = startAngle + sweep
    const largeArc = sweep > Math.PI ? 1 : 0
    const segment = {
      ...item,
      startAngle,
      endAngle,
      largeArc,
      path: createPieArcPath(startAngle, endAngle, largeArc),
      chartPercent:
        visibleModelTokens.value > 0
          ? ((item.totalTokens || 0) / visibleModelTokens.value) * 100
          : 0,
    }
    startAngle = endAngle
    return segment
  })
})

function toLocalDateTime(value: Date) {
  const offset = value.getTimezoneOffset() * 60_000
  return new Date(value.getTime() - offset).toISOString().slice(0, 16)
}

function initializeCustomRange() {
  const end = new Date()
  const start = new Date(end.getTime() - 7 * 24 * 60 * 60 * 1000)
  customStart.value = toLocalDateTime(start)
  customEnd.value = toLocalDateTime(end)
}

function selectedRange(): { startAt?: number; endAt?: number } {
  if (rangePreset.value === 'all') return {}
  if (rangePreset.value === 'custom') {
    const startAt = new Date(customStart.value).getTime()
    const endAt = new Date(customEnd.value).getTime()
    return Number.isFinite(startAt) && Number.isFinite(endAt) ? { startAt, endAt } : {}
  }
  const durations: Record<Exclude<RangePreset, 'all' | 'custom'>, number> = {
    '24h': 24 * 60 * 60 * 1000,
    '7d': 7 * 24 * 60 * 60 * 1000,
    '30d': 30 * 24 * 60 * 60 * 1000,
  }
  const endAt = Date.now()
  return { startAt: endAt - durations[rangePreset.value], endAt }
}

async function loadStatistics() {
  const sequence = ++loadSequence
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await aiGatewayApi.tokenStatistics({ page: 1, pageSize: 1, ...selectedRange() })
    if (sequence !== loadSequence) return
    const modelStatistics = result.modelStatistics || []
    statistics.value = {
      ...result,
      items: result.items || [],
      modelStatistics,
    }
    if (!modelStatistics.some((item) => item.model === selectedModel.value)) {
      selectedModel.value = modelStatistics[0]?.model || ''
    }
  } catch (error) {
    if (sequence !== loadSequence) return
    errorMessage.value = error instanceof Error ? error.message : '用量统计加载失败'
    toast.error(errorMessage.value)
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

async function loadLogs() {
  const sequence = ++logSequence
  logsLoading.value = true
  try {
    const page = await aiGatewayApi.logPage({
      page: currentPage.value,
      pageSize: pageSize.value,
      keyword: searchQuery.value.trim() || undefined,
      success: statusFilter.value === 'all' ? undefined : statusFilter.value === 'success' ? 1 : 2,
      ...selectedRange(),
    })
    if (sequence !== logSequence) return
    logs.value = page.records || []
    total.value = page.total || 0
    currentPage.value = page.page || currentPage.value
    pageSize.value = page.pageSize || pageSize.value
  } catch (error) {
    if (sequence !== logSequence) return
    toast.error(error instanceof Error ? error.message : '调用日志加载失败')
  } finally {
    if (sequence === logSequence) logsLoading.value = false
  }
}

function refreshAll() {
  void Promise.all([loadStatistics(), loadLogs()])
}

function setRange(value: RangePreset) {
  rangePreset.value = value
  currentPage.value = 1
  if (value === 'custom' && !customStart.value) initializeCustomRange()
  if (value !== 'custom') refreshAll()
}

function applyCustomRange() {
  const { startAt, endAt } = selectedRange()
  if (!startAt || !endAt || endAt < startAt) {
    toast.error('请选择有效的开始和结束时间')
    return
  }
  currentPage.value = 1
  refreshAll()
}

function changePage(page: number) {
  const totalPages = Math.max(1, Math.ceil(total.value / pageSize.value))
  if (page < 1 || page > totalPages || page === currentPage.value || logsLoading.value) return
  currentPage.value = page
  void loadLogs()
}

function changePageSize(size: number) {
  if (logsLoading.value || size === pageSize.value) return
  pageSize.value = size
  currentPage.value = 1
  void loadLogs()
}

function formatTokens(value?: number) {
  return integerFormatter.format(value || 0)
}

function formatCompact(value?: number) {
  return compactFormatter.format(value || 0)
}

function formatAmount(value?: number) {
  return `$${(value || 0).toFixed(6)}`
}

function formatTime(value?: number) {
  return value ? dateTimeFormatter.format(new Date(value)) : '-'
}

function barWidth(value: number) {
  return `${Math.max(value ? 2 : 0, (value / maxTokenPart.value) * 100)}%`
}

function modelSuccessRate(callCount: number, successCount: number) {
  return callCount ? (successCount / callCount) * 100 : 0
}

function modelBarWidth(value: number) {
  return `${Math.max(value ? 2 : 0, (value / maxModelTokenPart.value) * 100)}%`
}

function formatLatency(value?: number) {
  const latency = value || 0
  return latency >= 1000 ? `${(latency / 1000).toFixed(2)}s` : `${latency}ms`
}

function selectModel(model: string) {
  selectedModel.value = model
  tooltip.value = null
}

function clearPieTooltip() {
  tooltip.value = null
}

function polarPoint(center: number, radius: number, angle: number) {
  return {
    x: center + radius * Math.cos(angle),
    y: center + radius * Math.sin(angle),
  }
}

function createPieArcPath(startAngle: number, endAngle: number, largeArc: number) {
  const center = 120
  const outerRadius = 112
  const innerRadius = 66
  if (endAngle - startAngle >= Math.PI * 2 - 0.0001) {
    return [
      `M ${center} ${center - outerRadius}`,
      `A ${outerRadius} ${outerRadius} 0 1 1 ${center} ${center + outerRadius}`,
      `A ${outerRadius} ${outerRadius} 0 1 1 ${center} ${center - outerRadius}`,
      `M ${center} ${center - innerRadius}`,
      `A ${innerRadius} ${innerRadius} 0 1 0 ${center} ${center + innerRadius}`,
      `A ${innerRadius} ${innerRadius} 0 1 0 ${center} ${center - innerRadius}`,
      'Z',
    ].join(' ')
  }
  const startOuter = polarPoint(center, outerRadius, startAngle)
  const endOuter = polarPoint(center, outerRadius, endAngle)
  const startInner = polarPoint(center, innerRadius, endAngle)
  const endInner = polarPoint(center, innerRadius, startAngle)
  return [
    `M ${startOuter.x.toFixed(2)} ${startOuter.y.toFixed(2)}`,
    `A ${outerRadius} ${outerRadius} 0 ${largeArc} 1 ${endOuter.x.toFixed(2)} ${endOuter.y.toFixed(2)}`,
    `L ${startInner.x.toFixed(2)} ${startInner.y.toFixed(2)}`,
    `A ${innerRadius} ${innerRadius} 0 ${largeArc} 0 ${endInner.x.toFixed(2)} ${endInner.y.toFixed(2)}`,
    'Z',
  ].join(' ')
}

function clearModelSearch() {
  modelSearchQuery.value = ''
}

async function viewModelLogs(model: string) {
  searchQuery.value = model
  statusFilter.value = 'all'
  currentPage.value = 1
  await nextTick()
  document.querySelector('#call-logs')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function showPieTooltip(event: MouseEvent, item: ModelPieItem) {
  const current = event.currentTarget as SVGElement | null
  const container = current?.ownerSVGElement?.parentElement
  const rect = container?.getBoundingClientRect()
  if (!rect) return
  const tooltipWidth = 190
  const tooltipHeight = 130
  tooltip.value = {
    item,
    x: Math.min(
      Math.max(12, event.clientX - rect.left),
      Math.max(12, rect.width - tooltipWidth - 12),
    ),
    y: Math.min(
      Math.max(12, event.clientY - rect.top - 18),
      Math.max(12, rect.height - tooltipHeight - 12),
    ),
  }
}

function logout() {
  userStore.logout()
  void router.replace({ name: 'Login' })
}

watch([searchQuery, statusFilter], () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    void loadLogs()
  }, 300)
})

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
  loadSequence += 1
  logSequence += 1
})

refreshAll()
</script>

<template>
  <div class="min-h-screen bg-background text-foreground">
    <header class="sticky top-0 z-30 border-b bg-card">
      <div class="mx-auto flex min-h-16 max-w-[1440px] items-center gap-3 px-4 sm:px-6 lg:px-8">
        <div
          class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground"
        >
          <KeyRound class="h-5 w-5" />
        </div>
        <div class="min-w-0">
          <p class="truncate text-sm font-semibold">{{ userStore.displayName }}</p>
          <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Clock3 class="h-3 w-3" />
            <span class="truncate">授权至 {{ expiryText }}</span>
          </div>
        </div>
        <div class="ml-auto flex items-center gap-1">
          <ThemeSwitcher />
          <Button
            variant="ghost"
            size="icon"
            title="退出授权"
            aria-label="退出授权"
            @click="logout"
          >
            <LogOut class="h-5 w-5" />
          </Button>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-[1360px] px-4 py-6 sm:px-6 lg:px-8">
      <section class="border-b pb-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p class="text-sm font-medium text-primary">API 密钥用量</p>
            <h1 class="mt-1 text-2xl font-semibold">Token 消耗分析</h1>
          </div>
          <div class="flex flex-col gap-2 sm:items-end">
            <div
              class="flex max-w-full overflow-x-auto rounded-md border bg-muted/30 p-0.5"
              aria-label="统计时间范围"
            >
              <Button
                v-for="option in rangeOptions"
                :key="option.value"
                size="sm"
                :variant="rangePreset === option.value ? 'secondary' : 'ghost'"
                class="h-8 shrink-0 rounded-sm px-3"
                :disabled="loading"
                @click="setRange(option.value)"
              >
                {{ option.label }}
              </Button>
            </div>
            <div
              v-if="rangePreset === 'custom'"
              class="flex flex-col gap-2 sm:flex-row sm:items-center"
            >
              <Input v-model="customStart" type="datetime-local" class="h-9 sm:w-48" />
              <span class="hidden text-xs text-muted-foreground sm:inline">至</span>
              <Input v-model="customEnd" type="datetime-local" class="h-9 sm:w-48" />
              <Button size="sm" class="h-9" :disabled="loading" @click="applyCustomRange"
                >查询</Button
              >
            </div>
          </div>
        </div>
      </section>

      <div
        v-if="errorMessage"
        class="mt-5 flex items-center justify-between gap-4 rounded-md border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      >
        <span>{{ errorMessage }}</span>
        <Button variant="outline" size="sm" @click="loadStatistics">重试</Button>
      </div>

      <section class="py-5">
        <div
          v-if="loading && !statistics.items.length"
          class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
        >
          <Skeleton v-for="index in 4" :key="index" class="h-28 rounded-md" />
        </div>
        <div v-else class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <div class="rounded-md border bg-card p-4 shadow-sm">
            <div class="flex items-center justify-between text-sm text-muted-foreground">
              <span>总 Token</span>
              <span
                class="flex size-8 items-center justify-center rounded-md bg-blue-500/10 text-blue-600 dark:text-blue-400"
              >
                <ChartNoAxesCombined class="h-4 w-4" />
              </span>
            </div>
            <p class="mt-3 text-2xl font-semibold tabular-nums">
              {{ formatTokens(statistics.totalTokens) }}
            </p>
            <p class="mt-1 text-xs text-muted-foreground">所选时间范围</p>
          </div>
          <div class="rounded-md border bg-card p-4 shadow-sm">
            <div class="flex items-center justify-between text-sm text-muted-foreground">
              <span>调用次数</span>
              <span
                class="flex size-8 items-center justify-center rounded-md bg-cyan-500/10 text-cyan-600 dark:text-cyan-400"
              >
                <Activity class="h-4 w-4" />
              </span>
            </div>
            <p class="mt-3 text-2xl font-semibold tabular-nums">
              {{ formatTokens(statistics.callCount) }}
            </p>
            <p class="mt-1 text-xs text-muted-foreground">
              成功 {{ formatTokens(statistics.successCount) }} · 失败
              {{ formatTokens(statistics.failureCount) }}
            </p>
          </div>
          <div class="rounded-md border bg-card p-4 shadow-sm">
            <div class="flex items-center justify-between text-sm text-muted-foreground">
              <span>成功率</span>
              <span
                class="flex size-8 items-center justify-center rounded-md bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
              >
                <CheckCircle2 class="h-4 w-4" />
              </span>
            </div>
            <p class="mt-3 text-2xl font-semibold tabular-nums">{{ successRate.toFixed(1) }}%</p>
            <p class="mt-1 text-xs text-muted-foreground">按调用结果计算</p>
          </div>
          <div class="rounded-md border bg-card p-4 shadow-sm">
            <div class="flex items-center justify-between text-sm text-muted-foreground">
              <span>最近调用</span>
              <span
                class="flex size-8 items-center justify-center rounded-md bg-amber-500/10 text-amber-600 dark:text-amber-400"
              >
                <Clock3 class="h-4 w-4" />
              </span>
            </div>
            <p class="mt-3 text-base font-semibold">{{ formatTime(usage?.lastUsedAt) }}</p>
            <p class="mt-2 text-xs text-muted-foreground">
              推理 Token {{ formatTokens(usage?.reasoningTokens) }}
            </p>
          </div>
        </div>
      </section>

      <section class="border-y py-6">
        <div class="mb-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div class="flex items-center gap-3">
            <span
              class="flex size-9 items-center justify-center rounded-md border bg-card text-muted-foreground"
            >
              <ChartPie class="h-4 w-4" />
            </span>
            <div>
              <h2 class="text-base font-semibold">模型使用分析</h2>
              <p class="mt-1 text-xs text-muted-foreground">
                按 Token 占比展示，共 {{ statistics.modelStatistics.length }} 个模型
              </p>
            </div>
          </div>
          <div class="flex flex-col gap-2 sm:flex-row">
            <div class="relative sm:w-64">
              <Search
                class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
              />
              <Input v-model="modelSearchQuery" class="h-9 pl-9 pr-9" placeholder="筛选模型" />
              <button
                v-if="modelSearchQuery"
                type="button"
                class="absolute right-2 top-1/2 flex size-6 -translate-y-1/2 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground"
                title="清除筛选"
                aria-label="清除模型筛选"
                @click="clearModelSearch"
              >
                <X class="h-3.5 w-3.5" />
              </button>
            </div>
            <select
              v-model="modelSort"
              class="h-9 rounded-md border bg-background px-3 text-sm"
              aria-label="模型排序"
            >
              <option value="tokens">按 Token 排序</option>
              <option value="calls">按调用次数排序</option>
              <option value="cost">按成本排序</option>
              <option value="latency">按平均延迟排序</option>
            </select>
          </div>
        </div>

        <div
          v-if="!statistics.modelStatistics.length"
          class="flex min-h-40 items-center justify-center rounded-md border border-dashed text-sm text-muted-foreground"
        >
          所选时间范围暂无模型调用
        </div>
        <div
          v-else-if="!visibleModelRows.length"
          class="flex min-h-40 items-center justify-center rounded-md border border-dashed text-sm text-muted-foreground"
        >
          没有匹配的模型
        </div>
        <div
          v-else
          class="grid overflow-hidden rounded-md border bg-card shadow-sm xl:grid-cols-[minmax(0,1fr)_360px] xl:items-stretch"
        >
          <div
            class="grid gap-5 p-4 sm:p-5 lg:grid-cols-[minmax(260px,0.9fr)_minmax(260px,1.1fr)] lg:items-center xl:min-h-[420px] xl:grid-rows-1"
          >
            <div class="relative min-h-[300px] self-stretch">
              <div class="absolute inset-0 flex items-center justify-center">
                <div class="relative aspect-square w-full max-w-[288px]">
                  <svg
                    viewBox="0 0 240 240"
                    class="h-full w-full"
                    role="img"
                    aria-label="模型 Token 使用饼图"
                    @mouseleave="clearPieTooltip"
                  >
                    <circle
                      cx="120"
                      cy="120"
                      r="89"
                      fill="none"
                      stroke-width="46"
                      class="stroke-muted"
                    />
                    <path
                      v-for="segment in modelPieSegments"
                      :key="segment.model || 'unknown-model'"
                      :d="segment.path"
                      :fill="segment.color"
                      fill-rule="evenodd"
                      :stroke-width="selectedVisibleModel === segment.model ? 3 : 1"
                      :class="[
                        selectedVisibleModel === segment.model
                          ? 'stroke-border'
                          : 'stroke-transparent',
                        'cursor-pointer transition-opacity hover:opacity-80',
                      ]"
                      @mouseenter="showPieTooltip($event, segment)"
                      @click="selectModel(segment.model)"
                    />
                  </svg>
                  <div
                    class="pointer-events-none absolute inset-0 flex items-center justify-center"
                  >
                    <div class="text-center">
                      <p class="text-2xl font-semibold tabular-nums">
                        {{ formatCompact(visibleModelTokens) }}
                      </p>
                      <p class="mt-1 text-xs text-muted-foreground">
                        {{ modelSearchQuery ? '筛选结果' : 'Token' }}
                      </p>
                    </div>
                  </div>
                  <div
                    v-if="tooltip"
                    class="pointer-events-none absolute z-20 w-[190px] rounded-md border bg-popover p-3 text-xs shadow-lg"
                    :style="tooltipStyle"
                  >
                    <p class="truncate font-semibold" :title="tooltip.item.model || '未标识模型'">
                      {{ tooltip.item.model || '未标识模型' }}
                    </p>
                    <p class="mt-1 text-muted-foreground">
                      Token {{ formatCompact(tooltip.item.totalTokens) }} ·
                      {{ tooltip.item.chartPercent.toFixed(1) }}%
                    </p>
                    <div class="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-muted-foreground">
                      <span>调用 {{ formatTokens(tooltip.item.callCount) }}</span>
                      <span
                        >成功率
                        {{
                          modelSuccessRate(
                            tooltip.item.callCount,
                            tooltip.item.successCount,
                          ).toFixed(1)
                        }}%</span
                      >
                      <span>成本 {{ formatAmount(tooltip.item.cost) }}</span>
                      <span>延迟 {{ formatLatency(tooltip.item.avgLatencyMs) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="self-center">
              <div
                class="mb-2 flex items-center justify-between px-2 text-xs text-muted-foreground"
              >
                <span>模型</span>
                <span>Token 占比</span>
              </div>
              <div class="max-h-[300px] space-y-1 overflow-y-auto pr-1">
                <button
                  v-for="(segment, index) in modelPieSegments"
                  :key="segment.model || 'unknown-model'"
                  type="button"
                  class="grid w-full grid-cols-[minmax(0,1fr)_auto] gap-2 rounded-md px-2 py-2 text-left transition-colors"
                  :class="
                    selectedVisibleModel === segment.model ? 'bg-primary/5' : 'hover:bg-muted/40'
                  "
                  :aria-pressed="selectedModel === segment.model"
                  @click="selectModel(segment.model)"
                >
                  <span class="flex min-w-0 items-center gap-2">
                    <span
                      class="w-4 shrink-0 text-right text-[11px] tabular-nums text-muted-foreground"
                      >{{ index + 1 }}</span
                    >
                    <span
                      class="size-2.5 shrink-0 rounded-sm"
                      :style="{ backgroundColor: segment.color }"
                    ></span>
                    <span
                      class="truncate text-sm font-medium"
                      :title="segment.model || '未标识模型'"
                      >{{ segment.model || '未标识模型' }}</span
                    >
                  </span>
                  <span class="text-right text-xs text-muted-foreground"
                    >{{ segment.chartPercent.toFixed(1) }}%</span
                  >
                  <span
                    class="col-span-2 mt-1 flex items-center justify-between gap-3 pl-10 text-xs text-muted-foreground"
                  >
                    <span
                      >{{ formatCompact(segment.totalTokens) }} Token ·
                      {{ formatTokens(segment.callCount) }} 次</span
                    >
                    <span>{{ formatAmount(segment.cost) }}</span>
                  </span>
                </button>
              </div>
            </div>
          </div>

          <aside class="border-t bg-muted/10 p-4 xl:border-l xl:border-t-0">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div
                  class="mb-1.5 flex items-center gap-2 text-xs font-medium text-muted-foreground"
                >
                  <Layers3 class="h-3.5 w-3.5" /> 模型详情
                </div>
                <h3
                  class="truncate text-base font-semibold"
                  :title="activeModelItem?.model || '未标识模型'"
                >
                  {{ activeModelItem?.model || '未标识模型' }}
                </h3>
                <p class="mt-1 text-xs text-muted-foreground">
                  Token 占比 {{ (activeModelItem?.percent || 0).toFixed(1) }}% · 最近
                  {{ formatTime(activeModelItem?.lastUsedAt) }}
                </p>
              </div>
            </div>

            <div class="mt-3 grid grid-cols-2 border-y text-sm">
              <div class="border-b border-r p-2.5">
                <p class="text-xs text-muted-foreground">总 Token</p>
                <p class="mt-0.5 text-base font-semibold tabular-nums">
                  {{ formatTokens(activeModelItem?.totalTokens) }}
                </p>
              </div>
              <div class="border-b p-2.5">
                <p class="text-xs text-muted-foreground">调用 / 失败</p>
                <p class="mt-0.5 text-base font-semibold tabular-nums">
                  {{ formatTokens(activeModelItem?.callCount)
                  }}<span class="ml-1 text-xs font-normal text-muted-foreground"
                    >/ {{ formatTokens(activeModelItem?.failureCount) }}</span
                  >
                </p>
              </div>
              <div class="border-b border-r p-2.5">
                <p class="text-xs text-muted-foreground">成功率</p>
                <p class="mt-0.5 text-base font-semibold tabular-nums">
                  {{
                    modelSuccessRate(
                      activeModelItem?.callCount || 0,
                      activeModelItem?.successCount || 0,
                    ).toFixed(1)
                  }}%
                </p>
              </div>
              <div class="border-b p-2.5">
                <p class="text-xs text-muted-foreground">成本</p>
                <p class="mt-0.5 text-base font-semibold tabular-nums">
                  {{ formatAmount(activeModelItem?.cost) }}
                </p>
              </div>
              <div class="border-r p-2.5">
                <p class="text-xs text-muted-foreground">平均延迟</p>
                <p class="mt-0.5 text-base font-semibold tabular-nums">
                  {{ formatLatency(activeModelItem?.avgLatencyMs) }}
                </p>
              </div>
              <div class="p-2.5">
                <p class="text-xs text-muted-foreground">推理 Token</p>
                <p class="mt-0.5 text-base font-semibold tabular-nums">
                  {{ formatTokens(activeModelItem?.reasoningTokens) }}
                </p>
              </div>
            </div>

            <div class="mt-4">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h4 class="text-sm font-semibold">模型 Token 构成</h4>
                  <p class="mt-1 text-xs text-muted-foreground">推理 Token 已包含在输出 Token 中</p>
                </div>
                <span class="rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground">{{
                  formatCompact(activeModelItem?.totalTokens)
                }}</span>
              </div>
              <div class="mt-2.5 space-y-2.5">
                <div
                  v-for="part in modelTokenParts"
                  :key="part.label"
                  class="grid grid-cols-[68px_minmax(0,1fr)_auto] items-center gap-2 text-xs"
                >
                  <span class="text-muted-foreground">{{ part.label }}</span>
                  <div class="h-1.5 overflow-hidden rounded-sm bg-muted">
                    <div
                      class="h-full rounded-sm"
                      :class="part.color"
                      :style="{ width: modelBarWidth(part.value) }"
                    ></div>
                  </div>
                  <span class="min-w-14 text-right font-medium tabular-nums">{{
                    formatTokens(part.value)
                  }}</span>
                </div>
              </div>
            </div>

            <Button
              variant="outline"
              class="mt-4 h-9 w-full"
              :disabled="!activeModelItem?.model"
              @click="viewModelLogs(activeModelItem?.model || '')"
            >
              <Eye class="mr-2 h-4 w-4" />
              查看该模型调用明细
            </Button>
          </aside>
        </div>
      </section>

      <section class="py-5">
        <div class="mb-4 flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold">密钥 Token 构成</h2>
            <p class="mt-1 text-xs text-muted-foreground">推理 Token 已包含在输出 Token 中</p>
          </div>
          <Button
            variant="outline"
            size="icon"
            :disabled="loading || logsLoading"
            title="刷新数据"
            aria-label="刷新数据"
            @click="refreshAll"
          >
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading || logsLoading }" />
          </Button>
        </div>
        <div class="rounded-md border bg-card p-4">
          <div class="space-y-4">
            <div
              v-for="part in tokenParts"
              :key="part.label"
              class="grid grid-cols-[76px_minmax(0,1fr)_auto] items-center gap-3 text-sm"
            >
              <span class="text-muted-foreground">{{ part.label }}</span>
              <div class="h-2 overflow-hidden rounded-sm bg-muted">
                <div
                  class="h-full rounded-sm"
                  :class="part.color"
                  :style="{ width: barWidth(part.value) }"
                ></div>
              </div>
              <span class="min-w-16 text-right font-medium tabular-nums">{{
                formatTokens(part.value)
              }}</span>
            </div>
          </div>
        </div>
      </section>

      <section id="call-logs" class="scroll-mt-20 pt-6">
        <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div>
            <h2 class="text-lg font-semibold">调用日志</h2>
            <p class="mt-1 text-sm text-muted-foreground">{{ total }} 条记录</p>
          </div>
          <div class="flex flex-col gap-2 sm:flex-row">
            <div class="relative sm:w-80">
              <Search
                class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
              />
              <Input
                v-model="searchQuery"
                class="h-9 pl-9"
                placeholder="搜索请求、模型、路径或错误"
              />
            </div>
            <select
              v-model="statusFilter"
              class="h-9 rounded-md border bg-background px-3 text-sm"
              :disabled="logsLoading"
              aria-label="调用状态"
            >
              <option value="all">全部状态</option>
              <option value="success">成功</option>
              <option value="failed">失败</option>
            </select>
          </div>
        </div>

        <div class="mt-4 overflow-hidden rounded-md border bg-card">
          <div class="hidden overflow-x-auto md:block">
            <table class="w-full min-w-[920px] text-sm">
              <thead class="border-b bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th class="px-4 py-3">时间</th>
                  <th class="px-4 py-3">模型 / 请求</th>
                  <th class="px-4 py-3">状态</th>
                  <th class="px-4 py-3">Token</th>
                  <th class="px-4 py-3">成本</th>
                  <th class="px-4 py-3">延迟</th>
                  <th class="px-4 py-3">错误</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="!logsLoading && !logs.length">
                  <td colspan="7" class="px-4 py-12 text-center text-muted-foreground">
                    暂无调用日志
                  </td>
                </tr>
                <tr
                  v-for="item in logs"
                  :key="item.id"
                  class="border-b last:border-0 hover:bg-muted/20"
                >
                  <td class="whitespace-nowrap px-4 py-3">{{ formatTime(item.createdAt) }}</td>
                  <td class="px-4 py-3">
                    <p class="font-medium">{{ item.model || '-' }}</p>
                    <p
                      class="mt-1 max-w-56 truncate text-xs text-muted-foreground"
                      :title="item.requestId"
                    >
                      {{ item.requestId }}
                    </p>
                  </td>
                  <td class="px-4 py-3">
                    <Badge :variant="item.success === 1 ? 'default' : 'destructive'">{{
                      item.statusCode
                    }}</Badge>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3">
                    <p class="font-medium tabular-nums">{{ formatTokens(item.totalTokens) }}</p>
                    <p class="mt-1 text-xs text-muted-foreground">
                      入 {{ formatTokens(item.promptTokens) }} / 出
                      {{ formatTokens(item.completionTokens) }}
                    </p>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3">{{ formatAmount(item.cost) }}</td>
                  <td class="whitespace-nowrap px-4 py-3">{{ item.latencyMs }} ms</td>
                  <td class="max-w-64 px-4 py-3">
                    <span v-if="item.errorMessage" class="flex items-start gap-1.5 text-destructive"
                      ><ShieldAlert class="mt-0.5 h-3.5 w-3.5 shrink-0" /><span
                        class="truncate"
                        :title="item.errorMessage"
                        >{{ item.errorMessage }}</span
                      ></span
                    ><span v-else class="text-muted-foreground">-</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="divide-y md:hidden">
            <div
              v-if="!logsLoading && !logs.length"
              class="px-4 py-12 text-center text-sm text-muted-foreground"
            >
              暂无调用日志
            </div>
            <article v-for="item in logs" :key="item.id" class="space-y-3 p-4">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <p class="truncate font-medium">{{ item.model || '-' }}</p>
                  <p class="mt-1 truncate text-xs text-muted-foreground">{{ item.requestId }}</p>
                </div>
                <Badge class="shrink-0" :variant="item.success === 1 ? 'default' : 'destructive'">{{
                  item.statusCode
                }}</Badge>
              </div>
              <div class="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <p class="text-xs text-muted-foreground">时间</p>
                  <p class="mt-1">{{ formatTime(item.createdAt) }}</p>
                </div>
                <div>
                  <p class="text-xs text-muted-foreground">Token</p>
                  <p class="mt-1 font-medium">{{ formatTokens(item.totalTokens) }}</p>
                </div>
                <div>
                  <p class="text-xs text-muted-foreground">成本</p>
                  <p class="mt-1">{{ formatAmount(item.cost) }}</p>
                </div>
                <div>
                  <p class="text-xs text-muted-foreground">延迟</p>
                  <p class="mt-1">{{ item.latencyMs }} ms</p>
                </div>
              </div>
              <p v-if="item.errorMessage" class="flex items-start gap-1.5 text-sm text-destructive">
                <ShieldAlert class="mt-0.5 h-3.5 w-3.5 shrink-0" /><span class="break-words">{{
                  item.errorMessage
                }}</span>
              </p>
            </article>
          </div>
          <AppPagination
            :total="total"
            :page="currentPage"
            :page-size="pageSize"
            :loading="logsLoading"
            @change-page="changePage"
            @change-page-size="changePageSize"
          />
        </div>
      </section>
    </main>
  </div>
</template>
