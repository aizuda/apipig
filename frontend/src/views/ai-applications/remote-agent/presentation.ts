import type { AgentStatus } from '@/api/ai-applications/remote-agent'

export function formatTime(value?: number) {
  return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
}

export function formatBytes(value?: number) {
  if (!value || value <= 0) return '-'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  let amount = value
  let unit = 0
  while (amount >= 1024 && unit < units.length - 1) {
    amount /= 1024
    unit += 1
  }
  return `${amount.toFixed(unit > 1 ? 1 : 0)} ${units[unit]}`
}

export function formatPercent(value?: number) {
  return `${Math.max(0, value || 0).toFixed(1)}%`
}

export function agentStatusLabel(status: AgentStatus) {
  return {
    ONLINE: '在线',
    OFFLINE: '离线',
    BUSY: '执行中',
    DISABLED: '已禁用',
  }[status]
}

export function agentStatusVariant(status: AgentStatus) {
  return status === 'ONLINE' ? 'default' : status === 'BUSY' ? 'secondary' : 'outline'
}
