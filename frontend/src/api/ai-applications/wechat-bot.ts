import { get, post } from '../request'
import type { PageResult } from '../ai-gateway'

export type WechatBotStatus =
  | 'CONNECTING'
  | 'ONLINE'
  | 'OFFLINE'
  | 'DISABLED'
  | 'SESSION_EXPIRED'
  | 'ERROR'

export interface WechatBot {
  id: string
  name: string
  botId: string
  baseUrl: string
  iLinkUserId: string
  status: WechatBotStatus
  enabled: boolean
  lastError: string
  messageCount: number
  lastMessageAt: number
  connectedAt: number
  createdAt: number
}

export interface WechatContact {
  id: string
  userId: string
  lastMessage: string
  messageCount: number
  lastActiveAt: number
  canSend: boolean
}

export interface WechatMessage {
  id: string
  externalMessageId: string
  direction: 'inbound' | 'outbound'
  userId: string
  content: string
  contentType: 'text' | 'image' | 'voice' | 'file' | 'video' | 'unknown'
  sessionId: string
  groupId: string
  occurredAt: number
}

export interface BindStartResult {
  sessionId: string
  qrCode: string
}

export interface BindStatusResult {
  status: 'WAITING' | 'SCANNED' | 'EXPIRED' | 'CONFIRMED' | string
  qrCode?: string
  bot?: WechatBot
}

export interface WechatWebhookCredentials {
  webhookUrl: string
  webhookSecret?: string
}

export const wechatBotApi = {
  page: (params: { page: number; pageSize: number; keyword?: string; status?: string }) =>
    post<PageResult<WechatBot>>('/ai-applications/wechat-bot/bot/page', params),
  list: (params: { name?: string; status?: string; enabled?: boolean }) =>
    post<WechatBot[]>('/ai-applications/wechat-bot/bot/list', params),
  startBind: (params: { name?: string; botId?: string }) =>
    post<BindStartResult>('/ai-applications/wechat-bot/bot/bind/start', params),
  pollBind: (sessionId: string) =>
    get<BindStatusResult>(
      `/ai-applications/wechat-bot/bot/bind/status/${encodeURIComponent(sessionId)}`,
    ),
  rename: (id: string, name: string) =>
    post<boolean>('/ai-applications/wechat-bot/bot/rename', { id, name }),
  reconnect: (id: string) => post<boolean>('/ai-applications/wechat-bot/bot/reconnect', { id }),
  setEnabled: (id: string, enabled: boolean) =>
    post<boolean>('/ai-applications/wechat-bot/bot/enabled', { id, enabled }),
  delete: (id: string) => post<boolean>('/ai-applications/wechat-bot/bot/delete', { id }),
  contacts: (botId: string) => {
    const query = new URLSearchParams({ botId })
    return get<WechatContact[]>(`/ai-applications/wechat-bot/bot/contacts?${query}`)
  },
  messages: (botId: string, userId: string, page = 1, pageSize = 100) =>
    post<PageResult<WechatMessage>>('/ai-applications/wechat-bot/bot/messages', {
      botId,
      userId,
      page,
      pageSize,
    }),
  send: (botId: string, userId: string, content: string) =>
    post<WechatMessage>('/ai-applications/wechat-bot/bot/send', { botId, userId, content }),
  webhookCredentials: (id: string, rotateSecret = false) =>
    post<WechatWebhookCredentials>('/ai-applications/wechat-bot/bot/webhook/credentials', {
      id,
      rotateSecret,
    }),
}
