import { ref } from 'vue'
import { remoteAgentApi, type RemoteAgentEvent } from '@/api/ai-applications/remote-agent'

export type RemoteAgentStreamEvent =
  | { type: 'ready' }
  | { type: 'agent'; data: RemoteAgentEvent }

type Listener = (event: RemoteAgentStreamEvent) => void

const listeners = new Set<Listener>()
const connected = ref(false)
let controller: AbortController | null = null
let reconnectTimer: number | undefined
let reconnectDelay = 1500

function emit(event: RemoteAgentStreamEvent) {
  listeners.forEach((listener) => listener(event))
}

function parseEvent(frame: string) {
  let type = 'message'
  const data: string[] = []
  for (const line of frame.split(/\r?\n/)) {
    if (line.startsWith('event:')) type = line.slice(6).trim()
    if (line.startsWith('data:')) data.push(line.slice(5).trimStart())
  }
  if (!data.length) return
  if (type === 'ready') {
    connected.value = true
    reconnectDelay = 1500
    emit({ type: 'ready' })
    return
  }
  if (type !== 'agent') return
  try {
    emit({ type: 'agent', data: JSON.parse(data.join('\n')) as RemoteAgentEvent })
  } catch {
    // Ignore malformed events; the next ready event triggers a full sync.
  }
}

async function consume(signal: AbortSignal) {
  try {
    const response = await remoteAgentApi.streamAgentEvents(signal)
    const reader = response.body?.getReader()
    if (!reader) throw new Error('Agent SSE stream is unavailable')
    const decoder = new TextDecoder()
    let buffer = ''
    while (!signal.aborted) {
      const { done, value } = await reader.read()
      buffer += decoder.decode(value || new Uint8Array(), { stream: !done }).replace(/\r\n/g, '\n')
      let boundary = buffer.indexOf('\n\n')
      while (boundary >= 0) {
        parseEvent(buffer.slice(0, boundary))
        buffer = buffer.slice(boundary + 2)
        boundary = buffer.indexOf('\n\n')
      }
      if (done) break
    }
  } catch {
    if (!signal.aborted) {
      // Reconnect is scheduled below without surfacing transient network errors to every view.
    }
  } finally {
    if (controller?.signal === signal) {
      controller = null
      connected.value = false
      scheduleReconnect()
    }
  }
}

function scheduleReconnect() {
  if (!listeners.size || reconnectTimer !== undefined) return
  const delay = reconnectDelay
  reconnectDelay = Math.min(reconnectDelay * 2, 30_000)
  reconnectTimer = window.setTimeout(() => {
    reconnectTimer = undefined
    ensureConnection()
  }, delay)
}

function ensureConnection() {
  if (!listeners.size || controller) return
  controller = new AbortController()
  void consume(controller.signal)
}

export function subscribeRemoteAgentEvents(listener: Listener) {
  listeners.add(listener)
  ensureConnection()
  return () => {
    listeners.delete(listener)
    if (!listeners.size) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = undefined
      controller?.abort()
      controller = null
      connected.value = false
      reconnectDelay = 1500
    }
  }
}

export function useRemoteAgentEvents() {
  return { connected, subscribe: subscribeRemoteAgentEvents }
}
