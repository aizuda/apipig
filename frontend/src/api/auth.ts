import { get, post } from './request'
import type {
  CaptchaInfo,
  LoginParams,
  LoginResult,
  PublicKeyInfo,
  TokenLoginParams,
} from '@/types/auth'

export function fetchPublicKey(): Promise<PublicKeyInfo> {
  return post<PublicKeyInfo>('/public-key', {})
}

export function fetchCaptcha(uuid: string): Promise<CaptchaInfo> {
  return get<CaptchaInfo>(`/captcha?uuid=${encodeURIComponent(uuid)}`)
}

export function fetchLogin(params: LoginParams): Promise<LoginResult> {
  return post<LoginResult>('/login', params)
}

export function fetchTokenLogin(params: TokenLoginParams): Promise<LoginResult> {
  return post<LoginResult>('/token-login', params)
}
