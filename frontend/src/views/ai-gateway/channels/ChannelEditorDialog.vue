<script setup lang="ts">
import { computed, toRef } from 'vue'
import { Plus, Trash2 } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent, Input, Label } from '@tabtab/ui'
import type { Channel, ModelPricingRule, Provider, ProxyNode } from '@/api/ai-gateway'

type PricingNumberField = Exclude<keyof ModelPricingRule, 'model' | 'tokenPriceUnit'>
const pricingFields: Array<{ key: PricingNumberField; label: string }> = [
  { key: 'inputPricePerMTokens', label: '输入 Token' },
  { key: 'cacheReadPricePerMTokens', label: '缓存读' },
  { key: 'cacheWritePricePerMTokens', label: '缓存写' },
  { key: 'outputPricePerMTokens', label: '输出 Token' },
  { key: 'inputImagePricePerMTokens', label: '图片输入 Token' },
  { key: 'outputImagePricePerMTokens', label: '图片输出 Token' },
  { key: 'inputImagePricePerImage', label: '输入图片/张' },
  { key: 'outputImagePricePerImage', label: '输出图片/张' },
]

const props = defineProps<{
  open: boolean
  loading: boolean
  errorMessage: string
  form: Channel
  providers: Provider[]
  proxies: ProxyNode[]
  availableModels: string[]
  pricingRules: ModelPricingRule[]
}>()

const emit = defineEmits<{
  close: []
  submit: []
  providerChange: []
  addPricing: []
  removePricing: [index: number]
}>()

const channelForm = toRef(props, 'form')
const providers = toRef(props, 'providers')
const proxies = toRef(props, 'proxies')
const availableChannelModels = toRef(props, 'availableModels')
const channelPricingRules = toRef(props, 'pricingRules')
const editorOpen = computed(() => props.open)
const editorErrorMessage = computed(() => props.errorMessage)

function closeEditor() {
  emit('close')
}

function saveChannel() {
  emit('submit')
}

function handleProviderChange() {
  emit('providerChange')
}

function addChannelPricingRule() {
  emit('addPricing')
}

function removeChannelPricingRule(index: number) {
  emit('removePricing', index)
}

function updatePricingRule(rule: ModelPricingRule, field: PricingNumberField, value: unknown) {
  rule[field] = Number(value) || 0
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
      :title="channelForm.id ? '编辑渠道' : '新增渠道'"
      description="配置路由优先级、限流和成本倍率；API Key 请在账户管理中维护。"
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
          <Label for="channel-provider">所属供应商 <span class="text-destructive">*</span></Label
          ><select
            id="channel-provider"
            v-model="channelForm.providerId"
            class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            @change="handleProviderChange"
          >
            <option value="">请选择供应商</option>
            <option v-for="provider in providers" :key="provider.id" :value="provider.id">
              {{ provider.name }}
            </option>
          </select>
          <p class="text-xs text-muted-foreground">决定该渠道使用的协议、Base URL 和可计价模型。</p>
        </div>
        <div class="space-y-1.5">
          <Label for="channel-name">渠道名称 <span class="text-destructive">*</span></Label
          ><Input id="channel-name" v-model="channelForm.name" placeholder="例如 OpenAI 主账号" />
          <p class="text-xs text-muted-foreground">用于区分同一供应商下的不同账号或线路。</p>
        </div>
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <div>
              <Label>模型计价</Label>
              <p class="text-xs text-muted-foreground">
                可按 USD / Token 或 USD / 1M Token 配置，图片单位 USD / 张。
              </p>
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              :disabled="!channelForm.providerId"
              @click="addChannelPricingRule"
              ><Plus class="mr-1.5 size-4" />添加模型</Button
            >
          </div>
          <div
            v-if="!channelForm.providerId"
            class="rounded-md border border-dashed p-4 text-sm text-muted-foreground"
          >
            请先选择供应商。
          </div>
          <div
            v-else-if="availableChannelModels.length === 0"
            class="rounded-md border border-dashed p-4 text-sm text-muted-foreground"
          >
            当前供应商未配置支持模型，请先在 LLM 供应商中添加模型标签。
          </div>
          <div v-else class="space-y-2">
            <div
              v-if="!channelPricingRules.length"
              class="rounded-md bg-muted/40 p-4 text-center text-xs text-muted-foreground"
            >
              未配置计价，调用只记录用量。
            </div>
            <div
              v-for="(rule, index) in channelPricingRules"
              :key="index"
              class="space-y-2 rounded-md border p-3"
            >
              <div class="flex gap-2">
                <select
                  v-model="rule.model"
                  class="h-9 min-w-0 flex-1 rounded-md border bg-background px-3 text-sm"
                >
                  <option value="">选择上游模型</option>
                  <option
                    v-for="modelName in availableChannelModels"
                    :key="modelName"
                    :value="modelName"
                  >
                    {{ modelName }}
                  </option>
                  <option value="*">* (默认规则)</option></select
                ><select
                  v-model="rule.tokenPriceUnit"
                  class="h-9 rounded-md border bg-background px-3 text-sm"
                >
                  <option value="perMillion">USD / 1M Token</option>
                  <option value="perToken">USD / Token</option></select
                ><Button variant="ghost" size="icon" @click="removeChannelPricingRule(index)"
                  ><Trash2 class="size-4 text-destructive"
                /></Button>
              </div>
              <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                <label
                  v-for="field in pricingFields"
                  :key="field.key"
                  class="space-y-1 text-xs text-muted-foreground"
                  ><span
                    >{{ field.label
                    }}{{
                      field.key.endsWith('PerMTokens')
                        ? rule.tokenPriceUnit === 'perToken'
                          ? ' (USD / Token)'
                          : ' (USD / 1M)'
                        : ''
                    }}</span
                  ><Input
                    :model-value="rule[field.key]"
                    type="number"
                    min="0"
                    step="0.000001"
                    class="h-8 text-foreground"
                    @update:model-value="updatePricingRule(rule, field.key, $event)"
                /></label>
              </div>
            </div>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="channel-priority">路由优先级</Label
            ><Input id="channel-priority" v-model.number="channelForm.priority" type="number" />
            <p class="text-xs text-muted-foreground">
              数值越大越优先，只在最高优先级组内进行加权选择。
            </p>
          </div>
          <div class="space-y-1.5">
            <Label for="channel-weight">同级路由权重</Label
            ><Input
              id="channel-weight"
              v-model.number="channelForm.weight"
              type="number"
              min="1"
              max="10000"
            />
            <p class="text-xs text-muted-foreground">同优先级渠道的流量分配比例，默认 1。</p>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="channel-rpm">RPM 请求上限</Label
            ><Input id="channel-rpm" v-model.number="channelForm.rpm" type="number" min="0" />
            <p class="text-xs text-muted-foreground">每分钟最大请求数，0 表示不限制。</p>
          </div>
          <div class="space-y-1.5">
            <Label for="channel-tpm">TPM Token 上限</Label
            ><Input id="channel-tpm" v-model.number="channelForm.tpm" type="number" min="0" />
            <p class="text-xs text-muted-foreground">每分钟最大 Token 数，0 表示不限制。</p>
          </div>
        </div>
        <div class="space-y-1.5">
          <Label for="channel-cost-multiplier">成本倍率</Label
          ><Input
            id="channel-cost-multiplier"
            v-model.number="channelForm.costMultiplier"
            type="number"
            min="0"
            max="1000"
            step="0.01"
          />
          <p class="text-xs text-muted-foreground">
            最终扣费 = 供应商标准成本 × 倍率；小于等于 0 时按 1 计算。
          </p>
        </div>
        <div class="space-y-1.5">
          <Label for="channel-proxy">出站代理</Label
          ><select
            id="channel-proxy"
            v-model="channelForm.proxyId"
            class="h-10 w-full rounded-md border bg-background px-3 text-sm"
          >
            <option value="">不使用代理，直接访问上游</option>
            <option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">
              {{ proxy.name }}
            </option>
          </select>
          <p class="text-xs text-muted-foreground">选择后，该渠道所有上游请求通过指定代理发送。</p>
        </div>
        <div class="space-y-1.5">
          <Label for="channel-status">启用状态</Label
          ><select
            id="channel-status"
            v-model.number="channelForm.status"
            class="h-10 w-full rounded-md border bg-background px-3 text-sm"
          >
            <option :value="1">启用渠道</option>
            <option :value="2">禁用渠道</option>
          </select>
          <p class="text-xs text-muted-foreground">禁用后渠道立即退出模型路由，不影响历史日志。</p>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="loading" @click="closeEditor">取消</Button>
        <Button :disabled="loading" @click="saveChannel">{{
          channelForm.id ? '保存修改' : '创建渠道'
        }}</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
