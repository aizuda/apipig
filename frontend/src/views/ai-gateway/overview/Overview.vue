<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import {
  Activity,
  CalendarDays,
  CircleDollarSign,
  DatabaseZap,
  Gauge,
  KeyRound,
  Network,
  ServerCog,
} from '@lucide/vue'
import { Card, CardContent, Progress, Skeleton, toast } from '@tabtab/ui'
import { aiGatewayApi, type GatewaySummary, type GatewayTokenTrend } from '@/api/ai-gateway'
import AppPageHeader from '@/components/AppPageHeader.vue'

defineOptions({ name: 'AiGatewayOverview' })

type TrendMetric = 'promptTokens' | 'completionTokens' | 'cacheTokens' | 'totalTokens'

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
const loading = ref(false)
const hasLoaded = ref(false)
const errorMessage = ref('')
const rangePreset = ref<'7' | '30' | 'custom'>('7')
const endDate = ref(formatDateInput(new Date()))
const startDate = ref(formatDateInput(addDays(new Date(), -6)))
let loadSequence = 0
const summary = ref<GatewaySummary>({
  providerCount: 0,
  channelCount: 0,
  tokenCount: 0,
  proxyCount: 0,
  callCount: 0,
  successCount: 0,
  errorCount: 0,
  totalTokens: 0,
  promptTokens: 0,
  completionTokens: 0,
  reasoningTokens: 0,
  cacheReadTokens: 0,
  cacheWriteTokens: 0,
  standardCost: 0,
  totalCost: 0,
  modelDistribution: [],
  tokenTrend: [],
  channelStatistics: [],
})

const successRateValue = computed(() =>
  summary.value.callCount > 0 ? (summary.value.successCount / summary.value.callCount) * 100 : 0,
)
const cacheHitRate = computed(() => {
  const totalInput = summary.value.promptTokens + summary.value.cacheReadTokens
  return totalInput > 0 ? (summary.value.cacheReadTokens / totalInput) * 100 : 0
})
const averageCost = computed(() =>
  summary.value.successCount > 0 ? summary.value.totalCost / summary.value.successCount : 0,
)
const costDelta = computed(() => summary.value.totalCost - summary.value.standardCost)
const totalModelTokens = computed(() =>
  summary.value.modelDistribution.reduce((total, item) => total + item.tokenCount, 0),
)
const modelRows = computed(() =>
  summary.value.modelDistribution.map((item, index) => ({
    ...item,
    color: modelColors[index % modelColors.length],
    percent: totalModelTokens.value > 0 ? (item.tokenCount / totalModelTokens.value) * 100 : 0,
  })),
)
const donutBackground = computed(() => {
  if (!modelRows.value.length || totalModelTokens.value <= 0)
    return 'conic-gradient(hsl(var(--muted)) 0 100%)'
  let offset = 0
  const segments = modelRows.value.map((item) => {
    const start = offset
    offset += item.percent
    return `${item.color} ${start}% ${offset}%`
  })
  return `conic-gradient(${segments.join(', ')})`
})
const trendMax = computed(() =>
  Math.max(1, ...summary.value.tokenTrend.map((item) => item.totalTokens)),
)
const latestTrend = computed(() => summary.value.tokenTrend.at(-1))
const averageDailyTokens = computed(() =>
  summary.value.tokenTrend.length > 0
    ? summary.value.tokenTrend.reduce((total, item) => total + item.totalTokens, 0) /
      summary.value.tokenTrend.length
    : 0,
)
const rangeLabel = computed(() => `${startDate.value} 至 ${endDate.value}`)
const maxChannelTokens = computed(() =>
  Math.max(1, ...summary.value.channelStatistics.map((item) => item.tokenCount)),
)
const trendAxisItems = computed(() => {
  const data = summary.value.tokenTrend
  if (data.length <= 7) return data
  const indexes = new Set([0, data.length - 1])
  for (let index = 1; index < 6; index += 1) {
    indexes.add(Math.round((index / 6) * (data.length - 1)))
  }
  return [...indexes]
    .sort((left, right) => left - right)
    .map((index) => data[index])
    .filter((item): item is GatewayTokenTrend => Boolean(item))
})

async function loadData() {
  const sequence = ++loadSequence
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await aiGatewayApi.summary(summaryRangeParams())
    if (sequence === loadSequence) {
      summary.value = {
        ...result,
        modelDistribution: result.modelDistribution || [],
        tokenTrend: result.tokenTrend || [],
        channelStatistics: result.channelStatistics || [],
      }
      hasLoaded.value = true
    }
  } catch (error) {
    if (sequence !== loadSequence) return
    errorMessage.value = error instanceof Error ? error.message : '概览数据加载失败'
    toast.error(errorMessage.value)
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

function addDays(value: Date, days: number) {
  const result = new Date(value)
  result.setDate(result.getDate() + days)
  return result
}

function formatDateInput(value: Date) {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, '0')
  const day = String(value.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function parseDateInput(value: string, endOfDay = false) {
  const date = new Date(`${value}T${endOfDay ? '23:59:59.999' : '00:00:00.000'}`)
  return date.getTime()
}

function summaryRangeParams() {
  if (!startDate.value || !endDate.value) throw new Error('请选择完整的开始和结束日期')
  const startAt = parseDateInput(startDate.value)
  const endAt = parseDateInput(endDate.value, true)
  if (startAt > endAt) throw new Error('开始日期不能晚于结束日期')
  if ((endAt - startAt) / 86_400_000 + 1 > 90) throw new Error('时间范围不能超过 90 天')
  return { startAt, endAt }
}

function setRangeDays(days: 7 | 30) {
  const end = new Date()
  rangePreset.value = String(days) as '7' | '30'
  endDate.value = formatDateInput(end)
  startDate.value = formatDateInput(addDays(end, -(days - 1)))
  void loadData()
}

function applyCustomRange() {
  rangePreset.value = 'custom'
  void loadData()
}

function formatAmount(value?: number) {
  return '$' + (value || 0).toFixed(6)
}

function formatTokens(value?: number) {
  return (value || 0).toLocaleString()
}

function formatCompact(value?: number) {
  return new Intl.NumberFormat('zh-CN', { notation: 'compact', maximumFractionDigits: 1 }).format(
    value || 0,
  )
}

function formatDay(value: string) {
  return value.slice(5).replace('-', '/')
}

function trendPoints(metric: TrendMetric) {
  const data = summary.value.tokenTrend
  if (!data.length) return ''
  const width = 676
  const left = 12
  const top = 18
  const height = 154
  return data
    .map((item, index) => {
      const x = left + (data.length === 1 ? width / 2 : (index / (data.length - 1)) * width)
      const y = top + height - (item[metric] / trendMax.value) * height
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')
}

function trendAreaPoints() {
  const points = trendPoints('totalTokens')
  return points ? `12,178 ${points} 688,178` : ''
}

function trendLabel(item?: GatewayTokenTrend) {
  if (!item) return '暂无调用'
  return `输入 ${formatCompact(item.promptTokens)} · 输出 ${formatCompact(item.completionTokens)} · 缓存 ${formatCompact(item.cacheTokens)}`
}

function channelTokenPercent(value: number) {
  return Math.max(0, Math.min(100, (value / maxChannelTokens.value) * 100))
}

function formatLatency(value?: number) {
  const latency = value || 0
  return latency >= 1000 ? `${(latency / 1000).toFixed(2)}s` : `${latency}ms`
}

onBeforeUnmount(() => {
  loadSequence += 1
})

void loadData()
</script>

<template>
  <div class="-mt-4 space-y-2 [&_[data-slot=card]]:gap-0 [&_[data-slot=card]]:py-0">
    <AppPageHeader
      title="数据概览"
      description="紧凑展示网关资源、模型调用分布、Token 趋势与成本效率。"
      :loading="loading"
      @refresh="loadData"
    />

    <div
      v-if="errorMessage"
      class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
    >
      {{ errorMessage }}
    </div>

    <div
      v-if="loading && !hasLoaded"
      class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6"
    >
      <Skeleton v-for="index in 6" :key="index" class="h-20 rounded-xl" />
      <Skeleton class="h-64 rounded-xl lg:col-span-2" />
      <Skeleton class="h-64 rounded-xl lg:col-span-3 xl:col-span-4" />
    </div>

    <template v-if="hasLoaded">
      <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        <Card>
          <CardContent class="flex items-center justify-between px-3 py-2.5">
            <div>
              <p class="text-xs text-muted-foreground">供应商</p>
              <p class="mt-1 text-xl font-semibold">{{ summary.providerCount }}</p>
            </div>
            <ServerCog class="size-5 text-indigo-500" />
          </CardContent>
        </Card>
        <Card>
          <CardContent class="flex items-center justify-between px-3 py-2.5">
            <div>
              <p class="text-xs text-muted-foreground">渠道号池</p>
              <p class="mt-1 text-xl font-semibold">{{ summary.channelCount }}</p>
            </div>
            <Network class="size-5 text-cyan-500" />
          </CardContent>
        </Card>
        <Card>
          <CardContent class="flex items-center justify-between px-3 py-2.5">
            <div>
              <p class="text-xs text-muted-foreground">API 密钥</p>
              <p class="mt-1 text-xl font-semibold">{{ summary.tokenCount }}</p>
            </div>
            <KeyRound class="size-5 text-amber-500" />
          </CardContent>
        </Card>
        <Card>
          <CardContent class="flex items-center justify-between px-3 py-2.5">
            <div>
              <p class="text-xs text-muted-foreground">调用 / 成功率</p>
              <p class="mt-1 text-xl font-semibold">
                {{ formatCompact(summary.callCount) }}
                <span class="text-sm text-emerald-600">{{ successRateValue.toFixed(1) }}%</span>
              </p>
            </div>
            <Activity class="size-5 text-emerald-500" />
          </CardContent>
        </Card>
        <Card>
          <CardContent class="flex items-center justify-between px-3 py-2.5">
            <div>
              <p class="text-xs text-muted-foreground">Token 消耗</p>
              <p class="mt-1 text-xl font-semibold">{{ formatCompact(summary.totalTokens) }}</p>
              <p class="text-[11px] text-muted-foreground">
                {{ formatTokens(summary.totalTokens) }}
              </p>
            </div>
            <Gauge class="size-5 text-violet-500" />
          </CardContent>
        </Card>
        <Card>
          <CardContent class="flex items-center justify-between px-3 py-2.5">
            <div>
              <p class="text-xs text-muted-foreground">总成本</p>
              <p class="mt-1 text-xl font-semibold">{{ formatAmount(summary.totalCost) }}</p>
              <p class="text-[11px] text-muted-foreground">
                标准 {{ formatAmount(summary.standardCost) }}
              </p>
            </div>
            <CircleDollarSign class="size-5 text-rose-500" />
          </CardContent>
        </Card>
      </div>

      <div
        class="flex flex-wrap items-center justify-between gap-2 rounded-lg border bg-card px-3 py-2"
      >
        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <CalendarDays class="size-4 text-primary" />
          <span>图表时间范围</span>
          <span class="font-medium text-foreground">{{ rangeLabel }}</span>
        </div>
        <div class="flex flex-wrap items-center gap-1.5">
          <button
            v-for="days in [7, 30] as const"
            :key="days"
            type="button"
            class="h-7 rounded-md border px-2.5 text-xs transition-colors hover:bg-muted"
            :class="rangePreset === String(days) ? 'border-primary bg-primary/10 text-primary' : ''"
            :disabled="loading"
            @click="setRangeDays(days)"
          >
            近 {{ days }} 天
          </button>
          <input
            v-model="startDate"
            type="date"
            class="h-7 rounded-md border bg-background px-2 text-xs"
            :max="endDate"
            @change="rangePreset = 'custom'"
          />
          <span class="text-xs text-muted-foreground">至</span>
          <input
            v-model="endDate"
            type="date"
            class="h-7 rounded-md border bg-background px-2 text-xs"
            :min="startDate"
            :max="formatDateInput(new Date())"
            @change="rangePreset = 'custom'"
          />
          <button
            type="button"
            class="h-7 rounded-md bg-primary px-3 text-xs text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            :disabled="loading"
            @click="applyCustomRange"
          >
            查询
          </button>
        </div>
      </div>

      <div class="grid gap-2 xl:grid-cols-5">
        <Card class="xl:col-span-2">
          <CardContent class="p-3">
            <div class="mb-2 flex items-start justify-between gap-2">
              <div>
                <h3 class="text-sm font-semibold">模型分布</h3>
                <p class="text-xs text-muted-foreground">所选时间范围内按成功调用 Token 统计</p>
              </div>
              <span class="rounded-md bg-muted px-2 py-1 text-xs text-muted-foreground"
                >{{ modelRows.length }} 个模型</span
              >
            </div>
            <div v-if="modelRows.length" class="grid items-center gap-3 sm:grid-cols-[120px_1fr]">
              <div
                class="relative mx-auto size-28 rounded-full"
                :style="{ background: donutBackground }"
              >
                <div
                  class="absolute inset-[18px] flex flex-col items-center justify-center rounded-full bg-card shadow-inner"
                >
                  <span class="text-xl font-semibold">{{ formatCompact(totalModelTokens) }}</span>
                  <span class="text-[11px] text-muted-foreground">Token</span>
                </div>
              </div>
              <div class="max-h-40 space-y-1 overflow-auto pr-1">
                <div
                  v-for="item in modelRows"
                  :key="item.model"
                  class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 rounded-md px-2 py-1 hover:bg-muted/50"
                >
                  <div class="min-w-0">
                    <div class="flex items-center gap-2">
                      <span
                        class="size-2 shrink-0 rounded-full"
                        :style="{ backgroundColor: item.color }"
                      ></span
                      ><span class="truncate text-xs font-medium" :title="item.model">{{
                        item.model
                      }}</span>
                    </div>
                    <div class="ml-4 text-[11px] text-muted-foreground">
                      {{ item.callCount }} 次 · {{ formatAmount(item.cost) }}
                    </div>
                  </div>
                  <div class="text-right">
                    <div class="text-xs font-medium">{{ item.percent.toFixed(1) }}%</div>
                    <div class="text-[11px] text-muted-foreground">
                      {{ formatCompact(item.tokenCount) }}
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div
              v-else
              class="flex h-40 items-center justify-center rounded-lg border border-dashed text-sm text-muted-foreground"
            >
              暂无模型调用数据
            </div>
          </CardContent>
        </Card>

        <Card class="xl:col-span-3">
          <CardContent class="p-3">
            <div class="mb-1 flex flex-wrap items-start justify-between gap-2">
              <div>
                <h3 class="text-sm font-semibold">Token 使用趋势</h3>
                <p class="text-xs text-muted-foreground">所选时间范围内成功调用用量</p>
              </div>
              <div class="text-right">
                <p class="text-lg font-semibold">
                  最新日 {{ formatCompact(latestTrend?.totalTokens) }}
                </p>
                <p class="text-[11px] text-muted-foreground">{{ trendLabel(latestTrend) }}</p>
              </div>
            </div>
            <div class="flex items-center gap-4 text-[11px] text-muted-foreground">
              <span class="flex items-center gap-1"
                ><i class="h-0.5 w-3 bg-indigo-500"></i>输入</span
              >
              <span class="flex items-center gap-1"
                ><i class="h-0.5 w-3 bg-emerald-500"></i>输出</span
              >
              <span class="flex items-center gap-1"
                ><i class="h-0.5 w-3 bg-amber-500"></i>缓存</span
              >
              <span class="ml-auto">日均 {{ formatCompact(averageDailyTokens) }}</span>
            </div>
            <div class="mt-1.5 h-36 w-full overflow-hidden rounded-lg bg-muted/20">
              <svg
                viewBox="0 0 700 190"
                class="h-full w-full"
                preserveAspectRatio="none"
                role="img"
                aria-label="最近七天 Token 使用趋势"
              >
                <line
                  v-for="y in [30, 78, 126, 174]"
                  :key="y"
                  x1="12"
                  x2="688"
                  :y1="y"
                  :y2="y"
                  class="stroke-border"
                  stroke-dasharray="4 5"
                />
                <polygon :points="trendAreaPoints()" fill="#6366f1" fill-opacity="0.08" />
                <polyline
                  :points="trendPoints('promptTokens')"
                  fill="none"
                  stroke="#6366f1"
                  stroke-width="3"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <polyline
                  :points="trendPoints('completionTokens')"
                  fill="none"
                  stroke="#10b981"
                  stroke-width="3"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <polyline
                  :points="trendPoints('cacheTokens')"
                  fill="none"
                  stroke="#f59e0b"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </div>
            <div
              class="mt-0.5 grid gap-1"
              :style="{
                gridTemplateColumns: `repeat(${trendAxisItems.length || 1}, minmax(0, 1fr))`,
              }"
            >
              <div
                v-for="item in trendAxisItems"
                :key="item.date"
                class="min-w-0 rounded px-1 py-0.5 text-center hover:bg-muted/60"
                :title="`${item.date}：${formatTokens(item.totalTokens)} Token`"
              >
                <div class="text-[10px] text-muted-foreground">{{ formatDay(item.date) }}</div>
                <div class="truncate text-[11px] font-medium">
                  {{ formatCompact(item.totalTokens) }}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="grid gap-2 lg:grid-cols-3">
        <Card
          ><CardContent class="px-3 py-2.5"
            ><div class="flex items-center justify-between">
              <div>
                <p class="text-xs text-muted-foreground">缓存命中率</p>
                <p class="mt-0.5 text-lg font-semibold">{{ cacheHitRate.toFixed(1) }}%</p>
              </div>
              <DatabaseZap class="size-5 text-primary" />
            </div>
            <Progress class="mt-2 h-1.5" :model-value="cacheHitRate" />
            <p class="mt-1.5 text-[11px] text-muted-foreground">
              缓存读 {{ formatTokens(summary.cacheReadTokens) }} · 普通输入
              {{ formatTokens(summary.promptTokens) }}
            </p></CardContent
          ></Card
        >
        <Card
          ><CardContent class="flex items-center justify-between gap-3 px-3 py-2.5"
            ><div>
              <p class="text-xs text-muted-foreground">平均成功调用成本</p>
              <p class="mt-0.5 text-lg font-semibold">{{ formatAmount(averageCost) }}</p>
              <p class="mt-1 text-[11px] text-muted-foreground">
                倍率影响 {{ costDelta >= 0 ? '+' : '' }}{{ formatAmount(costDelta) }}
              </p>
            </div>
            <CircleDollarSign class="size-6 text-emerald-600" /></CardContent
        ></Card>
        <Card
          ><CardContent class="flex items-center justify-between gap-3 px-3 py-2.5"
            ><div>
              <p class="text-xs text-muted-foreground">Token 构成</p>
              <p class="mt-0.5 text-lg font-semibold">
                输出 {{ formatCompact(summary.completionTokens) }}
              </p>
              <p class="mt-1 text-[11px] text-muted-foreground">
                推理 {{ formatTokens(summary.reasoningTokens) }} · 缓存写
                {{ formatTokens(summary.cacheWriteTokens) }}
              </p>
            </div>
            <Gauge class="size-6 text-amber-600" /></CardContent
        ></Card>
      </div>

      <Card>
        <CardContent class="p-3">
          <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 class="text-sm font-semibold">渠道统计分析</h3>
              <p class="text-xs text-muted-foreground">
                所选时间范围内按成功 Token 排名，成功率按全部调用计算
              </p>
            </div>
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <Network class="size-4 text-primary" />
              <span>{{ summary.channelStatistics.length }} 个活跃渠道</span>
            </div>
          </div>

          <div v-if="summary.channelStatistics.length" class="overflow-x-auto rounded-lg border">
            <table class="w-full min-w-[760px] text-left text-xs">
              <thead class="bg-muted/50 text-muted-foreground">
                <tr>
                  <th class="w-12 px-3 py-2 font-medium">排名</th>
                  <th class="px-3 py-2 font-medium">渠道</th>
                  <th class="min-w-48 px-3 py-2 font-medium">Token 使用</th>
                  <th class="px-3 py-2 text-right font-medium">调用 / 成功率</th>
                  <th class="px-3 py-2 text-right font-medium">成本</th>
                  <th class="px-3 py-2 text-right font-medium">平均延迟</th>
                </tr>
              </thead>
              <tbody class="divide-y">
                <tr
                  v-for="(item, index) in summary.channelStatistics"
                  :key="item.channelId"
                  class="transition-colors hover:bg-muted/30"
                >
                  <td class="px-3 py-2 text-muted-foreground">{{ index + 1 }}</td>
                  <td class="px-3 py-2">
                    <div class="max-w-52 truncate font-medium" :title="item.channelName">
                      {{ item.channelName }}
                    </div>
                    <div class="mt-0.5 text-[10px] text-muted-foreground">
                      成功 {{ item.successCount }} 次
                    </div>
                  </td>
                  <td class="px-3 py-2">
                    <div class="flex items-center justify-between gap-3">
                      <span class="font-medium">{{ formatTokens(item.tokenCount) }}</span>
                      <span class="text-[10px] text-muted-foreground">
                        {{ channelTokenPercent(item.tokenCount).toFixed(1) }}%
                      </span>
                    </div>
                    <div class="mt-1 h-1.5 overflow-hidden rounded-full bg-muted">
                      <div
                        class="h-full rounded-full bg-primary transition-all"
                        :style="{ width: `${channelTokenPercent(item.tokenCount)}%` }"
                      ></div>
                    </div>
                  </td>
                  <td class="px-3 py-2 text-right">
                    <div class="font-medium">{{ item.callCount }} 次</div>
                    <div
                      class="mt-0.5 text-[10px]"
                      :class="
                        item.successRate >= 95
                          ? 'text-emerald-600'
                          : item.successRate >= 80
                            ? 'text-amber-600'
                            : 'text-rose-600'
                      "
                    >
                      {{ item.successRate.toFixed(1) }}%
                    </div>
                  </td>
                  <td class="px-3 py-2 text-right font-medium">{{ formatAmount(item.cost) }}</td>
                  <td class="px-3 py-2 text-right text-muted-foreground">
                    {{ formatLatency(item.avgLatencyMs) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div
            v-else
            class="flex h-24 items-center justify-center rounded-lg border border-dashed text-sm text-muted-foreground"
          >
            当前时间范围内暂无渠道调用数据
          </div>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
