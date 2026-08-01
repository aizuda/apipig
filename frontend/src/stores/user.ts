import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { BackendUser, LoginResult, LoginType, UserInfo } from '@/types/auth'

const TOKEN_KEY = 'apipig_token'
const REFRESH_TOKEN_KEY = 'apipig_refreshToken'
const USER_KEY = 'apipig_userInfo'
const LOGIN_TYPE_KEY = 'apipig_loginType'
const EXPIRES_AT_KEY = 'apipig_expiresAt'
export const DEFAULT_AVATAR_URL = '/avatar.jpg'

function normalizeUser(user: BackendUser): UserInfo {
  return {
    ...user,
    name: user.realName || user.nickName || user.username,
  }
}

function getStoredUser(): UserInfo | null {
  const value = localStorage.getItem(USER_KEY)
  if (!value) return null
  try {
    return JSON.parse(value) as UserInfo
  } catch {
    localStorage.removeItem(USER_KEY)
    return null
  }
}

export const useUserStore = defineStore('user', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const refreshToken = ref<string | null>(localStorage.getItem(REFRESH_TOKEN_KEY))
  const user = ref<UserInfo | null>(getStoredUser())
  const loginType = ref<LoginType>(
    localStorage.getItem(LOGIN_TYPE_KEY) === 'api_token' ? 'api_token' : 'account',
  )
  const expiresAt = ref(Number(localStorage.getItem(EXPIRES_AT_KEY)) || 0)
  const permissions = ref<string[]>([])

  const isAuthenticated = computed(() => Boolean(token.value))
  const isAPITokenSession = computed(() => loginType.value === 'api_token')
  const displayName = computed(
    () =>
      user.value?.nickName?.trim() ||
      user.value?.realName?.trim() ||
      user.value?.username?.trim() ||
      'Admin',
  )
  const displayEmail = computed(() => user.value?.email?.trim() || '')
  const defaultAvatar = computed(() => user.value?.avatar?.trim() || DEFAULT_AVATAR_URL)
  const avatarFallback = computed(() => {
    return Array.from(displayName.value)[0]?.toLocaleUpperCase() || 'A'
  })

  function setSession(session: LoginResult) {
    const normalizedUser = normalizeUser(session.user)
    token.value = session.token
    refreshToken.value = session.refreshToken || null
    user.value = normalizedUser
    loginType.value = session.loginType || 'account'
    expiresAt.value = session.expiresAt || 0
    localStorage.setItem(TOKEN_KEY, session.token)
    if (session.refreshToken) localStorage.setItem(REFRESH_TOKEN_KEY, session.refreshToken)
    else localStorage.removeItem(REFRESH_TOKEN_KEY)
    localStorage.setItem(USER_KEY, JSON.stringify(normalizedUser))
    localStorage.setItem(LOGIN_TYPE_KEY, loginType.value)
    if (expiresAt.value) localStorage.setItem(EXPIRES_AT_KEY, String(expiresAt.value))
    else localStorage.removeItem(EXPIRES_AT_KEY)
  }

  function setUserProfile(profile: BackendUser) {
    const normalizedUser = normalizeUser(profile)
    user.value = normalizedUser
    localStorage.setItem(USER_KEY, JSON.stringify(normalizedUser))
  }

  function logout() {
    token.value = null
    refreshToken.value = null
    user.value = null
    loginType.value = 'account'
    expiresAt.value = 0
    permissions.value = []
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    localStorage.removeItem(LOGIN_TYPE_KEY)
    localStorage.removeItem(EXPIRES_AT_KEY)
  }

  function setPermissions(perms: string[]) {
    permissions.value = perms
  }

  return {
    token,
    refreshToken,
    user,
    loginType,
    expiresAt,
    permissions,
    isAuthenticated,
    isAPITokenSession,
    displayName,
    displayEmail,
    defaultAvatar,
    avatarFallback,
    setSession,
    setUserProfile,
    logout,
    setPermissions,
  }
})
