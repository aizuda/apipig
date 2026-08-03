<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ArrowDownToLine, Eraser, Radio } from '@lucide/vue'
import { Badge, Button } from '@tabtab/ui'
import { remoteAgentApi, type TaskLog, type TaskStatus } from '@/api/ai-applications/remote-agent'

const props = withDefaults(
  defineProps<{
    taskId: string
    status: TaskStatus
    initialLogs?: TaskLog[]
  }>(),
  { initialLogs: () => [] },
)

const emit = defineEmits<{
  done: []
}>()

const logs = ref<TaskLog[]>([])
const connected = ref(false)
const follow = ref(true)
const viewport = ref<HTMLElement | null>(null)
const lastSequence = ref(0)
let controller: AbortController | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false

function resetLogs() {
  logs.value = [...props.initialLogs].sort((a, b) => a.sequence - b.sequence)
  lastSequence.value = logs.value.reduce((max, item) => Math.max(max, item.sequence), 0)
}

async function scrollToBottom() {
  if (!follow.value) return
  await nextTick()
  if (viewport.value) viewport.value.scrollTop = viewport.value.scrollHeight
}

function appendLog(log: TaskLog) {
  if (log.sequence <= lastSequence.value) return
  logs.value.push(log)
  lastSequence.value = log.sequence
  void scrollToBottom()
}

function parseFrame(frame: string) {
  let event = 'message'
  const data: string[] = []
  for (const line of frame.split(/\r?\n/)) {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    if (line.startsWith('data:')) data.push(line.slice(5).trimStart())
  }
  if (!data.length) return false
  if (event === 'log') appendLog(JSON.parse(data.join('\n')) as TaskLog)
  if (event === 'done') {
    connected.value = false
    emit('done')
    return true
  }
  return false
}

function clearLogs() {
  logs.value = []
}

function isActive() {
  return props.status === 'PENDING' || props.status === 'RUNNING'
}

function clearReconnectTimer() {
  if (reconnectTimer) clearTimeout(reconnectTimer)
  reconnectTimer = null
}

function scheduleReconnect(delay = 1500) {
  clearReconnectTimer()
  if (disposed || !isActive()) return
  reconnectTimer = setTimeout(() => void connect(), delay)
}

async function connect() {
  clearReconnectTimer()
  controller?.abort()
  if (disposed || !props.taskId) return
  const requestController = new AbortController()
  controller = requestController
  let streamDone = false
  try {
    connected.value = true
    const response = await remoteAgentApi.streamTaskLogs(
      props.taskId,
      lastSequence.value,
      requestController.signal,
    )
    const reader = response.body?.getReader()
    if (!reader) throw new Error('日志流不可读')
    const decoder = new TextDecoder()
    let buffer = ''
    while (true) {
      const { done, value } = await reader.read()
      buffer += decoder.decode(value || new Uint8Array(), { stream: !done }).replace(/\r\n/g, '\n')
      let boundary = buffer.indexOf('\n\n')
      while (boundary >= 0) {
        streamDone = parseFrame(buffer.slice(0, boundary)) || streamDone
        buffer = buffer.slice(boundary + 2)
        boundary = buffer.indexOf('\n\n')
      }
      if (done) break
    }
  } catch (error) {
    if (!(error instanceof DOMException && error.name === 'AbortError')) scheduleReconnect()
  } finally {
    if (controller === requestController) {
      controller = null
      connected.value = false
      if (!streamDone) scheduleReconnect()
    }
  }
}

watch(
  () => props.taskId,
  () => {
    resetLogs()
    void connect()
  },
)

watch(
  () => props.status,
  () => {
    if (isActive()) {
      if (!connected.value) scheduleReconnect(0)
      return
    }
    clearReconnectTimer()
    controller?.abort()
  },
)

onMounted(() => {
  resetLogs()
  void scrollToBottom()
  void connect()
})
onBeforeUnmount(() => {
  disposed = true
  clearReconnectTimer()
  controller?.abort()
})
</script>

<template>
  <section class="overflow-hidden rounded-md border bg-[#111315] text-[#e7e9ea]">
    <header class="flex h-11 items-center justify-between border-b border-white/10 px-3">
      <div class="flex items-center gap-2 text-xs">
        <Radio class="h-3.5 w-3.5" :class="connected ? 'text-emerald-400' : 'text-zinc-500'" />
        <span class="font-medium">执行日志</span>
        <Badge variant="outline" class="border-white/15 text-[10px] text-zinc-300">
          {{ connected ? '实时' : status }}
        </Badge>
      </div>
      <div class="flex items-center gap-1">
        <Button
          size="icon"
          variant="ghost"
          class="h-8 w-8 text-zinc-300 hover:bg-white/10 hover:text-white"
          title="跟随最新日志"
          @click="follow = !follow"
        >
          <ArrowDownToLine class="h-4 w-4" :class="follow ? 'text-emerald-400' : ''" />
        </Button>
        <Button
          size="icon"
          variant="ghost"
          class="h-8 w-8 text-zinc-300 hover:bg-white/10 hover:text-white"
          title="清空当前显示"
          @click="clearLogs"
        >
          <Eraser class="h-4 w-4" />
        </Button>
      </div>
    </header>
    <div
      ref="viewport"
      class="h-[420px] overflow-auto p-4 font-mono text-xs leading-5"
      @scroll="
        follow = Boolean(
          viewport && viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight < 32,
        )
      "
    >
      <div v-if="logs.length" class="space-y-0.5">
        <div
          v-for="log in logs"
          :key="log.sequence"
          class="grid grid-cols-[3rem_minmax(0,1fr)] gap-2"
          :class="log.stream === 'stderr' ? 'text-red-300' : 'text-zinc-200'"
        >
          <span class="select-none text-right text-zinc-600">{{ log.sequence }}</span>
          <span class="whitespace-pre-wrap break-all">{{ log.content }}</span>
        </div>
      </div>
      <div v-else class="flex h-full items-center justify-center text-zinc-600">暂无执行日志</div>
    </div>
  </section>
</template>
