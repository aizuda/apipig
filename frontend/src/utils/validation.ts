/**
 * 校验登录密码强度
 * @returns 校验错误信息，校验通过返回 undefined
 */
export function validateLoginPassword(password: string): string | undefined {
  if (!password || password.length < 6) {
    return '登录密码至少需要 6 位'
  }
  const hasLetter = /[a-zA-Z]/.test(password)
  const hasDigit = /\d/.test(password)
  const hasSpecial = /[^a-zA-Z0-9]/.test(password)
  const types = [hasLetter, hasDigit, hasSpecial].filter(Boolean).length
  if (types < 2) {
    return '登录密码必须包含字母、数字、特殊字符中的至少两种'
  }
  return undefined
}
