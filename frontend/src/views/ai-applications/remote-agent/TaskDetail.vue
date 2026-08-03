<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowLeft, Ban, FileCode2, GitBranch, ListChecks, Server } from '@lucide/vue'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  Card,
  CardContent,
  toast,
} from '@tabtab/ui'
import { useRoute, useRouter } from 'vue-router'
import { remoteAgentApi, type TaskDetailResult } from '@/api/ai-applications/remote-agent'
import AppPageHeader from '@/components/AppPageHeader.vue'
import RemoteAgentSectionNav from './components/RemoteAgentSectionNav.vue'
import TaskLogConsole from './components/TaskLogConsole.vue'
import { formatTime, parseChangedFiles, taskStatusLabel, taskStatusVariant } from './presentation'

defineOptions({ name: 'RemoteAgentTaskDetail' })

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const cancelling = ref(false)
const cancelOpen = ref(false)
const detail = ref<TaskDetailResult | null>(null)

const taskId = computed(() => String(route.params.id || ''))
const task = computed(() => detail.value?.task)
const changedFiles = computed(() => parseChangedFiles(task.value?.changedFiles))
const cancellable = computed(() => task.value && ['PENDING', 'RUNNING'].includes(task.value.status))

async function loadDetail() {
  if (!taskId.value) return
  loading.value = true
  try {
    detail.value = await remoteAgentApi.getTask(taskId.value)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载任务详情失败')
  } finally {
    loading.value = false
  }
}

async function cancelTask() {
  if (!task.value) return
  cancelling.value = true
  try {
    await remoteAgentApi.cancelTask(task.value.id)
    toast.success('任务已取消')
    cancelOpen.value = false
    await loadDetail()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '取消任务失败')
  } finally {
    cancelling.value = false
  }
}

function commandTypeLabel(value: string) {
  return value === 'EXECUTE_TASK' ? '执行任务' : value === 'STOP_TASK' ? '停止任务' : value
}

function commandStatusLabel(value: string) {
  const labels: Record<string, string> = {
    PENDING: '等待分发',
    DISPATCHED: '已分发',
    ACKNOWLEDGED: 'Agent 已确认',
    COMPLETED: '已完成',
    FAILED: '失败',
    CANCELLED: '已取消',
  }
  return labels[value] || value
}

onMounted(loadDetail)
</script>

<template>
  <div class="space-y-6">
    <AppPageHeader
      :title="task?.name || '任务详情'"
      description="查看任务参数、命令分发、实时输出和最终执行结果。"
      :loading="loading"
      @refresh="loadDetail"
    >
      <template #actions>
        <Button
          variant="outline"
          size="sm"
          class="gap-2"
          @click="router.push('/ai-applications/remote-agent/tasks')"
        >
          <ArrowLeft class="h-4 w-4" />返回列表
        </Button>
        <Button
          v-if="cancellable"
          variant="destructive"
          size="sm"
          class="gap-2"
          @click="cancelOpen = true"
        >
          <Ban class="h-4 w-4" />取消任务
        </Button>
      </template>
    </AppPageHeader>

    <RemoteAgentSectionNav />

    <template v-if="task && detail">
      <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-md border p-4">
          <div class="flex items-center justify-between text-sm text-muted-foreground">
            执行状态<ListChecks class="h-4 w-4" />
          </div>
          <Badge class="mt-3" :variant="taskStatusVariant(task.status)">{{
            taskStatusLabel(task.status)
          }}</Badge>
          <div class="mt-2 text-xs text-muted-foreground">
            创建于 {{ formatTime(task.createdAt) }}
          </div>
        </div>
        <div class="rounded-md border p-4">
          <div class="flex items-center justify-between text-sm text-muted-foreground">
            Agent 节点<Server class="h-4 w-4" />
          </div>
          <Button
            variant="link"
            class="mt-1 h-auto max-w-full px-0 font-mono text-xs"
            @click="router.push(`/ai-applications/remote-agent/agents/${task.agentId}`)"
          >
            {{ task.agentId }}
          </Button>
          <div class="mt-2 text-xs text-muted-foreground">
            Workspace {{ task.workspaceId || '-' }}
          </div>
        </div>
        <div class="rounded-md border p-4">
          <div class="text-sm text-muted-foreground">开始时间</div>
          <div class="mt-2 font-medium">{{ formatTime(task.startedAt) }}</div>
          <div class="mt-1 text-xs text-muted-foreground">等待 Agent 确认后记录</div>
        </div>
        <div class="rounded-md border p-4">
          <div class="text-sm text-muted-foreground">完成时间</div>
          <div class="mt-2 font-medium">{{ formatTime(task.finishedAt) }}</div>
          <div class="mt-1 text-xs text-muted-foreground">终态结果上报时间</div>
        </div>
      </section>

      <Card class="gap-0 py-0">
        <CardContent class="p-0">
          <div class="border-b px-4 py-3"><h2 class="font-medium">任务配置</h2></div>
          <dl class="grid gap-x-8 gap-y-4 p-4 text-sm sm:grid-cols-2 xl:grid-cols-4">
            <div>
              <dt class="text-xs text-muted-foreground">任务 ID</dt>
              <dd class="mt-1 break-all font-mono text-xs">{{ task.id }}</dd>
            </div>
            <div class="sm:col-span-2">
              <dt class="text-xs text-muted-foreground">Git 仓库</dt>
              <dd class="mt-1 break-all">
                <a
                  :href="task.repositoryUrl"
                  target="_blank"
                  rel="noreferrer"
                  class="hover:underline"
                  >{{ task.repositoryUrl }}</a
                >
              </dd>
            </div>
            <div>
              <dt class="text-xs text-muted-foreground">工作目录</dt>
              <dd class="mt-1 font-mono text-xs">{{ task.workingDir || '仓库根目录' }}</dd>
            </div>
            <div class="sm:col-span-2 xl:col-span-4">
              <dt class="text-xs text-muted-foreground">Prompt</dt>
              <dd
                class="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/40 p-3 leading-6"
              >
                {{ task.prompt }}
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <section class="space-y-3">
        <div>
          <h2 class="font-medium">实时执行日志</h2>
          <p class="text-sm text-muted-foreground">stdout 与 stderr 按 Agent 上传顺序合并展示。</p>
        </div>
        <TaskLogConsole
          :task-id="task.id"
          :status="task.status"
          :initial-logs="detail.logs"
          @done="loadDetail"
        />
      </section>

      <div class="grid gap-4 xl:grid-cols-2">
        <Card class="gap-0 py-0">
          <CardContent class="p-0">
            <div class="border-b px-4 py-3">
              <h2 class="font-medium">执行结果</h2>
            </div>
            <div class="space-y-4 p-4 text-sm">
              <div
                v-if="task.errorMessage"
                class="rounded-md border border-destructive/40 bg-destructive/5 p-3"
              >
                <div class="text-xs font-medium text-destructive">错误信息</div>
                <pre
                  class="mt-2 whitespace-pre-wrap break-words font-sans leading-6 text-destructive"
                  >{{ task.errorMessage }}</pre
                >
              </div>
              <div>
                <div class="text-xs text-muted-foreground">结果摘要</div>
                <pre
                  class="mt-2 max-h-[360px] overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/40 p-3 font-sans leading-6"
                  >{{ task.result || '暂无结果' }}</pre
                >
              </div>
            </div>
          </CardContent>
        </Card>

        <Card class="gap-0 py-0">
          <CardContent class="p-0">
            <div class="border-b px-4 py-3">
              <h2 class="font-medium">修改文件</h2>
              <p class="mt-0.5 text-xs text-muted-foreground">{{ changedFiles.length }} 个文件</p>
            </div>
            <div v-if="changedFiles.length" class="max-h-[470px] overflow-auto p-4">
              <ul class="space-y-1.5 text-sm">
                <li
                  v-for="file in changedFiles"
                  :key="file"
                  class="flex min-w-0 items-center gap-2 rounded-md border px-3 py-2"
                >
                  <FileCode2 class="h-4 w-4 shrink-0 text-muted-foreground" />
                  <span class="min-w-0 break-all font-mono text-xs">{{ file }}</span>
                </li>
              </ul>
            </div>
            <div
              v-else
              class="flex min-h-40 items-center justify-center text-sm text-muted-foreground"
            >
              暂无文件变更
            </div>
          </CardContent>
        </Card>
      </div>

      <Card class="gap-0 overflow-hidden py-0">
        <CardContent class="p-0">
          <div class="border-b px-4 py-3">
            <h2 class="font-medium">命令历史</h2>
            <p class="mt-0.5 text-xs text-muted-foreground">Controller 到 Agent 的任务级指令记录</p>
          </div>
          <div class="mobile-table-scroll" aria-label="任务命令历史表格">
            <table class="w-full min-w-[820px] text-sm">
              <thead class="border-b bg-muted/20 text-left text-muted-foreground">
                <tr>
                  <th class="px-4 py-3 font-medium">命令</th>
                  <th class="px-4 py-3 font-medium">状态</th>
                  <th class="px-4 py-3 font-medium">尝试次数</th>
                  <th class="px-4 py-3 font-medium">分发时间</th>
                  <th class="px-4 py-3 font-medium">确认时间</th>
                  <th class="px-4 py-3 font-medium">命令 ID</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="command in detail.commands"
                  :key="command.id"
                  class="border-b last:border-0"
                >
                  <td class="px-4 py-3">
                    <span class="inline-flex items-center gap-2"
                      ><GitBranch class="h-4 w-4 text-muted-foreground" />{{
                        commandTypeLabel(command.type)
                      }}</span
                    >
                  </td>
                  <td class="px-4 py-3">
                    <Badge variant="outline">{{ commandStatusLabel(command.status) }}</Badge>
                  </td>
                  <td class="px-4 py-3 tabular-nums">{{ command.attempt }}</td>
                  <td class="whitespace-nowrap px-4 py-3 text-muted-foreground">
                    {{ formatTime(command.dispatchedAt) }}
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-muted-foreground">
                    {{ formatTime(command.acknowledgedAt) }}
                  </td>
                  <td class="px-4 py-3 font-mono text-xs text-muted-foreground">
                    {{ command.id }}
                  </td>
                </tr>
                <tr v-if="!detail.commands.length">
                  <td colspan="6" class="px-4 py-12 text-center text-muted-foreground">
                    暂无命令记录
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </template>

    <div v-else-if="loading" class="py-20 text-center text-sm text-muted-foreground">
      正在加载任务详情...
    </div>

    <AlertDialog
      :open="cancelOpen"
      @update:open="(open) => !open && !cancelling && (cancelOpen = false)"
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认取消任务“{{ task?.name }}”？</AlertDialogTitle>
          <AlertDialogDescription
            >等待中的任务会直接取消；执行中的任务将向 Agent 下发停止指令。</AlertDialogDescription
          >
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="cancelling" @click="cancelOpen = false"
            >返回</AlertDialogCancel
          >
          <Button variant="destructive" :disabled="cancelling" @click="cancelTask">确认取消</Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
