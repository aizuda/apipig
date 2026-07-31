import { get, post } from '../request'
import type { PageResult } from '../ai-gateway'

export interface ReviewProject {
  id?: string
  name: string
  provider: 'github' | 'gitlab' | 'gitee'
  repositoryUrl: string
  defaultBranch: string
  webhookKey?: string
  accessTokenId: string
  model: string
  reviewPrompt: string
  branchPattern: string
  ignorePatterns: string
  pushEnabled: boolean
  pullRequestEnabled: boolean
  status: number
  remark: string
  webhookSecret?: string
  repositoryToken?: string
  rotateWebhookSecret?: boolean
}

export interface ReviewProjectSaveResult {
  project: ReviewProject
  webhookSecret?: string
  webhookUrl: string
}

export interface ReviewProjectPageParams {
  page: number
  pageSize: number
  keyword?: string
  provider?: string
  status?: number
}

export interface ReviewTaskPageParams {
  page: number
  pageSize: number
  projectId?: string
  eventType?: string
  status?: string
  keyword?: string
}

export interface ReviewTask {
  id: string
  projectId: string
  eventType: string
  repositoryName: string
  ref: string
  baseSha: string
  headSha: string
  author: string
  title: string
  status: string
  attempt: number
  changedFiles: number
  additions: number
  deletions: number
  riskLevel: string
  summary: string
  report: string
  diffTruncated: boolean
  errorMessage: string
  createdAt: number
  finishedAt: number
}

export const codeReviewApi = {
  projectPage: (params: ReviewProjectPageParams = { page: 1, pageSize: 100 }) =>
    post<PageResult<ReviewProject>>('/ai-applications/code-review/project/page', params),
  saveProject: (project: ReviewProject) =>
    post<ReviewProjectSaveResult>('/ai-applications/code-review/project/save', project),
  deleteProject: (id: string) =>
    post<boolean>('/ai-applications/code-review/project/delete', { ids: [id] }),
  taskPage: (params: ReviewTaskPageParams) =>
    post<PageResult<ReviewTask>>('/ai-applications/code-review/task/page', params),
  getTask: (id: string) => get<ReviewTask>(`/ai-applications/code-review/task/get?id=${id}`),
  retryTask: (id: string) => post<boolean>('/ai-applications/code-review/task/retry', { id }),
}
