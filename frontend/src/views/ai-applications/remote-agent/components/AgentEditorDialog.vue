<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { Button, Dialog, DialogFixedContent, Input, Label, Textarea } from '@tabtab/ui'
import type { RemoteAgent, RemoteAgentSaveParams } from '@/api/ai-applications/remote-agent'

const defaultCodexArgs = [
  '--ask-for-approval',
  'never',
  'exec',
  '--json',
  '--sandbox',
  'workspace-write',
  '-c',
  'sandbox_workspace_write.network_access=true',
  '--skip-git-repo-check',
  '-',
]

const props = defineProps<{ open: boolean; loading: boolean; agent: RemoteAgent | null }>()
const emit = defineEmits<{ close: []; submit: [form: RemoteAgentSaveParams] }>()
type AgentForm = Omit<RemoteAgentSaveParams, 'codexArgs' | 'claudeArgs'> & {
  codexArgsText: string
  claudeArgsText: string
}

const form = reactive<AgentForm>({
  agentKey: '',
  name: '',
  workspaceRoot: '.',
  codexCommand: 'codex',
  codexArgsText: defaultCodexArgs.join('\n'),
  claudeCommand: 'claude',
  claudeArgsText: '-p\n--permission-mode\nacceptEdits',
  pollWaitSeconds: 25,
  requestTimeoutSeconds: 40,
  logFile: './logs/remote-agent.log',
})
const isEditing = computed(() => Boolean(props.agent?.id))
const agentKeyValid = computed(() => /^[A-Za-z0-9_-]+$/.test(form.agentKey.trim()))

watch(
  () => [props.open, props.agent] as const,
  ([open, agent]) => {
    if (!open) return
    Object.assign(form, {
      id: agent?.id,
      agentKey: agent?.agentKey || '',
      name: agent?.name || '',
      workspaceRoot: agent?.workspaceRoot || '.',
      codexCommand: agent?.codexCommand || 'codex',
      codexArgsText: (agent?.codexArgs || defaultCodexArgs).join('\n'),
      claudeCommand: agent?.claudeCommand || 'claude',
      claudeArgsText: (agent?.claudeArgs || ['-p', '--permission-mode', 'acceptEdits']).join('\n'),
      pollWaitSeconds: agent?.pollWaitSeconds || 25,
      requestTimeoutSeconds: agent?.requestTimeoutSeconds || 40,
      logFile: agent?.logFile || './logs/remote-agent.log',
    })
  },
  { immediate: true },
)

function submit() {
  if (!agentKeyValid.value) return
  emit('submit', {
    id: form.id,
    agentKey: form.agentKey.trim(),
    name: form.name.trim(),
    workspaceRoot: form.workspaceRoot.trim(),
    codexCommand: form.codexCommand.trim(),
    codexArgs: form.codexArgsText
      .split(/\r?\n/)
      .map((item) => item.trim())
      .filter(Boolean),
    claudeCommand: form.claudeCommand.trim(),
    claudeArgs: form.claudeArgsText
      .split(/\r?\n/)
      .map((item) => item.trim())
      .filter(Boolean),
    pollWaitSeconds: Number(form.pollWaitSeconds),
    requestTimeoutSeconds: Number(form.requestTimeoutSeconds),
    logFile: form.logFile.trim(),
  })
}

function updateAgentKey(value: string | number) {
  form.agentKey = String(value).replace(/[^A-Za-z0-9_-]/g, '')
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && !loading && emit('close')">
    <DialogFixedContent
      :title="isEditing ? '编辑 Agent' : '添加 Agent'"
      description="一个配置记录仅允许注册一台工作站。"
      class="sm:max-w-3xl"
    >
      <div class="grid max-h-[65vh] gap-4 overflow-y-auto pr-1 sm:grid-cols-2">
        <div class="space-y-1.5">
          <Label for="agent-name">Agent 名称 <span class="text-destructive">*</span></Label>
          <Input id="agent-name" v-model="form.name" placeholder="开发工作站" />
        </div>
        <div class="space-y-1.5">
          <Label for="agent-key">Agent Key <span class="text-destructive">*</span></Label>
          <Input
            id="agent-key"
            :model-value="form.agentKey"
            :disabled="Boolean(agent?.hostname)"
            maxlength="100"
            placeholder="dev-workstation-01"
            @update:model-value="updateAgentKey"
          />
          <p class="text-xs text-muted-foreground">仅支持字母、数字、中划线和下划线</p>
        </div>
        <div class="space-y-1.5">
          <Label for="agent-workspace">项目根目录</Label>
          <Input id="agent-workspace" v-model="form.workspaceRoot" />
          <p class="text-xs text-muted-foreground">必须是客户端上已存在且允许远程开发的目录</p>
        </div>
        <div class="space-y-1.5">
          <Label for="agent-log">Agent 日志文件</Label>
          <Input id="agent-log" v-model="form.logFile" />
        </div>
        <div class="space-y-1.5 sm:col-span-2">
          <Label for="agent-codex">Codex 命令</Label>
          <Input id="agent-codex" v-model="form.codexCommand" />
        </div>
        <div class="space-y-1.5 sm:col-span-2">
          <Label for="agent-codex-args">Codex 参数</Label>
          <Textarea
            id="agent-codex-args"
            v-model="form.codexArgsText"
            rows="4"
            class="font-mono text-xs"
          />
        </div>
        <div class="space-y-1.5 sm:col-span-2">
          <Label for="agent-claude">Claude 命令</Label>
          <Input id="agent-claude" v-model="form.claudeCommand" />
        </div>
        <div class="space-y-1.5 sm:col-span-2">
          <Label for="agent-claude-args">Claude 参数</Label>
          <Textarea
            id="agent-claude-args"
            v-model="form.claudeArgsText"
            rows="4"
            class="font-mono text-xs"
          />
        </div>
        <div class="space-y-1.5">
          <Label for="agent-poll">轮询等待（秒）</Label>
          <Input
            id="agent-poll"
            v-model.number="form.pollWaitSeconds"
            type="number"
            min="1"
            max="25"
          />
        </div>
        <div class="space-y-1.5">
          <Label for="agent-timeout">请求超时（秒）</Label>
          <Input
            id="agent-timeout"
            v-model.number="form.requestTimeoutSeconds"
            type="number"
            min="2"
          />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="loading" @click="emit('close')">取消</Button>
        <Button :disabled="loading || !form.name.trim() || !agentKeyValid" @click="submit">
          {{ isEditing ? '保存配置' : '添加并生成配置' }}
        </Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
