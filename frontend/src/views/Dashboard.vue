<script setup lang="ts">
import {
  Activity,
  ArrowRight,
  Bot,
  Cable,
  CircleDollarSign,
  KeyRound,
  Network,
  RefreshCw,
  Route,
  ShieldCheck,
  Sparkles,
  Zap,
} from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Badge, Button, Card, CardContent, CardHeader, CardTitle } from '@tabtab/ui'
import { aiGatewayApi, type GatewaySummary } from '@/api/ai-gateway'
import { useUserStore } from '@/stores/user'

const { t, locale } = useI18n()
const router = useRouter()
const userStore = useUserStore()

const loading = ref(false)
const errorMessage = ref('')
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
  tokenDailyStatistics: [],
})

const dateRange = computed(() => {
  const formatter = new Intl.DateTimeFormat(locale.value, { month: 'short', day: 'numeric' })
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - 6)
  return `${formatter.format(start)} - ${formatter.format(end)}`
})

const successRate = computed(() =>
  summary.value.callCount > 0 ? (summary.value.successCount / summary.value.callCount) * 100 : 0,
)

const averageCost = computed(() =>
  summary.value.successCount > 0 ? summary.value.totalCost / summary.value.successCount : 0,
)

const resourceTotal = computed(
  () =>
    summary.value.providerCount +
    summary.value.channelCount +
    summary.value.tokenCount +
    summary.value.proxyCount,
)

const metrics = computed(() => [
  {
    label: t('dashboard.metrics.calls'),
    value: formatCompact(summary.value.callCount),
    detail: t('dashboard.metricDetails.calls', { count: formatCompact(summary.value.errorCount) }),
    icon: Activity,
    tone: 'indigo',
  },
  {
    label: t('dashboard.metrics.successRate'),
    value: `${successRate.value.toFixed(2)}%`,
    detail: t('dashboard.metricDetails.success', {
      count: formatCompact(summary.value.successCount),
    }),
    icon: ShieldCheck,
    tone: 'emerald',
  },
  {
    label: t('dashboard.metrics.tokens'),
    value: formatCompact(summary.value.totalTokens),
    detail: t('dashboard.metricDetails.tokens', {
      input: formatCompact(summary.value.promptTokens),
      output: formatCompact(summary.value.completionTokens),
    }),
    icon: Zap,
    tone: 'amber',
  },
  {
    label: t('dashboard.metrics.cost'),
    value: formatAmount(summary.value.totalCost),
    detail: t('dashboard.metricDetails.cost', { amount: formatAmount(averageCost.value) }),
    icon: CircleDollarSign,
    tone: 'cyan',
  },
])

const resources = computed(() => [
  {
    label: t('dashboard.resources.providers'),
    value: summary.value.providerCount,
    description: t('dashboard.resourceDetails.providers'),
    icon: Network,
    path: '/ai-gateway/providers',
    tone: 'violet',
  },
  {
    label: t('dashboard.resources.channels'),
    value: summary.value.channelCount,
    description: t('dashboard.resourceDetails.channels'),
    icon: Route,
    path: '/ai-gateway/channels',
    tone: 'blue',
  },
  {
    label: t('dashboard.resources.tokens'),
    value: summary.value.tokenCount,
    description: t('dashboard.resourceDetails.tokens'),
    icon: KeyRound,
    path: '/ai-gateway/tokens',
    tone: 'emerald',
  },
  {
    label: t('dashboard.resources.proxies'),
    value: summary.value.proxyCount,
    description: t('dashboard.resourceDetails.proxies'),
    icon: Cable,
    path: '/ai-gateway/proxies',
    tone: 'orange',
  },
])

const quickActions = computed(() => [
  {
    title: t('dashboard.actions.addProvider'),
    description: t('dashboard.actions.addProviderDesc'),
    icon: Network,
    path: '/ai-gateway/providers',
  },
  {
    title: t('dashboard.actions.configureChannel'),
    description: t('dashboard.actions.configureChannelDesc'),
    icon: Route,
    path: '/ai-gateway/channels',
  },
  {
    title: t('dashboard.actions.issueToken'),
    description: t('dashboard.actions.issueTokenDesc'),
    icon: KeyRound,
    path: '/ai-gateway/tokens',
  },
  {
    title: t('dashboard.actions.inspectLogs'),
    description: t('dashboard.actions.inspectLogsDesc'),
    icon: Activity,
    path: '/ai-gateway/logs',
  },
])

const topChannels = computed(() => summary.value.channelStatistics.slice(0, 5))

const trendPoints = computed(() => {
  const data = summary.value.tokenTrend
  if (!data.length) return ''
  const maximum = Math.max(1, ...data.map((item) => item.totalTokens))
  return data
    .map((item, index) => {
      const x = data.length === 1 ? 350 : 16 + (index / (data.length - 1)) * 668
      const y = 178 - (item.totalTokens / maximum) * 142
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')
})

const trendAreaPoints = computed(() =>
  trendPoints.value ? `16,184 ${trendPoints.value} 684,184` : '',
)

const trendLabels = computed(() => {
  const data = summary.value.tokenTrend
  if (data.length <= 7) return data
  return data.filter((_, index) => index === 0 || index === data.length - 1 || index % 2 === 0)
})

function dateTimestamp(date: Date, endOfDay = false) {
  const target = new Date(date)
  target.setHours(endOfDay ? 23 : 0, endOfDay ? 59 : 0, endOfDay ? 59 : 0, endOfDay ? 999 : 0)
  return target.getTime()
}

async function loadDashboard() {
  loading.value = true
  errorMessage.value = ''
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - 6)

  try {
    const result = await aiGatewayApi.summary({
      startAt: dateTimestamp(start),
      endAt: dateTimestamp(end, true),
    })
    summary.value = {
      ...result,
      modelDistribution: result.modelDistribution || [],
      tokenTrend: result.tokenTrend || [],
      channelStatistics: result.channelStatistics || [],
      tokenDailyStatistics: result.tokenDailyStatistics || [],
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('dashboard.loadFailed')
  } finally {
    loading.value = false
  }
}

function formatCompact(value: number) {
  return new Intl.NumberFormat(locale.value, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value || 0)
}

function formatAmount(value: number) {
  return `$${(value || 0).toFixed(value >= 1 ? 2 : 4)}`
}

function formatDay(value: string) {
  return value.slice(5).replace('-', '/')
}

function go(path: string) {
  void router.push(path)
}

onMounted(loadDashboard)
</script>

<template>
  <div class="mx-auto -mt-4 w-full max-w-[1600px] space-y-3">
    <section
      class="relative overflow-hidden rounded-2xl border border-blue-100/80 bg-gradient-to-br from-white via-blue-50/80 to-sky-50/70 p-4 text-slate-950 shadow-lg shadow-blue-100/40 dark:border-blue-900/50 dark:from-slate-950 dark:via-blue-950 dark:to-slate-900 dark:text-white dark:shadow-blue-950/20 sm:p-5"
    >
      <div
        class="absolute -right-20 -top-24 h-72 w-72 rounded-full bg-blue-400/15 blur-3xl dark:bg-blue-500/20"
      />
      <div
        class="absolute -bottom-28 left-1/3 h-64 w-64 rounded-full bg-sky-300/20 blur-3xl dark:bg-cyan-400/10"
      />
      <div class="relative flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div class="max-w-3xl">
          <Badge
            class="mb-3 border-blue-200/80 bg-blue-100/80 text-blue-700 hover:bg-blue-100 dark:border-white/15 dark:bg-white/10 dark:text-white dark:hover:bg-white/10"
          >
            <Sparkles class="mr-1.5 h-3.5 w-3.5" />
            {{ t('dashboard.consoleBadge') }}
          </Badge>
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">
            {{ t('dashboard.welcome', { name: userStore.displayName }) }}
          </h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">
            {{ t('dashboard.subtitle') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div
            class="rounded-lg border border-blue-200/70 bg-white/70 px-3 py-2 shadow-sm backdrop-blur dark:border-white/10 dark:bg-white/5 dark:shadow-none"
          >
            <p class="text-[11px] uppercase tracking-[0.18em] text-blue-600/80 dark:text-slate-400">
              {{ t('dashboard.dataRange') }}
            </p>
            <p class="mt-1 text-sm font-medium text-slate-900 dark:text-white">{{ dateRange }}</p>
          </div>
          <Button
            class="h-9 shadow-sm shadow-blue-200/60 dark:shadow-none"
            :disabled="loading"
            @click="loadDashboard"
          >
            <RefreshCw class="mr-2 h-4 w-4" :class="{ 'animate-spin': loading }" />
            {{ t('dashboard.refresh') }}
          </Button>
        </div>
      </div>
    </section>

    <div
      v-if="errorMessage"
      class="flex items-center justify-between gap-4 rounded-2xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
    >
      <span>{{ errorMessage }}</span>
      <Button size="sm" variant="outline" @click="loadDashboard">{{ t('dashboard.retry') }}</Button>
    </div>

    <section class="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
      <Card
        v-for="metric in metrics"
        :key="metric.label"
        class="gap-0 overflow-hidden border-border/70 py-0"
      >
        <CardContent class="p-4">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="text-sm text-muted-foreground">{{ metric.label }}</p>
              <p class="mt-2 text-2xl font-semibold tracking-tight">{{ metric.value }}</p>
            </div>
            <div :class="`dashboard-tone dashboard-tone-${metric.tone}`">
              <component :is="metric.icon" class="h-5 w-5" />
            </div>
          </div>
          <p class="mt-4 truncate text-xs text-muted-foreground">{{ metric.detail }}</p>
        </CardContent>
      </Card>
    </section>

    <section class="grid gap-3 xl:grid-cols-[1.55fr_1fr]">
      <Card class="gap-4 border-border/70 py-4">
        <CardHeader class="flex flex-row items-start justify-between gap-4 px-4">
          <div>
            <CardTitle class="text-base">{{ t('dashboard.trafficTrend') }}</CardTitle>
            <p class="mt-1 text-sm text-muted-foreground">{{ t('dashboard.trafficTrendDesc') }}</p>
          </div>
          <Button variant="ghost" size="sm" @click="go('/ai-gateway/overview')">
            {{ t('dashboard.viewDetails') }}
            <ArrowRight class="ml-1.5 h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent class="px-4">
          <div v-if="summary.tokenTrend.length" class="rounded-2xl border bg-muted/20 p-4">
            <svg viewBox="0 0 700 200" class="h-52 w-full" preserveAspectRatio="none">
              <defs>
                <linearGradient id="dashboardTrend" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="#6366f1" stop-opacity="0.34" />
                  <stop offset="100%" stop-color="#6366f1" stop-opacity="0" />
                </linearGradient>
              </defs>
              <line
                v-for="y in [40, 88, 136, 184]"
                :key="y"
                x1="16"
                :y1="y"
                x2="684"
                :y2="y"
                stroke="currentColor"
                class="text-border"
                stroke-dasharray="4 6"
              />
              <polygon :points="trendAreaPoints" fill="url(#dashboardTrend)" />
              <polyline
                :points="trendPoints"
                fill="none"
                stroke="#6366f1"
                stroke-width="4"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            <div class="mt-1 flex justify-between text-[11px] text-muted-foreground">
              <span v-for="item in trendLabels" :key="item.date">{{ formatDay(item.date) }}</span>
            </div>
          </div>
          <div
            v-else
            class="flex h-64 flex-col items-center justify-center rounded-2xl border border-dashed text-center"
          >
            <Activity class="h-8 w-8 text-muted-foreground/50" />
            <p class="mt-3 text-sm font-medium">{{ t('dashboard.noTraffic') }}</p>
            <p class="mt-1 text-xs text-muted-foreground">{{ t('dashboard.noTrafficDesc') }}</p>
          </div>
        </CardContent>
      </Card>

      <Card class="gap-4 border-border/70 py-4">
        <CardHeader class="px-4">
          <div class="flex items-center justify-between gap-4">
            <div>
              <CardTitle class="text-base">{{ t('dashboard.resourceCenter') }}</CardTitle>
              <p class="mt-1 text-sm text-muted-foreground">
                {{ t('dashboard.resourceCenterDesc') }}
              </p>
            </div>
            <Badge variant="secondary">{{
              t('dashboard.resourceCount', { count: resourceTotal })
            }}</Badge>
          </div>
        </CardHeader>
        <CardContent class="grid gap-3 px-4 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2">
          <button
            v-for="resource in resources"
            :key="resource.label"
            class="group flex items-center gap-3 rounded-2xl border p-3 text-left transition hover:-translate-y-0.5 hover:border-primary/30 hover:bg-muted/40 hover:shadow-sm"
            @click="go(resource.path)"
          >
            <div :class="`dashboard-tone dashboard-tone-${resource.tone}`">
              <component :is="resource.icon" class="h-5 w-5" />
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-baseline justify-between gap-2">
                <span class="truncate text-sm font-medium">{{ resource.label }}</span>
                <span class="text-lg font-semibold">{{ resource.value }}</span>
              </div>
              <p class="mt-0.5 truncate text-xs text-muted-foreground">
                {{ resource.description }}
              </p>
            </div>
          </button>
        </CardContent>
      </Card>
    </section>

    <section class="grid gap-3 xl:grid-cols-[1.1fr_1fr]">
      <Card class="gap-4 border-border/70 py-4">
        <CardHeader class="flex flex-row items-start justify-between gap-4 px-4">
          <div>
            <CardTitle class="text-base">{{ t('dashboard.channelRanking') }}</CardTitle>
            <p class="mt-1 text-sm text-muted-foreground">
              {{ t('dashboard.channelRankingDesc') }}
            </p>
          </div>
          <Button variant="ghost" size="sm" @click="go('/ai-gateway/channels')">
            {{ t('dashboard.manage') }}
            <ArrowRight class="ml-1.5 h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent class="px-4">
          <div v-if="topChannels.length" class="space-y-2">
            <div
              v-for="(channel, index) in topChannels"
              :key="channel.channelId"
              class="grid grid-cols-[2rem_minmax(0,1fr)_auto] items-center gap-3 rounded-xl px-2 py-2.5 hover:bg-muted/40"
            >
              <span
                class="flex h-7 w-7 items-center justify-center rounded-lg bg-muted text-xs font-semibold"
                >{{ index + 1 }}</span
              >
              <div class="min-w-0">
                <div class="flex items-center justify-between gap-3">
                  <span class="truncate text-sm font-medium">{{ channel.channelName }}</span>
                  <span class="text-xs text-muted-foreground"
                    >{{ channel.successRate.toFixed(1) }}%</span
                  >
                </div>
                <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
                  <div
                    class="h-full rounded-full bg-gradient-to-r from-indigo-500 to-cyan-400"
                    :style="{ width: `${Math.max(3, channel.successRate)}%` }"
                  />
                </div>
              </div>
              <div class="text-right">
                <p class="text-sm font-semibold">{{ formatCompact(channel.tokenCount) }}</p>
                <p class="text-[11px] text-muted-foreground">{{ t('dashboard.tokensUnit') }}</p>
              </div>
            </div>
          </div>
          <div
            v-else
            class="flex h-52 flex-col items-center justify-center rounded-2xl border border-dashed text-center"
          >
            <Bot class="h-8 w-8 text-muted-foreground/50" />
            <p class="mt-3 text-sm font-medium">{{ t('dashboard.noChannelData') }}</p>
          </div>
        </CardContent>
      </Card>

      <Card class="gap-4 border-border/70 py-4">
        <CardHeader class="px-4">
          <CardTitle class="text-base">{{ t('dashboard.quickActions') }}</CardTitle>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('dashboard.quickActionsDesc') }}</p>
        </CardHeader>
        <CardContent class="grid gap-3 px-4 sm:grid-cols-2">
          <button
            v-for="action in quickActions"
            :key="action.title"
            class="group rounded-2xl border p-4 text-left transition hover:-translate-y-0.5 hover:border-primary/30 hover:bg-muted/30 hover:shadow-sm"
            @click="go(action.path)"
          >
            <div class="flex items-center justify-between">
              <div class="rounded-xl bg-primary/10 p-2.5 text-primary">
                <component :is="action.icon" class="h-5 w-5" />
              </div>
              <ArrowRight
                class="h-4 w-4 text-muted-foreground transition group-hover:translate-x-1 group-hover:text-primary"
              />
            </div>
            <p class="mt-4 text-sm font-semibold">{{ action.title }}</p>
            <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ action.description }}</p>
          </button>
        </CardContent>
      </Card>
    </section>
  </div>
</template>

<style scoped>
.dashboard-tone {
  display: flex;
  height: 2.75rem;
  width: 2.75rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 0.875rem;
}

.dashboard-tone-indigo,
.dashboard-tone-violet {
  background: color-mix(in oklab, #6366f1 13%, transparent);
  color: #6366f1;
}

.dashboard-tone-emerald {
  background: color-mix(in oklab, #10b981 13%, transparent);
  color: #10b981;
}

.dashboard-tone-amber,
.dashboard-tone-orange {
  background: color-mix(in oklab, #f59e0b 14%, transparent);
  color: #f59e0b;
}

.dashboard-tone-cyan,
.dashboard-tone-blue {
  background: color-mix(in oklab, #0891b2 13%, transparent);
  color: #0891b2;
}
</style>
