<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  AlertTriangle,
  Bell,
  CheckCheck,
  CheckCircle,
  Info,
  MessageSquare,
  Trash2,
  X,
} from '@lucide/vue'
import { Badge, Button, Card } from '@tabtab/ui'
import AppPageHeader from '@/components/AppPageHeader.vue'
import {
  useNotificationStore,
  type Notification,
  type NotificationType,
} from '@/stores/notification'

defineOptions({ name: 'MyMessages' })

type FilterType = 'all' | 'unread' | 'read'
type GroupKey = 'unread' | 'today' | 'yesterday' | 'earlier'

const { t } = useI18n()
const notificationStore = useNotificationStore()
const currentFilter = ref<FilterType>('all')
const currentTypeFilter = ref<NotificationType | 'all'>('all')

const notificationTypeConfig: Record<
  NotificationType,
  { icon: typeof Info; color: string; bgColor: string; labelKey: string }
> = {
  info: {
    icon: Info,
    color: 'text-blue-600 dark:text-blue-400',
    bgColor: 'bg-blue-500/10',
    labelKey: 'common.notification.typeInfo',
  },
  warning: {
    icon: AlertTriangle,
    color: 'text-amber-600 dark:text-amber-400',
    bgColor: 'bg-amber-500/10',
    labelKey: 'common.notification.typeWarning',
  },
  success: {
    icon: CheckCircle,
    color: 'text-emerald-600 dark:text-emerald-400',
    bgColor: 'bg-emerald-500/10',
    labelKey: 'common.notification.typeSuccess',
  },
  error: {
    icon: AlertTriangle,
    color: 'text-red-600 dark:text-red-400',
    bgColor: 'bg-red-500/10',
    labelKey: 'common.notification.typeError',
  },
  message: {
    icon: MessageSquare,
    color: 'text-violet-600 dark:text-violet-400',
    bgColor: 'bg-violet-500/10',
    labelKey: 'common.notification.typeMessage',
  },
}

const filterTabs: Array<{ key: FilterType; label: string }> = [
  { key: 'all', label: 'common.notification.filterAll' },
  { key: 'unread', label: 'common.notification.filterUnread' },
  { key: 'read', label: 'common.notification.filterRead' },
]

const typeFilters: Array<{ key: NotificationType | 'all'; label: string }> = [
  { key: 'all', label: 'common.notification.filterTypeAll' },
  { key: 'info', label: 'common.notification.filterTypeInfo' },
  { key: 'warning', label: 'common.notification.filterTypeWarning' },
  { key: 'success', label: 'common.notification.filterTypeSuccess' },
  { key: 'error', label: 'common.notification.typeError' },
  { key: 'message', label: 'common.notification.filterTypeMessage' },
]

const notifications = computed(() => notificationStore.notificationList)
const unreadCount = computed(() => notificationStore.unreadCount)
const readCount = computed(() => notifications.value.length - unreadCount.value)

const filteredNotifications = computed(() => {
  return notifications.value.filter((notification) => {
    const matchesRead =
      currentFilter.value === 'all' ||
      (currentFilter.value === 'unread' && !notification.read) ||
      (currentFilter.value === 'read' && notification.read)
    const matchesType =
      currentTypeFilter.value === 'all' || notification.type === currentTypeFilter.value
    return matchesRead && matchesType
  })
})

const groupedNotifications = computed<Record<GroupKey, Notification[]>>(() => {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const yesterday = today - 24 * 60 * 60 * 1000
  const groups: Record<GroupKey, Notification[]> = {
    unread: [],
    today: [],
    yesterday: [],
    earlier: [],
  }

  filteredNotifications.value.forEach((notification) => {
    const date = new Date(
      notification.createdAt.getFullYear(),
      notification.createdAt.getMonth(),
      notification.createdAt.getDate(),
    ).getTime()
    if (!notification.read) groups.unread.push(notification)
    else if (date >= today) groups.today.push(notification)
    else if (date >= yesterday) groups.yesterday.push(notification)
    else groups.earlier.push(notification)
  })
  return groups
})

const groupEntries = computed(() => {
  const labels: Record<GroupKey, string> = {
    unread: t('common.notification.unread'),
    today: t('common.notification.today'),
    yesterday: t('common.notification.yesterday'),
    earlier: t('common.notification.earlier'),
  }
  return (Object.keys(groupedNotifications.value) as GroupKey[])
    .map((key) => ({ key, label: labels[key], items: groupedNotifications.value[key] }))
    .filter((group) => group.items.length > 0)
})

function formatTime(date: Date) {
  const diff = Date.now() - date.getTime()
  const minutes = Math.floor(diff / 60_000)
  const hours = Math.floor(diff / 3_600_000)
  const days = Math.floor(diff / 86_400_000)
  if (minutes < 1) return t('common.notification.justNow')
  if (minutes < 60) return t('common.notification.minutesAgo', { minutes })
  if (hours < 24) return t('common.notification.hoursAgo', { hours })
  if (days < 7) return t('common.notification.daysAgo', { days })
  return date.toLocaleDateString()
}

function openNotification(notification: Notification) {
  if (!notification.read) notificationStore.markAsRead(notification.id)
}
</script>

<template>
  <div class="space-y-5">
    <AppPageHeader
      :title="t('settings.myMessages')"
      :description="t('settings.myMessagesDesc')"
      :show-refresh="false"
    />

    <div
      class="flex flex-col gap-3 rounded-xl border bg-card p-3 lg:flex-row lg:items-center lg:justify-between"
    >
      <div class="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
        <div class="flex rounded-lg bg-muted p-1">
          <button
            v-for="tab in filterTabs"
            :key="tab.key"
            type="button"
            class="rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
            :class="
              currentFilter === tab.key
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            "
            @click="currentFilter = tab.key"
          >
            {{ t(tab.label) }}
          </button>
        </div>
        <select
          v-model="currentTypeFilter"
          class="h-8 rounded-md border bg-background px-2.5 text-xs text-foreground"
        >
          <option v-for="filter in typeFilters" :key="filter.key" :value="filter.key">
            {{ t(filter.label) }}
          </option>
        </select>
        <div class="flex items-center gap-2 whitespace-nowrap text-xs text-muted-foreground">
          <span>共 {{ notifications.length }} 条</span>
          <span class="size-1 rounded-full bg-border" />
          <span class="text-primary">{{ unreadCount }} 条未读</span>
          <span class="hidden sm:inline">{{ readCount }} 条已读</span>
        </div>
      </div>
      <div class="flex gap-2">
        <Button
          size="sm"
          variant="outline"
          :disabled="unreadCount === 0"
          @click="notificationStore.markAllAsRead"
        >
          <CheckCheck class="mr-1.5 size-4" />{{ t('common.notification.markAllRead') }}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          class="text-destructive hover:text-destructive"
          :disabled="notifications.length === 0"
          @click="notificationStore.clearAll"
        >
          <Trash2 class="mr-1.5 size-4" />{{ t('common.notification.clearAll') }}
        </Button>
      </div>
    </div>

    <Card class="gap-0 overflow-hidden py-0">
      <div
        v-if="filteredNotifications.length === 0"
        class="flex min-h-72 flex-col items-center justify-center px-6 py-12 text-center"
      >
        <div class="flex size-12 items-center justify-center rounded-full bg-muted">
          <Bell class="size-6 text-muted-foreground" />
        </div>
        <h2 class="mt-3 font-semibold">{{ t('common.notification.empty') }}</h2>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('common.notification.emptyDesc') }}</p>
      </div>

      <div v-else>
        <section v-for="group in groupEntries" :key="group.key">
          <div
            class="flex items-center justify-between border-b bg-muted/30 px-4 py-2 text-xs font-medium text-muted-foreground"
          >
            <span>{{ group.label }}</span>
            <span>{{ group.items.length }}</span>
          </div>
          <article
            v-for="notification in group.items"
            :key="notification.id"
            class="group grid cursor-pointer grid-cols-[auto_minmax(0,1fr)_auto] gap-3 border-b px-4 py-3 transition-colors last:border-b-0 hover:bg-muted/30"
            :class="!notification.read ? 'bg-primary/[0.025]' : ''"
            @click="openNotification(notification)"
          >
            <div
              class="mt-0.5 flex size-8 items-center justify-center rounded-lg"
              :class="notificationTypeConfig[notification.type].bgColor"
            >
              <component
                :is="notificationTypeConfig[notification.type].icon"
                class="size-4"
                :class="notificationTypeConfig[notification.type].color"
              />
            </div>

            <div class="min-w-0">
              <div class="flex min-w-0 items-center gap-2">
                <span v-if="!notification.read" class="size-1.5 shrink-0 rounded-full bg-primary" />
                <h3
                  class="truncate text-sm"
                  :class="notification.read ? 'font-medium text-foreground/80' : 'font-semibold'"
                >
                  {{ notification.title }}
                </h3>
                <Badge variant="outline" class="hidden h-5 px-1.5 text-[10px] sm:inline-flex">
                  {{ t(notificationTypeConfig[notification.type].labelKey) }}
                </Badge>
              </div>
              <p class="mt-0.5 line-clamp-2 text-sm leading-5 text-muted-foreground">
                {{ notification.message }}
              </p>
              <RouterLink
                v-if="notification.actionUrl"
                :to="notification.actionUrl"
                class="mt-1 inline-flex text-xs font-medium text-primary hover:underline"
                @click.stop="notificationStore.markAsRead(notification.id)"
              >
                {{ notification.actionLabel || t('common.notification.viewAll') }}
              </RouterLink>
            </div>

            <div class="flex items-start gap-1">
              <time class="whitespace-nowrap pt-1 text-xs text-muted-foreground">
                {{ formatTime(notification.createdAt) }}
              </time>
              <Button
                variant="ghost"
                size="icon"
                class="size-7 text-muted-foreground opacity-60 hover:text-destructive sm:opacity-0 sm:group-hover:opacity-100"
                title="删除消息"
                @click.stop="notificationStore.removeNotification(notification.id)"
              >
                <X class="size-3.5" />
              </Button>
            </div>
          </article>
        </section>
      </div>
    </Card>
  </div>
</template>
