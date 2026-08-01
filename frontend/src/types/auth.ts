export interface PublicKeyInfo {
  uuid: string
  publicKey: string
}

export interface CaptchaInfo {
  token: string
  base64Image: string
}

export interface LoginParams {
  uuid: string
  username: string
  password: string
  captchaToken: string
  captchaCode: string
}

export type LoginType = 'account' | 'api_token'

export interface TokenAuthorizationParams {
  token: string
  captchaToken: string
  captchaCode: string
}

export interface BackendUser {
  id: string
  username: string
  realName?: string
  nickName?: string
  avatar?: string
  sex?: number
  phone?: string
  phoneVerified?: number
  email?: string
  emailVerified?: number
  status?: number
  jobNum?: string
  loginTime?: number
  pwdTime?: number
  createdAt?: number
  updatedAt?: number
}

export interface UserInfo extends BackendUser {
  name: string
}

export interface LoginResult {
  user: BackendUser
  token: string
  refreshToken?: string
  loginType: LoginType
  expiresAt?: number
}
