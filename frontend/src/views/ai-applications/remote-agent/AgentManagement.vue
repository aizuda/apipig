<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  Ellipsis,
  KeyRound,
  MessageSquare,
  MonitorCog,
  Pencil,
  Plus,
  Power,
  Search,
  Trash2,
} from '@lucide/vue'
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Input,
  toast,
} from '@tabtab/ui'
import { useRouter } from 'vue-router'
import {
  remoteAgentApi,
  type AgentStatus,
  type RemoteAgent,
  type RemoteAgentCredential,
  type RemoteAgentSaveParams,
} from '@/api/ai-applications/remote-agent'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import AgentCredentialDialog from './components/AgentCredentialDialog.vue'
import AgentEditorDialog from './components/AgentEditorDialog.vue'
import {
  agentStatusLabel,
  agentStatusVariant,
  formatBytes,
  formatPercent,
  formatTime,
} from './presentation'

defineOptions({ name: 'RemoteAgentAgents' })
const router = useRouter()
const loading = ref(false)
const saving = ref(false)
const agents = ref<RemoteAgent[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const status = ref<AgentStatus | ''>('')
const editorOpen = ref(false)
const editingAgent = ref<RemoteAgent | null>(null)
const credential = ref<RemoteAgentCredential | null>(null)
const statusTarget = ref<RemoteAgent | null>(null)
const updatingStatus = ref(false)
const resetTokenTarget = ref<RemoteAgent | null>(null)
const resettingToken = ref(false)
const deleteTarget = ref<RemoteAgent | null>(null)
const deletingAgent = ref(false)

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
    toast.error(error instanceof Error ? error.message : '加载 Agent 失败')
  } finally {
    loading.value = false
  }
}

function openAdd() {
  editingAgent.value = null
  editorOpen.value = true
}
function openEdit(agent: RemoteAgent) {
  editingAgent.value = agent
  editorOpen.value = true
}

async function saveAgent(form: RemoteAgentSaveParams) {
  saving.value = true
  try {
    if (form.id) {
      await remoteAgentApi.updateAgent(form)
      toast.success('Agent 配置已保存')
    } else {
      credential.value = await remoteAgentApi.createAgent(form)
      toast.success('Agent 已添加')
    }
    editorOpen.value = false
    await loadAgents()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '保存 Agent 失败')
  } finally {
    saving.value = false
  }
}

function canResetToken(agent: RemoteAgent) {
  return agent.status === 'OFFLINE' || agent.status === 'DISABLED'
}

async function confirmResetToken() {
  if (!resetTokenTarget.value || !canResetToken(resetTokenTarget.value) || resettingToken.value)
    return
  const agent = resetTokenTarget.value
  resettingToken.value = true
  try {
    const nextCredential = await remoteAgentApi.rotateAgentToken(agent.id)
    resetTokenTarget.value = null
    credential.value = nextCredential
    toast.success('接入配置已重新生成')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '重置 Token 失败')
  } finally {
    resettingToken.value = false
  }
}

async function confirmStatusChange() {
  if (!statusTarget.value || updatingStatus.value) return
  const agent = statusTarget.value
  updatingStatus.value = true
  try {
    await remoteAgentApi.setAgentStatus(agent.id, agent.status === 'DISABLED')
    statusTarget.value = null
    toast.success(agent.status === 'DISABLED' ? 'Agent 已启用，请重新启动客户端' : 'Agent 已禁用')
    await loadAgents()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '更新状态失败')
  } finally {
    updatingStatus.value = false
  }
}

async function confirmDelete() {
  if (!deleteTarget.value || deletingAgent.value) return
  deletingAgent.value = true
  try {
    await remoteAgentApi.deleteAgent(deleteTarget.value.id)
    deleteTarget.value = null
    toast.success('Agent 及相关存档已删除')
    if (agents.value.length === 1 && page.value > 1) page.value--
    await loadAgents()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '删除 Agent 失败')
  } finally {
    deletingAgent.value = false
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
function memoryLabel(agent: RemoteAgent) {
  if (!agent.memoryTotal) return formatBytes(agent.memoryUsed)
  return `${formatBytes(agent.memoryUsed)} / ${formatBytes(agent.memoryTotal)} (${formatPercent((agent.memoryUsed / agent.memoryTotal) * 100)})`
}
onMounted(loadAgents)
</script>

<template>
  <div class="space-y-6">
    <AppPageHeader
      title="Remote Agent"
      description="管理 Agent 接入配置、在线状态与 Web 对话控制台。"
      :loading="loading"
      @refresh="loadAgents"
    >
      <template #actions>
        <Button size="sm" class="gap-2" @click="openAdd"><Plus class="h-4 w-4" />添加 Agent</Button>
      </template>
    </AppPageHeader>

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
            <Input v-model="keyword" class="pl-9" placeholder="搜索名称、Agent Key、主机名或 IP" />
          </div>
          <select
            v-model="status"
            class="h-10 rounded-md border bg-background px-3 text-sm lg:w-40"
          >
            <option value="">全部状态</option>
            <option value="ONLINE">在线</option>
            <option value="BUSY">对话中</option>
            <option value="OFFLINE">离线</option>
            <option value="DISABLED">已禁用</option>
          </select>
          <div class="flex gap-2">
            <Button type="submit" variant="outline" :disabled="loading">查询</Button
            ><Button type="button" variant="ghost" :disabled="loading" @click="resetFilters"
              >重置</Button
            >
          </div>
        </form>

        <div v-if="agents.length" class="mobile-table-scroll" aria-label="Agent 表格">
          <table class="w-full min-w-[1120px] text-sm">
            <thead class="border-b bg-muted/20 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3 font-medium">Agent</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 font-medium">机器</th>
                <th class="px-4 py-3 font-medium">负载</th>
                <th class="px-4 py-3 font-medium">版本</th>
                <th class="px-4 py-3 font-medium">最后在线</th>
                <th
                  class="sticky right-0 z-20 w-[128px] min-w-[128px] max-w-[128px] bg-muted px-2 py-3 text-center font-medium shadow-[-4px_0_8px_-6px_rgba(0,0,0,0.35)]"
                >
                  操作
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="agent in agents"
                :key="agent.id"
                class="group border-b last:border-0 hover:bg-muted/20"
              >
                <td class="px-4 py-4">
                  <button
                    type="button"
                    class="font-medium hover:underline"
                    @click="router.push(`/ai-applications/remote-agent/agents/${agent.id}`)"
                  >
                    {{ agent.name }}
                  </button>
                  <div class="mt-1 font-mono text-xs text-muted-foreground">
                    {{ agent.agentKey }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <Badge :variant="agentStatusVariant(agent.status)">{{
                    agentStatusLabel(agent.status)
                  }}</Badge>
                </td>
                <td class="px-4 py-4">
                  <div>{{ agent.hostname || '尚未注册' }}</div>
                  <div class="mt-1 text-xs text-muted-foreground">
                    {{ agent.ipAddress || '-' }} · {{ agent.operatingSystem || '-' }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <div>CPU {{ formatPercent(agent.cpuUsage) }}</div>
                  <div class="mt-1 text-xs text-muted-foreground">{{ memoryLabel(agent) }}</div>
                </td>
                <td class="px-4 py-4">
                  <div>{{ agent.codexVersion || '-' }}</div>
                  <div class="mt-1 text-xs text-muted-foreground">
                    Agent {{ agent.agentVersion || '-' }}
                  </div>
                </td>
                <td class="whitespace-nowrap px-4 py-4 text-muted-foreground">
                  {{ formatTime(agent.lastSeenAt) }}
                </td>
                <td
                  class="sticky right-0 z-10 w-[128px] min-w-[128px] max-w-[128px] whitespace-nowrap bg-card px-2 py-4 text-center shadow-[-4px_0_8px_-6px_rgba(0,0,0,0.35)] transition-colors group-hover:bg-muted"
                >
                  <div class="flex justify-center gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      title="Web Agent 控制台"
                      :disabled="agent.status === 'DISABLED'"
                      @click="router.push(`/ai-applications/remote-agent/agents/${agent.id}`)"
                      ><MessageSquare class="h-4 w-4"
                    /></Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      :title="agent.status === 'DISABLED' ? '启用 Agent' : '禁用 Agent'"
                      @click="statusTarget = agent"
                      ><Power class="h-4 w-4"
                    /></Button>
                    <DropdownMenu>
                      <DropdownMenuTrigger as-child>
                        <Button variant="ghost" size="icon" title="更多操作">
                          <Ellipsis class="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" class="w-44">
                        <DropdownMenuItem class="cursor-pointer" @click="openEdit(agent)">
                          <Pencil />编辑配置
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          class="cursor-pointer"
                          :disabled="!canResetToken(agent)"
                          @click="resetTokenTarget = agent"
                        >
                          <KeyRound />重置注册 Token
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          variant="destructive"
                          class="cursor-pointer"
                          @click="deleteTarget = agent"
                        >
                          <Trash2 />删除 Agent
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else-if="!loading" class="px-6 py-16 text-center">
          <MonitorCog class="mx-auto h-10 w-10 text-muted-foreground/50" />
          <h3 class="mt-4 font-medium">暂无 Agent</h3>
          <p class="mt-1 text-sm text-muted-foreground">
            添加 Agent 后下载配置并在目标工作站启动客户端。
          </p>
          <Button class="mt-4 gap-2" @click="openAdd"><Plus class="h-4 w-4" />添加 Agent</Button>
        </div>
        <div v-else class="px-6 py-16 text-center text-sm text-muted-foreground">
          正在加载 Agent...
        </div>
        <AppPagination
          :total="total"
          :page="page"
          :page-size="pageSize"
          :loading="loading"
          @change-page="
            (value) => {
              page = value
              loadAgents()
            }
          "
          @change-page-size="
            (value) => {
              pageSize = value
              page = 1
              loadAgents()
            }
          "
        />
      </CardContent>
    </Card>

    <AgentEditorDialog
      :open="editorOpen"
      :loading="saving"
      :agent="editingAgent"
      @close="editorOpen = false"
      @submit="saveAgent"
    />
    <AlertDialog
      :open="Boolean(statusTarget)"
      @update:open="(open) => !open && !updatingStatus && (statusTarget = null)"
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {{ statusTarget?.status === 'DISABLED' ? '启用' : '禁用' }}“{{ statusTarget?.name }}”？
          </AlertDialogTitle>
          <AlertDialogDescription>
            <template v-if="statusTarget?.status === 'DISABLED'">
              启用后 Agent 状态将变为离线，请重新启动客户端以完成注册并恢复连接。
            </template>
            <template v-else>
              禁用后当前运行 Token 将立即失效，Agent 将无法继续连接；正在响应时不能禁用。
            </template>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="updatingStatus" @click="statusTarget = null">
            取消
          </AlertDialogCancel>
          <Button
            :variant="statusTarget?.status === 'DISABLED' ? 'default' : 'destructive'"
            :disabled="updatingStatus"
            @click="confirmStatusChange"
          >
            确认{{ statusTarget?.status === 'DISABLED' ? '启用' : '禁用' }}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
    <AlertDialog
      :open="Boolean(resetTokenTarget)"
      @update:open="(open) => !open && !resettingToken && (resetTokenTarget = null)"
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>重置“{{ resetTokenTarget?.name }}”的注册 Token？</AlertDialogTitle>
          <AlertDialogDescription>
            重置后，旧的注册 Token 和当前运行 Token
            将立即失效。请下载新配置，替换工作站上的完整配置文件并重启 Agent。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="resettingToken" @click="resetTokenTarget = null"
            >取消</AlertDialogCancel
          >
          <Button :disabled="resettingToken" @click="confirmResetToken">确认重置</Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
    <AlertDialog
      :open="Boolean(deleteTarget)"
      @update:open="(open) => !open && !deletingAgent && (deleteTarget = null)"
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除“{{ deleteTarget?.name }}”？</AlertDialogTitle>
          <AlertDialogDescription>
            删除后将同时永久删除该 Agent 的会话、消息、执行记录和心跳存档，且无法恢复。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="deletingAgent" @click="deleteTarget = null">
            取消
          </AlertDialogCancel>
          <Button variant="destructive" :disabled="deletingAgent" @click="confirmDelete">
            确认删除
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
    <AgentCredentialDialog
      :open="Boolean(credential)"
      :credential="credential"
      @close="credential = null"
    />
  </div>
</template>
