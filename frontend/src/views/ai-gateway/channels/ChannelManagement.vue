<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Plus, Search, Trash2 } from '@lucide/vue'
import { Badge, Button, Card, CardContent, Input, toast } from '@tabtab/ui'
import {
  aiGatewayApi,
  type Channel,
  type ModelPricingRule,
  type Provider,
  type ProxyNode,
} from '@/api/ai-gateway'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import GatewayResourceConfirmDialogs from '../components/GatewayResourceConfirmDialogs.vue'
import { useGatewayResourcePage } from '../composables/useGatewayResourcePage'
import ChannelEditorDialog from './ChannelEditorDialog.vue'

defineOptions({ name: 'AiGatewayChannels' })

const providers = ref<Provider[]>([])
const proxies = ref<ProxyNode[]>([])
const channelPricingRules = ref<ModelPricingRule[]>([])
const channelForm = reactive<Channel>({
  providerId: '',
  name: '',
  modelPricing: '[]',
  priority: 10,
  weight: 1,
  proxyId: '',
  rpm: 0,
  tpm: 0,
  costMultiplier: 1,
  status: 1,
  remark: '',
})

const {
  loading,
  errorMessage,
  editorErrorMessage,
  editorOpen,
  searchQuery,
  records: channels,
  currentPage,
  pageSize,
  total,
  deleteTarget,
  statusTarget,
  statusChangingKey,
  loadData,
  run,
  changePage,
  changePageSize,
  requestDelete: requestResourceDelete,
  confirmDelete,
  requestStatusChange: requestResourceStatusChange,
  confirmStatusChange,
  closeEditor,
  statusLabel,
  nextStatus,
  statusVariant,
} = useGatewayResourcePage<Channel>({
  fetchPage: aiGatewayApi.channelPage,
  deleteRecord: aiGatewayApi.deleteChannel,
  changeStatus: aiGatewayApi.changeChannelStatus,
})

const selectedChannelProvider = computed(() =>
  providers.value.find((provider) => provider.id === channelForm.providerId),
)
const availableChannelModels = computed(() => splitModels(selectedChannelProvider.value?.models))

function splitModels(value?: string) {
  return (value || '')
    .split(',')
    .map((model) => model.trim())
    .filter(Boolean)
}

function resetChannelForm() {
  channelPricingRules.value = []
  Object.assign(channelForm, {
    id: undefined,
    providerId: providers.value[0]?.id || '',
    name: '',
    modelPricing: '[]',
    priority: 10,
    weight: 1,
    proxyId: '',
    rpm: 0,
    tpm: 0,
    costMultiplier: 1,
    status: 1,
    remark: '',
  })
}

function parseChannelPricing(value?: string): ModelPricingRule[] {
  try {
    const rules = JSON.parse(value || '[]') as ModelPricingRule[]
    return rules.map((rule) => ({
      ...rule,
      tokenPriceUnit: rule.tokenPriceUnit || inferTokenPriceUnit(rule),
    }))
  } catch {
    return []
  }
}

function inferTokenPriceUnit(rule: ModelPricingRule): 'perMillion' | 'perToken' {
  const prices = [
    rule.inputPricePerMTokens,
    rule.outputPricePerMTokens,
    rule.cacheReadPricePerMTokens,
    rule.cacheWritePricePerMTokens,
    rule.inputImagePricePerMTokens,
    rule.outputImagePricePerMTokens,
  ].filter((price) => price > 0)
  return prices.length > 0 && prices.every((price) => price <= 0.0001) ? 'perToken' : 'perMillion'
}

function addChannelPricingRule() {
  channelPricingRules.value.push({
    model:
      availableChannelModels.value.find(
        (model) => !channelPricingRules.value.some((rule) => rule.model === model),
      ) || '',
    tokenPriceUnit: 'perMillion',
    inputPricePerMTokens: 0,
    outputPricePerMTokens: 0,
    cacheReadPricePerMTokens: 0,
    cacheWritePricePerMTokens: 0,
    inputImagePricePerImage: 0,
    outputImagePricePerImage: 0,
    inputImagePricePerMTokens: 0,
    outputImagePricePerMTokens: 0,
  })
}

function removeChannelPricingRule(index: number) {
  channelPricingRules.value.splice(index, 1)
}

function handleChannelProviderChange() {
  channelPricingRules.value = []
}

function showEditorError(message: string) {
  editorErrorMessage.value = message
  toast.error(message)
}

function prepareChannelPricing() {
  const allowedModels = new Set(availableChannelModels.value)
  const seen = new Set<string>()
  for (const rule of channelPricingRules.value) {
    rule.model = rule.model.trim()
    if (!rule.model || (rule.model !== '*' && !allowedModels.has(rule.model))) {
      showEditorError('请选择当前供应商的计价模型')
      return false
    }
    if (seen.has(rule.model)) {
      showEditorError('计价模型 ' + rule.model + ' 不能重复')
      return false
    }
    seen.add(rule.model)
  }
  channelForm.modelPricing = JSON.stringify(channelPricingRules.value)
  return true
}

async function loadEditorLookups() {
  const [providerPage, proxyPage] = await Promise.all([
    aiGatewayApi.providerPage({ page: 1, pageSize: 1000 }),
    aiGatewayApi.proxyPage({ page: 1, pageSize: 1000 }),
  ])
  providers.value = providerPage.records
  proxies.value = proxyPage.records
}

async function saveChannel() {
  if (!prepareChannelPricing()) return
  await run(
    async () => {
      await aiGatewayApi.saveChannel({ ...channelForm })
      editorOpen.value = false
      resetChannelForm()
      await loadData()
    },
    channelForm.id ? '渠道配置已更新' : '渠道已创建',
  )
}

async function editChannel(item: Channel) {
  editorErrorMessage.value = ''
  await run(async () => {
    await loadEditorLookups()
    Object.assign(channelForm, item)
    channelPricingRules.value = parseChannelPricing(item.modelPricing)
    editorOpen.value = true
  })
}

async function openCreateEditor() {
  editorErrorMessage.value = ''
  await run(async () => {
    await loadEditorLookups()
    resetChannelForm()
    editorOpen.value = true
  })
}

function requestDelete(_kind: 'channel', id: string | undefined, name: string) {
  requestResourceDelete(id, name)
}

function requestStatusChange(_kind: 'channel', item: Channel) {
  requestResourceStatusChange(item)
}
</script>

<template>
  <div class="-mt-4 space-y-4">
    <AppPageHeader
      title="渠道号池"
      description="渠道号池独立管理。"
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
        <Input v-model="searchQuery" class="pl-9" placeholder="搜索渠道名称、模型或标识" />
      </div>
      <div
        class="flex w-full flex-col gap-2 text-sm text-muted-foreground sm:w-auto sm:flex-row sm:items-center"
      >
        <span>{{ total }} 个渠道</span>
        <Button size="sm" class="w-full sm:w-auto" @click="openCreateEditor">
          <Plus class="mr-1.5 h-4 w-4" />
          新增渠道号池
        </Button>
      </div>
    </div>

    <section>
      <ChannelEditorDialog
        :open="editorOpen"
        :loading="loading"
        :error-message="editorErrorMessage"
        :form="channelForm"
        :providers="providers"
        :proxies="proxies"
        :available-models="availableChannelModels"
        :pricing-rules="channelPricingRules"
        @close="closeEditor"
        @submit="saveChannel"
        @provider-change="handleChannelProviderChange"
        @add-pricing="addChannelPricingRule"
        @remove-pricing="removeChannelPricingRule"
      />

      <Card class="gap-0 py-0">
        <CardContent class="p-0">
          <div class="mobile-table-scroll" aria-label="渠道号数据表格，可横向滚动">
            <table class="w-full text-sm">
              <thead class="border-b bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th class="px-4 py-3">名称</th>
                  <th class="px-4 py-3">供应商</th>
                  <th class="px-4 py-3">限流</th>
                  <th class="px-4 py-3">代理</th>
                  <th class="px-4 py-3">成本倍率</th>
                  <th class="px-4 py-3">状态</th>
                  <th class="px-4 py-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in channels"
                  :key="item.id"
                  class="border-b last:border-0 hover:bg-muted/30"
                >
                  <td class="px-4 py-3 font-medium">{{ item.name }}</td>
                  <td class="px-4 py-3">
                    {{ item.providerName || item.providerId }}
                  </td>
                  <td class="px-4 py-3">
                    RPM {{ item.rpm || '不限' }} / TPM {{ item.tpm || '不限' }}
                  </td>
                  <td class="px-4 py-3">{{ item.proxyName || '-' }}</td>
                  <td class="px-4 py-3">× {{ item.costMultiplier || 1 }}</td>
                  <td class="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      class="group h-auto rounded-full p-0"
                      :disabled="loading || Boolean(statusChangingKey)"
                      :aria-label="`${item.name}当前${statusLabel(item.status)}，点击切换为${statusLabel(nextStatus(item.status))}`"
                      @click="requestStatusChange('channel', item)"
                    >
                      <Badge
                        :variant="statusVariant(item.status)"
                        class="transition-opacity group-hover:opacity-80"
                        >{{ statusLabel(item.status) }}</Badge
                      >
                    </Button>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <Button variant="ghost" size="sm" @click="editChannel(item)">编辑</Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      @click="requestDelete('channel', item.id, item.name)"
                    >
                      <Trash2 class="h-4 w-4 text-destructive" />
                    </Button>
                  </td>
                </tr>
              </tbody>
            </table>
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
    </section>

    <GatewayResourceConfirmDialogs
      :delete-target="deleteTarget"
      :status-target="statusTarget"
      :loading="loading"
      :status-changing="Boolean(statusChangingKey)"
      @close-delete="deleteTarget = null"
      @confirm-delete="confirmDelete"
      @close-status="statusTarget = null"
      @confirm-status="confirmStatusChange"
    />
  </div>
</template>
