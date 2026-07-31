<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { Search, ShieldAlert } from '@lucide/vue'
import { Badge, Card, CardContent, Input, toast } from '@tabtab/ui'
import { aiGatewayApi, type CallLog } from '@/api/ai-gateway'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'

defineOptions({ name: 'AiGatewayCallLogsManagement' })

const loading = ref(false)
const errorMessage = ref('')
const searchQuery = ref('')
const statusFilter = ref<'all' | 'success' | 'failed'>('all')
const logs = ref<CallLog[]>([])
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)
let loadSequence = 0
let searchTimer: ReturnType<typeof setTimeout> | undefined

async function loadData() {
  const sequence = ++loadSequence
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await aiGatewayApi.logPage({
      page: currentPage.value,
      pageSize: pageSize.value,
      keyword: searchQuery.value.trim() || undefined,
      success: statusFilter.value === 'all' ? undefined : statusFilter.value === 'success' ? 1 : 2,
    })
    if (sequence !== loadSequence) return
    logs.value = page.records
    total.value = page.total
    currentPage.value = page.page || currentPage.value
    pageSize.value = page.pageSize || pageSize.value
  } catch (error) {
    if (sequence !== loadSequence) return
    errorMessage.value = error instanceof Error ? error.message : '调用日志加载失败'
    toast.error(errorMessage.value)
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

function changePage(page: number) {
  const totalPages = Math.max(1, Math.ceil(total.value / pageSize.value))
  if (page < 1 || page > totalPages || page === currentPage.value || loading.value) return
  currentPage.value = page
  loadData()
}

function changePageSize(size: number) {
  if (loading.value || size === pageSize.value) return
  pageSize.value = size
  currentPage.value = 1
  loadData()
}

function formatAmount(value?: number) {
  return `$${(value || 0).toFixed(6)}`
}

function formatTime(value?: number) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function formatTokens(value?: number) {
  return (value || 0).toLocaleString()
}

watch([searchQuery, statusFilter], () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadData()
  }, 300)
})

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
  loadSequence += 1
})

loadData()
</script>

<template>
  <div class="-mt-4 space-y-4">
    <AppPageHeader
      title="调用日志"
      description="查询 AI 网关调用结果、Token 用量、成本和响应延迟。"
      :loading="loading"
      @refresh="loadData"
    />

    <div
      v-if="errorMessage"
      class="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ errorMessage }}
    </div>

    <div
      class="flex flex-col gap-3 rounded-xl border bg-card p-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <div class="relative w-full sm:max-w-md">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          v-model="searchQuery"
          class="pl-9"
          placeholder="搜索请求 ID、模型、路径、IP 或错误信息"
        />
      </div>
      <div
        class="flex w-full flex-col gap-2 text-sm text-muted-foreground sm:w-auto sm:flex-row sm:items-center"
      >
        <select
          v-model="statusFilter"
          class="h-9 w-full rounded-md border bg-background px-3 text-sm text-foreground sm:w-auto"
          :disabled="loading"
        >
          <option value="all">全部状态</option>
          <option value="success">仅成功</option>
          <option value="failed">仅失败</option>
        </select>
        <span>{{ total }} 条记录</span>
      </div>
    </div>

    <Card class="gap-0 py-0">
      <CardContent class="p-0">
        <div class="hidden md:block mobile-table-scroll" aria-label="调用日志表格，可横向滚动">
          <table class="w-full text-sm">
            <thead class="border-b bg-muted/40 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3">时间</th>
                <th class="px-4 py-3">模型</th>
                <th class="px-4 py-3">路径</th>
                <th class="px-4 py-3">状态</th>
                <th class="px-4 py-3">Token</th>
                <th class="px-4 py-3">成本</th>
                <th class="px-4 py-3">延迟</th>
                <th class="px-4 py-3">错误</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!loading && logs.length === 0">
                <td colspan="8" class="px-4 py-10 text-center text-muted-foreground">
                  暂无调用日志
                </td>
              </tr>
              <tr
                v-for="item in logs"
                :key="item.id"
                class="border-b last:border-0 hover:bg-muted/30"
              >
                <td class="px-4 py-3 whitespace-nowrap">{{ formatTime(item.createdAt) }}</td>
                <td class="px-4 py-3">
                  <div class="font-medium">{{ item.model }}</div>
                  <div class="text-xs text-muted-foreground">{{ item.requestId }}</div>
                </td>
                <td class="px-4 py-3">{{ item.method }} {{ item.path }}</td>
                <td class="px-4 py-3">
                  <Badge :variant="item.success === 1 ? 'default' : 'destructive'">
                    {{ item.statusCode }}
                  </Badge>
                </td>
                <td class="px-4 py-3 whitespace-nowrap">
                  {{ formatTokens(item.totalTokens) }}<br />
                  <span class="text-xs text-muted-foreground">
                    入 {{ formatTokens(item.promptTokens) }} / 出
                    {{ formatTokens(item.completionTokens) }} / 推理
                    {{ formatTokens(item.reasoningTokens) }}
                  </span>
                  <br />
                  <span class="text-xs text-muted-foreground">
                    缓存读 {{ formatTokens(item.cacheReadTokens) }} / 写
                    {{ formatTokens(item.cacheWriteTokens) }}
                  </span>
                  <template v-if="item.inputImages || item.outputImages">
                    <br />
                    <span class="text-xs text-muted-foreground"
                      >图片入 {{ item.inputImages }} / 出 {{ item.outputImages }}</span
                    >
                  </template>
                  <template v-if="item.inputImageTokens || item.outputImageTokens">
                    <br />
                    <span class="text-xs text-muted-foreground"
                      >图片 Token 入 {{ formatTokens(item.inputImageTokens) }} / 出
                      {{ formatTokens(item.outputImageTokens) }}</span
                    >
                  </template>
                </td>
                <td class="px-4 py-3 whitespace-nowrap">
                  {{ formatAmount(item.cost) }}<br />
                  <span class="text-xs text-muted-foreground">
                    标准 {{ formatAmount(item.standardCost) }} × {{ item.costMultiplier || 1 }}
                  </span>
                  <template v-if="item.pricingModel">
                    <br /><span class="text-xs text-muted-foreground"
                      >计价 {{ item.pricingModel }}</span
                    >
                  </template>
                </td>
                <td class="px-4 py-3">{{ item.latencyMs }}ms</td>
                <td class="px-4 py-3 max-w-[280px] truncate">
                  <span
                    v-if="item.errorMessage"
                    class="inline-flex items-center gap-1 text-destructive"
                  >
                    <ShieldAlert class="h-3.5 w-3.5" />
                    {{ item.errorMessage }}
                  </span>
                  <span v-else>-</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="divide-y md:hidden">
          <div
            v-if="!loading && logs.length === 0"
            class="px-4 py-10 text-center text-muted-foreground"
          >
            暂无调用日志
          </div>
          <article v-for="item in logs" :key="item.id" class="space-y-3 p-4">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="truncate font-medium">{{ item.model }}</div>
                <div class="mt-0.5 break-all text-xs text-muted-foreground">
                  {{ item.requestId }}
                </div>
              </div>
              <Badge class="shrink-0" :variant="item.success === 1 ? 'default' : 'destructive'">{{
                item.statusCode
              }}</Badge>
            </div>
            <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
              <div>
                <div class="text-xs text-muted-foreground">时间</div>
                <div class="mt-0.5">{{ formatTime(item.createdAt) }}</div>
              </div>
              <div>
                <div class="text-xs text-muted-foreground">延迟</div>
                <div class="mt-0.5">{{ item.latencyMs }}ms</div>
              </div>
              <div>
                <div class="text-xs text-muted-foreground">Token</div>
                <div class="mt-0.5">{{ formatTokens(item.totalTokens) }}</div>
              </div>
              <div>
                <div class="text-xs text-muted-foreground">成本</div>
                <div class="mt-0.5">{{ formatAmount(item.cost) }}</div>
              </div>
            </div>
            <div class="rounded-lg bg-muted/50 px-3 py-2 text-sm">
              <div class="text-xs text-muted-foreground">请求路径</div>
              <div class="mt-1 break-all font-mono text-xs">{{ item.method }} {{ item.path }}</div>
            </div>
            <div v-if="item.errorMessage" class="flex items-start gap-1.5 text-sm text-destructive">
              <ShieldAlert class="mt-0.5 h-3.5 w-3.5 shrink-0" /><span class="break-words">{{
                item.errorMessage
              }}</span>
            </div>
          </article>
        </div>
        <AppPagination
          :total="total"
          :page="currentPage"
          :page-size="pageSize"
          :loading="loading"
          @change-page="changePage"
          @change-page-size="changePageSize"
        />
      </CardContent>
    </Card>
  </div>
</template>
