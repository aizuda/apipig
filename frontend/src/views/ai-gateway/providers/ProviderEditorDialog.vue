<script setup lang="ts">
import { computed, toRef } from 'vue'
import { X } from '@lucide/vue'
import { Badge, Button, Dialog, DialogFixedContent, Input, Label, Textarea } from '@tabtab/ui'
import type { Provider } from '@/api/ai-gateway'
import ProviderIconPicker from './ProviderIconPicker.vue'

const props = defineProps<{
  open: boolean
  loading: boolean
  errorMessage: string
  form: Provider
  modelTags: string[]
  modelInput: string
}>()

const emit = defineEmits<{
  close: []
  submit: []
  commitModels: []
  modelKeydown: [event: KeyboardEvent]
  removeModel: [index: number]
  'update:modelInput': [value: string]
}>()

const providerForm = toRef(props, 'form')
const providerModelTags = toRef(props, 'modelTags')
const providerModelInput = computed({
  get: () => props.modelInput,
  set: (value) => emit('update:modelInput', value),
})
const editorOpen = computed(() => props.open)
const editorErrorMessage = computed(() => props.errorMessage)

function closeEditor() {
  emit('close')
}

function saveProvider() {
  emit('submit')
}

function commitProviderModelInput() {
  emit('commitModels')
}

function handleProviderModelKeydown(event: KeyboardEvent) {
  emit('modelKeydown', event)
}

function removeProviderModel(index: number) {
  emit('removeModel', index)
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
      :title="providerForm.id ? '编辑供应商' : '新增供应商'"
      description="配置协议接入、基础地址与支持模型。"
      class="sm:max-w-2xl"
    >
      <div class="space-y-3">
        <div
          v-if="editorErrorMessage"
          class="sticky top-0 z-20 rounded-md border border-destructive/40 bg-background px-4 py-3 text-sm text-destructive shadow-sm"
        >
          {{ editorErrorMessage }}
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="provider-name">供应商名称 <span class="text-destructive">*</span></Label>
            <Input id="provider-name" v-model="providerForm.name" placeholder="例如 OpenAI" />
            <p class="text-xs text-muted-foreground">用于管理界面和渠道选择中展示。</p>
          </div>
          <div class="space-y-1.5">
            <Label for="provider-code">供应商编码 <span class="text-destructive">*</span></Label>
            <Input id="provider-code" v-model="providerForm.code" placeholder="例如 openai" />
            <p class="text-xs text-muted-foreground">系统内部唯一标识，建议使用小写英文。</p>
          </div>
        </div>
        <div class="space-y-1.5">
          <Label>供应商图标</Label>
          <ProviderIconPicker
            v-model="providerForm.icon"
            :fallback="providerForm.code"
            :secondary-fallback="providerForm.protocol"
            :disabled="loading"
          />
        </div>
        <div class="space-y-1.5">
          <Label for="provider-protocol">上游协议 <span class="text-destructive">*</span></Label>
          <select
            id="provider-protocol"
            v-model="providerForm.protocol"
            class="h-10 w-full rounded-md border bg-background px-3 text-sm"
          >
            <option value="openai">OpenAI</option>
            <option value="anthropic">Anthropic</option>
            <option value="codex">Codex</option>
            <option value="grok">Grok / xAI</option>
            <option value="gemini">Gemini</option>
            <option value="qwen">Qwen / DashScope</option>
            <option value="custom">OpenAI 兼容协议</option>
          </select>
          <p class="text-xs text-muted-foreground">决定请求认证方式和协议转换实现。</p>
        </div>
        <div class="space-y-1.5">
          <Label for="provider-base-url"
            >API 基础地址 <span class="text-destructive">*</span></Label
          >
          <Input
            id="provider-base-url"
            v-model="providerForm.baseUrl"
            placeholder="例如 https://api.openai.com/v1"
          />
          <p class="text-xs text-muted-foreground">
            填写供应商 API 根地址，不要包含用户名、密码或具体模型路径。
          </p>
        </div>
        <div class="space-y-1.5">
          <Label for="provider-models">支持模型</Label>
          <div v-if="providerModelTags.length" class="flex flex-wrap gap-1.5 rounded-md border p-2">
            <Badge
              v-for="(modelName, index) in providerModelTags"
              :key="[modelName, index].join('-')"
              variant="secondary"
              class="gap-1 font-normal"
            >
              {{ modelName }}
              <button
                type="button"
                class="-mr-1 inline-flex size-4 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted-foreground/15 hover:text-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                :aria-label="`删除模型 ${modelName}`"
                @click="removeProviderModel(index)"
              >
                <X class="size-3" />
              </button>
            </Badge>
          </div>
          <Input
            id="provider-models"
            v-model="providerModelInput"
            placeholder="输入模型后按 Enter 或英文逗号添加"
            @keydown="handleProviderModelKeydown"
            @blur="commitProviderModelInput"
          />
          <p class="text-xs text-muted-foreground">
            支持逐个添加或粘贴英文逗号分隔模型；保存时转换为英文逗号分隔数据。
          </p>
        </div>
        <div class="space-y-1.5">
          <Label for="provider-timeout">请求超时（毫秒）</Label
          ><Input
            id="provider-timeout"
            v-model.number="providerForm.timeoutMs"
            type="number"
            min="1000"
          />
          <p class="text-xs text-muted-foreground">允许范围 1000–600000，默认 60000。</p>
        </div>
        <div class="space-y-1.5">
          <Label for="provider-status">启用状态</Label
          ><select
            id="provider-status"
            v-model.number="providerForm.status"
            class="h-10 w-full rounded-md border bg-background px-3 text-sm"
          >
            <option :value="1">启用</option>
            <option :value="2">禁用</option>
          </select>
          <p class="text-xs text-muted-foreground">禁用后，该供应商下所有渠道不参与路由。</p>
        </div>
        <div class="space-y-1.5">
          <Label for="provider-remark">备注</Label
          ><Textarea
            id="provider-remark"
            v-model="providerForm.remark"
            placeholder="记录用途、账号归属或维护说明"
          />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="loading" @click="closeEditor">取消</Button>
        <Button :disabled="loading" @click="saveProvider">{{
          providerForm.id ? '保存修改' : '创建供应商'
        }}</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
