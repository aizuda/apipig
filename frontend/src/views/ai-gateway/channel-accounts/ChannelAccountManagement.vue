<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { Plus, Search, Trash2 } from '@lucide/vue'
import { Badge, Button, Card, CardContent, Input, toast } from '@tabtab/ui'
import { aiGatewayApi, type Channel, type ChannelAccount } from '@/api/ai-gateway'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import ChannelAccountEditorDialog from './ChannelAccountEditorDialog.vue'
import GatewayResourceConfirmDialogs from '../components/GatewayResourceConfirmDialogs.vue'

defineOptions({ name: 'AiGatewayChannelAccounts' })

type AccountModelMapping = { gatewayModel: string; providerModel: string }

const loading = ref(false)
const errorMessage = ref('')
const editorErrorMessage = ref('')
const editorOpen = ref(false)
const searchQuery = ref('')
const deleteTarget = ref<ChannelAccount | null>(null)
const statusTarget = ref<{ item: ChannelAccount; targetStatus: number } | null>(null)
const accounts = ref<ChannelAccount[]>([])
const channels = ref<Channel[]>([])
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)
const apiKeyVisible = ref(false)
const accountModelMappings = ref<AccountModelMapping[]>([])
let loadSequence = 0
let searchTimer: ReturnType<typeof setTimeout> | undefined

const form = reactive<ChannelAccount>({
  channelId: '',
  name: '',
  apiKey: '',
  models: '{}',
  status: 1,
  remark: '',
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

function resetForm() {
  accountModelMappings.value = []
  apiKeyVisible.value = false
  editorErrorMessage.value = ''
  Object.assign(form, {
    id: undefined,
    channelId: channels.value[0]?.id || '',
    name: '',
    apiKey: '',
    models: '{}',
    status: 1,
    remark: '',
  })
}

function parseAccountMappings(value?: string): AccountModelMapping[] {
  try {
    return Object.entries(JSON.parse(value || '{}') as Record<string, string>).map(
      ([gatewayModel, providerModel]) => ({ gatewayModel, providerModel }),
    )
  } catch {
    return []
  }
}

function addAccountMapping() {
  const channel = channels.value.find((item) => item.id === form.channelId)
  const providerModel = channel?.providerModels?.[0] || ''
  accountModelMappings.value.push({ gatewayModel: providerModel, providerModel })
}

function removeAccountMapping(index: number) {
  accountModelMappings.value.splice(index, 1)
}

function handleAccountChannelChange() {
  accountModelMappings.value = []
}

function prepareAccountModels() {
  if (!accountModelMappings.value.length) {
    editorErrorMessage.value = '请至少添加一个模型映射'
    toast.error(editorErrorMessage.value)
    return false
  }
  const mappings: Record<string, string> = {}
  for (const item of accountModelMappings.value) {
    const gatewayModel = item.gatewayModel.trim()
    const providerModel = item.providerModel.trim()
    if (!gatewayModel || !providerModel || mappings[gatewayModel]) {
      editorErrorMessage.value = '请完整填写且不要重复网关模型'
      toast.error(editorErrorMessage.value)
      return false
    }
    mappings[gatewayModel] = providerModel
  }
  form.models = JSON.stringify(mappings)
  return true
}

async function run(action: () => Promise<void>, successMessage?: string) {
  loading.value = true
  if (editorOpen.value) {
    editorErrorMessage.value = ''
  } else {
    errorMessage.value = ''
  }
  try {
    await action()
    if (successMessage) toast.success(successMessage)
  } catch (error) {
    const message = error instanceof Error ? error.message : '操作失败'
    if (editorOpen.value) {
      editorErrorMessage.value = message
    } else {
      errorMessage.value = message
    }
    toast.error(message)
  } finally {
    loading.value = false
  }
}

async function loadData() {
  const sequence = ++loadSequence
  loading.value = true
  errorMessage.value = ''
  try {
    const accountPage = await aiGatewayApi.channelAccountPage({
      page: currentPage.value,
      pageSize: pageSize.value,
      keyword: searchQuery.value.trim() || undefined,
    })
    if (sequence !== loadSequence) return
    accounts.value = accountPage.records
    total.value = accountPage.total
    currentPage.value = accountPage.page || currentPage.value
    pageSize.value = accountPage.pageSize || pageSize.value
  } catch (error) {
    if (sequence !== loadSequence) return
    errorMessage.value = error instanceof Error ? error.message : '账户数据加载失败'
    toast.error(errorMessage.value)
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

function refreshData() {
  return loadData()
}

function changePage(page: number) {
  if (page < 1 || page > totalPages.value || page === currentPage.value || loading.value) return
  currentPage.value = page
  loadData()
}

function changePageSize(size: number) {
  pageSize.value = size
  currentPage.value = 1
  loadData()
}

async function loadEditorChannels() {
  const channelPage = await aiGatewayApi.channelPage({ page: 1, pageSize: 1000 })
  channels.value = channelPage.records
}

async function openCreate() {
  await run(async () => {
    await loadEditorChannels()
    resetForm()
    editorOpen.value = true
  })
}

async function openEdit(item: ChannelAccount) {
  editorErrorMessage.value = ''
  await run(async () => {
    await loadEditorChannels()
    apiKeyVisible.value = false
    Object.assign(form, item)
    accountModelMappings.value = parseAccountMappings(item.models)
    editorOpen.value = true
  })
}

async function saveAccount() {
  if (!form.channelId || !form.name.trim() || (!form.id && !form.apiKey.trim())) {
    const message = '请选择关联渠道，并填写账户名称和 API Key'
    editorErrorMessage.value = message
    toast.error(message)
    return
  }
  if (!prepareAccountModels()) return
  await run(
    async () => {
      const success = await aiGatewayApi.saveChannelAccount({ ...form })
      if (!success) throw new Error('账户保存失败')
      editorOpen.value = false
      resetForm()
      await loadData()
    },
    form.id ? '账户已更新' : '账户已创建',
  )
}

function closeEditor() {
  if (!loading.value) {
    editorOpen.value = false
    editorErrorMessage.value = ''
  }
}

function requestStatusChange(item: ChannelAccount) {
  if (!item.id || loading.value) return
  statusTarget.value = {
    item,
    targetStatus: item.status === 1 ? 2 : 1,
  }
}

async function confirmStatusChange() {
  const target = statusTarget.value
  if (!target?.item.id) return
  await run(
    async () => {
      const success = await aiGatewayApi.changeChannelAccountStatus(
        target.item.id!,
        target.targetStatus,
      )
      if (!success) throw new Error('状态切换失败')
      target.item.status = target.targetStatus
      statusTarget.value = null
    },
    target.targetStatus === 1 ? '账户已启用' : '账户已禁用',
  )
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target?.id) return
  await run(async () => {
    const success = await aiGatewayApi.deleteChannelAccount(target.id!)
    if (!success) throw new Error('账户删除失败')
    deleteTarget.value = null
    if (accounts.value.length === 1 && currentPage.value > 1) currentPage.value -= 1
    await loadData()
  }, `${target.name} 已删除`)
}

function channelDescription(item: ChannelAccount) {
  if (!item.channelName) return '关联渠道不存在'
  return `${item.providerName || '未知供应商'} / ${item.channelName}`
}

function maskSecret(value?: string) {
  if (!value) return '-'
  if (value.length <= 10) return '********'
  return `${value.slice(0, 6)}...${value.slice(-4)}`
}

watch(searchQuery, () => {
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
  <div class="space-y-5">
    <AppPageHeader
      title="账户管理"
      description="管理 OpenAI、DeepSeek、Anthropic 等上游平台注册账户，并关联到具体渠道号。"
      :loading="loading"
      @refresh="refreshData"
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
        <Input v-model="searchQuery" class="pl-9" placeholder="搜索账户名称、渠道或供应商" />
      </div>
      <div
        class="flex w-full flex-col gap-2 text-sm text-muted-foreground sm:w-auto sm:flex-row sm:items-center"
      >
        <span>{{ total }} 个账户</span>
        <Button size="sm" :disabled="loading" @click="openCreate">
          <Plus class="mr-1.5 h-4 w-4" />新增账户
        </Button>
      </div>
    </div>

    <Card class="gap-0 py-0">
      <CardContent class="p-0">
        <div
          v-if="accounts.length === 0 && !loading"
          class="p-10 text-center text-sm text-muted-foreground"
        >
          暂无账户数据
        </div>
        <div v-else class="mobile-table-scroll" aria-label="渠道账号数据表格，可横向滚动">
          <table class="w-full text-sm">
            <thead class="border-b bg-muted/40 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3">名称</th>
                <th class="px-4 py-3">关联渠道</th>
                <th class="px-4 py-3">API Key</th>
                <th class="px-4 py-3">备注</th>
                <th class="px-4 py-3">状态</th>
                <th class="px-4 py-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="item in accounts"
                :key="item.id"
                class="border-b last:border-0 hover:bg-muted/30"
              >
                <td class="px-4 py-3 font-medium">{{ item.name }}</td>
                <td class="px-4 py-3">{{ channelDescription(item) }}</td>
                <td class="px-4 py-3 font-mono text-xs">{{ maskSecret(item.apiKey) }}</td>
                <td class="max-w-xs truncate px-4 py-3 text-muted-foreground">
                  {{ item.remark || '-' }}
                </td>
                <td class="px-4 py-3">
                  <Button
                    variant="ghost"
                    size="sm"
                    class="group h-auto rounded-full p-0"
                    :disabled="loading"
                    :aria-label="`${item.name}当前${item.status === 1 ? '启用' : '禁用'}，点击切换状态`"
                    @click="requestStatusChange(item)"
                  >
                    <Badge
                      :variant="item.status === 1 ? 'default' : 'secondary'"
                      class="transition-opacity group-hover:opacity-80"
                    >
                      {{ item.status === 1 ? '启用' : '禁用' }}
                    </Badge>
                  </Button>
                </td>
                <td class="px-4 py-3 text-right">
                  <Button variant="ghost" size="sm" @click="openEdit(item)">编辑</Button>
                  <Button variant="ghost" size="icon" @click="deleteTarget = item">
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

    <ChannelAccountEditorDialog
      :open="editorOpen"
      :loading="loading"
      :error-message="editorErrorMessage"
      :form="form"
      :channels="channels"
      :model-mappings="accountModelMappings"
      v-model:api-key-visible="apiKeyVisible"
      @close="closeEditor"
      @submit="saveAccount"
      @channel-change="handleAccountChannelChange"
      @add-mapping="addAccountMapping"
      @remove-mapping="removeAccountMapping"
    />

    <GatewayResourceConfirmDialogs
      :delete-target="deleteTarget"
      :status-target="statusTarget"
      :loading="loading"
      :status-changing="loading"
      delete-description="删除后无法恢复，但不会删除关联的渠道号。"
      @close-delete="deleteTarget = null"
      @confirm-delete="confirmDelete"
      @close-status="statusTarget = null"
      @confirm-status="confirmStatusChange"
    />
  </div>
</template>
