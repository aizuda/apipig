<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Bot, LoaderCircle, RotateCcw, Send, Settings2, Sparkles, Square, User } from '@lucide/vue'
import {
  Badge,
  Button,
  Card,
  Dialog,
  DialogFixedContent,
  Input,
  Label,
  Textarea,
  toast,
} from '@tabtab/ui'
import { aiGatewayApi, type AccessToken, type AIChatMessage, type Channel } from '@/api/ai-gateway'
import AppPageHeader from '@/components/AppPageHeader.vue'

defineOptions({ name: 'AiGatewayChat' })

const props = withDefaults(defineProps<{ embedded?: boolean }>(), {
  embedded: false,
})

type DisplayMessage = AIChatMessage & {
  id: number
  requestId?: string
  usage?: { prompt_tokens: number; completion_tokens: number; total_tokens: number }
}

const tokens = ref<AccessToken[]>([])
const channels = ref<Channel[]>([])
const tokenId = ref('')
const model = ref('')
const systemPrompt = ref('你是一个乐于助人的 AI 助手。')
const temperature = ref(0.7)
const maxTokens = ref(2048)
const contextRounds = ref(6)
const input = ref('')
const messages = ref<DisplayMessage[]>([])
const loadingOptions = ref(false)
const sending = ref(false)
const errorMessage = ref('')
const configOpen = ref(false)
const messageList = ref<HTMLElement | null>(null)
const autoScrollEnabled = ref(true)
let messageId = 0
let activeRequestController: AbortController | null = null

const MAX_SSE_EVENT_CHARS = 2 * 1024 * 1024

const selectedToken = computed(() =>
  tokens.value.find((token) => String(token.id) === tokenId.value),
)
const availableModels = computed(() => {
  const token = selectedToken.value
  if (!token) return []
  const authorized = splitModels(token.models)
  if (authorized.length) return authorized
  const relatedChannels = token.channelId
    ? channels.value.filter((channel) => String(channel.id) === String(token.channelId))
    : channels.value.filter((channel) => channel.status === 1)
  return [...new Set(relatedChannels.flatMap((channel) => channel.availableModels || []))]
})
const canSend = computed(
  () => Boolean(tokenId.value && model.value && input.value.trim()) && !sending.value,
)
const effectiveContextRounds = computed(() =>
  Math.min(20, Math.max(1, Math.floor(Number(contextRounds.value) || 1))),
)

function splitModels(value?: string) {
  return (value || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function createMessage(role: AIChatMessage['role'], content: string): DisplayMessage {
  return { id: ++messageId, role, content }
}

function selectSlidingContext(conversation: DisplayMessage[], rounds: number): AIChatMessage[] {
  const normalizedRounds = Math.min(20, Math.max(1, Math.floor(Number(rounds) || 1)))
  let startIndex = 0
  let userRounds = 0
  for (let index = conversation.length - 1; index >= 0; index -= 1) {
    if (conversation[index]?.role !== 'user') continue
    userRounds += 1
    if (userRounds === normalizedRounds) {
      startIndex = index
      break
    }
  }
  return conversation.slice(startIndex).map(({ role, content }) => ({ role, content }))
}

async function scrollToBottom(force = false) {
  await nextTick()
  const element = messageList.value
  if (!element || (!force && !autoScrollEnabled.value)) return
  element.scrollTop = element.scrollHeight
}

function handleMessageScroll() {
  const element = messageList.value
  if (!element) return
  const remainingDistance = element.scrollHeight - element.scrollTop - element.clientHeight
  autoScrollEnabled.value = remainingDistance < 96
}

async function loadOptions() {
  loadingOptions.value = true
  errorMessage.value = ''
  try {
    const [tokenPage, channelPage] = await Promise.all([
      aiGatewayApi.tokenPage({ page: 1, pageSize: 500 }),
      aiGatewayApi.channelPage({ page: 1, pageSize: 500 }),
    ])
    tokens.value = tokenPage.records.filter((token) => token.status === 1)
    channels.value = channelPage.records
    if (!tokenId.value && tokens.value[0]?.id) tokenId.value = String(tokens.value[0].id)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '聊天配置加载失败'
    toast.error(errorMessage.value)
  } finally {
    loadingOptions.value = false
  }
}

async function sendMessage() {
  const content = input.value.trim()
  if (!canSend.value || !content) return
  errorMessage.value = ''
  const userMessage = createMessage('user', content)
  const assistantMessage = createMessage('assistant', '')
  let assistantAdded = false
  messages.value.push(userMessage)
  input.value = ''
  sending.value = true
  const requestController = new AbortController()
  activeRequestController = requestController
  autoScrollEnabled.value = true
  await scrollToBottom(true)
  try {
    const requestMessages: AIChatMessage[] = [
      ...(systemPrompt.value.trim()
        ? [{ role: 'system' as const, content: systemPrompt.value.trim() }]
        : []),
      ...selectSlidingContext(messages.value, effectiveContextRounds.value),
    ]
    const response = await aiGatewayApi.chatStream(
      {
        tokenId: tokenId.value,
        model: model.value,
        messages: requestMessages,
        temperature: Math.min(2, Math.max(0, Number(temperature.value) || 0)),
        maxTokens: Math.min(32768, Math.max(1, Math.floor(Number(maxTokens.value) || 1))),
      },
      requestController.signal,
    )
    await consumeChatStream(response, assistantMessage, () => {
      if (assistantAdded) return
      messages.value.push(assistantMessage)
      assistantAdded = true
    })
    if (!assistantAdded) {
      assistantMessage.content = '模型未返回文本内容。'
      messages.value.push(assistantMessage)
    } else if (!assistantMessage.content) {
      assistantMessage.content = '模型未返回文本内容。'
    }
  } catch (error) {
    if (isAbortError(error)) {
      if (!assistantAdded) {
        assistantMessage.content = '已停止生成。'
        messages.value.push(assistantMessage)
      } else if (!assistantMessage.content) {
        assistantMessage.content = '已停止生成。'
      }
    } else {
      messages.value = messages.value.filter(
        (message) => message.id !== userMessage.id && message.id !== assistantMessage.id,
      )
      input.value = content
      errorMessage.value = error instanceof Error ? error.message : '聊天请求失败'
      toast.error(errorMessage.value)
    }
  } finally {
    if (activeRequestController === requestController) activeRequestController = null
    sending.value = false
    await scrollToBottom()
  }
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}

function cancelStreaming() {
  activeRequestController?.abort()
}

async function consumeChatStream(
  response: Response,
  assistantMessage: DisplayMessage,
  ensureAssistant: () => void,
) {
  if (!response.body) throw new Error('流式响应内容为空')
  const reader = response.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''
  let pendingContent = ''
  let animationFrame: number | undefined

  const flushPendingContent = () => {
    animationFrame = undefined
    if (!pendingContent) return
    ensureAssistant()
    assistantMessage.content += pendingContent
    pendingContent = ''
    void scrollToBottom()
  }

  const scheduleContentFlush = () => {
    if (animationFrame !== undefined) return
    animationFrame = window.requestAnimationFrame(flushPendingContent)
  }

  const processEvent = (eventText: string) => {
    if (eventText.length > MAX_SSE_EVENT_CHARS) throw new Error('流式响应事件过大')
    const data = eventText
      .split(/\r?\n/)
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).trimStart())
      .join('\n')
    if (!data || data === '[DONE]') return

    const payload = JSON.parse(data) as {
      id?: string
      error?: { message?: string }
      choices?: Array<{ delta?: { content?: string } }>
      usage?: { prompt_tokens: number; completion_tokens: number; total_tokens: number }
    }
    if (payload.error) throw new Error(payload.error.message || '模型流式调用失败')
    const content = payload.choices?.[0]?.delta?.content
    if (content) {
      pendingContent += content
      scheduleContentFlush()
    }
    if (payload.id && !assistantMessage.requestId) assistantMessage.requestId = payload.id
    if (payload.usage) {
      if (animationFrame !== undefined) window.cancelAnimationFrame(animationFrame)
      flushPendingContent()
      ensureAssistant()
      assistantMessage.usage = payload.usage
    }
  }

  try {
    while (true) {
      const { value, done } = await reader.read()
      buffer += decoder.decode(value, { stream: !done })
      if (buffer.length > MAX_SSE_EVENT_CHARS) throw new Error('流式响应缓冲区过大')
      const events = buffer.split(/\r?\n\r?\n/)
      buffer = events.pop() || ''
      for (const eventText of events) processEvent(eventText)
      if (done) break
    }
    if (buffer.trim()) processEvent(buffer)
    if (animationFrame !== undefined) window.cancelAnimationFrame(animationFrame)
    flushPendingContent()
  } catch (error) {
    if (animationFrame !== undefined) window.cancelAnimationFrame(animationFrame)
    await reader.cancel().catch(() => undefined)
    throw error
  } finally {
    reader.releaseLock()
  }
}

function clearConversation() {
  messages.value = []
  errorMessage.value = ''
  autoScrollEnabled.value = true
}

function handleComposerKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    void sendMessage()
  }
}

watch(tokenId, () => {
  if (!availableModels.value.includes(model.value)) model.value = availableModels.value[0] || ''
})
watch(availableModels, (models) => {
  if (!models.includes(model.value)) model.value = models[0] || ''
})
onMounted(loadOptions)
onBeforeUnmount(cancelStreaming)
</script>

<template>
  <div :class="props.embedded ? 'h-full' : '-mt-5 space-y-2'">
    <AppPageHeader
      v-if="!props.embedded"
      title="AI Chat"
      description="选择 API 密钥测试大模型多轮对话；当前会话仅保留在浏览器内存中，不保存聊天数据。"
      :loading="loadingOptions"
      @refresh="loadOptions"
    />

    <div
      v-if="errorMessage && !props.embedded"
      class="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ errorMessage }}
    </div>

    <Card
      class="flex min-w-0 gap-0 overflow-hidden py-0 shadow-sm"
      :class="
        props.embedded
          ? 'h-full min-h-0 rounded-none border-0 shadow-none'
          : 'min-h-[calc(100vh-10.5rem)]'
      "
    >
      <div class="flex flex-wrap items-center justify-between gap-2 border-b px-4 py-2.5">
        <div class="flex items-center gap-2.5">
          <div class="rounded-lg bg-primary/10 p-1.5 text-primary">
            <Sparkles class="h-4 w-4" />
          </div>
          <div>
            <div class="font-medium leading-none">临时聊天会话</div>
            <div class="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
              <span>{{ selectedToken?.name || '未选择 API 密钥' }}</span>
              <span class="text-border">/</span>
              <span>{{ model || '未选择模型' }}</span>
              <span class="text-border">/</span>
              <span>上下文 {{ effectiveContextRounds }} 轮</span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-1.5">
          <Badge variant="secondary">不保存数据</Badge>
          <Button variant="outline" size="sm" class="h-8 px-2.5" @click="configOpen = true">
            <Settings2 class="mr-1.5 h-3.5 w-3.5" />会话配置
          </Button>
          <Button
            variant="ghost"
            size="sm"
            class="h-8 px-2.5 text-muted-foreground"
            :disabled="sending || messages.length === 0"
            @click="clearConversation"
          >
            <RotateCcw class="mr-1.5 h-3.5 w-3.5" />清空会话
          </Button>
        </div>
      </div>

      <div
        v-if="errorMessage && props.embedded"
        class="border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-sm text-destructive"
      >
        {{ errorMessage }}
      </div>

      <div
        ref="messageList"
        class="min-h-0 flex-1 space-y-3.5 overflow-y-auto bg-muted/5 px-4 py-4 sm:px-5"
        @scroll.passive="handleMessageScroll"
      >
        <div
          v-if="messages.length === 0"
          class="flex h-full min-h-64 flex-col items-center justify-center text-center"
        >
          <div class="rounded-2xl border border-primary/10 bg-primary/10 p-3.5 text-primary">
            <Bot class="h-7 w-7" />
          </div>
          <h3 class="mt-3 font-semibold">开始测试大模型对话</h3>
          <p class="mt-1.5 max-w-md text-sm leading-6 text-muted-foreground">
            选择 API 密钥与模型后发送消息。多轮上下文会随每次请求发送，刷新或离开页面后即清除。
          </p>
          <Button variant="outline" size="sm" class="mt-3" @click="configOpen = true">
            <Settings2 class="mr-1.5 h-3.5 w-3.5" />打开会话配置
          </Button>
        </div>

        <div
          v-for="message in messages"
          :key="message.id"
          class="flex gap-2.5"
          :class="message.role === 'user' ? 'justify-end' : ''"
        >
          <div
            v-if="message.role === 'assistant'"
            class="mt-0.5 h-8 w-8 shrink-0 rounded-full bg-primary/10 p-2 text-primary"
          >
            <Bot class="h-4 w-4" />
          </div>
          <div class="max-w-[88%] space-y-1.5 lg:max-w-[78%]">
            <div
              class="whitespace-pre-wrap break-words rounded-2xl px-4 py-2.5 text-sm leading-6 shadow-xs"
              :class="
                message.role === 'user'
                  ? 'rounded-br-md bg-primary text-primary-foreground'
                  : 'rounded-bl-md border bg-background'
              "
            >
              {{ message.content }}
              <span
                v-if="
                  sending &&
                  message.role === 'assistant' &&
                  messages[messages.length - 1]?.id === message.id
                "
                class="ml-0.5 inline-block h-4 w-0.5 animate-pulse align-middle bg-current/70"
              />
            </div>
            <div
              v-if="message.usage"
              class="flex flex-wrap gap-2 text-[11px] text-muted-foreground"
            >
              <span>输入 {{ message.usage.prompt_tokens }}</span>
              <span>输出 {{ message.usage.completion_tokens }}</span>
              <span>总计 {{ message.usage.total_tokens }}</span>
              <span v-if="message.requestId">请求 {{ message.requestId }}</span>
            </div>
          </div>
          <div
            v-if="message.role === 'user'"
            class="mt-0.5 h-8 w-8 shrink-0 rounded-full bg-muted p-2"
          >
            <User class="h-4 w-4" />
          </div>
        </div>

        <div
          v-if="sending && messages[messages.length - 1]?.role !== 'assistant'"
          class="flex gap-2.5"
        >
          <div class="mt-0.5 h-8 w-8 rounded-full bg-primary/10 p-2 text-primary">
            <Bot class="h-4 w-4" />
          </div>
          <div
            class="flex items-center gap-2 rounded-2xl rounded-bl-md border bg-background px-4 py-2.5 text-sm text-muted-foreground shadow-xs"
          >
            <LoaderCircle class="h-4 w-4 animate-spin" />模型正在思考…
          </div>
        </div>
      </div>

      <div class="border-t bg-background p-2.5 sm:p-3">
        <div class="rounded-xl border bg-muted/20 p-2 shadow-xs">
          <div class="flex items-end gap-2">
            <Textarea
              v-model="input"
              class="min-h-16 resize-none border-0 bg-transparent shadow-none focus-visible:ring-0"
              placeholder="输入消息，Enter 发送，Shift + Enter 换行"
              :disabled="sending"
              @keydown="handleComposerKeydown"
            />
            <Button
              class="h-9 shrink-0 px-3"
              :disabled="!sending && !canSend"
              @click="sending ? cancelStreaming() : sendMessage()"
            >
              <Square v-if="sending" class="h-4 w-4 fill-current sm:mr-1.5" />
              <Send v-else class="h-4 w-4 sm:mr-1.5" />
              <span class="hidden sm:inline">{{ sending ? '停止' : '发送' }}</span>
            </Button>
          </div>
        </div>
      </div>
    </Card>

    <Dialog :open="configOpen" @update:open="configOpen = $event">
      <DialogFixedContent
        title="会话配置"
        description="选择测试密钥与模型，并调整本次临时会话参数。调用仍计入密钥额度与调用日志。"
        class="sm:max-w-xl"
      >
        <div class="space-y-4">
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label for="chat-token">API 密钥</Label>
              <select
                id="chat-token"
                v-model="tokenId"
                class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
                :disabled="loadingOptions || sending"
              >
                <option value="">请选择 API 密钥</option>
                <option v-for="token in tokens" :key="token.id" :value="String(token.id)">
                  {{ token.name }}{{ token.channelName ? ` · ${token.channelName}` : '' }}
                </option>
              </select>
            </div>

            <div class="space-y-1.5">
              <Label for="chat-model">模型</Label>
              <select
                id="chat-model"
                v-model="model"
                class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
                :disabled="!tokenId || sending"
              >
                <option value="">请选择模型</option>
                <option v-for="item in availableModels" :key="item" :value="item">
                  {{ item }}
                </option>
              </select>
            </div>
          </div>

          <div v-if="tokens.length === 0 && !loadingOptions" class="text-xs text-destructive">
            暂无启用的 API 密钥，请先创建并启用密钥。
          </div>
          <div v-else-if="tokenId && availableModels.length === 0" class="text-xs text-destructive">
            所选密钥未找到可测试模型，请检查密钥授权和渠道模型配置。
          </div>

          <div class="space-y-1.5">
            <Label for="system-prompt">系统提示词</Label>
            <Textarea
              id="system-prompt"
              v-model="systemPrompt"
              rows="4"
              class="resize-none"
              placeholder="定义助手行为，可留空"
              :disabled="sending"
            />
          </div>

          <div class="grid gap-3 sm:grid-cols-3">
            <div class="space-y-1.5">
              <Label for="chat-temperature">Temperature</Label>
              <Input
                id="chat-temperature"
                v-model.number="temperature"
                type="number"
                min="0"
                max="2"
                step="0.1"
                :disabled="sending"
              />
            </div>
            <div class="space-y-1.5">
              <Label for="chat-max-tokens">最大 Token</Label>
              <Input
                id="chat-max-tokens"
                v-model.number="maxTokens"
                type="number"
                min="1"
                max="32768"
                step="1"
                :disabled="sending"
              />
            </div>
            <div class="space-y-1.5">
              <Label for="chat-context-rounds">上下文轮数</Label>
              <Input
                id="chat-context-rounds"
                v-model.number="contextRounds"
                type="number"
                min="1"
                max="20"
                step="1"
                :disabled="sending"
              />
            </div>
          </div>

          <p class="text-xs leading-5 text-muted-foreground">
            每次仅向模型发送最近 {{ effectiveContextRounds }}
            个用户轮次（包含当前问题）；更早的消息仍保留在页面，但不会占用本次输入 Token。
          </p>

          <div class="flex justify-end">
            <Button @click="configOpen = false">完成</Button>
          </div>
        </div>
      </DialogFixedContent>
    </Dialog>
  </div>
</template>
