<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Eye, MonitorCog, Plus, Search } from '@lucide/vue'
import { Badge, Button, Card, CardContent, Input, toast } from '@tabtab/ui'
import {
  remoteAgentApi,
  type AgentStatus,
  type RemoteAgent,
  type TaskCreateParams,
} from '@/api/ai-applications/remote-agent'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import RemoteAgentSectionNav from './components/RemoteAgentSectionNav.vue'
import TaskEditorDialog from './components/TaskEditorDialog.vue'
import {
  agentStatusLabel,
  agentStatusVariant,
  formatBytes,
  formatPercent,
  formatTime,
} from './presentation'
import { useRouter } from 'vue-router'

defineOptions({ name: 'RemoteAgentAgents' })

const router = useRouter()
const loading = ref(false)
const creating = ref(false)
const agents = ref<RemoteAgent[]>([])
const availableAgents = ref<RemoteAgent[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const status = ref<AgentStatus | ''>('')
const editorOpen = ref(false)
const initialAgentId = ref('')

async function loadAgents() {
  loading.value = true
  try {
    const result = await remoteAgentApi.agentPage({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      status: status.value,
    })
    agents.value = result.records || []
    total.value = result.total || 0
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载 Agent 节点失败')
  } finally {
    loading.value = false
  }
}

async function loadAvailableAgents() {
  try {
    const result = await remoteAgentApi.agentPage({ page: 1, pageSize: 100, status: 'ONLINE' })
    availableAgents.value = result.records || []
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载在线 Agent 失败')
  }
}

async function openCreate(agentId = '') {
  initialAgentId.value = agentId
  await loadAvailableAgents()
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

function searchAgents() {
  page.value = 1
  void loadAgents()
}

function resetFilters() {
  keyword.value = ''
  status.value = ''
  searchAgents()
}

function changePage(value: number) {
  page.value = value
  void loadAgents()
}

function changePageSize(value: number) {
  pageSize.value = value
  page.value = 1
  void loadAgents()
}

function memoryLabel(agent: RemoteAgent) {
  if (!agent.memoryTotal) return formatBytes(agent.memoryUsed)
  const percent = (agent.memoryUsed / agent.memoryTotal) * 100
  return `${formatBytes(agent.memoryUsed)} / ${formatBytes(agent.memoryTotal)} (${formatPercent(percent)})`
}

onMounted(loadAgents)
</script>

<template>
  <div class="space-y-6">
    <AppPageHeader
      title="Agent 节点"
      description="查看远程工作节点的在线状态、运行负载和当前任务。"
      :loading="loading"
      @refresh="loadAgents"
    >
      <template #actions>
        <Button size="sm" class="gap-2" @click="openCreate()">
          <Plus class="h-4 w-4" />创建任务
        </Button>
      </template>
    </AppPageHeader>

    <RemoteAgentSectionNav />

    <Card class="gap-0 overflow-hidden py-0">
      <CardContent class="p-0">
        <form
          class="flex flex-col gap-3 border-b p-4 lg:flex-row lg:items-center"
          @submit.prevent="searchAgents"
        >
          <div class="relative min-w-0 flex-1">
            <Search
              class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            />
            <Input v-model="keyword" class="pl-9" placeholder="搜索节点名称、主机名或 IP" />
          </div>
          <select
            v-model="status"
            class="h-10 rounded-md border bg-background px-3 text-sm lg:w-40"
          >
            <option value="">全部状态</option>
            <option value="ONLINE">在线</option>
            <option value="BUSY">执行中</option>
            <option value="OFFLINE">离线</option>
            <option value="DISABLED">已禁用</option>
          </select>
          <div class="flex gap-2">
            <Button type="submit" variant="outline" :disabled="loading">查询</Button>
            <Button type="button" variant="ghost" :disabled="loading" @click="resetFilters">
              重置
            </Button>
          </div>
        </form>

        <div v-if="agents.length" class="mobile-table-scroll" aria-label="Agent 节点表格">
          <table class="w-full min-w-[1080px] text-sm">
            <thead class="border-b bg-muted/20 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3 font-medium">节点</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 font-medium">CPU</th>
                <th class="px-4 py-3 font-medium">内存</th>
                <th class="px-4 py-3 font-medium">Codex / Agent</th>
                <th class="px-4 py-3 font-medium">当前任务</th>
                <th class="px-4 py-3 font-medium">最后在线</th>
                <th class="px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="agent in agents"
                :key="agent.id"
                class="border-b last:border-0 hover:bg-muted/20"
              >
                <td class="px-4 py-4">
                  <button
                    type="button"
                    class="font-medium hover:underline"
                    @click="router.push(`/ai-applications/remote-agent/agents/${agent.id}`)"
                  >
                    {{ agent.name }}
                  </button>
                  <div class="mt-1 text-xs text-muted-foreground">
                    {{ agent.hostname || '-' }} · {{ agent.ipAddress || '-' }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <Badge :variant="agentStatusVariant(agent.status)">
                    {{ agentStatusLabel(agent.status) }}
                  </Badge>
                </td>
                <td class="max-w-[220px] px-4 py-4">
                  <div class="tabular-nums">{{ formatPercent(agent.cpuUsage) }}</div>
                  <div class="mt-1 truncate text-xs text-muted-foreground">
                    {{ agent.cpuInfo || '-' }}
                  </div>
                </td>
                <td class="whitespace-nowrap px-4 py-4 tabular-nums">
                  {{ memoryLabel(agent) }}
                </td>
                <td class="px-4 py-4">
                  <div>{{ agent.codexVersion || '-' }}</div>
                  <div class="mt-1 text-xs text-muted-foreground">
                    Agent {{ agent.agentVersion || '-' }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <Button
                    v-if="agent.currentTaskId"
                    variant="link"
                    size="sm"
                    class="h-auto px-0 font-mono text-xs"
                    @click="
                      router.push(`/ai-applications/remote-agent/tasks/${agent.currentTaskId}`)
                    "
                  >
                    {{ agent.currentTaskId }}
                  </Button>
                  <span v-else class="text-muted-foreground">-</span>
                </td>
                <td class="whitespace-nowrap px-4 py-4 text-muted-foreground">
                  {{ formatTime(agent.lastSeenAt) }}
                </td>
                <td class="px-4 py-4">
                  <div class="flex justify-end gap-1">
                    <Button
                      v-if="agent.status === 'ONLINE'"
                      variant="ghost"
                      size="icon"
                      title="在此节点创建任务"
                      @click="openCreate(agent.id)"
                    >
                      <Plus class="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      title="查看节点"
                      @click="router.push(`/ai-applications/remote-agent/agents/${agent.id}`)"
                    >
                      <Eye class="h-4 w-4" />
                    </Button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else-if="!loading" class="px-6 py-16 text-center">
          <MonitorCog class="mx-auto h-10 w-10 text-muted-foreground/50" />
          <h3 class="mt-4 font-medium">暂无 Agent 节点</h3>
          <p class="mt-1 text-sm text-muted-foreground">Agent 注册并开始心跳后会显示在这里。</p>
        </div>
        <div v-else class="px-6 py-16 text-center text-sm text-muted-foreground">
          正在加载 Agent 节点...
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
      :initial-agent-id="initialAgentId"
      @close="editorOpen = false"
      @submit="createTask"
    />
  </div>
</template>
