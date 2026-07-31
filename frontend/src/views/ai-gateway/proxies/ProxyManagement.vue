<script setup lang="ts">
import { reactive } from 'vue'
import { Plus, Search, Trash2 } from '@lucide/vue'
import { Badge, Button, Card, CardContent, Input } from '@tabtab/ui'
import { aiGatewayApi, type ProxyNode } from '@/api/ai-gateway'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import GatewayResourceConfirmDialogs from '../components/GatewayResourceConfirmDialogs.vue'
import { useGatewayResourcePage } from '../composables/useGatewayResourcePage'
import ProxyEditorDialog from './ProxyEditorDialog.vue'

defineOptions({ name: 'AiGatewayProxies' })

const proxyForm = reactive<ProxyNode>({
  name: '',
  scheme: 'http',
  host: '',
  port: 8080,
  username: '',
  password: '',
  region: '',
  status: 1,
  remark: '',
})

const {
  loading,
  errorMessage,
  editorErrorMessage,
  editorOpen,
  searchQuery,
  records: proxies,
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
} = useGatewayResourcePage<ProxyNode>({
  fetchPage: aiGatewayApi.proxyPage,
  deleteRecord: aiGatewayApi.deleteProxy,
  changeStatus: aiGatewayApi.changeProxyStatus,
})

function resetProxyForm() {
  Object.assign(proxyForm, {
    id: undefined,
    name: '',
    scheme: 'http',
    host: '',
    port: 8080,
    username: '',
    password: '',
    region: '',
    status: 1,
    remark: '',
  })
}

async function saveProxy() {
  await run(
    async () => {
      await aiGatewayApi.saveProxy({ ...proxyForm })
      editorOpen.value = false
      resetProxyForm()
      await loadData()
    },
    proxyForm.id ? '代理配置已更新' : '代理已创建',
  )
}

function editProxy(item: ProxyNode) {
  editorErrorMessage.value = ''
  Object.assign(proxyForm, item)
  editorOpen.value = true
}

function openCreateEditor() {
  editorErrorMessage.value = ''
  resetProxyForm()
  editorOpen.value = true
}

function requestDelete(_kind: 'proxy', id: string | undefined, name: string) {
  requestResourceDelete(id, name)
}

function requestStatusChange(_kind: 'proxy', item: ProxyNode) {
  requestResourceStatusChange(item)
}
</script>

<template>
  <div class="-mt-4 space-y-4">
    <AppPageHeader
      title="IP 代理池"
      description="IP 代理池独立管理。"
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
        <Input v-model="searchQuery" class="pl-9" placeholder="搜索代理名称、地址或地区" />
      </div>
      <div
        class="flex w-full flex-col gap-2 text-sm text-muted-foreground sm:w-auto sm:flex-row sm:items-center"
      >
        <span>{{ total }} 个代理</span>
        <Button size="sm" class="w-full sm:w-auto" @click="openCreateEditor">
          <Plus class="mr-1.5 h-4 w-4" />
          新增 IP 代理池
        </Button>
      </div>
    </div>

    <section>
      <ProxyEditorDialog
        :open="editorOpen"
        :loading="loading"
        :error-message="editorErrorMessage"
        :form="proxyForm"
        @close="closeEditor"
        @submit="saveProxy"
      />

      <Card class="gap-0 py-0">
        <CardContent class="p-0">
          <div class="mobile-table-scroll" aria-label="路由策略数据表格，可横向滚动">
            <table class="w-full text-sm">
              <thead class="border-b bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th class="px-4 py-3">名称</th>
                  <th class="px-4 py-3">地址</th>
                  <th class="px-4 py-3">区域</th>
                  <th class="px-4 py-3">状态</th>
                  <th class="px-4 py-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in proxies"
                  :key="item.id"
                  class="border-b last:border-0 hover:bg-muted/30"
                >
                  <td class="px-4 py-3 font-medium">{{ item.name }}</td>
                  <td class="px-4 py-3">{{ item.scheme }}://{{ item.host }}:{{ item.port }}</td>
                  <td class="px-4 py-3">{{ item.region || '-' }}</td>
                  <td class="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      class="group h-auto rounded-full p-0"
                      :disabled="loading || Boolean(statusChangingKey)"
                      :aria-label="`${item.name}当前${statusLabel(item.status)}，点击切换为${statusLabel(nextStatus(item.status))}`"
                      @click="requestStatusChange('proxy', item)"
                    >
                      <Badge
                        :variant="statusVariant(item.status)"
                        class="transition-opacity group-hover:opacity-80"
                        >{{ statusLabel(item.status) }}</Badge
                      >
                    </Button>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <Button variant="ghost" size="sm" @click="editProxy(item)">编辑</Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      @click="requestDelete('proxy', item.id, item.name)"
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
