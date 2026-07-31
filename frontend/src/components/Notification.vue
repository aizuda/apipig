<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  Bell,
  Check,
  CheckCheck,
  Trash2,
  X,
  Info,
  CheckCircle,
  AlertTriangle,
  MessageSquare,
} from '@lucide/vue'
import { useNotificationStore, type Notification } from '@/stores/notification'
import { Button, ScrollArea } from '@tabtab/ui'
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from '@tabtab/ui'

const { t } = useI18n()
const router = useRouter()
const notificationStore = useNotificationStore()

/**
 * 閫氱煡绫诲瀷閰嶇疆
 */
const notificationTypeConfig = {
  info: {
    icon: Info,
    color: 'text-blue-500',
    bgColor: 'bg-gradient-to-br from-blue-500/20 to-blue-600/10',
  },
  warning: {
    icon: AlertTriangle,
    color: 'text-amber-500',
    bgColor: 'bg-gradient-to-br from-amber-500/20 to-orange-600/10',
  },
  success: {
    icon: CheckCircle,
    color: 'text-emerald-500',
    bgColor: 'bg-gradient-to-br from-emerald-500/20 to-green-600/10',
  },
  error: {
    icon: AlertTriangle,
    color: 'text-red-500',
    bgColor: 'bg-gradient-to-br from-red-500/20 to-red-600/10',
  },
  message: {
    icon: MessageSquare,
    color: 'text-purple-500',
    bgColor: 'bg-gradient-to-br from-purple-500/20 to-violet-600/10',
  },
}

type NotificationType = keyof typeof notificationTypeConfig

const notificationList = computed(() => notificationStore.notificationList)
const unreadCount = computed(() => notificationStore.unreadCount)
const hasUnread = computed(() => unreadCount.value > 0)

/**
 * 鍒嗙粍閫氱煡
 */
const groupedNotifications = computed(() => {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1000)

  const groups = {
    unread: [] as Notification[],
    today: [] as Notification[],
    yesterday: [] as Notification[],
    earlier: [] as Notification[],
  }

  notificationList.value.forEach((notification) => {
    const notifDate = new Date(notification.createdAt)
    notifDate.setHours(0, 0, 0, 0)
    today.setHours(0, 0, 0, 0)

    if (!notification.read) {
      groups.unread.push(notification)
    } else if (notifDate.getTime() === today.getTime()) {
      groups.today.push(notification)
    } else if (notifDate.getTime() === yesterday.getTime()) {
      groups.yesterday.push(notification)
    } else {
      groups.earlier.push(notification)
    }
  })

  return groups
})

/**
 * 妫€鏌ュ垎缁勬槸鍚︽湁鍐呭
 */
function hasGroupItems(group: keyof typeof groupedNotifications.value): boolean {
  return groupedNotifications.value[group].length > 0
}

/**
 * 鑾峰彇鍒嗙粍鏍囬
 */
function getGroupTitle(group: keyof typeof groupedNotifications.value): string {
  const titles: Record<keyof typeof groupedNotifications.value, string> = {
    unread: t('common.notification.unread'),
    today: t('common.notification.today'),
    yesterday: t('common.notification.yesterday'),
    earlier: t('common.notification.earlier'),
  }
  return titles[group]
}

/**
 * 鏍煎紡鍖栨椂闂? */
function formatTime(date: Date): string {
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (minutes < 1) {
    return t('common.notification.justNow')
  }
  if (minutes < 60) {
    return t('common.notification.minutesAgo', { minutes })
  }
  if (hours < 24) {
    return t('common.notification.hoursAgo', { hours })
  }
  if (days < 7) {
    return t('common.notification.daysAgo', { days })
  }
  return date.toLocaleDateString()
}

/**
 * 澶勭悊鏍囪宸茶
 */
function handleMarkAsRead(id: string) {
  notificationStore.markAsRead(id)
}

/**
 * 澶勭悊鍏ㄩ儴鏍囪涓哄凡璇? */
function handleMarkAllAsRead() {
  notificationStore.markAllAsRead()
}

/**
 * 澶勭悊鍒犻櫎閫氱煡
 */
function handleDeleteNotification(event: Event, id: string) {
  event.stopPropagation()
  notificationStore.removeNotification(id)
}

/**
 * 澶勭悊娓呯┖鎵€鏈夐€氱煡
 */
function handleClearAll() {
  notificationStore.clearAll()
}

/**
 * 璺宠浆鍒伴€氱煡鍒楄〃椤甸潰
 */
function handleViewAll() {
  router.push('/settings/messages')
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        variant="ghost"
        size="icon"
        class="h-8 w-8 rounded-lg hover:bg-primary/10 hover:text-primary relative transition-all duration-200"
      >
        <Bell class="h-4 w-4" />
        <span v-if="hasUnread" class="absolute top-1.5 right-1.5 flex h-2.5 w-2.5">
          <span
            class="relative inline-flex rounded-full h-2.5 w-2.5 bg-red-500 ring-2 ring-background"
          />
        </span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-[calc(100vw-1rem)] max-w-80 p-0" :side-offset="8">
      <div class="flex items-center justify-between px-3 py-2.5 border-b border-border/50 gap-2">
        <div class="flex items-center gap-2 min-w-0">
          <Bell class="h-4 w-4 text-muted-foreground flex-shrink-0" />
          <span class="font-semibold text-sm truncate">{{ t('common.notification.title') }}</span>
          <span
            v-if="unreadCount > 0"
            class="bg-primary text-primary-foreground text-[10px] font-medium px-1.5 py-0.5 rounded-full min-w-[18px] text-center flex-shrink-0"
          >
            {{ unreadCount }}
          </span>
        </div>
        <div class="flex items-center gap-0.5 flex-shrink-0">
          <Button
            v-if="hasUnread"
            variant="ghost"
            size="sm"
            class="h-7 px-1.5 text-xs hover:bg-primary/10 hover:text-primary whitespace-nowrap"
            @click="handleMarkAllAsRead"
          >
            <CheckCheck class="h-3.5 w-3.5 mr-1 flex-shrink-0" />
            <span class="truncate">{{ t('common.notification.markAllRead') }}</span>
          </Button>
          <Button
            v-if="notificationList.length > 0"
            variant="ghost"
            size="icon"
            class="h-7 w-7 text-muted-foreground hover:text-destructive hover:bg-destructive/10 flex-shrink-0"
            @click="handleClearAll"
          >
            <Trash2 class="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <ScrollArea class="h-[360px]">
        <div
          v-if="notificationList.length === 0"
          class="flex flex-col items-center justify-center py-12 text-center"
        >
          <div class="relative mb-4">
            <div
              class="h-16 w-16 rounded-2xl bg-gradient-to-br from-muted to-muted/50 flex items-center justify-center shadow-sm"
            >
              <Bell class="h-7 w-7 text-muted-foreground/60" />
            </div>
            <div
              class="absolute -bottom-1 -right-1 h-5 w-5 rounded-full bg-background border-2 border-muted flex items-center justify-center"
            >
              <Check class="h-3 w-3 text-muted-foreground/60" />
            </div>
          </div>
          <p class="text-sm font-medium text-foreground/80 mb-1">
            {{ t('common.notification.empty') }}
          </p>
          <p class="text-xs text-muted-foreground">
            {{ t('common.notification.emptyDesc') }}
          </p>
        </div>

        <template v-else>
          <template v-if="hasGroupItems('unread')">
            <div class="px-4 py-2 bg-primary/5 border-b border-border/30">
              <span class="text-xs font-semibold text-primary flex items-center gap-1.5">
                <span class="h-1.5 w-1.5 rounded-full bg-primary" />
                {{ getGroupTitle('unread') }}
                <span class="bg-primary/20 text-primary text-[10px] px-1.5 py-0.5 rounded-full">
                  {{ groupedNotifications.unread.length }}
                </span>
              </span>
            </div>
            <div
              v-for="notification in groupedNotifications.unread"
              :key="notification.id"
              class="group flex items-start gap-3 px-4 py-3 transition-all duration-200 cursor-pointer border-b border-border/20 relative overflow-hidden"
              @click="handleMarkAsRead(notification.id)"
            >
              <div
                class="absolute left-0 top-0 bottom-0 w-[3px] bg-gradient-to-b from-primary to-primary/50"
              />
              <span
                class="absolute inset-0 bg-gradient-to-r from-transparent via-primary/5 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-700 ease-in-out pointer-events-none"
                aria-hidden="true"
              />
              <div
                class="h-9 w-9 rounded-xl flex items-center justify-center flex-shrink-0 transition-all duration-300 relative z-10 shadow-sm ring-1 ring-black/5 dark:ring-white/5"
                :class="
                  notificationTypeConfig[notification.type as NotificationType]?.bgColor ||
                  notificationTypeConfig.info.bgColor
                "
              >
                <component
                  :is="
                    notificationTypeConfig[notification.type as NotificationType]?.icon ||
                    notificationTypeConfig.info.icon
                  "
                  class="h-4 w-4 transition-colors duration-200"
                  :class="
                    notificationTypeConfig[notification.type as NotificationType]?.color ||
                    notificationTypeConfig.info.color
                  "
                />
              </div>
              <div class="flex-1 min-w-0 relative z-10">
                <div class="flex items-start justify-between gap-2">
                  <p class="text-sm font-medium text-foreground truncate">
                    {{ notification.title }}
                  </p>
                  <span
                    class="text-[10px] text-muted-foreground flex-shrink-0 bg-muted/50 px-1.5 py-0.5 rounded"
                  >
                    {{ formatTime(notification.createdAt) }}
                  </span>
                </div>
                <p class="text-xs text-muted-foreground line-clamp-2 mt-1 leading-relaxed">
                  {{ notification.message }}
                </p>
              </div>
              <div class="flex flex-col items-center gap-1 flex-shrink-0 relative z-10">
                <div class="h-2 w-2 rounded-full bg-primary" />
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-6 w-6 opacity-0 group-hover:opacity-100 transition-all duration-200 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                  @click.stop="handleDeleteNotification($event, notification.id)"
                >
                  <X class="h-3 w-3" />
                </Button>
              </div>
            </div>
          </template>

          <template v-if="hasGroupItems('today')">
            <div class="px-4 py-2 bg-muted/30 border-b border-border/30">
              <span class="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <span class="h-1 w-1 rounded-full bg-muted-foreground/50" />
                {{ getGroupTitle('today') }}
              </span>
            </div>
            <div
              v-for="notification in groupedNotifications.today"
              :key="notification.id"
              class="group flex items-start gap-3 px-4 py-3 transition-all duration-200 cursor-pointer border-b border-border/20 relative overflow-hidden hover:bg-muted/50"
              @click="handleMarkAsRead(notification.id)"
            >
              <span
                class="absolute inset-0 bg-gradient-to-r from-transparent via-muted/30 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-700 ease-in-out pointer-events-none"
                aria-hidden="true"
              />
              <div
                class="h-8 w-8 rounded-lg flex items-center justify-center flex-shrink-0 transition-all duration-200 relative z-10"
                :class="
                  notificationTypeConfig[notification.type as NotificationType]?.bgColor ||
                  notificationTypeConfig.info.bgColor
                "
              >
                <component
                  :is="
                    notificationTypeConfig[notification.type as NotificationType]?.icon ||
                    notificationTypeConfig.info.icon
                  "
                  class="h-3.5 w-3.5 transition-colors duration-200"
                  :class="
                    notificationTypeConfig[notification.type as NotificationType]?.color ||
                    notificationTypeConfig.info.color
                  "
                />
              </div>
              <div class="flex-1 min-w-0 relative z-10">
                <div class="flex items-start justify-between gap-2">
                  <p
                    class="text-sm text-foreground/80 truncate group-hover:text-foreground transition-colors"
                  >
                    {{ notification.title }}
                  </p>
                  <span class="text-[10px] text-muted-foreground flex-shrink-0">
                    {{ formatTime(notification.createdAt) }}
                  </span>
                </div>
                <p class="text-xs text-muted-foreground line-clamp-2 mt-0.5 leading-relaxed">
                  {{ notification.message }}
                </p>
              </div>
              <Button
                variant="ghost"
                size="icon"
                class="h-6 w-6 opacity-0 group-hover:opacity-100 transition-all duration-200 text-muted-foreground hover:text-destructive hover:bg-destructive/10 flex-shrink-0 relative z-10"
                @click.stop="handleDeleteNotification($event, notification.id)"
              >
                <X class="h-3 w-3" />
              </Button>
            </div>
          </template>

          <template v-if="hasGroupItems('yesterday')">
            <div class="px-4 py-2 bg-muted/30 border-b border-border/30">
              <span class="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <span class="h-1 w-1 rounded-full bg-muted-foreground/40" />
                {{ getGroupTitle('yesterday') }}
              </span>
            </div>
            <div
              v-for="notification in groupedNotifications.yesterday"
              :key="notification.id"
              class="group flex items-start gap-3 px-4 py-3 transition-all duration-200 cursor-pointer border-b border-border/20 relative overflow-hidden hover:bg-muted/50"
              @click="handleMarkAsRead(notification.id)"
            >
              <span
                class="absolute inset-0 bg-gradient-to-r from-transparent via-muted/30 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-700 ease-in-out pointer-events-none"
                aria-hidden="true"
              />
              <div
                class="h-8 w-8 rounded-lg flex items-center justify-center flex-shrink-0 transition-all duration-200 relative z-10 opacity-80"
                :class="
                  notificationTypeConfig[notification.type as NotificationType]?.bgColor ||
                  notificationTypeConfig.info.bgColor
                "
              >
                <component
                  :is="
                    notificationTypeConfig[notification.type as NotificationType]?.icon ||
                    notificationTypeConfig.info.icon
                  "
                  class="h-3.5 w-3.5 transition-colors duration-200"
                  :class="
                    notificationTypeConfig[notification.type as NotificationType]?.color ||
                    notificationTypeConfig.info.color
                  "
                />
              </div>
              <div class="flex-1 min-w-0 relative z-10">
                <div class="flex items-start justify-between gap-2">
                  <p
                    class="text-sm text-foreground/70 truncate group-hover:text-foreground/90 transition-colors"
                  >
                    {{ notification.title }}
                  </p>
                  <span class="text-[10px] text-muted-foreground flex-shrink-0">
                    {{ formatTime(notification.createdAt) }}
                  </span>
                </div>
                <p class="text-xs text-muted-foreground/80 line-clamp-2 mt-0.5 leading-relaxed">
                  {{ notification.message }}
                </p>
              </div>
              <Button
                variant="ghost"
                size="icon"
                class="h-6 w-6 opacity-0 group-hover:opacity-100 transition-all duration-200 text-muted-foreground hover:text-destructive hover:bg-destructive/10 flex-shrink-0 relative z-10"
                @click.stop="handleDeleteNotification($event, notification.id)"
              >
                <X class="h-3 w-3" />
              </Button>
            </div>
          </template>

          <template v-if="hasGroupItems('earlier')">
            <div class="px-4 py-2 bg-muted/30 border-b border-border/30">
              <span class="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <span class="h-1 w-1 rounded-full bg-muted-foreground/30" />
                {{ getGroupTitle('earlier') }}
              </span>
            </div>
            <div
              v-for="notification in groupedNotifications.earlier"
              :key="notification.id"
              class="group flex items-start gap-3 px-4 py-3 transition-all duration-200 cursor-pointer border-b border-border/20 relative overflow-hidden hover:bg-muted/50"
              @click="handleMarkAsRead(notification.id)"
            >
              <span
                class="absolute inset-0 bg-gradient-to-r from-transparent via-muted/30 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-700 ease-in-out pointer-events-none"
                aria-hidden="true"
              />
              <div
                class="h-8 w-8 rounded-lg flex items-center justify-center flex-shrink-0 transition-all duration-200 relative z-10 opacity-60"
                :class="
                  notificationTypeConfig[notification.type as NotificationType]?.bgColor ||
                  notificationTypeConfig.info.bgColor
                "
              >
                <component
                  :is="
                    notificationTypeConfig[notification.type as NotificationType]?.icon ||
                    notificationTypeConfig.info.icon
                  "
                  class="h-3.5 w-3.5 transition-colors duration-200"
                  :class="
                    notificationTypeConfig[notification.type as NotificationType]?.color ||
                    notificationTypeConfig.info.color
                  "
                />
              </div>
              <div class="flex-1 min-w-0 relative z-10">
                <div class="flex items-start justify-between gap-2">
                  <p
                    class="text-sm text-foreground/60 truncate group-hover:text-foreground/80 transition-colors"
                  >
                    {{ notification.title }}
                  </p>
                  <span class="text-[10px] text-muted-foreground flex-shrink-0">
                    {{ formatTime(notification.createdAt) }}
                  </span>
                </div>
                <p class="text-xs text-muted-foreground/70 line-clamp-2 mt-0.5 leading-relaxed">
                  {{ notification.message }}
                </p>
              </div>
              <Button
                variant="ghost"
                size="icon"
                class="h-6 w-6 opacity-0 group-hover:opacity-100 transition-all duration-200 text-muted-foreground hover:text-destructive hover:bg-destructive/10 flex-shrink-0 relative z-10"
                @click.stop="handleDeleteNotification($event, notification.id)"
              >
                <X class="h-3 w-3" />
              </Button>
            </div>
          </template>
        </template>
      </ScrollArea>

      <div v-if="notificationList.length > 0" class="border-t border-border/50 p-2">
        <Button
          variant="ghost"
          size="sm"
          class="w-full text-xs text-muted-foreground hover:text-primary hover:bg-primary/10 transition-all duration-200 rounded-lg"
          @click="handleViewAll"
        >
          {{ t('common.notification.viewAll') }}
        </Button>
      </div>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
