<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Eye, ListChecks, Plus, Search } from '@lucide/vue'
import { Badge, Button, Card, CardContent, Input, toast } from '@tabtab/ui'
import { useRouter } from 'vue-router'
import {
  remoteAgentApi,
  type RemoteAgent,
  type RemoteTask,
  type TaskCreateParams,
  type TaskStatus,
} from '@/api/ai-applications/remote-agent'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import RemoteAgentSectionNav from './components/RemoteAgentSectionNav.vue'
import TaskEditorDialog from './components/TaskEditorDialog.vue'
import { formatTime, taskStatusLabel, taskStatusVariant } from './presentation'

defineOptions({ name: 'RemoteAgentTasks' })

const router = useRouter()
const loading = ref(false)
const creating = ref(false)
const tasks = ref<RemoteTask[]>([])
const agents = ref<RemoteAgent[]>([])
const availableAgents = ref<RemoteAgent[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const agentId = ref('')
const status = ref<TaskStatus | ''>('')
const editorOpen = ref(false)

async function loadTasks() {
  loading.value = true
  try {
    const result = await remoteAgentApi.taskPage({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      agentId: agentId.value || undefined,
      status: status.value,
    })
    tasks.value = result.records || []
    total.value = result.total || 0
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载远程任务失败')
  } finally {
    loading.value = false
  }
}

async function loadAgents() {
  try {
    const result = await remoteAgentApi.agentPage({ page: 1, pageSize: 100 })
    agents.value = result.records || []
    availableAgents.value = agents.value.filter((agent) => agent.status === 'ONLINE')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载 Agent 节点失败')
  }
}

async function openCreate() {
  await loadAgents()
  if (!availableAgents.value.length) {
    toast.error('当前没有可接收任务的在线 Agent')
    return
  }
  editorOpen.value = true
}

async function createTask(params: TaskCreateParams) {
  creating.value = true
  try {
    const task = await remoteAgentApi.createTask(params)
    toast.success('远程任务已创建')
    editorOpen.value = false
    await router.push(`/ai-applications/remote-agent/tasks/${task.id}`)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '创建远程任务失败')
  } finally {
    creating.value = false
  }
}

function searchTasks() {
  page.value = 1
  void loadTasks()
}

function resetFilters() {
  keyword.value = ''
  agentId.value = ''
  status.value = ''
  searchTasks()
}

function changePage(value: number) {
  page.value = value
  void loadTasks()
}

function changePageSize(value: number) {
  pageSize.value = value
  page.value = 1
  void loadTasks()
}

function agentName(id: string) {
  const agent = agents.value.find((item) => item.id === id)
  return agent ? agent.name : id
}

onMounted(() => Promise.all([loadTasks(), loadAgents()]))
</script>

<template>
  <div class="space-y-6">
    <AppPageHeader
      title="远程任务"
      description="创建远程 Coding 任务并跟踪分发、执行和完成状态。"
      :loading="loading"
      @refresh="loadTasks"
    >
      <template #actions>
        <Button size="sm" class="gap-2" @click="openCreate">
          <Plus class="h-4 w-4" />创建任务
        </Button>
      </template>
    </AppPageHeader>

    <RemoteAgentSectionNav />

    <Card class="gap-0 overflow-hidden py-0">
      <CardContent class="p-0">
        <form
          class="grid gap-3 border-b p-4 md:grid-cols-2 xl:grid-cols-[minmax(260px,1fr)_200px_160px_auto]"
          @submit.prevent="searchTasks"
        >
          <div class="relative">
            <Search
              class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            />
            <Input v-model="keyword" class="pl-9" placeholder="搜索任务名称或仓库地址" />
          </div>
          <select v-model="agentId" class="h-10 rounded-md border bg-background px-3 text-sm">
            <option value="">全部 Agent</option>
            <option v-for="agent in agents" :key="agent.id" :value="agent.id">
              {{ agent.name }} · {{ agent.hostname }}
            </option>
          </select>
          <select v-model="status" class="h-10 rounded-md border bg-background px-3 text-sm">
            <option value="">全部状态</option>
            <option value="PENDING">等待执行</option>
            <option value="RUNNING">执行中</option>
            <option value="SUCCESS">成功</option>
            <option value="FAILED">失败</option>
            <option value="CANCELLED">已取消</option>
          </select>
          <div class="flex gap-2 md:col-span-2 xl:col-span-1">
            <Button type="submit" variant="outline" :disabled="loading">查询</Button>
            <Button type="button" variant="ghost" :disabled="loading" @click="resetFilters">
              重置
            </Button>
          </div>
        </form>

        <div v-if="tasks.length" class="mobile-table-scroll" aria-label="远程任务表格">
          <table class="w-full min-w-[1100px] text-sm">
            <thead class="border-b bg-muted/20 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3 font-medium">任务</th>
                <th class="px-4 py-3 font-medium">Agent</th>
                <th class="px-4 py-3 font-medium">仓库 / 目录</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 font-medium">创建时间</th>
                <th class="px-4 py-3 font-medium">开始 / 完成</th>
                <th class="px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="task in tasks"
                :key="task.id"
                class="border-b last:border-0 hover:bg-muted/20"
              >
                <td class="max-w-[280px] px-4 py-4">
                  <button
                    type="button"
                    class="block max-w-full truncate text-left font-medium hover:underline"
                    @click="router.push(`/ai-applications/remote-agent/tasks/${task.id}`)"
                  >
                    {{ task.name }}
                  </button>
                  <div class="mt-1 truncate font-mono text-xs text-muted-foreground">
                    {{ task.id }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <Button
                    variant="link"
                    size="sm"
                    class="h-auto px-0"
                    @click="router.push(`/ai-applications/remote-agent/agents/${task.agentId}`)"
                  >
                    {{ agentName(task.agentId) }}
                  </Button>
                </td>
                <td class="max-w-[320px] px-4 py-4">
                  <a
                    :href="task.repositoryUrl"
                    target="_blank"
                    rel="noreferrer"
                    class="block truncate hover:underline"
                  >
                    {{ task.repositoryUrl }}
                  </a>
                  <div class="mt-1 text-xs text-muted-foreground">
                    {{ task.workingDir || '仓库根目录' }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <Badge :variant="taskStatusVariant(task.status)">{{
                    taskStatusLabel(task.status)
                  }}</Badge>
                </td>
                <td class="whitespace-nowrap px-4 py-4 text-muted-foreground">
                  {{ formatTime(task.createdAt) }}
                </td>
                <td class="whitespace-nowrap px-4 py-4 text-xs text-muted-foreground">
                  <div>{{ formatTime(task.startedAt) }}</div>
                  <div class="mt-1">{{ formatTime(task.finishedAt) }}</div>
                </td>
                <td class="px-4 py-4 text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    class="gap-1.5"
                    @click="router.push(`/ai-applications/remote-agent/tasks/${task.id}`)"
                  >
                    <Eye class="h-4 w-4" />查看
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else-if="!loading" class="px-6 py-16 text-center">
          <ListChecks class="mx-auto h-10 w-10 text-muted-foreground/50" />
          <h3 class="mt-4 font-medium">暂无远程任务</h3>
          <p class="mt-1 text-sm text-muted-foreground">在线 Agent 可用后即可创建第一个任务。</p>
          <Button class="mt-4" @click="openCreate">创建任务</Button>
        </div>
        <div v-else class="px-6 py-16 text-center text-sm text-muted-foreground">
          正在加载远程任务...
        </div>

        <AppPagination
          :total="total"
          :page="page"
          :page-size="pageSize"
          :loading="loading"
          @change-page="changePage"
          @change-page-size="changePageSize"
        />
      </CardContent>
    </Card>

    <TaskEditorDialog
      :open="editorOpen"
      :loading="creating"
      :agents="availableAgents"
      @close="editorOpen = false"
      @submit="createTask"
    />
  </div>
</template>
