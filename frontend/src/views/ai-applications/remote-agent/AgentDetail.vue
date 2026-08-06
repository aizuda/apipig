<script setup lang="ts">
import {
  computed,
  nextTick,
  onActivated,
  onBeforeUnmount,
  onDeactivated,
  onMounted,
  ref,
  watch,
} from 'vue'
import {
  ArrowLeft,
  Bot,
  Ellipsis,
  MessageSquarePlus,
  Pencil,
  Pin,
  Send,
  Smartphone,
  Trash2,
  User,
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
  Dialog,
  DialogFixedContent,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Input,
  Label,
  Textarea,
  toast,
} from '@tabtab/ui'
import { useRoute, useRouter } from 'vue-router'
import {
  remoteAgentApi,
  type AgentDetailResult,
  type CLIType,
  type MessageChunk,
  type RemoteConversation,
  type RemoteMessage,
} from '@/api/ai-applications/remote-agent'
import {
  wechatBotApi,
  type WechatBot,
  type WechatContact,
} from '@/api/ai-applications/wechat-bot'
import { useTabsStore } from '@/stores/tabs'
import MarkdownContent from './components/MarkdownContent.vue'
import { agentStatusLabel, agentStatusVariant, formatTime } from './presentation'

defineOptions({ name: 'RemoteAgentAgentDetail' })
const route = useRoute()
const router = useRouter()
const tabsStore = useTabsStore()
const detail = ref<AgentDetailResult | null>(null)
const conversations = ref<RemoteConversation[]>([])
const selectedConversation = ref<RemoteConversation | null>(null)
const messages = ref<RemoteMessage[]>([])
const prompt = ref('')
const loading = ref(false)
const sending = ref(false)
const pinningConversationId = ref('')
const renameConversationTarget = ref<RemoteConversation | null>(null)
const renameConversationTitle = ref('')
const renamingConversation = ref(false)
const deleteConversationTarget = ref<RemoteConversation | null>(null)
const deletingConversation = ref(false)
const newConversationOpen = ref(false)
const newConversationTitle = ref('')
const newConversationCLI = ref<CLIType>('CODEX')
const newConversationWorkingDirectory = ref('.')
const creatingConversation = ref(false)
const refreshingAgentStatus = ref(false)
const takeoverOpen = ref(false)
const takeoverBots = ref<WechatBot[]>([])
const takeoverContacts = ref<WechatContact[]>([])
const takeoverBotId = ref('')
const takeoverUserId = ref('')
const takeoverLoading = ref(false)
const takeoverSyncing = ref(false)
const viewport = ref<HTMLElement | null>(null)
const agentId = computed(() => String(route.params.id || ''))
const agent = computed(() => detail.value?.agent)
const canSend = computed(
  () =>
    agent.value?.status === 'ONLINE' &&
    Boolean(selectedConversation.value) &&
    selectedConversation.value?.controlMode !== 'WECHAT' &&
    !sending.value,
)

async function openTakeover() {
  if (!selectedConversation.value) return
  takeoverOpen.value = true
  takeoverLoading.value = true
  takeoverBotId.value = ''
  takeoverUserId.value = ''
  takeoverContacts.value = []
  try {
    const result = await wechatBotApi.page({ page: 1, pageSize: 100, status: 'ONLINE' })
    takeoverBots.value = (result.records || []).filter((bot) => bot.enabled && bot.status === 'ONLINE')
    if (takeoverBots.value.length === 1) {
      takeoverBotId.value = takeoverBots.value[0]!.id
      await loadTakeoverContacts()
    }
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载微信 Bot 失败')
  } finally {
    takeoverLoading.value = false
  }
}

async function loadTakeoverContacts() {
  takeoverUserId.value = ''
  takeoverContacts.value = []
  if (!takeoverBotId.value) return
  takeoverLoading.value = true
  try {
    takeoverContacts.value = (await wechatBotApi.contacts(takeoverBotId.value)).filter(
      (contact) => contact.canSend,
    )
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载微信联系人失败')
  } finally {
    takeoverLoading.value = false
  }
}

async function startTakeover() {
  if (!selectedConversation.value || !takeoverBotId.value || !takeoverUserId.value) return
  takeoverLoading.value = true
  try {
    const updated = await remoteAgentApi.startTakeover(
      selectedConversation.value.id,
      takeoverBotId.value,
      takeoverUserId.value,
    )
    selectedConversation.value = updated
    conversations.value = conversations.value.map((item) => (item.id === updated.id ? updated : item))
    prompt.value = ''
    takeoverOpen.value = false
    toast.success('微信 Bot 已接管当前会话')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '开启微信接管失败')
  } finally {
    takeoverLoading.value = false
  }
}

async function stopTakeover() {
  if (!selectedConversation.value || takeoverLoading.value) return
  takeoverLoading.value = true
  try {
    const updated = await remoteAgentApi.stopTakeover(selectedConversation.value.id)
    selectedConversation.value = updated
    conversations.value = conversations.value.map((item) => (item.id === updated.id ? updated : item))
    toast.success('已恢复 Web 端控制')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '结束微信接管失败')
  } finally {
    takeoverLoading.value = false
  }
}
const canRenameConversation = computed(() => {
  const title = renameConversationTitle.value.trim()
  return (
    Boolean(renameConversationTarget.value) &&
    title !== '' &&
    Array.from(title).length <= 200 &&
    title !== renameConversationTarget.value?.title
  )
})
let streamController: AbortController | null = null
let statusRefreshTimer: number | undefined
let scrollFrame: number | undefined
let viewportResizeObserver: ResizeObserver | undefined

async function scrollToBottom() {
  await nextTick()
  window.cancelAnimationFrame(scrollFrame || 0)
  scrollFrame = window.requestAnimationFrame(() => {
    if (viewport.value) viewport.value.scrollTop = viewport.value.scrollHeight
    scrollFrame = undefined
  })
}

watch(
  () =>
    messages.value.map((message) => [
      message.id,
      message.status,
      message.content,
      message.errorMessage,
    ]),
  () => void scrollToBottom(),
  { flush: 'post' },
)

watch(viewport, (element) => {
  viewportResizeObserver?.disconnect()
  viewportResizeObserver = undefined
  if (!element) return
  viewportResizeObserver = new ResizeObserver(() => void scrollToBottom())
  viewportResizeObserver.observe(element)
  void scrollToBottom()
})

async function loadAgent() {
  if (!agentId.value) return
  const requestedAgentId = agentId.value
  loading.value = true
  try {
    detail.value = await remoteAgentApi.getAgent(requestedAgentId)
    if (agentId.value !== requestedAgentId) return
    const tabTitle = `${detail.value.agent.name} (${detail.value.agent.agentKey})`
    tabsStore.updateTabTitle(route.fullPath, tabTitle)
    document.title = `${tabTitle} | ApiPig`
    await loadConversations()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载 Agent 控制台失败')
  } finally {
    loading.value = false
  }
}

async function refreshAgentStatus() {
  if (!agentId.value || !detail.value || refreshingAgentStatus.value) return
  refreshingAgentStatus.value = true
  try {
    const latest = await remoteAgentApi.getAgentStatus(agentId.value)
    if (detail.value && latest.id === agentId.value) detail.value.agent = latest
  } catch {
    // A transient status refresh failure must not interrupt the active conversation.
  } finally {
    refreshingAgentStatus.value = false
  }
}

async function syncTakeoverConversation() {
  const current = selectedConversation.value
  if (current?.controlMode !== 'WECHAT' || takeoverSyncing.value || streamController) return
  takeoverSyncing.value = true
  try {
    const result = await remoteAgentApi.getConversation(current.id)
    if (selectedConversation.value?.id !== current.id) return
    selectedConversation.value = result.conversation
    conversations.value = conversations.value.map((item) =>
      item.id === result.conversation.id ? result.conversation : item,
    )
    const previousLastId = messages.value.at(-1)?.id
    const nextLastId = result.messages.at(-1)?.id
    if (previousLastId !== nextLastId) {
      messages.value = result.messages || []
      const active = [...messages.value]
        .reverse()
        .find(
          (item) =>
            item.role === 'ASSISTANT' &&
            (item.status === 'PENDING' || item.status === 'STREAMING'),
        )
      if (active) void streamMessage(active)
    }
  } catch {
    // The next poll will retry without interrupting the visible conversation.
  } finally {
    takeoverSyncing.value = false
  }
}

async function loadConversations(preferredId = '') {
  const result = await remoteAgentApi.conversationPage({
    page: 1,
    pageSize: 100,
    agentId: agentId.value,
  })
  conversations.value = result.records || []
  const target =
    conversations.value.find(
      (item) => item.id === (preferredId || selectedConversation.value?.id),
    ) || conversations.value[0]
  if (target) await selectConversation(target)
  else {
    selectedConversation.value = null
    messages.value = []
  }
}

async function createConversation() {
  const workingDirectory = newConversationWorkingDirectory.value.trim()
  if (creatingConversation.value) return
  creatingConversation.value = true
  try {
    const conversation = await remoteAgentApi.createConversation(
      agentId.value,
      newConversationCLI.value,
      workingDirectory,
      newConversationTitle.value.trim(),
    )
    newConversationOpen.value = false
    newConversationTitle.value = ''
    newConversationWorkingDirectory.value = '.'
    await loadConversations(conversation.id)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '创建会话失败')
  } finally {
    creatingConversation.value = false
  }
}

async function selectConversation(conversation: RemoteConversation) {
  streamController?.abort()
  streamController = null
  selectedConversation.value = conversation
  try {
    const result = await remoteAgentApi.getConversation(conversation.id)
    selectedConversation.value = result.conversation
    messages.value = result.messages || []
    const active = [...messages.value]
      .reverse()
      .find(
        (item) =>
          item.role === 'ASSISTANT' && (item.status === 'PENDING' || item.status === 'STREAMING'),
      )
    if (active) {
      active.content = ''
      void streamMessage(active)
    }
    await scrollToBottom()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载会话失败')
  }
}

async function sendMessage() {
  const content = prompt.value.trim()
  if (!content || !selectedConversation.value || !canSend.value) return
  sending.value = true
  try {
    const result = await remoteAgentApi.sendMessage(selectedConversation.value.id, content)
    prompt.value = ''
    messages.value.push(result.userMessage, result.assistantMessage)
    if (detail.value) detail.value.agent.status = 'BUSY'
    await scrollToBottom()
    void streamMessage(result.assistantMessage)
  } catch (error) {
    sending.value = false
    toast.error(error instanceof Error ? error.message : '发送消息失败')
  }
}

async function streamMessage(message: RemoteMessage) {
  streamController?.abort()
  const controller = new AbortController()
  streamController = controller
  const streamState = { lastChunkSequence: 0, streamedContent: '' }
  const syncTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') {
      void syncStreamingMessage(message, streamState).then((terminal) => {
        if (terminal && streamController === controller) controller.abort()
      })
    }
  }, 2000)
  let retryCount = 0
  try {
    let finished = false
    while (!finished && !controller.signal.aborted) {
      try {
        const response = await remoteAgentApi.streamMessage(
          message.id,
          streamState.lastChunkSequence,
          controller.signal,
        )
        const reader = response.body?.getReader()
        if (!reader) throw new Error('消息流不可读')
        const decoder = new TextDecoder()
        let buffer = ''
        while (!finished) {
          const { done, value } = await reader.read()
          buffer += decoder
            .decode(value || new Uint8Array(), { stream: !done })
            .replace(/\r\n/g, '\n')
          let boundary = buffer.indexOf('\n\n')
          while (boundary >= 0) {
            finished = parseFrame(message, buffer.slice(0, boundary), streamState) || finished
            retryCount = 0
            buffer = buffer.slice(boundary + 2)
            boundary = buffer.indexOf('\n\n')
          }
          if (done) break
        }
        if (!finished) throw new Error('实时消息连接已关闭')
      } catch (error) {
        if (controller.signal.aborted) break
        retryCount += 1
        const terminal = await syncStreamingMessage(message, streamState)
        if (terminal) {
          finished = true
          break
        }
        if (retryCount >= 5) throw error
        await new Promise((resolve) => window.setTimeout(resolve, 1000))
      }
    }
  } catch (error) {
    if (!(error instanceof DOMException && error.name === 'AbortError'))
      toast.error(error instanceof Error ? error.message : '实时消息连接中断')
  } finally {
    window.clearInterval(syncTimer)
    if (streamController === controller) {
      streamController = null
      sending.value = false
    }
  }
}

async function syncStreamingMessage(
  message: RemoteMessage,
  streamState: { lastChunkSequence: number; streamedContent: string },
) {
  const conversationId = selectedConversation.value?.id
  if (!conversationId) return false
  if (message.status === 'COMPLETED' || message.status === 'FAILED') return true
  try {
    const detailResult = await remoteAgentApi.getConversation(conversationId)
    const stored = detailResult.messages.find((item) => item.id === message.id)
    if (!stored) return false
    const terminal = stored.status === 'COMPLETED' || stored.status === 'FAILED'
    if (terminal) Object.assign(message, stored)
    else if (
      message.status !== 'COMPLETED' &&
      message.status !== 'FAILED' &&
      stored.content.length >= message.content.length
    ) {
      message.content = stored.content
      message.status = stored.status
      message.updatedAt = stored.updatedAt
      if (streamState.streamedContent.length > stored.content.length) {
        message.content = streamState.streamedContent
      }
    }
    if (terminal) {
      if (detail.value) detail.value.agent.status = 'ONLINE'
      void loadConversations(conversationId)
    }
    return terminal
  } catch {
    return false
  }
}

function parseFrame(
  message: RemoteMessage,
  frame: string,
  streamState: { lastChunkSequence: number; streamedContent: string },
) {
  let event = 'message'
  const data: string[] = []
  for (const line of frame.split(/\r?\n/)) {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    if (line.startsWith('data:')) data.push(line.slice(5).trimStart())
  }
  if (!data.length) return false
  if (event === 'chunk') {
    const chunk = JSON.parse(data.join('\n')) as MessageChunk
    if (chunk.sequence > streamState.lastChunkSequence) {
      streamState.streamedContent += chunk.content
      if (streamState.streamedContent.length >= message.content.length) {
        message.content = streamState.streamedContent
      }
      streamState.lastChunkSequence = chunk.sequence
      void scrollToBottom()
    }
  }
  if (event === 'status') message.status = (JSON.parse(data.join('\n')) as RemoteMessage).status
  if (event === 'done') {
    Object.assign(message, JSON.parse(data.join('\n')) as RemoteMessage)
    if (detail.value) detail.value.agent.status = 'ONLINE'
    void loadConversations(selectedConversation.value?.id)
    return true
  }
  if (event === 'error') {
    const payload = JSON.parse(data.join('\n')) as { message: string }
    toast.error(payload.message)
    return true
  }
  return false
}

function compareConversations(left: RemoteConversation, right: RemoteConversation) {
  if (left.pinned !== right.pinned) return left.pinned ? -1 : 1
  if (left.pinned && left.pinnedAt !== right.pinnedAt) return right.pinnedAt - left.pinnedAt
  if (left.lastMessageAt !== right.lastMessageAt) return right.lastMessageAt - left.lastMessageAt
  return right.createdAt - left.createdAt
}

async function toggleConversationPin(conversation: RemoteConversation) {
  if (pinningConversationId.value) return
  pinningConversationId.value = conversation.id
  try {
    const updated = await remoteAgentApi.pinConversation(conversation.id, !conversation.pinned)
    conversations.value = conversations.value
      .map((item) => (item.id === updated.id ? updated : item))
      .sort(compareConversations)
    if (selectedConversation.value?.id === updated.id) {
      Object.assign(selectedConversation.value, updated)
    }
    toast.success(updated.pinned ? '会话已置顶' : '已取消置顶')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '更新会话置顶状态失败')
  } finally {
    pinningConversationId.value = ''
  }
}

function openRenameConversation(conversation: RemoteConversation) {
  renameConversationTarget.value = conversation
  renameConversationTitle.value = conversation.title
}

function closeRenameConversation() {
  if (renamingConversation.value) return
  renameConversationTarget.value = null
  renameConversationTitle.value = ''
}

async function confirmRenameConversation() {
  if (!renameConversationTarget.value || !canRenameConversation.value || renamingConversation.value)
    return
  const target = renameConversationTarget.value
  renamingConversation.value = true
  try {
    const updated = await remoteAgentApi.renameConversation(
      target.id,
      renameConversationTitle.value.trim(),
    )
    conversations.value = conversations.value.map((item) =>
      item.id === updated.id ? updated : item,
    )
    if (selectedConversation.value?.id === updated.id) {
      Object.assign(selectedConversation.value, updated)
    }
    renameConversationTarget.value = null
    renameConversationTitle.value = ''
    toast.success('会话名称已更新')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '重命名会话失败')
  } finally {
    renamingConversation.value = false
  }
}

function handleRenameKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.isComposing) return
  event.preventDefault()
  void confirmRenameConversation()
}

async function confirmDeleteConversation() {
  if (!deleteConversationTarget.value || deletingConversation.value) return
  const target = deleteConversationTarget.value
  deletingConversation.value = true
  try {
    await remoteAgentApi.deleteConversation(target.id)
    if (selectedConversation.value?.id === target.id) {
      streamController?.abort()
      streamController = null
      selectedConversation.value = null
      messages.value = []
      sending.value = false
    }
    deleteConversationTarget.value = null
    await loadConversations()
    toast.success('会话及相关记录已删除')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '删除会话失败')
  } finally {
    deletingConversation.value = false
  }
}

function handleComposerKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    void sendMessage()
  }
}

function startStatusRefresh() {
  window.clearInterval(statusRefreshTimer)
  void refreshAgentStatus()
  statusRefreshTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') {
      void refreshAgentStatus()
      void syncTakeoverConversation()
    }
  }, 3000)
}

function stopStatusRefresh() {
  window.clearInterval(statusRefreshTimer)
  statusRefreshTimer = undefined
}

onMounted(() => {
  void loadAgent()
  startStatusRefresh()
})
onActivated(startStatusRefresh)
onDeactivated(stopStatusRefresh)
onBeforeUnmount(() => {
  streamController?.abort()
  stopStatusRefresh()
  viewportResizeObserver?.disconnect()
  window.cancelAnimationFrame(scrollFrame || 0)
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden border bg-background">
    <header class="flex min-h-14 flex-wrap items-center justify-between gap-3 border-b px-4 py-2">
      <div class="flex min-w-0 items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          title="返回 Agent 列表"
          @click="router.push('/ai-applications/remote-agent/agents')"
          ><ArrowLeft class="h-4 w-4"
        /></Button>
        <div class="min-w-0">
          <h1 class="truncate text-sm font-semibold">{{ agent?.name || 'Web Agent 控制台' }}</h1>
          <p class="truncate text-xs text-muted-foreground">
            {{ agent?.hostname || agent?.agentKey || '-' }} · {{ agent?.ipAddress || '未连接' }}
          </p>
        </div>
      </div>
      <Badge v-if="agent" :variant="agentStatusVariant(agent.status)">{{
        agentStatusLabel(agent.status)
      }}</Badge>
    </header>

    <div
      class="grid min-h-0 flex-1 grid-rows-[auto_minmax(0,1fr)] md:grid-rows-1 md:grid-cols-[280px_minmax(0,1fr)]"
    >
      <aside class="flex min-h-0 flex-col border-b md:border-b-0 md:border-r">
        <div class="flex items-center justify-between border-b px-3 py-2">
          <span class="text-sm font-medium">会话</span>
          <Button size="icon" variant="ghost" title="新建会话" @click="newConversationOpen = true"
            ><MessageSquarePlus class="h-4 w-4"
          /></Button>
        </div>
        <div class="max-h-48 flex-1 overflow-auto p-2 md:max-h-none">
          <div
            v-for="conversation in conversations"
            :key="conversation.id"
            class="group mb-1 flex w-full items-center border-l-2 hover:bg-muted/50"
            :class="
              selectedConversation?.id === conversation.id
                ? 'border-primary bg-muted'
                : 'border-transparent'
            "
          >
            <button
              type="button"
              class="min-w-0 flex-1 px-3 py-2 text-left"
              @click="selectConversation(conversation)"
            >
              <div class="truncate text-sm font-medium">{{ conversation.title }}</div>
              <div class="mt-1 truncate text-xs text-muted-foreground">
                {{ conversation.cliType }} · {{ conversation.workingDirectory }}
              </div>
              <div class="mt-1 flex items-center justify-between text-xs text-muted-foreground">
                <span>{{ formatTime(conversation.lastMessageAt) }}</span>
                <span class="flex items-center gap-2">
                  <Smartphone v-if="conversation.controlMode === 'WECHAT'" class="h-3.5 w-3.5" />
                  <Pin v-if="conversation.pinned" class="h-3.5 w-3.5" />
                </span>
              </div>
            </button>
            <DropdownMenu v-if="selectedConversation?.id === conversation.id">
              <DropdownMenuTrigger as-child>
                <Button
                  size="icon"
                  variant="ghost"
                  class="mr-1 h-8 w-8 shrink-0 text-muted-foreground"
                  title="更多操作"
                >
                  <Ellipsis class="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-36">
                <DropdownMenuItem
                  class="cursor-pointer"
                  :disabled="pinningConversationId === conversation.id"
                  @click="toggleConversationPin(conversation)"
                >
                  <Pin />{{ conversation.pinned ? '取消置顶' : '置顶会话' }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  class="cursor-pointer"
                  @click="openRenameConversation(conversation)"
                >
                  <Pencil />重命名
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="destructive"
                  class="cursor-pointer"
                  @click="deleteConversationTarget = conversation"
                >
                  <Trash2 />删除会话
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <div
            v-if="!conversations.length"
            class="px-3 py-10 text-center text-xs text-muted-foreground"
          >
            暂无会话
          </div>
        </div>
      </aside>

      <main class="flex min-h-0 min-w-0 flex-col">
        <div class="flex min-h-12 items-center justify-between gap-3 border-b px-4 py-2">
          <div class="min-w-0">
            <div class="truncate text-sm font-medium">
              {{ selectedConversation?.title || '选择或新建会话' }}
            </div>
            <div v-if="selectedConversation" class="truncate text-xs text-muted-foreground">
              {{ selectedConversation.cliType }} · {{ selectedConversation.workingDirectory }}
            </div>
          </div>
          <div v-if="selectedConversation" class="flex shrink-0 items-center gap-2">
            <Badge v-if="selectedConversation.controlMode === 'WECHAT'" variant="secondary">
              <Smartphone class="mr-1 h-3.5 w-3.5" />微信接管中
            </Badge>
            <Button
              size="sm"
              :variant="selectedConversation.controlMode === 'WECHAT' ? 'outline' : 'default'"
              :disabled="takeoverLoading || agent?.status === 'BUSY'"
              @click="selectedConversation.controlMode === 'WECHAT' ? stopTakeover() : openTakeover()"
            >
              <Smartphone class="h-4 w-4" />
              {{ selectedConversation.controlMode === 'WECHAT' ? '结束接管' : '微信接管' }}
            </Button>
          </div>
        </div>

        <div
          ref="viewport"
          class="min-h-0 flex-1 overflow-x-hidden overflow-y-auto overscroll-contain px-4 py-5 sm:px-8"
        >
          <div v-if="messages.length" class="mx-auto max-w-4xl space-y-6">
            <article
              v-for="message in messages"
              :key="message.id"
              class="grid gap-3"
              :class="
                message.role === 'USER'
                  ? 'grid-cols-[minmax(0,1fr)_28px]'
                  : 'grid-cols-[28px_minmax(0,1fr)]'
              "
            >
              <div
                class="flex h-7 w-7 items-center justify-center border bg-muted"
                :class="message.role === 'USER' ? 'col-start-2 row-start-1' : ''"
              >
                <User v-if="message.role === 'USER'" class="h-4 w-4" /><Bot
                  v-else
                  class="h-4 w-4"
                />
              </div>
              <div
                class="min-w-0"
                :class="message.role === 'USER' ? 'col-start-1 row-start-1 text-right' : ''"
              >
                <div
                  class="mb-1 flex items-center gap-2 text-xs font-medium"
                  :class="message.role === 'USER' ? 'justify-end' : ''"
                >
                  <span>{{ message.role === 'USER' ? '你' : agent?.name || 'Agent' }}</span
                  ><span class="font-normal text-muted-foreground">{{
                    formatTime(message.createdAt)
                  }}</span>
                </div>
                <MarkdownContent
                  v-if="message.content"
                  :content="message.content"
                  :align="message.role === 'USER' ? 'right' : 'left'"
                />
                <div v-else class="text-sm leading-6 text-muted-foreground">
                  {{ message.status === 'PENDING' ? '等待 Agent 响应...' : '正在响应...' }}
                </div>
                <p
                  v-if="message.errorMessage"
                  class="mt-2 whitespace-pre-wrap text-xs text-destructive"
                >
                  {{ message.errorMessage }}
                </p>
              </div>
            </article>
          </div>
          <div
            v-else
            class="flex h-full min-h-72 items-center justify-center text-center text-sm text-muted-foreground"
          >
            {{
              selectedConversation ? '发送第一条消息开始对话' : '新建会话后进入 Web Agent 控制台'
            }}
          </div>
        </div>

        <div class="border-t p-3 sm:p-4">
          <div class="mx-auto flex max-w-4xl items-end gap-2">
            <Textarea
              v-model="prompt"
              rows="3"
              class="min-h-20 flex-1 resize-none"
              :disabled="!selectedConversation || selectedConversation.controlMode === 'WECHAT'"
              :placeholder="
                selectedConversation?.controlMode === 'WECHAT'
                  ? '微信 Bot 接管中，Web 端已暂停控制'
                  : agent?.status === 'ONLINE'
                  ? '输入消息，Enter 发送，Shift + Enter 换行'
                  : 'Agent 在线后可发送消息'
              "
              @keydown="handleComposerKeydown"
            />
            <Button
              size="icon"
              class="h-10 w-10 shrink-0"
              title="发送消息"
              :disabled="!canSend || !prompt.trim()"
              @click="sendMessage"
              ><Send class="h-4 w-4"
            /></Button>
          </div>
        </div>
      </main>
    </div>

    <Dialog :open="takeoverOpen" @update:open="(open) => !takeoverLoading && (takeoverOpen = open)">
      <DialogFixedContent title="微信 Bot 接管" class="sm:max-w-md">
        <div class="space-y-4">
          <div class="space-y-1.5">
            <Label for="takeover-bot">微信 Bot</Label>
            <select
              id="takeover-bot"
              v-model="takeoverBotId"
              class="h-10 w-full border bg-background px-3 text-sm"
              :disabled="takeoverLoading"
              @change="loadTakeoverContacts"
            >
              <option value="">选择在线 Bot</option>
              <option v-for="botItem in takeoverBots" :key="botItem.id" :value="botItem.id">
                {{ botItem.name }}
              </option>
            </select>
          </div>
          <div class="space-y-1.5">
            <Label for="takeover-contact">接管联系人</Label>
            <select
              id="takeover-contact"
              v-model="takeoverUserId"
              class="h-10 w-full border bg-background px-3 text-sm"
              :disabled="takeoverLoading || !takeoverBotId"
            >
              <option value="">选择 24 小时内活跃的联系人</option>
              <option v-for="contact in takeoverContacts" :key="contact.id" :value="contact.userId">
                {{ contact.userId }} · {{ contact.lastMessage || '暂无消息' }}
              </option>
            </select>
          </div>
          <p class="text-xs leading-5 text-muted-foreground">
            接管后，该联系人的文本消息将发送给当前 Agent，回复自动回发微信；Web 端仍可查看消息，但暂停发送。
          </p>
          <div class="flex justify-end gap-2">
            <Button variant="outline" :disabled="takeoverLoading" @click="takeoverOpen = false">取消</Button>
            <Button :disabled="takeoverLoading || !takeoverBotId || !takeoverUserId" @click="startTakeover">
              <Smartphone class="h-4 w-4" />开始接管
            </Button>
          </div>
        </div>
      </DialogFixedContent>
    </Dialog>

    <Dialog
      :open="newConversationOpen"
      @update:open="(open) => !creatingConversation && (newConversationOpen = open)"
    >
      <DialogFixedContent title="新建会话" class="sm:max-w-md">
        <div class="space-y-4">
          <div class="space-y-1.5">
            <Label for="new-conversation-title">会话名称</Label>
            <Input
              id="new-conversation-title"
              v-model="newConversationTitle"
              placeholder="新会话"
            />
          </div>
          <div class="space-y-1.5">
            <Label>开发工具</Label>
            <div class="grid grid-cols-2 gap-2">
              <Button
                type="button"
                :variant="newConversationCLI === 'CODEX' ? 'default' : 'outline'"
                @click="newConversationCLI = 'CODEX'"
              >
                Codex CLI
              </Button>
              <Button
                type="button"
                :variant="newConversationCLI === 'CLAUDE' ? 'default' : 'outline'"
                @click="newConversationCLI = 'CLAUDE'"
              >
                Claude CLI
              </Button>
            </div>
          </div>
          <div class="space-y-1.5">
            <Label for="new-conversation-directory">项目目录</Label>
            <Input
              id="new-conversation-directory"
              v-model="newConversationWorkingDirectory"
              placeholder=". 或 project-name"
            />
            <p class="text-xs text-muted-foreground">
              相对于 Agent 工作区根目录；“.” 表示工作区根目录
            </p>
          </div>
        </div>
        <template #footer>
          <Button
            variant="outline"
            :disabled="creatingConversation"
            @click="newConversationOpen = false"
          >
            取消
          </Button>
          <Button :disabled="creatingConversation" @click="createConversation"> 创建 </Button>
        </template>
      </DialogFixedContent>
    </Dialog>

    <Dialog
      :open="Boolean(renameConversationTarget)"
      @update:open="(open) => !open && closeRenameConversation()"
    >
      <DialogFixedContent title="重命名会话" class="sm:max-w-md">
        <div class="space-y-1.5">
          <Label for="conversation-title">会话名称</Label>
          <Input
            id="conversation-title"
            v-model="renameConversationTitle"
            maxlength="200"
            autofocus
            @keydown="handleRenameKeydown"
          />
        </div>
        <template #footer>
          <Button
            variant="outline"
            :disabled="renamingConversation"
            @click="closeRenameConversation"
          >
            取消
          </Button>
          <Button
            :disabled="renamingConversation || !canRenameConversation"
            @click="confirmRenameConversation"
          >
            保存
          </Button>
        </template>
      </DialogFixedContent>
    </Dialog>

    <AlertDialog
      :open="Boolean(deleteConversationTarget)"
      @update:open="(open) => !open && !deletingConversation && (deleteConversationTarget = null)"
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除“{{ deleteConversationTarget?.title }}”？</AlertDialogTitle>
          <AlertDialogDescription>
            删除后将永久清除该会话的全部消息和相关执行记录，且无法恢复。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel
            :disabled="deletingConversation"
            @click="deleteConversationTarget = null"
          >
            取消
          </AlertDialogCancel>
          <Button
            variant="destructive"
            :disabled="deletingConversation"
            @click="confirmDeleteConversation"
          >
            确认删除
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
