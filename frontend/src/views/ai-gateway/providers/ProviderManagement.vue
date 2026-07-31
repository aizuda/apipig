<script setup lang="ts">
import { reactive, ref } from 'vue'
import { Plus, Search, Trash2 } from '@lucide/vue'
import { Badge, Button, Card, CardContent, Input } from '@tabtab/ui'
import { aiGatewayApi, type Provider } from '@/api/ai-gateway'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import GatewayResourceConfirmDialogs from '../components/GatewayResourceConfirmDialogs.vue'
import { useGatewayResourcePage } from '../composables/useGatewayResourcePage'
import ProviderEditorDialog from './ProviderEditorDialog.vue'

defineOptions({ name: 'AiGatewayProviders' })

const providerModelTags = ref<string[]>([])
const providerModelInput = ref('')
const providerForm = reactive<Provider>({
  name: '',
  code: '',
  protocol: 'openai',
  baseUrl: '',
  models: '',
  timeoutMs: 60000,
  status: 1,
  remark: '',
})

const {
  loading,
  errorMessage,
  editorErrorMessage,
  editorOpen,
  searchQuery,
  records: providers,
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
} = useGatewayResourcePage<Provider>({
  fetchPage: aiGatewayApi.providerPage,
  deleteRecord: aiGatewayApi.deleteProvider,
  changeStatus: aiGatewayApi.changeProviderStatus,
})

function splitModels(value?: string) {
  return (value || '')
    .split(',')
    .map((model) => model.trim())
    .filter(Boolean)
}

function resetProviderForm() {
  providerModelTags.value = []
  providerModelInput.value = ''
  Object.assign(providerForm, {
    id: undefined,
    name: '',
    code: '',
    protocol: 'openai',
    baseUrl: '',
    models: '',
    timeoutMs: 60000,
    status: 1,
    remark: '',
  })
}

function commitProviderModelInput() {
  const models = splitModels(providerModelInput.value)
  for (const modelName of models) {
    if (!providerModelTags.value.includes(modelName)) providerModelTags.value.push(modelName)
  }
  providerModelInput.value = ''
}

function handleProviderModelKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== ',') return
  event.preventDefault()
  commitProviderModelInput()
}

function removeProviderModel(index: number) {
  providerModelTags.value.splice(index, 1)
}

async function saveProvider() {
  commitProviderModelInput()
  providerForm.models = providerModelTags.value.join(',')
  await run(
    async () => {
      await aiGatewayApi.saveProvider({ ...providerForm })
      editorOpen.value = false
      resetProviderForm()
      await loadData()
    },
    providerForm.id ? '供应商配置已更新' : '供应商已创建',
  )
}

function editProvider(item: Provider) {
  editorErrorMessage.value = ''
  Object.assign(providerForm, item)
  providerModelTags.value = splitModels(item.models)
  providerModelInput.value = ''
  editorOpen.value = true
}

function openCreateEditor() {
  editorErrorMessage.value = ''
  resetProviderForm()
  editorOpen.value = true
}

function requestDelete(_kind: 'provider', id: string | undefined, name: string) {
  requestResourceDelete(id, name)
}

function requestStatusChange(_kind: 'provider', item: Provider) {
  requestResourceStatusChange(item)
}
</script>

<template>
  <div class="-mt-4 space-y-4">
    <AppPageHeader
      title="LLM 供应商"
      description="LLM 供应商独立管理。"
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
        <Input v-model="searchQuery" class="pl-9" placeholder="搜索供应商名称、模型或标识" />
      </div>
      <div
        class="flex w-full flex-col gap-2 text-sm text-muted-foreground sm:w-auto sm:flex-row sm:items-center"
      >
        <span>{{ total }} 个供应商</span>
        <Button size="sm" class="w-full sm:w-auto" @click="openCreateEditor">
          <Plus class="mr-1.5 h-4 w-4" />
          新增 LLM 供应商
        </Button>
      </div>
    </div>

    <section>
      <ProviderEditorDialog
        :open="editorOpen"
        :loading="loading"
        :error-message="editorErrorMessage"
        :form="providerForm"
        :model-tags="providerModelTags"
        v-model:model-input="providerModelInput"
        @close="closeEditor"
        @submit="saveProvider"
        @commit-models="commitProviderModelInput"
        @model-keydown="handleProviderModelKeydown"
        @remove-model="removeProviderModel"
      />
    </section>

    <section>
      <Card class="gap-0 py-0">
        <CardContent class="p-0">
          <div class="mobile-table-scroll" aria-label="供应商数据表格，可横向滚动">
            <table class="w-full text-sm">
              <thead class="border-b bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th class="px-4 py-3">名称</th>
                  <th class="px-4 py-3">编码</th>
                  <th class="px-4 py-3">协议</th>
                  <th class="px-4 py-3">模型</th>
                  <th class="px-4 py-3">状态</th>
                  <th class="px-4 py-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in providers"
                  :key="item.id"
                  class="border-b last:border-0 hover:bg-muted/30"
                >
                  <td class="px-4 py-3 font-medium">{{ item.name }}</td>
                  <td class="px-4 py-3">{{ item.code }}</td>
                  <td class="px-4 py-3">{{ item.protocol }}</td>
                  <td class="max-w-[320px] px-4 py-3">
                    <div v-if="splitModels(item.models).length" class="flex flex-wrap gap-1.5">
                      <Badge
                        v-for="(model, index) in splitModels(item.models)"
                        :key="[item.id, model, index].join('-')"
                        variant="secondary"
                        class="font-normal"
                      >
                        {{ model }}
                      </Badge>
                    </div>
                    <span v-else>-</span>
                  </td>
                  <td class="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      class="group h-auto rounded-full p-0"
                      :disabled="loading || Boolean(statusChangingKey)"
                      :aria-label="`${item.name}当前${statusLabel(item.status)}，点击切换为${statusLabel(nextStatus(item.status))}`"
                      @click="requestStatusChange('provider', item)"
                    >
                      <Badge
                        :variant="statusVariant(item.status)"
                        class="transition-opacity group-hover:opacity-80"
                        >{{ statusLabel(item.status) }}</Badge
                      >
                    </Button>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <Button variant="ghost" size="sm" @click="editProvider(item)">编辑</Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      @click="requestDelete('provider', item.id, item.name)"
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
