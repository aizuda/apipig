import type { WechatBotStatus } from '@/api/ai-applications/wechat-bot'

export function formatWechatTime(value?: number) {
  return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
}

export function wechatStatusLabel(status: WechatBotStatus) {
  return {
    CONNECTING: '连接中',
    ONLINE: '在线',
    OFFLINE: '离线',
    DISABLED: '已停用',
    SESSION_EXPIRED: '会话过期',
    ERROR: '连接异常',
  }[status]
}

export function wechatStatusVariant(status: WechatBotStatus) {
  if (status === 'ONLINE') return 'default'
  if (status === 'CONNECTING') return 'secondary'
  if (status === 'ERROR' || status === 'SESSION_EXPIRED') return 'destructive'
  return 'outline'
}
