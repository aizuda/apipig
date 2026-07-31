import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type NotificationType = 'info' | 'success' | 'warning' | 'error' | 'message'

export interface Notification {
  id: string
  title: string
  message: string
  type: NotificationType
  read: boolean
  createdAt: Date
  actionUrl?: string
  actionLabel?: string
}

const mockNotifications: Notification[] = [
  {
    id: '1',
    title: '系统更新完成',
    message: '系统已成功更新到最新版本 v2.0.0，新增多项功能优化和性能提升',
    type: 'success',
    read: false,
    createdAt: new Date(Date.now() - 1000 * 60 * 5),
    actionUrl: '/settings',
    actionLabel: '查看详情',
  },
  {
    id: '2',
    title: '新用户注册',
    message: '用户 "张三" 刚刚完成了注册，请及时审核',
    type: 'info',
    read: false,
    createdAt: new Date(Date.now() - 1000 * 60 * 30),
  },
  {
    id: '3',
    title: '存储空间警告',
    message: '您的存储空间使用率已超过 85%，请及时清理或升级存储方案',
    type: 'warning',
    read: true,
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 2),
  },
  {
    id: '4',
    title: '新消息',
    message: '您收到一条来自管理员的新消息，请查看详情',
    type: 'message',
    read: true,
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24),
  },
  {
    id: '5',
    title: '订单支付成功',
    message: '订单 #20240306001 已完成支付，金额 ¥1,299.00',
    type: 'success',
    read: true,
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 26),
  },
  {
    id: '6',
    title: '系统维护通知',
    message: '系统将于今晚 22:00-23:00 进行例行维护，届时服务可能暂时不可用',
    type: 'warning',
    read: true,
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 50),
  },
  {
    id: '7',
    title: '安全提醒',
    message: '检测到您的账号在新设备上登录，如非本人操作请及时修改密码',
    type: 'error',
    read: true,
    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 72),
  },
]

/**
 * 通知状态管理 Store
 * 管理应用内的通知消息列表、已读状态和通知操作
 */
export const useNotificationStore = defineStore('notification', () => {
  const notifications = ref<Notification[]>([...mockNotifications])

  const unreadCount = computed(() => {
    return notifications.value.filter((n) => !n.read).length
  })

  const notificationList = computed(() => {
    return [...notifications.value].sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime())
  })

  const typeCount = computed(() => {
    const counts: Record<NotificationType, number> = {
      info: 0,
      success: 0,
      warning: 0,
      error: 0,
      message: 0,
    }
    notifications.value.forEach((n) => {
      counts[n.type]++
    })
    return counts
  })

  function addNotification(notification: Omit<Notification, 'id' | 'createdAt' | 'read'>) {
    const newNotification: Notification = {
      ...notification,
      id: `notification-${Date.now()}`,
      createdAt: new Date(),
      read: false,
    }
    notifications.value.unshift(newNotification)
  }

  function markAsRead(id: string) {
    const notification = notifications.value.find((n) => n.id === id)
    if (notification) {
      notification.read = true
    }
  }

  function markAllAsRead() {
    notifications.value.forEach((n) => {
      n.read = true
    })
  }

  function removeNotification(id: string) {
    const index = notifications.value.findIndex((n) => n.id === id)
    if (index > -1) {
      notifications.value.splice(index, 1)
    }
  }

  function clearAll() {
    notifications.value = []
  }

  function resetToMock() {
    notifications.value = [...mockNotifications]
  }

  return {
    notifications,
    unreadCount,
    notificationList,
    typeCount,
    addNotification,
    markAsRead,
    markAllAsRead,
    removeNotification,
    clearAll,
    resetToMock,
  }
})
