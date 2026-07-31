<script setup lang="ts">
import { Badge, Button, Dialog, DialogFixedContent } from '@tabtab/ui'
import type { ReviewTask } from '@/api/ai-applications/code-review'

defineProps<{
  open: boolean
  loading: boolean
  task: ReviewTask | null
}>()

const emit = defineEmits<{
  close: []
  retry: [task: ReviewTask]
}>()

function statusVariant(status?: string) {
  return status === 'succeeded' ? 'default' : status === 'failed' ? 'destructive' : 'secondary'
}

function statusLabel(status?: string) {
  const labels: Record<string, string> = {
    queued: '等待执行',
    running: '执行中',
    succeeded: '已完成',
    failed: '失败',
    ignored: '已忽略',
  }
  return status ? labels[status] || status : '-'
}

function formatTime(value?: number) {
  return value ? new Date(value).toLocaleString() : '-'
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && !loading && emit('close')">
    <DialogFixedContent
      title="评审报告"
      :description="
        task
          ? `${task.repositoryName || '代码仓库'} · ${task.title || task.ref || task.id}`
          : '正在加载任务详情'
      "
      class="sm:max-w-5xl"
      body-class="bg-muted/10"
    >
      <div v-if="loading" class="py-16 text-center text-sm text-muted-foreground">
        正在加载评审报告...
      </div>
      <div v-else-if="task" class="space-y-5">
        <div class="flex flex-wrap items-center gap-2">
          <Badge :variant="statusVariant(task.status)">{{ statusLabel(task.status) }}</Badge>
          <Badge variant="outline">风险 {{ task.riskLevel || '-' }}</Badge>
          <Badge variant="outline">{{ task.eventType || '-' }}</Badge>
          <Badge v-if="task.diffTruncated" variant="secondary">Diff 已截断</Badge>
        </div>

        <div
          class="grid gap-3 rounded-lg border bg-background p-4 text-sm sm:grid-cols-2 lg:grid-cols-4"
        >
          <div>
            <span class="text-muted-foreground">提交人</span>
            <p class="mt-1 font-medium">{{ task.author || '-' }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">分支</span>
            <p class="mt-1 break-all font-medium">{{ task.ref || '-' }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">开始时间</span>
            <p class="mt-1 font-medium">{{ formatTime(task.createdAt) }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">完成时间</span>
            <p class="mt-1 font-medium">{{ formatTime(task.finishedAt) }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">变更文件</span>
            <p class="mt-1 font-medium">{{ task.changedFiles }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">新增行</span>
            <p class="mt-1 font-medium text-emerald-600">+{{ task.additions }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">删除行</span>
            <p class="mt-1 font-medium text-destructive">-{{ task.deletions }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">执行次数</span>
            <p class="mt-1 font-medium">{{ task.attempt }}</p>
          </div>
        </div>

        <section v-if="task.summary" class="rounded-lg border bg-background p-4">
          <h3 class="mb-2 text-sm font-semibold">评审摘要</h3>
          <p class="whitespace-pre-wrap text-sm leading-6">{{ task.summary }}</p>
        </section>

        <section v-if="task.report" class="rounded-lg border bg-background p-4">
          <h3 class="mb-3 text-sm font-semibold">完整报告</h3>
          <pre class="max-h-[52vh] overflow-auto whitespace-pre-wrap text-sm leading-6">{{
            task.report
          }}</pre>
        </section>

        <div
          v-else-if="task.errorMessage"
          class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"
        >
          {{ task.errorMessage }}
        </div>
        <div
          v-else
          class="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground"
        >
          任务尚未生成报告，请稍后刷新。
        </div>
      </div>

      <template #footer>
        <Button variant="outline" @click="emit('close')">关闭</Button>
        <Button
          v-if="task && ['failed', 'ignored'].includes(task.status)"
          :disabled="loading"
          @click="emit('retry', task)"
        >
          重新执行
        </Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
