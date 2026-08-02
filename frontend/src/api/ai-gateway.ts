import { get, post, postStream } from './request'

export interface PageResult<T> {
  total: number
  page: number
  pageSize: number
  records: T[]
}

export interface GatewayPageParams {
  page: number
  pageSize: number
  keyword?: string
}

export interface CallLogPageParams extends GatewayPageParams {
  success?: number
  startAt?: number
  endAt?: number
}

export interface Provider {
  id?: string
  name: string
  code: string
  icon?: string
  protocol: string
  baseUrl: string
  models: string
  timeoutMs: number
  status: number
  remark?: string
}

export interface Channel {
  id?: string
  providerId: string
  providerName?: string
  availableModels?: string[]
  providerModels?: string[]
  name: string
  modelPricing: string
  priority: number
  weight: number
  proxyId?: string
  proxyName?: string
  rpm: number
  tpm: number
  costMultiplier: number
  status: number
  remark?: string
}

export interface ChannelAccount {
  id?: string
  channelId: string
  channelName?: string
  providerName?: string
  name: string
  apiKey: string
  models: string
  status: number
  remark?: string
}

export interface ModelPricingRule {
  model: string
  tokenPriceUnit?: 'perMillion' | 'perToken'
  inputPricePerMTokens: number
  outputPricePerMTokens: number
  cacheReadPricePerMTokens: number
  cacheWritePricePerMTokens: number
  inputImagePricePerImage: number
  outputImagePricePerImage: number
  inputImagePricePerMTokens: number
  outputImagePricePerMTokens: number
}

export interface ChannelAccountPageParams {
  page: number
  pageSize: number
  keyword?: string
}

export interface AccessTokenIPRule {
  enabled: boolean
  whitelist: string[]
  blacklist: string[]
}

export interface AccessTokenRateLimitRule {
  enabled: boolean
  fiveHourAmount: number
  dayAmount: number
  sevenDayAmount: number
}

export interface AccessToken {
  id?: string
  channelId?: string
  channelName?: string
  providerName?: string
  name: string
  token?: string
  models: string
  ipRule: string
  rateLimitRule: string
  rpm: number
  tpm: number
  quotaAmount: number
  usedAmount?: number
  successCount?: number
  failureCount?: number
  promptTokensTotal?: number
  completionTokensTotal?: number
  reasoningTokensTotal?: number
  cacheReadTokensTotal?: number
  cacheWriteTokensTotal?: number
  lastUsedAt?: number
  expireAt: number
  status: number
  remark?: string
  tagIds?: string[]
  tags?: AccessTokenTag[]
}

export interface AccessTokenSaveResult {
  success: boolean
  token?: string
}

export interface AccessTokenStatisticRecord {
  tokenId: string
  tokenName: string
  status: number
  callCount: number
  successCount: number
  failureCount: number
  promptTokens: number
  completionTokens: number
  reasoningTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
  totalTokens: number
  lastUsedAt: number
}

export interface AccessTokenModelStatistic {
  model: string
  callCount: number
  successCount: number
  failureCount: number
  promptTokens: number
  completionTokens: number
  reasoningTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
  totalTokens: number
  cost: number
  avgLatencyMs: number
  lastUsedAt: number
}

export interface AccessTokenStatistics {
  startAt: number
  endAt: number
  total: number
  page: number
  pageSize: number
  tokenCount: number
  activeTokenCount: number
  callCount: number
  successCount: number
  failureCount: number
  totalTokens: number
  modelStatistics: AccessTokenModelStatistic[]
  items: AccessTokenStatisticRecord[]
}

export interface AccessTokenStatisticsParams {
  page?: number
  pageSize?: number
  keyword?: string
  tagId?: string
  startAt?: number
  endAt?: number
}

export interface AccessTokenTag {
  id?: string
  name: string
  remark: string
  sort: number
  createdAt?: number
  updatedAt?: number
}

export interface ProxyNode {
  id?: string
  name: string
  scheme: string
  host: string
  port: number
  username?: string
  password?: string
  region?: string
  status: number
  remark?: string
}

export interface CallLog {
  id: string
  requestId: string
  accessTokenId: string
  providerId: string
  channelId: string
  model: string
  path: string
  method: string
  clientIp: string
  statusCode: number
  promptTokens: number
  completionTokens: number
  reasoningTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
  inputImages: number
  outputImages: number
  inputImageTokens: number
  outputImageTokens: number
  totalTokens: number
  pricingModel: string
  pricingSnapshot: string
  standardCost: number
  cost: number
  standardCostMicroUsd: number
  costMicroUsd: number
  costMultiplier: number
  latencyMs: number
  success: number
  errorMessage: string
  createdAt: number
}

export interface GatewaySummary {
  providerCount: number
  channelCount: number
  tokenCount: number
  proxyCount: number
  callCount: number
  successCount: number
  errorCount: number
  totalTokens: number
  promptTokens: number
  completionTokens: number
  reasoningTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
  standardCost: number
  totalCost: number
  modelDistribution: GatewayModelDistribution[]
  tokenTrend: GatewayTokenTrend[]
  channelStatistics: GatewayChannelStatistic[]
}

export interface GatewayChannelStatistic {
  channelId: string
  channelName: string
  callCount: number
  successCount: number
  successRate: number
  tokenCount: number
  cost: number
  avgLatencyMs: number
}

export interface GatewaySummaryParams {
  startAt?: number
  endAt?: number
}

export interface GatewayModelDistribution {
  model: string
  callCount: number
  tokenCount: number
  cost: number
}

export interface GatewayTokenTrend {
  date: string
  promptTokens: number
  completionTokens: number
  cacheTokens: number
  totalTokens: number
}

export interface AIChatMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
}

export interface AIChatParams {
  tokenId: string
  model: string
  messages: AIChatMessage[]
  temperature?: number
  maxTokens?: number
}

export interface AIChatResponse {
  id: string
  model: string
  request_id: string
  choices: Array<{
    index: number
    message: {
      role: 'assistant'
      content: string | null
    }
    finish_reason: string
  }>
  usage: {
    prompt_tokens: number
    completion_tokens: number
    total_tokens: number
  }
}

export const aiGatewayApi = {
  summary: (params: GatewaySummaryParams = {}) => {
    const query = new URLSearchParams()
    if (params.startAt) query.set('startAt', String(params.startAt))
    if (params.endAt) query.set('endAt', String(params.endAt))
    const suffix = query.toString()
    return get<GatewaySummary>(`/ai/gateway/summary${suffix ? `?${suffix}` : ''}`)
  },
  chat: (data: AIChatParams) => post<AIChatResponse>('/ai/gateway/chat', data),
  chatStream: (data: AIChatParams, signal?: AbortSignal) =>
    postStream('/ai/gateway/chat/stream', data, signal),

  providerPage: (params: GatewayPageParams = { page: 1, pageSize: 100 }) =>
    post<PageResult<Provider>>('/ai/gateway/provider/page', params),
  providerList: () => post<Provider[]>('/ai/gateway/provider/list', {}),
  saveProvider: (data: Provider) => post<boolean>('/ai/gateway/provider/save', data),
  changeProviderStatus: (id: string, status: number) =>
    post<boolean>('/ai/gateway/provider/change-status', { id, status }),
  deleteProvider: (id: string) => post<boolean>('/ai/gateway/provider/delete', { ids: [id] }),

  channelPage: (params: GatewayPageParams = { page: 1, pageSize: 100 }) =>
    post<PageResult<Channel>>('/ai/gateway/channel/page', params),
  saveChannel: (data: Channel) =>
    post<boolean>('/ai/gateway/channel/save', {
      ...data,
      proxyId: data.proxyId?.trim() || undefined,
    }),
  changeChannelStatus: (id: string, status: number) =>
    post<boolean>('/ai/gateway/channel/change-status', { id, status }),
  deleteChannel: (id: string) => post<boolean>('/ai/gateway/channel/delete', { ids: [id] }),

  channelAccountPage: (params: ChannelAccountPageParams) =>
    post<PageResult<ChannelAccount>>('/ai/gateway/channel-account/page', params),
  saveChannelAccount: (data: ChannelAccount) => {
    const payload = { ...data }
    delete payload.channelName
    delete payload.providerName
    return post<boolean>('/ai/gateway/channel-account/save', payload)
  },
  changeChannelAccountStatus: (id: string, status: number) =>
    post<boolean>('/ai/gateway/channel-account/change-status', { id, status }),
  deleteChannelAccount: (id: string) =>
    post<boolean>('/ai/gateway/channel-account/delete', { ids: [id] }),

  tokenPage: (params: GatewayPageParams = { page: 1, pageSize: 100 }) =>
    post<PageResult<AccessToken>>('/ai/gateway/token/page', params),
  tokenStatistics: (params: AccessTokenStatisticsParams = {}) =>
    post<AccessTokenStatistics>('/ai/gateway/token/statistics', params),
  saveToken: (data: AccessToken) => {
    const payload = {
      ...data,
      channelId: data.channelId?.trim() || undefined,
      ipRule: data.ipRule || '{"enabled":false,"whitelist":[],"blacklist":[]}',
      rateLimitRule:
        data.rateLimitRule ||
        '{"enabled":false,"fiveHourAmount":0,"dayAmount":0,"sevenDayAmount":0}',
    }
    delete payload.token
    delete payload.channelName
    delete payload.providerName
    delete payload.tags
    return post<AccessTokenSaveResult>('/ai/gateway/token/save', payload)
  },
  updateTokenTags: (id: string, tagIds: string[]) =>
    post<boolean>('/ai/gateway/token/update-tags', { id, tagIds }),
  changeTokenStatus: (id: string, status: number) =>
    post<boolean>('/ai/gateway/token/change-status', { id, status }),
  deleteToken: (id: string) => post<boolean>('/ai/gateway/token/delete', { ids: [id] }),

  tokenTagList: () => post<AccessTokenTag[]>('/ai/gateway/token-tag/list', {}),
  saveTokenTag: (data: AccessTokenTag) => post<boolean>('/ai/gateway/token-tag/save', data),
  deleteTokenTag: (id: string) => post<boolean>('/ai/gateway/token-tag/delete', { ids: [id] }),
  sortTokenTags: (ids: string[]) => post<boolean>('/ai/gateway/token-tag/sort', { ids }),

  proxyPage: (params: GatewayPageParams = { page: 1, pageSize: 100 }) =>
    post<PageResult<ProxyNode>>('/ai/gateway/proxy/page', params),
  saveProxy: (data: ProxyNode) => post<boolean>('/ai/gateway/proxy/save', data),
  changeProxyStatus: (id: string, status: number) =>
    post<boolean>('/ai/gateway/proxy/change-status', { id, status }),
  deleteProxy: (id: string) => post<boolean>('/ai/gateway/proxy/delete', { ids: [id] }),

  logPage: (params: CallLogPageParams = { page: 1, pageSize: 50 }) =>
    post<PageResult<CallLog>>('/ai/gateway/log/page', params),
}
