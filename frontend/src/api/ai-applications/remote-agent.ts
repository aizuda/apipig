import { get, post, postStream } from '../request'
import type { PageResult } from '../ai-gateway'

export type AgentStatus = 'ONLINE' | 'OFFLINE' | 'BUSY' | 'DISABLED'
export type TaskStatus = 'PENDING' | 'RUNNING' | 'SUCCESS' | 'FAILED' | 'CANCELLED'

export interface RemoteAgent {
  id: string
  agentKey: string
  name: string
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
  currentTaskId?: string
  lastSeenAt: number
  createdAt: number
}

export interface AgentHeartbeat {
  id: string
  agentId: string
  status: AgentStatus
  cpuUsage: number
  memoryUsed: number
  occurredAt: number
}

export interface RemoteTask {
  id: string
  name: string
  agentId: string
  workspaceId?: string
  repositoryUrl: string
  workingDir: string
  prompt: string
  status: TaskStatus
  result: string
  errorMessage: string
  changedFiles: string
  startedAt: number
  finishedAt: number
  createdAt: number
}

export interface TaskLog {
  id: string
  taskId: string
  agentId: string
  sequence: number
  stream: 'stdout' | 'stderr'
  content: string
  createdAt: number
}

export interface RemoteWorkspace {
  id: string
  agentId: string
  name: string
  path: string
  repositoryUrl: string
  status: number
}

export interface RemoteCommand {
  id: string
  taskId: string
  type: string
  status: string
  attempt: number
  dispatchedAt: number
  acknowledgedAt: number
}

export interface AgentDetailResult {
  agent: RemoteAgent
  heartbeats: AgentHeartbeat[]
  tasks: RemoteTask[]
}

export interface TaskDetailResult {
  task: RemoteTask
  workspace: RemoteWorkspace
  commands: RemoteCommand[]
  logs: TaskLog[]
}

export interface AgentPageParams {
  page: number
  pageSize: number
  keyword?: string
  status?: AgentStatus | ''
}

export interface TaskPageParams {
  page: number
  pageSize: number
  agentId?: string
  status?: TaskStatus | ''
  keyword?: string
}

export interface TaskCreateParams {
  name: string
  agentId: string
  repositoryUrl: string
  workingDir?: string
  prompt: string
}

export const remoteAgentApi = {
  agentPage: (params: AgentPageParams) =>
    post<PageResult<RemoteAgent>>('/ai-applications/remote-agent/agent/page', params),
  getAgent: (id: string) =>
    get<AgentDetailResult>(
      `/ai-applications/remote-agent/agent/get?${new URLSearchParams({ id })}`,
    ),
  taskPage: (params: TaskPageParams) =>
    post<PageResult<RemoteTask>>('/ai-applications/remote-agent/task/page', params),
  createTask: (params: TaskCreateParams) =>
    post<RemoteTask>('/ai-applications/remote-agent/task/create', params),
  getTask: (id: string) =>
    get<TaskDetailResult>(`/ai-applications/remote-agent/task/get?${new URLSearchParams({ id })}`),
  cancelTask: (id: string) => post<boolean>('/ai-applications/remote-agent/task/cancel', { id }),
  streamTaskLogs: (taskId: string, afterSequence: number, signal?: AbortSignal) =>
    postStream('/ai-applications/remote-agent/task/log/stream', { taskId, afterSequence }, signal),
}
