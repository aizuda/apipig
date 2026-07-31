<script setup lang="ts">
import { computed, toRef } from 'vue'
import { X } from '@lucide/vue'
import {
  Badge,
  Button,
  Dialog,
  DialogFixedContent,
  Input,
  Label,
  Switch,
  Textarea,
} from '@tabtab/ui'
import type { AccessToken, AccessTokenTag, Channel } from '@/api/ai-gateway'

const props = defineProps<{
  open: boolean
  loading: boolean
  errorMessage: string
  form: AccessToken
  channels: Channel[]
  tags: AccessTokenTag[]
  availableModels: string[]
  modelTags: string[]
  modelSelection: string
  ipRuleForm: {
    enabled: boolean
    whitelist: string
    blacklist: string
  }
  rateLimitForm: {
    enabled: boolean
    fiveHourAmount: number
    dayAmount: number
    sevenDayAmount: number
  }
  validityForm: {
    enabled: boolean
    expireAt: string
  }
}>()

const emit = defineEmits<{
  close: []
  submit: []
  channelChange: []
  addModel: []
  removeModel: [index: number]
  'update:modelSelection': [value: string]
}>()

const tokenForm = toRef(props, 'form')
const channels = toRef(props, 'channels')
const sortedTokenTags = computed(() =>
  props.tags
    .filter((tag): tag is AccessTokenTag & { id: string } => Boolean(tag.id))
    .sort((left, right) => left.sort - right.sort),
)
const availableTokenModels = toRef(props, 'availableModels')
const tokenModelTags = toRef(props, 'modelTags')
const tokenIPRuleForm = toRef(props, 'ipRuleForm')
const tokenRateLimitForm = toRef(props, 'rateLimitForm')
const tokenValidityForm = toRef(props, 'validityForm')
const tokenModelSelection = computed({
  get: () => props.modelSelection,
  set: (value) => emit('update:modelSelection', value),
})
const editorOpen = computed(() => props.open)
const editorErrorMessage = computed(() => props.errorMessage)

function closeEditor() {
  emit('close')
}
function saveToken() {
  emit('submit')
}
function handleTokenChannelChange() {
  emit('channelChange')
}
function addTokenModel() {
  emit('addModel')
}
function removeTokenModel(index: number) {
  emit('removeModel', index)
}

function setValidityDays(days: number) {
  tokenValidityForm.value.enabled = true
  const expireAt = Date.now() + days * 24 * 60 * 60 * 1000
  const date = new Date(expireAt)
  const offset = date.getTimezoneOffset() * 60_000
  tokenValidityForm.value.expireAt = new Date(expireAt - offset).toISOString().slice(0, 16)
}

function handleValidityEnabled(enabled: boolean) {
  tokenValidityForm.value.enabled = enabled
  if (enabled && !tokenValidityForm.value.expireAt) setValidityDays(7)
}

function isValidityPreset(days: number) {
  if (!tokenValidityForm.value.enabled || !tokenValidityForm.value.expireAt) return false
  const difference = new Date(tokenValidityForm.value.expireAt).getTime() - Date.now()
  return Math.abs(difference - days * 24 * 60 * 60 * 1000) < 60_000
}

function openCustomValidity() {
  const input = document.querySelector<HTMLInputElement>('#api-key-expire-at')
  input?.focus()
  input?.showPicker?.()
}
</script>

<template>
  <Dialog
    :open="editorOpen"
    @update:open="
      (open) => {
        if (!open) closeEditor()
      }
    "
  >
    <DialogFixedContent
      :title="tokenForm.id ? '编辑 API 密钥' : '创建 API 密钥'"
      :description="
        tokenForm.id
          ? '编辑授权模型、IP 限制、消费速率限制和有效期；已生成的 API 密钥不可修改。'
          : '配置授权模型、IP 限制、消费速率限制和有效期；API 密钥由系统安全生成。'
      "
      class="sm:max-w-2xl"
    >
      <div class="space-y-3">
        <div
          v-if="editorErrorMessage"
          class="sticky top-0 z-20 rounded-md border border-destructive/40 bg-background px-4 py-3 text-sm text-destructive shadow-sm"
        >
          {{ editorErrorMessage }}
        </div>
        <div class="space-y-1.5">
          <Label for="api-key-name">密钥名称 <span class="text-destructive">*</span></Label
          ><Input id="api-key-name" v-model="tokenForm.name" placeholder="例如 生产环境服务" />
          <p class="text-xs text-muted-foreground">
            用于识别调用方或业务系统，不会作为实际认证密钥。
          </p>
        </div>
        <div class="space-y-1.5">
          <div class="flex items-center justify-between gap-3">
            <Label>关联标签（可选）</Label>
            <span class="text-xs text-muted-foreground">
              已选择 {{ tokenForm.tagIds?.length || 0 }} 个
            </span>
          </div>
          <div
            v-if="sortedTokenTags.length === 0"
            class="rounded-md border border-dashed p-4 text-sm text-muted-foreground"
          >
            暂无可选标签，请关闭窗口后通过“标签管理”新增标签。
          </div>
          <div v-else class="grid gap-2 sm:grid-cols-2">
            <label
              v-for="tag in sortedTokenTags"
              :key="tag.id"
              class="flex min-w-0 cursor-pointer items-start gap-2 rounded-lg border p-3 transition-colors hover:bg-muted/40"
              :class="{
                'border-primary/50 bg-primary/5': tokenForm.tagIds?.includes(tag.id),
              }"
            >
              <input
                v-model="tokenForm.tagIds"
                type="checkbox"
                :value="tag.id"
                class="mt-0.5 h-4 w-4 shrink-0 accent-primary"
              />
              <span class="min-w-0">
                <span class="block text-sm font-medium">{{ tag.name }}</span>
                <span class="mt-0.5 block break-words text-xs text-muted-foreground">
                  {{ tag.remark || '暂无备注' }}
                </span>
              </span>
            </label>
          </div>
        </div>
        <div class="space-y-1.5">
          <Label for="api-key-channel">关联渠道号池 <span class="text-destructive">*</span></Label>
          <select
            id="api-key-channel"
            v-model="tokenForm.channelId"
            class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            @change="handleTokenChannelChange"
          >
            <option value="">请选择渠道号池</option>
            <option v-for="channel in channels" :key="channel.id" :value="channel.id">
              {{ channel.providerName || '未知供应商' }} / {{ channel.name }}
            </option>
          </select>
          <p class="text-xs text-muted-foreground">选择渠道号池后，可配置该密钥允许访问的模型。</p>
        </div>
        <div class="space-y-1.5">
          <Label for="api-key-models">关联渠道模型 <span class="text-destructive">*</span></Label>
          <div
            v-if="!tokenForm.channelId"
            class="rounded-md border border-dashed p-4 text-sm text-muted-foreground"
          >
            请先选择渠道号池，再选择允许访问的模型。
          </div>
          <div
            v-else-if="availableTokenModels.length === 0"
            class="rounded-md border border-dashed p-4 text-sm text-muted-foreground"
          >
            当前号池没有可选择的模型，请先配置渠道模型映射或供应商支持模型。
          </div>
          <div v-else class="space-y-2">
            <div v-if="tokenModelTags.length" class="flex flex-wrap gap-1.5 rounded-md border p-2">
              <Badge
                v-for="(modelName, index) in tokenModelTags"
                :key="[modelName, index].join('-')"
                variant="secondary"
                class="gap-1 font-normal"
              >
                {{ modelName }}
                <button
                  type="button"
                  class="rounded-full text-muted-foreground hover:text-foreground"
                  :aria-label="`删除模型 ${modelName}`"
                  @click="removeTokenModel(index)"
                >
                  <X class="h-3 w-3" />
                </button>
              </Badge>
            </div>
            <select
              id="api-key-models"
              v-model="tokenModelSelection"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
              @change="addTokenModel"
            >
              <option value="">请选择模型</option>
              <option
                v-for="modelName in availableTokenModels"
                :key="modelName"
                :value="modelName"
                :disabled="tokenModelTags.includes(modelName)"
              >
                {{ modelName }}
              </option>
            </select>
          </div>
          <p class="text-xs text-muted-foreground">
            至少选择一个模型；模型只能从所选号池中选择，变更号池后需要重新选择。
          </p>
        </div>
        <div class="flex items-center justify-between gap-4 rounded-lg border p-4">
          <div class="space-y-1">
            <Label for="api-key-ip-restriction">IP 限制</Label>
            <p class="text-xs text-muted-foreground">开启后同时应用白名单和黑名单规则。</p>
          </div>
          <Switch
            id="api-key-ip-restriction"
            :model-value="tokenIPRuleForm.enabled"
            @update:model-value="tokenIPRuleForm.enabled = $event"
          />
        </div>
        <div v-if="tokenIPRuleForm.enabled" class="space-y-4">
          <div class="space-y-1.5">
            <Label for="api-key-ip-whitelist">IP 白名单</Label>
            <Textarea
              id="api-key-ip-whitelist"
              v-model="tokenIPRuleForm.whitelist"
              rows="4"
              placeholder="192.168.1.100&#10;10.0.0.0/8"
            />
            <p class="text-xs text-muted-foreground">
              每行一个 IP 或 CIDR；填写后仅允许这些 IP 使用此密钥。
            </p>
          </div>
          <div class="space-y-1.5">
            <Label for="api-key-ip-blacklist">IP 黑名单</Label>
            <Textarea
              id="api-key-ip-blacklist"
              v-model="tokenIPRuleForm.blacklist"
              rows="4"
              placeholder="1.2.3.4&#10;5.6.0.0/16"
            />
            <p class="text-xs text-muted-foreground">
              每行一个 IP 或 CIDR；这些 IP 将被禁止使用此密钥，且黑名单优先。
            </p>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="api-key-rpm">RPM 请求上限</Label
            ><Input id="api-key-rpm" v-model.number="tokenForm.rpm" type="number" min="0" />
            <p class="text-xs text-muted-foreground">
              调用方每分钟最大请求数，默认 60；0 表示不限制。
            </p>
          </div>
          <div class="space-y-1.5">
            <Label for="api-key-tpm">TPM Token 上限</Label
            ><Input id="api-key-tpm" v-model.number="tokenForm.tpm" type="number" min="0" />
            <p class="text-xs text-muted-foreground">调用方每分钟最大 Token 数，0 表示不限制。</p>
          </div>
        </div>
        <div class="space-y-4 rounded-lg border p-4">
          <div class="flex items-center justify-between gap-4">
            <div class="space-y-1">
              <Label for="api-key-rate-limit">速率限制</Label>
              <p class="text-xs text-muted-foreground">
                设置此密钥在指定时间窗口内的最大消费金额，0 表示不限制。
              </p>
            </div>
            <Switch
              id="api-key-rate-limit"
              :model-value="tokenRateLimitForm.enabled"
              @update:model-value="tokenRateLimitForm.enabled = $event"
            />
          </div>
          <div v-if="tokenRateLimitForm.enabled" class="grid gap-3">
            <div class="space-y-1.5">
              <Label for="api-key-five-hour-limit">5 小时限额（USD）</Label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                  >$</span
                >
                <Input
                  id="api-key-five-hour-limit"
                  v-model.number="tokenRateLimitForm.fiveHourAmount"
                  class="pl-8"
                  type="number"
                  min="0"
                  step="0.000001"
                />
              </div>
            </div>
            <div class="space-y-1.5">
              <Label for="api-key-day-limit">日限额（USD）</Label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                  >$</span
                >
                <Input
                  id="api-key-day-limit"
                  v-model.number="tokenRateLimitForm.dayAmount"
                  class="pl-8"
                  type="number"
                  min="0"
                  step="0.000001"
                />
              </div>
            </div>
            <div class="space-y-1.5">
              <Label for="api-key-seven-day-limit">7 天限额（USD）</Label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                  >$</span
                >
                <Input
                  id="api-key-seven-day-limit"
                  v-model.number="tokenRateLimitForm.sevenDayAmount"
                  class="pl-8"
                  type="number"
                  min="0"
                  step="0.000001"
                />
              </div>
            </div>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="api-key-quota">成本额度（美元）</Label
            ><Input
              id="api-key-quota"
              v-model.number="tokenForm.quotaAmount"
              type="number"
              min="0"
              step="0.000001"
            />
            <p class="text-xs text-muted-foreground">
              累计有效成本上限，0 表示不限制；已用金额不会因修改额度而清零。
            </p>
          </div>
          <div class="space-y-1.5">
            <Label for="api-key-status">启用状态</Label
            ><select
              id="api-key-status"
              v-model.number="tokenForm.status"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            >
              <option :value="1">启用密钥</option>
              <option :value="2">禁用密钥</option>
            </select>
            <p class="text-xs text-muted-foreground">禁用后所有使用该密钥的新请求立即拒绝。</p>
          </div>
        </div>
        <div class="space-y-4 rounded-lg border p-4">
          <div class="flex items-center justify-between gap-4">
            <div class="space-y-1">
              <Label for="api-key-validity">密钥有效期</Label>
              <p class="text-xs text-muted-foreground">关闭后密钥永久有效。</p>
            </div>
            <Switch
              id="api-key-validity"
              :model-value="tokenValidityForm.enabled"
              @update:model-value="handleValidityEnabled"
            />
          </div>
          <div v-if="tokenValidityForm.enabled" class="space-y-3">
            <div class="flex flex-wrap gap-2">
              <Button
                v-for="days in [7, 30, 90]"
                :key="days"
                type="button"
                :variant="isValidityPreset(days) ? 'default' : 'secondary'"
                @click="setValidityDays(days)"
              >
                +{{ days }} 天
              </Button>
              <Button type="button" variant="secondary" @click="openCustomValidity">自定义</Button>
            </div>
            <div class="space-y-1.5">
              <Label for="api-key-expire-at">过期时间</Label>
              <Input
                id="api-key-expire-at"
                v-model="tokenValidityForm.expireAt"
                type="datetime-local"
              />
              <p class="text-xs text-muted-foreground">选择此 API 密钥的过期时间。</p>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="loading" @click="closeEditor">取消</Button>
        <Button :disabled="loading" @click="saveToken">{{
          tokenForm.id ? '保存修改' : '创建 API 密钥'
        }}</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
