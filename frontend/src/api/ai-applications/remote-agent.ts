import { get, post, postStream } from '../request'
import type { PageResult } from '../ai-gateway'

export type AgentStatus = 'ONLINE' | 'OFFLINE' | 'BUSY' | 'DISABLED'
export type ConversationStatus = 'ACTIVE'
export type CLIType = 'CODEX' | 'CLAUDE'
export type MessageRole = 'USER' | 'ASSISTANT'
export type MessageStatus = 'PENDING' | 'STREAMING' | 'COMPLETED' | 'FAILED'

export interface RemoteAgent {
  id: string
  agentKey: string
  name: string
  workspaceRoot: string
  codexCommand: string
  codexArgs: string[]
  claudeCommand: string
  claudeArgs: string[]
  pollWaitSeconds: number
  requestTimeoutSeconds: number
  logFile: string
  ipAddress: string
  hostname: string
  operatingSystem: string
  architecture: string
  cpuInfo: string
  cpuUsage: number
  memoryTotal: number
  memoryUsed: number
  codexVersion: string
  agentVersion: string
  status: AgentStatus
  currentMessageId?: string
  lastSeenAt: number
  createdAt: number
  updatedAt: number
}

export interface RemoteAgentSaveParams {
  id?: string
  agentKey: string
  name: string
  workspaceRoot: string
  codexCommand: string
  codexArgs: string[]
  claudeCommand: string
  claudeArgs: string[]
  pollWaitSeconds: number
  requestTimeoutSeconds: number
  logFile: string
}

export interface RemoteAgentCredential {
  agent: RemoteAgent
  registrationToken: string
  configYaml: string
}

export interface AgentHeartbeat {
  id: string
  agentId: string
  status: AgentStatus
  cpuUsage: number
  memoryUsed: number
  occurredAt: number
}

export interface RemoteConversation {
  id: string
  agentId: string
  title: string
  cliType: CLIType
  workingDirectory: string
  status: ConversationStatus
  pinned: boolean
  pinnedAt: number
  lastMessageAt: number
  createdAt: number
  updatedAt: number
}

export interface RemoteMessage {
  id: string
  conversationId: string
  agentId: string
  sequence: number
  role: MessageRole
  status: MessageStatus
  content: string
  errorMessage: string
  createdAt: number
  updatedAt: number
}

export interface MessageChunk {
  id: string
  messageId: string
  sequence: number
  content: string
}

export interface AgentDetailResult {
  agent: RemoteAgent
  heartbeats: AgentHeartbeat[]
  conversations: RemoteConversation[]
}

export interface ConversationDetailResult {
  conversation: RemoteConversation
  messages: RemoteMessage[]
}

export interface SendMessageResult {
  userMessage: RemoteMessage
  assistantMessage: RemoteMessage
}

export interface AgentPageParams {
  page: number
  pageSize: number
  keyword?: string
  status?: AgentStatus | ''
}

export interface ConversationPageParams {
  page: number
  pageSize: number
  agentId: string
}

export const remoteAgentApi = {
  agentPage: (params: AgentPageParams) =>
    post<PageResult<RemoteAgent>>('/ai-applications/remote-agent/agent/page', params),
  getAgent: (id: string) =>
    get<AgentDetailResult>(
      `/ai-applications/remote-agent/agent/get?${new URLSearchParams({ id })}`,
    ),
  getAgentStatus: (id: string) =>
    get<RemoteAgent>(`/ai-applications/remote-agent/agent/status?${new URLSearchParams({ id })}`),
  createAgent: (params: RemoteAgentSaveParams) =>
    post<RemoteAgentCredential>('/ai-applications/remote-agent/agent/create', params),
  updateAgent: (params: RemoteAgentSaveParams) =>
    post<RemoteAgent>('/ai-applications/remote-agent/agent/update', params),
  rotateAgentToken: (id: string) =>
    post<RemoteAgentCredential>(
      `/ai-applications/remote-agent/agent/rotate-token?${new URLSearchParams({ id })}`,
      {},
    ),
  setAgentStatus: (id: string, enabled: boolean) =>
    post<boolean>('/ai-applications/remote-agent/agent/status', { id, enabled }),
  deleteAgent: (id: string) => post<boolean>('/ai-applications/remote-agent/agent/delete', { id }),
  conversationPage: (params: ConversationPageParams) =>
    post<PageResult<RemoteConversation>>('/ai-applications/remote-agent/conversation/page', params),
  createConversation: (agentId: string, cliType: CLIType, workingDirectory: string, title = '') =>
    post<RemoteConversation>('/ai-applications/remote-agent/conversation/create', {
      agentId,
      title,
      cliType,
      workingDirectory,
    }),
  getConversation: (id: string) =>
    get<ConversationDetailResult>(
      `/ai-applications/remote-agent/conversation/get?${new URLSearchParams({ id })}`,
    ),
  pinConversation: (id: string, pinned: boolean) =>
    post<RemoteConversation>('/ai-applications/remote-agent/conversation/pin', { id, pinned }),
  renameConversation: (id: string, title: string) =>
    post<RemoteConversation>('/ai-applications/remote-agent/conversation/rename', { id, title }),
  deleteConversation: (id: string) =>
    post<boolean>('/ai-applications/remote-agent/conversation/delete', { id }),
  sendMessage: (conversationId: string, content: string) =>
    post<SendMessageResult>('/ai-applications/remote-agent/conversation/message/send', {
      conversationId,
      content,
    }),
  streamMessage: (messageId: string, afterSequence: number, signal?: AbortSignal) =>
    postStream(
      '/ai-applications/remote-agent/conversation/message/stream',
      { messageId, afterSequence },
      signal,
    ),
}
