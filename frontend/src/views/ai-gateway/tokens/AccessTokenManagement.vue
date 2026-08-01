<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import {
  ChartBarStacked,
  Check,
  Copy,
  MessageSquareText,
  Plus,
  Search,
  Tags,
  Trash2,
} from '@lucide/vue'
import {
  Badge,
  Button,
  Card,
  CardContent,
  Dialog,
  DialogFixedContent,
  Input,
  Progress,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
  toast,
} from '@tabtab/ui'
import {
  aiGatewayApi,
  type AccessToken,
  type AccessTokenIPRule,
  type AccessTokenRateLimitRule,
  type AccessTokenTag,
  type Channel,
} from '@/api/ai-gateway'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import GatewayResourceConfirmDialogs from '../components/GatewayResourceConfirmDialogs.vue'
import { useGatewayResourcePage } from '../composables/useGatewayResourcePage'
import AIChat from '../chat/AIChat.vue'
import TokenTagManagerDialog from './TokenTagManagerDialog.vue'
import AccessTokenEditorDialog from './AccessTokenEditorDialog.vue'
import CreatedTokenDialog from './CreatedTokenDialog.vue'
import AccessTokenTagAssignmentDialog from './AccessTokenTagAssignmentDialog.vue'
import AccessTokenStatisticsDialog from './AccessTokenStatisticsDialog.vue'

defineOptions({ name: 'AiGatewayTokens' })

type AccessTokenIPRuleForm = { enabled: boolean; whitelist: string; blacklist: string }
type AccessTokenValidityForm = { enabled: boolean; expireAt: string }

const createdToken = ref('')
const copiedTokenId = ref('')
const channels = ref<Channel[]>([])
const tokenModelTags = ref<string[]>([])
const tokenModelSelection = ref('')
const tokenTagManagerOpen = ref(false)
const aiChatOpen = ref(false)
const tokenStatisticsOpen = ref(false)
const tokenTagAssignmentTarget = ref<AccessToken | null>(null)
const tokenTagAssignmentLoading = ref(false)
const tokenTagLoading = ref(false)
const tokenTagErrorMessage = ref('')
const editingTokenTagId = ref('')
const tokenTagForm = reactive({ name: '', remark: '' })
const tokenIPRuleForm = reactive<AccessTokenIPRuleForm>({
  enabled: false,
  whitelist: '',
  blacklist: '',
})
const tokenRateLimitForm = reactive<AccessTokenRateLimitRule>({
  enabled: false,
  fiveHourAmount: 0,
  dayAmount: 0,
  sevenDayAmount: 0,
})
const tokenValidityForm = reactive<AccessTokenValidityForm>({ enabled: false, expireAt: '' })
const tokenTags = ref<AccessTokenTag[]>([])
const sortedTokenTags = computed(() =>
  [...tokenTags.value].sort((left, right) => left.sort - right.sort),
)
const tokenForm = reactive<AccessToken>({
  channelId: '',
  name: '',
  models: '',
  ipRule: createEmptyAccessTokenIPRuleJSON(),
  rateLimitRule: createEmptyAccessTokenRateLimitRuleJSON(),
  rpm: 60,
  tpm: 0,
  quotaAmount: 0,
  expireAt: 0,
  status: 1,
  remark: '',
  tagIds: [],
})

const {
  loading,
  errorMessage,
  editorErrorMessage,
  editorOpen,
  searchQuery,
  records: tokens,
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
} = useGatewayResourcePage<AccessToken>({
  fetchPage: aiGatewayApi.tokenPage,
  deleteRecord: aiGatewayApi.deleteToken,
  changeStatus: aiGatewayApi.changeTokenStatus,
})

const selectedTokenChannel = computed(() =>
  channels.value.find((channel) => String(channel.id || '') === String(tokenForm.channelId || '')),
)
const availableTokenModels = computed(() => tokenSelectableModels(selectedTokenChannel.value))
function tokenSelectableModels(channel?: Channel) {
  if (!channel) return []
  return [...new Set(channel.availableModels || [])]
}

function resetTokenForm() {
  tokenModelTags.value = []
  tokenModelSelection.value = ''
  Object.assign(tokenIPRuleForm, { enabled: false, whitelist: '', blacklist: '' })
  Object.assign(tokenRateLimitForm, createEmptyAccessTokenRateLimitRule())
  Object.assign(tokenValidityForm, { enabled: false, expireAt: '' })
  Object.assign(tokenForm, {
    id: undefined,
    channelId: '',
    name: '',
    models: '',
    ipRule: createEmptyAccessTokenIPRuleJSON(),
    rateLimitRule: createEmptyAccessTokenRateLimitRuleJSON(),
    rpm: 60,
    tpm: 0,
    quotaAmount: 0,
    usedAmount: 0,
    expireAt: 0,
    status: 1,
    remark: '',
    tagIds: [],
  })
}

function handleTokenChannelChange() {
  tokenModelTags.value = []
  tokenModelSelection.value = ''
  tokenForm.models = ''
}

function addTokenModel() {
  const modelName = tokenModelSelection.value
  if (modelName && !tokenModelTags.value.includes(modelName)) tokenModelTags.value.push(modelName)
  tokenModelSelection.value = ''
}

function removeTokenModel(index: number) {
  tokenModelTags.value.splice(index, 1)
}

function prepareTokenModels() {
  tokenForm.models = tokenModelTags.value.join(',')
}

function showTokenEditorError(message: string) {
  editorErrorMessage.value = message
  toast.error(message)
}

function resetTokenTagForm() {
  editingTokenTagId.value = ''
  tokenTagForm.name = ''
  tokenTagForm.remark = ''
}

async function loadTokenTags() {
  tokenTagLoading.value = true
  tokenTagErrorMessage.value = ''
  try {
    tokenTags.value = await aiGatewayApi.tokenTagList()
  } catch (error) {
    const message = error instanceof Error ? error.message : '标签加载失败'
    tokenTagErrorMessage.value = message
    toast.error(message)
  } finally {
    tokenTagLoading.value = false
  }
}

async function openTokenTagManager() {
  tokenTagManagerOpen.value = true
  resetTokenTagForm()
  await loadTokenTags()
}

async function openTokenTagAssignment(item: AccessToken) {
  if (!item.id || tokenTagAssignmentLoading.value) return
  tokenTagAssignmentTarget.value = item
  tokenTagAssignmentLoading.value = true
  try {
    tokenTags.value = await aiGatewayApi.tokenTagList()
  } catch (error) {
    const message = error instanceof Error ? error.message : '\u6807\u7b7e\u52a0\u8f7d\u5931\u8d25'
    tokenTagAssignmentTarget.value = null
    toast.error(message)
  } finally {
    tokenTagAssignmentLoading.value = false
  }
}

function closeTokenTagAssignment() {
  if (!tokenTagAssignmentLoading.value) tokenTagAssignmentTarget.value = null
}

async function saveTokenTagAssignment(tagIds: string[]) {
  const target = tokenTagAssignmentTarget.value
  if (!target?.id || tokenTagAssignmentLoading.value) return
  tokenTagAssignmentLoading.value = true
  try {
    const success = await aiGatewayApi.updateTokenTags(target.id, tagIds)
    if (!success) throw new Error('API \u5bc6\u94a5\u6807\u7b7e\u4fdd\u5b58\u5931\u8d25')
    tokenTagAssignmentTarget.value = null
    toast.success('API \u5bc6\u94a5\u6807\u7b7e\u5df2\u66f4\u65b0')
    await loadData()
  } catch (error) {
    const message =
      error instanceof Error
        ? error.message
        : 'API \u5bc6\u94a5\u6807\u7b7e\u4fdd\u5b58\u5931\u8d25'
    toast.error(message)
  } finally {
    tokenTagAssignmentLoading.value = false
  }
}

async function saveTokenTag() {
  const name = tokenTagForm.name.trim()
  const remark = tokenTagForm.remark.trim()

  if (!name) {
    toast.error('请输入标签名称')
    return
  }

  const duplicate = tokenTags.value.some(
    (tag) => tag.id !== editingTokenTagId.value && tag.name.toLowerCase() === name.toLowerCase(),
  )
  if (duplicate) {
    toast.error('标签名称已存在')
    return
  }

  tokenTagLoading.value = true
  tokenTagErrorMessage.value = ''
  try {
    const editingTag = tokenTags.value.find((tag) => tag.id === editingTokenTagId.value)
    await aiGatewayApi.saveTokenTag({
      id: editingTokenTagId.value || undefined,
      name,
      remark,
      sort: editingTag?.sort || tokenTags.value.length + 1,
    })
    await loadTokenTags()
    toast.success(editingTokenTagId.value ? '标签已更新' : '标签已新增')
    resetTokenTagForm()
  } catch (error) {
    const message = error instanceof Error ? error.message : '标签保存失败'
    tokenTagErrorMessage.value = message
    toast.error(message)
  } finally {
    tokenTagLoading.value = false
  }
}

function editTokenTag(tag: AccessTokenTag) {
  editingTokenTagId.value = tag.id || ''
  tokenTagForm.name = tag.name
  tokenTagForm.remark = tag.remark
}

async function deleteTokenTag(id?: string) {
  if (!id) return
  tokenTagLoading.value = true
  tokenTagErrorMessage.value = ''
  try {
    await aiGatewayApi.deleteTokenTag(id)
    await loadTokenTags()
    if (editingTokenTagId.value === id) resetTokenTagForm()
    toast.success('标签已删除')
  } catch (error) {
    const message = error instanceof Error ? error.message : '标签删除失败'
    tokenTagErrorMessage.value = message
    toast.error(message)
  } finally {
    tokenTagLoading.value = false
  }
}

async function moveTokenTag(id: string | undefined, direction: -1 | 1) {
  if (!id) return
  const tags = sortedTokenTags.value
  const currentIndex = tags.findIndex((tag) => tag.id === id)
  const targetIndex = currentIndex + direction
  if (currentIndex < 0 || targetIndex < 0 || targetIndex >= tags.length) return

  const currentTag = tags[currentIndex]
  const targetTag = tags[targetIndex]
  if (!currentTag || !targetTag) return
  tags[currentIndex] = targetTag
  tags[targetIndex] = currentTag
  const ids = tags.flatMap((tag) => (tag.id ? [tag.id] : []))
  if (ids.length !== tags.length) return

  tokenTagLoading.value = true
  tokenTagErrorMessage.value = ''
  try {
    await aiGatewayApi.sortTokenTags(ids)
    tokenTags.value = tags.map((tag, index) => ({ ...tag, sort: index + 1 }))
  } catch (error) {
    const message = error instanceof Error ? error.message : '标签排序失败'
    tokenTagErrorMessage.value = message
    toast.error(message)
    await loadTokenTags()
  } finally {
    tokenTagLoading.value = false
  }
}

function maskSecret(value?: string) {
  if (!value) return '-'
  if (value.length <= 10) return '********'
  return `${value.slice(0, 6)}...${value.slice(-4)}`
}

function splitModels(value?: string) {
  return (value || '')
    .split(',')
    .map((model) => model.trim())
    .filter(Boolean)
}

function createEmptyAccessTokenIPRule(): AccessTokenIPRule {
  return { enabled: false, whitelist: [], blacklist: [] }
}

function createEmptyAccessTokenIPRuleJSON() {
  return JSON.stringify(createEmptyAccessTokenIPRule())
}

function splitIPRules(value?: string) {
  return [...new Set((value || '').split(/[\s,;]+/).filter(Boolean))]
}

function parseAccessTokenIPRule(value?: string): AccessTokenIPRule {
  if (!value) return createEmptyAccessTokenIPRule()
  try {
    const parsed = JSON.parse(value) as Partial<AccessTokenIPRule>
    return {
      enabled: parsed.enabled === true,
      whitelist: Array.isArray(parsed.whitelist)
        ? splitIPRules(parsed.whitelist.filter((item) => typeof item === 'string').join('\n'))
        : [],
      blacklist: Array.isArray(parsed.blacklist)
        ? splitIPRules(parsed.blacklist.filter((item) => typeof item === 'string').join('\n'))
        : [],
    }
  } catch {
    return createEmptyAccessTokenIPRule()
  }
}

function applyAccessTokenIPRuleForm(value?: string) {
  const rule = parseAccessTokenIPRule(value)
  Object.assign(tokenIPRuleForm, {
    enabled: rule.enabled,
    whitelist: rule.whitelist.join('\n'),
    blacklist: rule.blacklist.join('\n'),
  })
}

function serializeAccessTokenIPRule() {
  return JSON.stringify({
    enabled: tokenIPRuleForm.enabled,
    whitelist: splitIPRules(tokenIPRuleForm.whitelist),
    blacklist: splitIPRules(tokenIPRuleForm.blacklist),
  } satisfies AccessTokenIPRule)
}

function accessTokenIPRuleSummary(value?: string) {
  const rule = parseAccessTokenIPRule(value)
  return {
    ...rule,
    title: `白名单：${rule.whitelist.join('、') || '无'}\n黑名单：${rule.blacklist.join('、') || '无'}`,
  }
}

function createEmptyAccessTokenRateLimitRule(): AccessTokenRateLimitRule {
  return { enabled: false, fiveHourAmount: 0, dayAmount: 0, sevenDayAmount: 0 }
}

function createEmptyAccessTokenRateLimitRuleJSON() {
  return JSON.stringify(createEmptyAccessTokenRateLimitRule())
}

function parseAccessTokenRateLimitRule(value?: string): AccessTokenRateLimitRule {
  if (!value) return createEmptyAccessTokenRateLimitRule()
  try {
    const parsed = JSON.parse(value) as Partial<AccessTokenRateLimitRule>
    return {
      enabled: parsed.enabled === true,
      fiveHourAmount: Number(parsed.fiveHourAmount) || 0,
      dayAmount: Number(parsed.dayAmount) || 0,
      sevenDayAmount: Number(parsed.sevenDayAmount) || 0,
    }
  } catch {
    return createEmptyAccessTokenRateLimitRule()
  }
}

function applyAccessTokenRateLimitForm(value?: string) {
  Object.assign(tokenRateLimitForm, parseAccessTokenRateLimitRule(value))
}

function serializeAccessTokenRateLimitRule() {
  return JSON.stringify({ ...tokenRateLimitForm } satisfies AccessTokenRateLimitRule)
}

function accessTokenRateLimitSummary(value?: string) {
  const rule = parseAccessTokenRateLimitRule(value)
  return {
    ...rule,
    title: `5 小时：${formatRateLimitAmount(rule.fiveHourAmount)}\n1 天：${formatRateLimitAmount(rule.dayAmount)}\n7 天：${formatRateLimitAmount(rule.sevenDayAmount)}`,
  }
}

function formatDateTimeLocal(timestamp?: number) {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(timestamp - offset).toISOString().slice(0, 16)
}

function applyAccessTokenValidityForm(expireAt?: number) {
  Object.assign(tokenValidityForm, {
    enabled: Boolean(expireAt),
    expireAt: formatDateTimeLocal(expireAt),
  })
}

function formatAmount(value?: number) {
  return `$${(value || 0).toFixed(6)}`
}

function formatRateLimitAmount(value?: number) {
  return value && value > 0 ? formatAmount(value) : '不限'
}

function formatTime(value?: number) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function tokenPreview(item: AccessToken) {
  return maskSecret(item.token)
}

function tokenChannelDescription(item: AccessToken) {
  if (!item.channelId) return '未关联'
  if (!item.channelName) return '关联渠道不存在'
  return `${item.providerName || '未知供应商'} / ${item.channelName}`
}

function canCopyToken(item: AccessToken) {
  return Boolean(item.token && item.token !== '********')
}

function formatTokens(value?: number) {
  return (value || 0).toLocaleString()
}

function quotaPercent(item: AccessToken) {
  if (!item.quotaAmount) return 0
  return Math.min(100, ((item.usedAmount || 0) / item.quotaAmount) * 100)
}

async function copyCreatedToken(token: string, id = 'created') {
  if (!token || token === '********') return
  await navigator.clipboard.writeText(token)
  copiedTokenId.value = id
  toast.success('复制成功')
  window.setTimeout(() => {
    if (copiedTokenId.value === id) copiedTokenId.value = ''
  }, 3000)
}

async function copyToken(item: AccessToken) {
  const token = item.token
  if (!item.id || !token || !canCopyToken(item)) return
  await copyCreatedToken(token, item.id)
}

async function loadEditorLookups() {
  const [channelPage, tags] = await Promise.all([
    aiGatewayApi.channelPage({ page: 1, pageSize: 1000 }),
    aiGatewayApi.tokenTagList(),
  ])
  channels.value = channelPage.records
  tokenTags.value = tags
}

async function saveToken() {
  if (!tokenForm.channelId) {
    showTokenEditorError('请选择关联渠道号池')
    return
  }
  if (tokenModelTags.value.length === 0) {
    showTokenEditorError('请至少选择一个关联渠道模型')
    return
  }
  if (
    tokenIPRuleForm.enabled &&
    splitIPRules(tokenIPRuleForm.whitelist).length === 0 &&
    splitIPRules(tokenIPRuleForm.blacklist).length === 0
  ) {
    showTokenEditorError('启用 IP 限制时，白名单和黑名单不能同时为空')
    return
  }
  if (
    tokenRateLimitForm.fiveHourAmount < 0 ||
    tokenRateLimitForm.dayAmount < 0 ||
    tokenRateLimitForm.sevenDayAmount < 0
  ) {
    showTokenEditorError('消费速率限额不能为负数')
    return
  }
  if (tokenValidityForm.enabled) {
    const expireAt = new Date(tokenValidityForm.expireAt).getTime()
    if (!tokenValidityForm.expireAt || !Number.isFinite(expireAt)) {
      showTokenEditorError('请选择有效的密钥过期时间')
      return
    }
    if (expireAt <= Date.now()) {
      showTokenEditorError('密钥过期时间必须晚于当前时间')
      return
    }
    tokenForm.expireAt = expireAt
  } else {
    tokenForm.expireAt = 0
  }
  prepareTokenModels()
  tokenForm.ipRule = serializeAccessTokenIPRule()
  tokenForm.rateLimitRule = serializeAccessTokenRateLimitRule()
  await run(
    async () => {
      const result = await aiGatewayApi.saveToken({ ...tokenForm })
      createdToken.value = result.token || ''
      editorOpen.value = false
      resetTokenForm()
      await loadData()
    },
    tokenForm.id ? 'API 密钥已更新' : 'API 密钥已创建',
  )
}

async function editToken(item: AccessToken) {
  editorErrorMessage.value = ''
  await run(async () => {
    await loadEditorLookups()
    Object.assign(tokenForm, {
      channelId: '',
      ...item,
      ipRule: item.ipRule || createEmptyAccessTokenIPRuleJSON(),
      rateLimitRule: item.rateLimitRule || createEmptyAccessTokenRateLimitRuleJSON(),
      tagIds: [...(item.tagIds || [])],
      token: undefined,
    })
    applyAccessTokenIPRuleForm(item.ipRule)
    applyAccessTokenRateLimitForm(item.rateLimitRule)
    applyAccessTokenValidityForm(item.expireAt)
    const availableModels = new Set(availableTokenModels.value)
    tokenModelTags.value = splitModels(item.models).filter(
      (modelName) => availableModels.has('*') || availableModels.has(modelName),
    )
    tokenModelSelection.value = ''
    editorOpen.value = true
  })
}

async function openCreateEditor() {
  createdToken.value = ''
  editorErrorMessage.value = ''
  await run(async () => {
    await loadEditorLookups()
    resetTokenForm()
    editorOpen.value = true
  })
}

function requestDelete(_kind: 'token', id: string | undefined, name: string) {
  requestResourceDelete(id, name)
}

function requestStatusChange(_kind: 'token', item: AccessToken) {
  requestResourceStatusChange(item)
}
</script>

<template>
  <div class="-mt-4 space-y-4">
    <AppPageHeader
      title="API 密钥"
      description="API 密钥独立管理。"
      :loading="loading"
      @refresh="loadData"
    >
      <template #actions>
        <Button
          variant="outline"
          size="sm"
          class="w-full shrink-0 sm:w-auto"
          @click="tokenStatisticsOpen = true"
        >
          <ChartBarStacked class="mr-1.5 h-3.5 w-3.5" />
          统计分析
        </Button>
      </template>
    </AppPageHeader>

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
        <Input v-model="searchQuery" class="pl-9" placeholder="搜索 API 密钥名称、模型或标识" />
      </div>
      <div
        class="flex w-full flex-col gap-2 text-sm text-muted-foreground sm:w-auto sm:flex-row sm:items-center"
      >
        <span>{{ total }} 个 API 密钥</span>
        <Button variant="outline" size="sm" class="w-full sm:w-auto" @click="openTokenTagManager">
          <Tags class="mr-1.5 h-4 w-4" />
          标签管理
        </Button>
        <Button variant="outline" size="sm" class="w-full sm:w-auto" @click="aiChatOpen = true">
          <MessageSquareText class="mr-1.5 h-4 w-4" />
          AI Chat
        </Button>
        <Button size="sm" class="w-full sm:w-auto" @click="openCreateEditor">
          <Plus class="mr-1.5 h-4 w-4" />
          新增 API 密钥
        </Button>
      </div>
    </div>

    <TokenTagManagerDialog
      v-model:open="tokenTagManagerOpen"
      :loading="tokenTagLoading"
      :error-message="tokenTagErrorMessage"
      :form="tokenTagForm"
      :tags="sortedTokenTags"
      :editing-id="editingTokenTagId"
      @reset="resetTokenTagForm"
      @save="saveTokenTag"
      @move="moveTokenTag"
      @edit="editTokenTag"
      @delete="deleteTokenTag"
    />

    <Dialog :open="aiChatOpen" @update:open="aiChatOpen = $event">
      <DialogFixedContent
        title="AI Chat"
        description="选择 API 密钥测试大模型实时对话；会话仅保留在当前弹窗中。"
        class="h-[calc(100dvh-1rem)] sm:h-[min(860px,calc(100dvh-2rem))] sm:max-w-[min(1200px,calc(100vw-2rem))]"
        body-class="overflow-hidden p-0"
        @pointer-down-outside.prevent
      >
        <AIChat embedded />
      </DialogFixedContent>
    </Dialog>

    <AccessTokenStatisticsDialog v-model:open="tokenStatisticsOpen" />

    <section>
      <AccessTokenEditorDialog
        :open="editorOpen"
        :loading="loading"
        :error-message="editorErrorMessage"
        :form="tokenForm"
        :channels="channels"
        :tags="tokenTags"
        :available-models="availableTokenModels"
        :model-tags="tokenModelTags"
        :ip-rule-form="tokenIPRuleForm"
        :rate-limit-form="tokenRateLimitForm"
        :validity-form="tokenValidityForm"
        v-model:model-selection="tokenModelSelection"
        @close="closeEditor"
        @submit="saveToken"
        @channel-change="handleTokenChannelChange"
        @add-model="addTokenModel"
        @remove-model="removeTokenModel"
      />

      <AccessTokenTagAssignmentDialog
        :open="Boolean(tokenTagAssignmentTarget)"
        :loading="tokenTagAssignmentLoading"
        :token-name="tokenTagAssignmentTarget?.name || ''"
        :tags="tokenTags"
        :selected-tag-ids="tokenTagAssignmentTarget?.tagIds || []"
        @close="closeTokenTagAssignment"
        @save="saveTokenTagAssignment"
      />

      <Card class="gap-0 py-0">
        <CardContent class="p-0">
          <div class="mobile-table-scroll" aria-label="API 密钥数据表格，可横向滚动">
            <table class="w-full text-sm">
              <thead class="border-b bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th class="w-[180px] min-w-[180px] max-w-[180px] px-4 py-3">名称</th>
                  <th class="px-4 py-3">标签</th>
                  <th class="px-4 py-3">API 密钥</th>
                  <th class="px-4 py-3">关联号池</th>
                  <th class="px-4 py-3">模型</th>
                  <th class="px-4 py-3">调用/用量</th>
                  <th class="px-4 py-3">IP 限制</th>
                  <th class="px-4 py-3">速率限制</th>
                  <th class="px-4 py-3">额度</th>
                  <th class="px-4 py-3">有效期</th>
                  <th class="px-4 py-3">状态</th>
                  <th
                    class="sticky right-0 z-20 w-[120px] min-w-[120px] max-w-[120px] bg-muted px-4 py-3 text-center shadow-[-4px_0_8px_-6px_rgba(0,0,0,0.35)]"
                  >
                    操作
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in tokens"
                  :key="item.id"
                  class="group border-b last:border-0 hover:bg-muted/30"
                >
                  <td
                    class="w-[180px] min-w-[180px] max-w-[180px] truncate px-4 py-3 font-medium"
                    :title="item.name"
                  >
                    {{ item.name }}
                  </td>
                  <td class="min-w-[180px] px-4 py-3">
                    <button
                      type="button"
                      class="block w-full cursor-pointer rounded-md text-left outline-none transition-colors hover:bg-muted/60 focus-visible:ring-2 focus-visible:ring-ring"
                      :title="'\u70b9\u51fb\u7f16\u8f91 ' + item.name + ' \u7684\u6807\u7b7e'"
                      :aria-label="'\u7f16\u8f91 ' + item.name + ' \u7684\u6807\u7b7e'"
                      @click="openTokenTagAssignment(item)"
                    >
                      <div v-if="item.tags?.length" class="flex flex-wrap gap-1.5 p-1">
                        <Badge
                          v-for="tag in item.tags"
                          :key="tag.id"
                          variant="secondary"
                          class="cursor-pointer font-normal"
                          :title="tag.remark || tag.name"
                        >
                          {{ tag.name }}
                        </Badge>
                      </div>
                      <span v-else class="block p-1 text-muted-foreground">
                        {{ '\u70b9\u51fb\u9009\u62e9\u6807\u7b7e' }}
                      </span>
                    </button>
                  </td>
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-1.5 whitespace-nowrap">
                      <code class="text-xs">{{ tokenPreview(item) }}</code>
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger as-child>
                            <Button
                              variant="ghost"
                              size="icon"
                              class="h-7 w-7 cursor-pointer"
                              :disabled="!canCopyToken(item)"
                              aria-label="复制到剪切板"
                              @click="copyToken(item)"
                            >
                              <Check
                                v-if="copiedTokenId === item.id"
                                class="h-4 w-4 text-green-600"
                              />
                              <Copy v-else class="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent side="top">复制到剪切板</TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                  </td>
                  <td class="px-4 py-3 whitespace-nowrap">
                    {{ tokenChannelDescription(item) }}
                  </td>
                  <td class="max-w-[280px] px-4 py-3">
                    <div v-if="splitModels(item.models).length" class="flex flex-wrap gap-1.5">
                      <Badge
                        v-for="(modelName, index) in splitModels(item.models)"
                        :key="[item.id, modelName, index].join('-')"
                        variant="secondary"
                        class="max-w-[220px] font-normal"
                        :title="modelName"
                      >
                        <span class="truncate">{{ modelName }}</span>
                      </Badge>
                    </div>
                    <span v-else class="text-muted-foreground">未配置</span>
                  </td>
                  <td class="px-4 py-3 whitespace-nowrap">
                    成功 {{ item.successCount || 0 }} / 失败 {{ item.failureCount || 0 }}<br />
                    <span class="text-xs text-muted-foreground"
                      >入 {{ formatTokens(item.promptTokensTotal) }} / 出
                      {{ formatTokens(item.completionTokensTotal) }} / 缓存
                      {{ formatTokens(item.cacheReadTokensTotal) }}</span
                    ><br />
                    <span class="text-xs text-muted-foreground"
                      >最近 {{ formatTime(item.lastUsedAt) }}</span
                    >
                  </td>
                  <td class="min-w-[140px] px-4 py-3">
                    <div
                      class="flex items-center gap-2 whitespace-nowrap"
                      :title="accessTokenIPRuleSummary(item.ipRule).title"
                    >
                      <Badge variant="secondary">
                        {{ accessTokenIPRuleSummary(item.ipRule).enabled ? '已启用' : '未启用' }}
                      </Badge>
                      <span
                        v-if="accessTokenIPRuleSummary(item.ipRule).enabled"
                        class="text-xs text-muted-foreground"
                      >
                        白 {{ accessTokenIPRuleSummary(item.ipRule).whitelist.length }} / 黑
                        {{ accessTokenIPRuleSummary(item.ipRule).blacklist.length }}
                      </span>
                    </div>
                  </td>
                  <td class="min-w-[180px] px-4 py-3">
                    <div
                      class="whitespace-nowrap"
                      :title="accessTokenRateLimitSummary(item.rateLimitRule).title"
                    >
                      <div
                        v-if="accessTokenRateLimitSummary(item.rateLimitRule).enabled"
                        class="grid gap-1 text-xs"
                      >
                        <div class="flex items-center justify-between gap-3">
                          <span class="text-muted-foreground">5 小时</span>
                          <span class="font-medium text-foreground">
                            {{
                              formatRateLimitAmount(
                                accessTokenRateLimitSummary(item.rateLimitRule).fiveHourAmount,
                              )
                            }}
                          </span>
                        </div>
                        <div class="flex items-center justify-between gap-3">
                          <span class="text-muted-foreground">1 天</span>
                          <span class="font-medium text-foreground">
                            {{
                              formatRateLimitAmount(
                                accessTokenRateLimitSummary(item.rateLimitRule).dayAmount,
                              )
                            }}
                          </span>
                        </div>
                        <div class="flex items-center justify-between gap-3">
                          <span class="text-muted-foreground">7 天</span>
                          <span class="font-medium text-foreground">
                            {{
                              formatRateLimitAmount(
                                accessTokenRateLimitSummary(item.rateLimitRule).sevenDayAmount,
                              )
                            }}
                          </span>
                        </div>
                      </div>
                      <span v-else class="text-xs text-muted-foreground">未启用</span>
                    </div>
                  </td>
                  <td class="min-w-[190px] px-4 py-3">
                    <div class="flex justify-between gap-3 text-xs">
                      <span>{{ formatAmount(item.usedAmount) }}</span
                      ><span>{{ item.quotaAmount ? formatAmount(item.quotaAmount) : '不限' }}</span>
                    </div>
                    <Progress
                      v-if="item.quotaAmount"
                      class="mt-2 h-1.5"
                      :model-value="quotaPercent(item)"
                    />
                    <div v-if="item.quotaAmount" class="mt-1 text-xs text-muted-foreground">
                      已使用 {{ quotaPercent(item).toFixed(1) }}%
                    </div>
                  </td>
                  <td class="px-4 py-3 whitespace-nowrap">
                    <span v-if="item.expireAt">{{ formatTime(item.expireAt) }}</span>
                    <span v-else class="text-muted-foreground">永久有效</span>
                  </td>
                  <td class="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      class="group h-auto rounded-full p-0"
                      :disabled="loading || Boolean(statusChangingKey)"
                      :aria-label="`${item.name}当前${statusLabel(item.status)}，点击切换为${statusLabel(nextStatus(item.status))}`"
                      @click="requestStatusChange('token', item)"
                    >
                      <Badge
                        :variant="statusVariant(item.status)"
                        class="transition-opacity group-hover:opacity-80"
                        >{{ statusLabel(item.status) }}</Badge
                      >
                    </Button>
                  </td>
                  <td
                    class="sticky right-0 z-10 w-[120px] min-w-[120px] max-w-[120px] whitespace-nowrap bg-card px-4 py-3 text-center shadow-[-4px_0_8px_-6px_rgba(0,0,0,0.35)] transition-colors group-hover:bg-muted"
                  >
                    <div class="flex items-center justify-center gap-1">
                      <Button variant="ghost" size="sm" @click="editToken(item)">编辑</Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        @click="requestDelete('token', item.id, item.name)"
                      >
                        <Trash2 class="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
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

    <CreatedTokenDialog
      :token="createdToken"
      :copied="copiedTokenId === 'created'"
      @close="createdToken = ''"
      @copy="copyCreatedToken"
    />
  </div>
</template>
