import { useUserStore } from '@/stores/user'
import type { LoginResult } from '@/types/auth'

export interface ApiResponse<T> {
  code: string
  data: T
  msg: string
}

export class ApiError extends Error {
  constructor(
    message: string,
    readonly code: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

const DEFAULT_API_BASE = 'http://localhost:8090/v1'
const API_BASE = import.meta.env.VITE_API_BASE || DEFAULT_API_BASE
let refreshPromise: Promise<boolean> | null = null

export function apiUrl(path: string) {
  return `${API_BASE.replace(/\/+$/, '')}/${path.replace(/^\/+/, '')}`
}

function redirectToLogin() {
  if (!window.location.hash.startsWith('#/login')) {
    window.location.hash = '#/login'
  }
}

async function refreshAccessToken(): Promise<boolean> {
  const userStore = useUserStore()
  if (!userStore.refreshToken) return false

  try {
    const response = await fetch(apiUrl('/refresh-token'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: userStore.refreshToken }),
    })
    const payload = (await response.json()) as ApiResponse<LoginResult>
    if (!response.ok || payload.code !== '1' || !payload.data) return false
    userStore.setSession(payload.data)
    return true
  } catch {
    return false
  }
}

async function ensureFreshToken(): Promise<boolean> {
  if (!refreshPromise) {
    refreshPromise = refreshAccessToken().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

async function executeRequest<T>(url: string, options: RequestInit, canRetry: boolean): Promise<T> {
  const userStore = useUserStore()
  const headers = new Headers(options.headers)
  headers.set('Content-Type', headers.get('Content-Type') || 'application/json')
  if (userStore.token) headers.set('accessToken', userStore.token)

  const response = await fetch(apiUrl(url), { ...options, headers })
  const payload = (await response.json()) as ApiResponse<T>

  if (payload.code === '2' && canRetry && url !== '/refresh-token') {
    if (await ensureFreshToken()) return executeRequest<T>(url, options, false)
    userStore.logout()
    redirectToLogin()
    throw new ApiError('登录状态已过期，请重新登录', payload.code)
  }

  if (payload.code === '3') {
    userStore.logout()
    redirectToLogin()
  }

  if (!response.ok || payload.code !== '1') {
    throw new ApiError(payload.msg || '请求失败', payload.code)
  }
  return payload.data
}

async function executeStreamRequest(
  url: string,
  options: RequestInit,
  canRetry: boolean,
): Promise<Response> {
  const userStore = useUserStore()
  const headers = new Headers(options.headers)
  headers.set('Content-Type', headers.get('Content-Type') || 'application/json')
  headers.set('Accept', 'text/event-stream')
  if (userStore.token) headers.set('accessToken', userStore.token)

  const response = await fetch(apiUrl(url), { ...options, headers })
  const contentType = response.headers.get('Content-Type') || ''
  if (response.ok && contentType.includes('text/event-stream')) return response

  const responseText = await response.text()
  let payload: Partial<ApiResponse<unknown>> & { error?: { message?: string } } = {}
  if (responseText) {
    try {
      payload = JSON.parse(responseText) as typeof payload
    } catch {
      payload = {}
    }
  }

  if (payload.code === '2' && canRetry && url !== '/refresh-token') {
    if (await ensureFreshToken()) return executeStreamRequest(url, options, false)
    userStore.logout()
    redirectToLogin()
    throw new ApiError('登录状态已过期，请重新登录', payload.code)
  }

  if (payload.code === '3') {
    userStore.logout()
    redirectToLogin()
  }

  const message =
    payload.msg ||
    payload.error?.message ||
    responseText ||
    `流式请求失败 (${response.status || '无响应'})`
  throw new ApiError(message, payload.code || String(response.status || 0))
}

export function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  return executeRequest<T>(url, options, true)
}

export function post<T>(url: string, body: unknown): Promise<T> {
  return request<T>(url, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function postStream(url: string, body: unknown, signal?: AbortSignal): Promise<Response> {
  return executeStreamRequest(
    url,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
    true,
  )
}

export function get<T>(url: string): Promise<T> {
  return request<T>(url, { method: 'GET' })
}
