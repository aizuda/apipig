<script setup lang="ts">
import { reactive, watch } from 'vue'
import { Button, Dialog, DialogFixedContent, Input, Label, Textarea } from '@tabtab/ui'
import type { RemoteAgent, TaskCreateParams } from '@/api/ai-applications/remote-agent'

const props = defineProps<{
  open: boolean
  loading: boolean
  agents: RemoteAgent[]
  initialAgentId?: string
}>()

const emit = defineEmits<{
  close: []
  submit: [value: TaskCreateParams]
}>()

const form = reactive<TaskCreateParams>({
  name: '',
  agentId: '',
  repositoryUrl: '',
  workingDir: '',
  prompt: '',
})

watch(
  () => props.open,
  (open) => {
    if (!open) return
    Object.assign(form, {
      name: '',
      agentId: props.initialAgentId || props.agents[0]?.id || '',
      repositoryUrl: '',
      workingDir: '',
      prompt: '',
    })
  },
)

function submit() {
  emit('submit', {
    name: form.name.trim(),
    agentId: form.agentId,
    repositoryUrl: form.repositoryUrl.trim(),
    workingDir: form.workingDir?.trim(),
    prompt: form.prompt.trim(),
  })
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && !loading && emit('close')">
    <DialogFixedContent
      title="创建远程任务"
      description="选择在线节点并提交代码仓库与执行目标。"
      class="sm:max-w-2xl"
    >
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-1.5 sm:col-span-2">
          <Label for="remote-task-name">任务名称</Label>
          <Input id="remote-task-name" v-model="form.name" placeholder="例如 修复订单服务测试" />
        </div>
        <div class="space-y-1.5">
          <Label for="remote-task-agent">Agent 节点</Label>
          <select
            id="remote-task-agent"
            v-model="form.agentId"
            class="h-10 w-full rounded-md border bg-background px-3 text-sm"
          >
            <option value="">请选择在线节点</option>
            <option v-for="agent in agents" :key="agent.id" :value="agent.id">
              {{ agent.name }} · {{ agent.hostname }}
            </option>
          </select>
        </div>
        <div class="space-y-1.5">
          <Label for="remote-task-directory">仓库内工作目录</Label>
          <Input
            id="remote-task-directory"
            v-model="form.workingDir"
            placeholder="可选，例如 services/api"
          />
        </div>
        <div class="space-y-1.5 sm:col-span-2">
          <Label for="remote-task-repository">HTTPS Git 仓库</Label>
          <Input
            id="remote-task-repository"
            v-model="form.repositoryUrl"
            placeholder="https://github.com/org/repository.git"
          />
        </div>
        <div class="space-y-1.5 sm:col-span-2">
          <Label for="remote-task-prompt">Prompt</Label>
          <Textarea
            id="remote-task-prompt"
            v-model="form.prompt"
            rows="8"
            placeholder="描述需要 Codex 完成的代码任务"
          />
        </div>
      </div>

      <template #footer>
        <Button variant="outline" :disabled="loading" @click="emit('close')">取消</Button>
        <Button
          :disabled="
            loading ||
            !form.name.trim() ||
            !form.agentId ||
            !form.repositoryUrl.trim() ||
            !form.prompt.trim()
          "
          @click="submit"
        >
          {{ loading ? '正在创建...' : '创建任务' }}
        </Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
