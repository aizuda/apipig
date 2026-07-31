<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { FileSearch, RotateCcw, Search } from '@lucide/vue'
import { Badge, Button, Card, CardContent, Input, toast } from '@tabtab/ui'
import {
  codeReviewApi,
  type ReviewProject,
  type ReviewTask,
} from '@/api/ai-applications/code-review'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import CodeReviewSectionNav from './components/CodeReviewSectionNav.vue'
import TaskDetailDialog from './components/TaskDetailDialog.vue'

defineOptions({ name: 'CodeReviewTasks' })

const loading = ref(false)
const detailLoading = ref(false)
const tasks = ref<ReviewTask[]>([])
const projects = ref<ReviewProject[]>([])
const selectedTask = ref<ReviewTask | null>(null)
const detailOpen = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const projectId = ref('')
const status = ref('')
const eventType = ref('')

async function loadTasks() {
  loading.value = true
  try {
    const result = await codeReviewApi.taskPage({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      projectId: projectId.value || undefined,
      status: status.value || undefined,
      eventType: eventType.value || undefined,
    })
    tasks.value = result.records
    total.value = result.total
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载评审任务失败')
  } finally {
    loading.value = false
  }
}

async function loadProjects() {
  try {
    const result = await codeReviewApi.projectPage({ page: 1, pageSize: 100 })
    projects.value = result.records
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载项目筛选项失败')
  }
}

async function openTask(task: ReviewTask) {
  selectedTask.value = null
  detailOpen.value = true
  detailLoading.value = true
  try {
    selectedTask.value = await codeReviewApi.getTask(task.id)
  } catch (error) {
    detailOpen.value = false
    toast.error(error instanceof Error ? error.message : '加载评审报告失败')
  } finally {
    detailLoading.value = false
  }
}

async function retryTask(task: ReviewTask) {
  detailLoading.value = true
  try {
    await codeReviewApi.retryTask(task.id)
    toast.success('任务已重新进入执行队列')
    detailOpen.value = false
    selectedTask.value = null
    await loadTasks()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '重试任务失败')
  } finally {
    detailLoading.value = false
  }
}

function searchTasks() {
  page.value = 1
  void loadTasks()
}

function resetFilters() {
  keyword.value = ''
  projectId.value = ''
  status.value = ''
  eventType.value = ''
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

function statusVariant(value: string) {
  return value === 'succeeded' ? 'default' : value === 'failed' ? 'destructive' : 'secondary'
}

function statusLabel(value: string) {
  const labels: Record<string, string> = {
    queued: '等待执行',
    running: '执行中',
    succeeded: '已完成',
    failed: '失败',
    ignored: '已忽略',
  }
  return labels[value] || value
}

function eventLabel(value: string) {
  const labels: Record<string, string> = {
    push: 'Push',
    pull_request: 'Pull Request',
    merge_request: 'Merge Request',
  }
  return labels[value] || value || '-'
}

function projectName(id: string) {
  return projects.value.find((item) => item.id === id)?.name || ''
}

function formatTime(value?: number) {
  return value ? new Date(value).toLocaleString() : '-'
}

onMounted(async () => {
  await Promise.all([loadTasks(), loadProjects()])
})
</script>

<template>
  <div class="space-y-6">
    <AppPageHeader
      title="评审任务"
      description="集中查看 WebHook 触发记录、执行状态和 AI 评审报告。"
      :loading="loading"
      @refresh="loadTasks"
    />

    <CodeReviewSectionNav />

    <Card>
      <CardContent class="p-0">
        <form
          class="grid gap-3 border-b p-4 md:grid-cols-2 xl:grid-cols-[minmax(240px,1fr)_180px_150px_160px_auto]"
          @submit.prevent="searchTasks"
        >
          <div class="relative">
            <Search
              class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            />
            <Input v-model="keyword" class="pl-9" placeholder="搜索仓库、标题、提交人或 SHA" />
          </div>
          <select v-model="projectId" class="h-10 rounded-md border bg-background px-3 text-sm">
            <option value="">全部项目</option>
            <option v-for="project in projects" :key="project.id" :value="project.id">
              {{ project.name }}
            </option>
          </select>
          <select v-model="status" class="h-10 rounded-md border bg-background px-3 text-sm">
            <option value="">全部状态</option>
            <option value="queued">等待执行</option>
            <option value="running">执行中</option>
            <option value="succeeded">已完成</option>
            <option value="failed">失败</option>
            <option value="ignored">已忽略</option>
          </select>
          <select v-model="eventType" class="h-10 rounded-md border bg-background px-3 text-sm">
            <option value="">全部事件</option>
            <option value="push">Push</option>
            <option value="pull_request">Pull Request</option>
            <option value="merge_request">Merge Request</option>
          </select>
          <div class="flex gap-2 md:col-span-2 xl:col-span-1">
            <Button type="submit" variant="outline" :disabled="loading">查询</Button>
            <Button type="button" variant="ghost" :disabled="loading" @click="resetFilters"
              >重置</Button
            >
          </div>
        </form>

        <div v-if="tasks.length" class="overflow-x-auto">
          <table class="w-full min-w-[1050px] text-sm">
            <thead class="border-b bg-muted/20 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3 font-medium">任务</th>
                <th class="px-4 py-3 font-medium">项目 / 仓库</th>
                <th class="px-4 py-3 font-medium">事件</th>
                <th class="px-4 py-3 font-medium">变更</th>
                <th class="px-4 py-3 font-medium">风险</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 font-medium">触发时间</th>
                <th class="px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="task in tasks"
                :key="task.id"
                class="border-b last:border-0 hover:bg-muted/20"
              >
                <td class="max-w-[300px] px-4 py-4">
                  <button
                    type="button"
                    class="block max-w-full truncate text-left font-medium hover:underline"
                    @click="openTask(task)"
                  >
                    {{ task.title || task.ref || task.id }}
                  </button>
                  <div class="mt-1 truncate font-mono text-xs text-muted-foreground">
                    {{ task.headSha || '-' }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <div>{{ projectName(task.projectId) || task.repositoryName || '-' }}</div>
                  <div
                    v-if="projectName(task.projectId) && task.repositoryName"
                    class="mt-1 text-xs text-muted-foreground"
                  >
                    {{ task.repositoryName }}
                  </div>
                </td>
                <td class="px-4 py-4">{{ eventLabel(task.eventType) }}</td>
                <td class="px-4 py-4 tabular-nums">
                  <span>{{ task.changedFiles }} 文件</span>
                  <div class="mt-1 text-xs">
                    <span class="text-emerald-600">+{{ task.additions }}</span> /
                    <span class="text-destructive">-{{ task.deletions }}</span>
                  </div>
                </td>
                <td class="px-4 py-4">
                  <Badge variant="outline">{{ task.riskLevel || '-' }}</Badge>
                </td>
                <td class="px-4 py-4">
                  <Badge :variant="statusVariant(task.status)">{{
                    statusLabel(task.status)
                  }}</Badge>
                </td>
                <td class="px-4 py-4 whitespace-nowrap text-muted-foreground">
                  {{ formatTime(task.createdAt) }}
                </td>
                <td class="px-4 py-4 text-right">
                  <Button variant="ghost" size="sm" class="gap-1.5" @click="openTask(task)">
                    <FileSearch class="h-4 w-4" />查看报告
                  </Button>
                  <Button
                    v-if="['failed', 'ignored'].includes(task.status)"
                    variant="ghost"
                    size="sm"
                    class="gap-1.5"
                    @click="retryTask(task)"
                  >
                    <RotateCcw class="h-4 w-4" />重试
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else-if="!loading" class="px-6 py-16 text-center">
          <FileSearch class="mx-auto h-10 w-10 text-muted-foreground/50" />
          <h3 class="mt-4 font-medium">暂无评审任务</h3>
          <p class="mt-1 text-sm text-muted-foreground">
            配置 WebHook 并推送代码后，任务会出现在这里。
          </p>
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

    <TaskDetailDialog
      :open="detailOpen"
      :loading="detailLoading"
      :task="selectedTask"
      @close="detailOpen = false"
      @retry="retryTask"
    />
  </div>
</template>
