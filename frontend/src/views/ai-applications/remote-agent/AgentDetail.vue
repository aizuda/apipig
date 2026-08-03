<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Activity, ArrowLeft, ListChecks, Plus, Server, TerminalSquare } from '@lucide/vue'
import { Badge, Button, Card, CardContent, toast } from '@tabtab/ui'
import { useRoute, useRouter } from 'vue-router'
import {
  remoteAgentApi,
  type AgentDetailResult,
  type TaskCreateParams,
  type TaskDetailResult,
} from '@/api/ai-applications/remote-agent'
import AppPageHeader from '@/components/AppPageHeader.vue'
import RemoteAgentSectionNav from './components/RemoteAgentSectionNav.vue'
import TaskEditorDialog from './components/TaskEditorDialog.vue'
import TaskLogConsole from './components/TaskLogConsole.vue'
import {
  agentStatusLabel,
  agentStatusVariant,
  formatBytes,
  formatPercent,
  formatTime,
  taskStatusLabel,
  taskStatusVariant,
} from './presentation'

defineOptions({ name: 'RemoteAgentAgentDetail' })

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const creating = ref(false)
const editorOpen = ref(false)
const detail = ref<AgentDetailResult | null>(null)
const currentTask = ref<TaskDetailResult | null>(null)

const agentId = computed(() => String(route.params.id || ''))
const agent = computed(() => detail.value?.agent)
const canCreateTask = computed(() => agent.value?.status === 'ONLINE')

async function loadDetail() {
  if (!agentId.value) return
  loading.value = true
  try {
    const result = await remoteAgentApi.getAgent(agentId.value)
    detail.value = result
    currentTask.value = null
    if (result.agent.currentTaskId) {
      currentTask.value = await remoteAgentApi.getTask(result.agent.currentTaskId)
    }
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载 Agent 详情失败')
  } finally {
    loading.value = false
  }
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

function memoryUsage() {
  if (!agent.value) return '-'
  if (!agent.value.memoryTotal) return formatBytes(agent.value.memoryUsed)
  const percent = (agent.value.memoryUsed / agent.value.memoryTotal) * 100
  return `${formatBytes(agent.value.memoryUsed)} / ${formatBytes(agent.value.memoryTotal)} (${formatPercent(percent)})`
}

onMounted(loadDetail)
</script>

<template>
  <div class="space-y-6">
    <AppPageHeader
      :title="agent?.name || 'Agent 详情'"
      :description="
        agent
          ? `${agent.hostname || '-'} · ${agent.ipAddress || '-'}`
          : '查看节点运行信息与任务历史。'
      "
      :loading="loading"
      @refresh="loadDetail"
    >
      <template #actions>
        <Button
          variant="outline"
          size="sm"
          class="gap-2"
          @click="router.push('/ai-applications/remote-agent/agents')"
        >
          <ArrowLeft class="h-4 w-4" />返回列表
        </Button>
        <Button v-if="canCreateTask" size="sm" class="gap-2" @click="editorOpen = true">
          <Plus class="h-4 w-4" />创建任务
        </Button>
      </template>
    </AppPageHeader>

    <RemoteAgentSectionNav />

    <template v-if="agent && detail">
      <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-md border p-4">
          <div class="flex items-center justify-between text-sm text-muted-foreground">
            节点状态<Server class="h-4 w-4" />
          </div>
          <Badge class="mt-3" :variant="agentStatusVariant(agent.status)">
            {{ agentStatusLabel(agent.status) }}
          </Badge>
          <div class="mt-2 text-xs text-muted-foreground">
            最后在线 {{ formatTime(agent.lastSeenAt) }}
          </div>
        </div>
        <div class="rounded-md border p-4">
          <div class="flex items-center justify-between text-sm text-muted-foreground">
            CPU 使用率<Activity class="h-4 w-4" />
          </div>
          <div class="mt-2 text-2xl font-semibold tabular-nums">
            {{ formatPercent(agent.cpuUsage) }}
          </div>
          <div class="mt-1 truncate text-xs text-muted-foreground">{{ agent.cpuInfo || '-' }}</div>
        </div>
        <div class="rounded-md border p-4">
          <div class="text-sm text-muted-foreground">内存占用</div>
          <div class="mt-2 text-lg font-semibold tabular-nums">{{ memoryUsage() }}</div>
          <div class="mt-1 text-xs text-muted-foreground">物理内存实时采样</div>
        </div>
        <div class="rounded-md border p-4">
          <div class="flex items-center justify-between text-sm text-muted-foreground">
            Codex CLI<TerminalSquare class="h-4 w-4" />
          </div>
          <div class="mt-2 text-lg font-semibold">{{ agent.codexVersion || '-' }}</div>
          <div class="mt-1 text-xs text-muted-foreground">
            Agent {{ agent.agentVersion || '-' }}
          </div>
        </div>
      </section>

      <Card class="gap-0 py-0">
        <CardContent class="p-0">
          <div class="border-b px-4 py-3">
            <h2 class="font-medium">基础信息</h2>
          </div>
          <dl class="grid gap-x-8 gap-y-4 p-4 text-sm sm:grid-cols-2 xl:grid-cols-4">
            <div>
              <dt class="text-xs text-muted-foreground">Agent ID</dt>
              <dd class="mt-1 break-all font-mono text-xs">{{ agent.id }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">节点标识</dt>
              <dd class="mt-1 break-all font-mono text-xs">{{ agent.agentKey }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">主机</dt>
              <dd class="mt-1">{{ agent.hostname || '-' }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">IP 地址</dt>
              <dd class="mt-1">{{ agent.ipAddress || '-' }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">操作系统</dt>
              <dd class="mt-1">{{ agent.operatingSystem || '-' }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">系统架构</dt>
              <dd class="mt-1">{{ agent.architecture || '-' }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">注册时间</dt>
              <dd class="mt-1">{{ formatTime(agent.createdAt) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">当前任务</dt>
              <dd class="mt-1 font-mono text-xs">{{ agent.currentTaskId || '-' }}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <section v-if="currentTask" class="space-y-3">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="font-medium">当前任务日志</h2>
            <p class="text-sm text-muted-foreground">{{ currentTask.task.name }}</p>
          </div>
          <Button
            variant="outline"
            size="sm"
            class="gap-2"
            @click="router.push(`/ai-applications/remote-agent/tasks/${currentTask.task.id}`)"
          >
            <ListChecks class="h-4 w-4" />任务详情
          </Button>
        </div>
        <TaskLogConsole
          :task-id="currentTask.task.id"
          :status="currentTask.task.status"
          :initial-logs="currentTask.logs"
          @done="loadDetail"
        />
      </section>

      <div class="grid gap-4 xl:grid-cols-2">
        <Card class="gap-0 overflow-hidden py-0">
          <CardContent class="p-0">
            <div class="border-b px-4 py-3">
              <h2 class="font-medium">心跳历史</h2>
              <p class="mt-0.5 text-xs text-muted-foreground">
                最近 {{ detail.heartbeats.length }} 次采样
              </p>
            </div>
            <div class="max-h-[430px] overflow-auto">
              <table class="w-full min-w-[580px] text-sm">
                <thead class="sticky top-0 border-b bg-card text-left text-muted-foreground">
                  <tr>
                    <th class="px-4 py-3 font-medium">时间</th>
                    <th class="px-4 py-3 font-medium">状态</th>
                    <th class="px-4 py-3 font-medium">CPU</th>
                    <th class="px-4 py-3 font-medium">内存</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="heartbeat in detail.heartbeats"
                    :key="heartbeat.id"
                    class="border-b last:border-0"
                  >
                    <td class="whitespace-nowrap px-4 py-3 text-muted-foreground">
                      {{ formatTime(heartbeat.occurredAt) }}
                    </td>
                    <td class="px-4 py-3">
                      <Badge variant="outline">{{ agentStatusLabel(heartbeat.status) }}</Badge>
                    </td>
                    <td class="px-4 py-3 tabular-nums">{{ formatPercent(heartbeat.cpuUsage) }}</td>
                    <td class="px-4 py-3 tabular-nums">{{ formatBytes(heartbeat.memoryUsed) }}</td>
                  </tr>
                  <tr v-if="!detail.heartbeats.length">
                    <td colspan="4" class="px-4 py-12 text-center text-muted-foreground">
                      暂无心跳记录
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>

        <Card class="gap-0 overflow-hidden py-0">
          <CardContent class="p-0">
            <div class="border-b px-4 py-3">
              <h2 class="font-medium">最近任务</h2>
              <p class="mt-0.5 text-xs text-muted-foreground">
                最近 {{ detail.tasks.length }} 条执行记录
              </p>
            </div>
            <div class="max-h-[430px] overflow-auto">
              <table class="w-full min-w-[620px] text-sm">
                <thead class="sticky top-0 border-b bg-card text-left text-muted-foreground">
                  <tr>
                    <th class="px-4 py-3 font-medium">任务</th>
                    <th class="px-4 py-3 font-medium">状态</th>
                    <th class="px-4 py-3 font-medium">创建时间</th>
                    <th class="px-4 py-3 text-right font-medium">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="task in detail.tasks" :key="task.id" class="border-b last:border-0">
                    <td class="max-w-[240px] px-4 py-3">
                      <div class="truncate font-medium">{{ task.name }}</div>
                      <div class="mt-1 truncate text-xs text-muted-foreground">
                        {{ task.repositoryUrl }}
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <Badge :variant="taskStatusVariant(task.status)">{{
                        taskStatusLabel(task.status)
                      }}</Badge>
                    </td>
                    <td class="whitespace-nowrap px-4 py-3 text-muted-foreground">
                      {{ formatTime(task.createdAt) }}
                    </td>
                    <td class="px-4 py-3 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        @click="router.push(`/ai-applications/remote-agent/tasks/${task.id}`)"
                        >查看</Button
                      >
                    </td>
                  </tr>
                  <tr v-if="!detail.tasks.length">
                    <td colspan="4" class="px-4 py-12 text-center text-muted-foreground">
                      暂无任务记录
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>

    <div v-else-if="loading" class="py-20 text-center text-sm text-muted-foreground">
      正在加载 Agent 详情...
    </div>

    <TaskEditorDialog
      :open="editorOpen"
      :loading="creating"
      :agents="agent && canCreateTask ? [agent] : []"
      :initial-agent-id="agentId"
      @close="editorOpen = false"
      @submit="createTask"
    />
  </div>
</template>
