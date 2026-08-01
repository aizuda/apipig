<script setup lang="ts">
import { computed, toRef } from 'vue'
import { Eye, EyeOff, Plus, Trash2 } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent, Input, Label, Textarea } from '@tabtab/ui'
import type { Channel, ChannelAccount } from '@/api/ai-gateway'

type AccountModelMapping = { gatewayModel: string; providerModel: string }

const props = defineProps<{
  open: boolean
  loading: boolean
  errorMessage: string
  form: ChannelAccount
  channels: Channel[]
  apiKeyVisible: boolean
  modelMappings: AccountModelMapping[]
}>()

const emit = defineEmits<{
  close: []
  submit: []
  'update:apiKeyVisible': [value: boolean]
  channelChange: []
  addMapping: []
  removeMapping: [index: number]
}>()

const form = toRef(props, 'form')
const channels = toRef(props, 'channels')
const modelMappings = toRef(props, 'modelMappings')
const editorOpen = computed(() => props.open)
const editorErrorMessage = computed(() => props.errorMessage)
const apiKeyVisible = computed({
  get: () => props.apiKeyVisible,
  set: (value) => emit('update:apiKeyVisible', value),
})
const selectedChannel = computed(() =>
  channels.value.find((item) => item.id === form.value.channelId),
)
const providerModels = computed(() => selectedChannel.value?.providerModels || [])

function closeEditor() {
  emit('close')
}

function saveAccount() {
  emit('submit')
}

function handleChannelChange() {
  emit('channelChange')
}
function addMapping() {
  emit('addMapping')
}
function removeMapping(index: number) {
  emit('removeMapping', index)
}

function syncGatewayModel(mapping: AccountModelMapping) {
  mapping.gatewayModel = mapping.providerModel
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
      :title="form.id ? '编辑账户' : '新增账户'"
      description="维护渠道 API Key，并配置网关模型到上游模型的映射。"
      class="sm:max-w-4xl"
    >
      <div class="space-y-3">
        <div
          v-if="editorErrorMessage"
          class="sticky top-0 z-20 rounded-md border border-destructive/40 bg-background px-4 py-3 text-sm text-destructive shadow-sm"
        >
          {{ editorErrorMessage }}
        </div>
        <div class="space-y-1.5">
          <Label for="account-channel">关联渠道 <span class="text-destructive">*</span></Label>
          <select
            id="account-channel"
            v-model="form.channelId"
            class="h-9 w-full rounded-md border bg-background px-3 text-sm"
            @change="handleChannelChange"
          >
            <option value="">请选择渠道</option>
            <option v-for="channel in channels" :key="channel.id" :value="channel.id">
              {{ channel.providerName || '未知供应商' }} / {{ channel.name }}
            </option>
          </select>
          <p class="text-xs text-muted-foreground">账户必须关联渠道号池中的一个具体渠道。</p>
        </div>
        <div class="space-y-1.5">
          <Label for="account-name">账户名称 <span class="text-destructive">*</span></Label>
          <Input id="account-name" v-model="form.name" placeholder="例如：OpenAI 主账户" />
          <p class="text-xs text-muted-foreground">用于区分同一平台或渠道下的不同注册账户。</p>
        </div>
        <div class="space-y-1.5">
          <Label for="account-api-key">API Key <span class="text-destructive">*</span></Label>
          <div class="relative">
            <Input
              id="account-api-key"
              v-model="form.apiKey"
              :type="apiKeyVisible ? 'text' : 'password'"
              class="pr-10"
              autocomplete="new-password"
              :placeholder="form.id ? '可直接修改当前 API Key' : '请输入该账户的 API Key'"
            />
            <button
              type="button"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
              :aria-label="apiKeyVisible ? '隐藏 API Key' : '显示 API Key'"
              @click="apiKeyVisible = !apiKeyVisible"
            >
              <Eye v-if="!apiKeyVisible" class="h-4 w-4" />
              <EyeOff v-else class="h-4 w-4" />
            </button>
          </div>
          <p class="text-xs text-muted-foreground">
            API Key 将加密保存，并由关联渠道进行网关调用。
          </p>
        </div>
        <div class="space-y-1.5">
          <Label for="account-status">状态</Label>
          <select
            id="account-status"
            v-model.number="form.status"
            class="h-9 w-full rounded-md border bg-background px-3 text-sm"
          >
            <option :value="1">启用</option>
            <option :value="2">禁用</option>
          </select>
        </div>
        <div class="space-y-2 rounded-lg border p-3">
          <div class="flex items-center justify-between">
            <div>
              <Label>模型映射</Label>
              <p class="text-xs text-muted-foreground">
                网关模型面向调用方，上游模型由当前渠道供应商提供。
              </p>
            </div>
            <Button
              type="button"
              size="sm"
              variant="outline"
              :disabled="!form.channelId"
              @click="addMapping"
              ><Plus class="mr-1.5 size-4" />添加映射</Button
            >
          </div>
          <div
            v-if="!modelMappings.length"
            class="rounded-md bg-muted/40 p-4 text-center text-xs text-muted-foreground"
          >
            请至少添加一个模型映射。
          </div>
          <div
            v-for="(mapping, index) in modelMappings"
            :key="index"
            class="grid gap-2 rounded-md border p-2 sm:grid-cols-[1fr_1fr_auto]"
          >
            <Input v-model="mapping.gatewayModel" placeholder="网关模型名称" /><select
              v-model="mapping.providerModel"
              class="h-10 rounded-md border bg-background px-3 text-sm"
              @change="syncGatewayModel(mapping)"
            >
              <option value="">选择上游模型</option>
              <option v-for="model in providerModels" :key="model" :value="model">
                {{ model }}
              </option></select
            ><Button type="button" size="icon" variant="ghost" @click="removeMapping(index)"
              ><Trash2 class="size-4 text-destructive"
            /></Button>
          </div>
        </div>
        <div class="space-y-1.5">
          <Label for="account-remark">备注</Label>
          <Textarea
            id="account-remark"
            v-model="form.remark"
            placeholder="可记录注册区域、套餐或其他说明"
          />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="loading" @click="closeEditor">取消</Button>
        <Button :disabled="loading" @click="saveAccount">{{
          form.id ? '保存修改' : '创建账户'
        }}</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
