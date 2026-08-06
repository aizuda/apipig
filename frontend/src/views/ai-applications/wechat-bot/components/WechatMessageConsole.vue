<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { LoaderCircle, MessageCircle, RefreshCw, Send } from '@lucide/vue'
import { Badge, Button, Dialog, DialogFixedContent, Textarea, toast } from '@tabtab/ui'
import {
  wechatBotApi,
  type WechatBot,
  type WechatContact,
  type WechatMessage,
} from '@/api/ai-applications/wechat-bot'
import { formatWechatTime } from '../presentation'

const props = defineProps<{ open: boolean; bot: WechatBot | null }>()
const emit = defineEmits<{ close: [] }>()

const contacts = ref<WechatContact[]>([])
const activeContact = ref<WechatContact | null>(null)
const messages = ref<WechatMessage[]>([])
const loadingContacts = ref(false)
const loadingMessages = ref(false)
const sending = ref(false)
const draft = ref('')
const messageViewport = ref<HTMLElement | null>(null)
let refreshTimer: number | undefined

const orderedMessages = computed(() => [...messages.value].reverse())
const canSend = computed(
  () =>
    props.bot?.status === 'ONLINE' &&
    Boolean(activeContact.value?.canSend) &&
    Boolean(draft.value.trim()),
)

async function loadContacts(silent = false) {
  if (!props.bot || (loadingContacts.value && silent)) return
  if (!silent) loadingContacts.value = true
  try {
    const rows = await wechatBotApi.contacts(props.bot.id)
    contacts.value = rows
    if (!activeContact.value && rows.length) {
      await selectContact(rows[0])
      return
    }
    if (activeContact.value) {
      activeContact.value = rows.find((item) => item.id === activeContact.value?.id) || null
    }
  } catch (error) {
    if (!silent) toast.error(error instanceof Error ? error.message : '加载联系人失败')
  } finally {
    if (!silent) loadingContacts.value = false
  }
}

async function loadMessages(silent = false) {
  if (!props.bot || !activeContact.value || (loadingMessages.value && silent)) return
  if (!silent) loadingMessages.value = true
  try {
    const result = await wechatBotApi.messages(props.bot.id, activeContact.value.userId)
    messages.value = result.records || []
    await nextTick()
    if (messageViewport.value) messageViewport.value.scrollTop = messageViewport.value.scrollHeight
  } catch (error) {
    if (!silent) toast.error(error instanceof Error ? error.message : '加载消息失败')
  } finally {
    if (!silent) loadingMessages.value = false
  }
}

async function selectContact(contact: WechatContact) {
  activeContact.value = contact
  draft.value = ''
  await loadMessages()
}

async function sendMessage() {
  if (!props.bot || !activeContact.value || !canSend.value || sending.value) return
  const content = draft.value.trim()
  sending.value = true
  try {
    await wechatBotApi.send(props.bot.id, activeContact.value.userId, content)
    draft.value = ''
    await Promise.all([loadMessages(), loadContacts(true)])
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '消息发送失败')
  } finally {
    sending.value = false
  }
}

function onDraftKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    void sendMessage()
  }
}

function stopRefresh() {
  window.clearInterval(refreshTimer)
  refreshTimer = undefined
}

function startRefresh() {
  stopRefresh()
  refreshTimer = window.setInterval(() => {
    if (document.visibilityState !== 'visible') return
    void Promise.all([loadContacts(true), loadMessages(true)])
  }, 3000)
}

watch(
  () => [props.open, props.bot?.id] as const,
  ([open]) => {
    stopRefresh()
    contacts.value = []
    activeContact.value = null
    messages.value = []
    draft.value = ''
    if (open && props.bot) {
      void loadContacts()
      startRefresh()
    }
  },
  { immediate: true },
)

onBeforeUnmount(stopRefresh)
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && emit('close')">
    <DialogFixedContent
      class="sm:max-w-6xl"
      body-class="p-0"
      :title="`${bot?.name || '微信 Bot'} 消息控制台`"
      :description="bot?.status === 'ONLINE' ? '连接在线' : '当前连接不可用'"
    >
      <div
        class="grid h-[min(72vh,720px)] min-h-[480px] grid-rows-[180px_minmax(0,1fr)] lg:grid-cols-[280px_minmax(0,1fr)] lg:grid-rows-1"
      >
        <aside class="min-h-0 border-b lg:border-b-0 lg:border-r">
          <div class="flex h-12 items-center justify-between border-b px-4">
            <span class="text-sm font-medium">最近联系人</span>
            <Button variant="ghost" size="icon" title="刷新联系人" @click="loadContacts()">
              <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loadingContacts }" />
            </Button>
          </div>
          <div class="h-[calc(100%-3rem)] overflow-y-auto">
            <button
              v-for="contact in contacts"
              :key="contact.id"
              type="button"
              class="block w-full border-b px-4 py-3 text-left hover:bg-muted/40"
              :class="{ 'bg-muted': activeContact?.id === contact.id }"
              @click="selectContact(contact)"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="truncate font-mono text-xs font-medium">{{ contact.userId }}</span>
                <Badge v-if="!contact.canSend" variant="outline" class="shrink-0">已过期</Badge>
              </div>
              <p class="mt-1 truncate text-xs text-muted-foreground">
                {{ contact.lastMessage || '非文本消息' }}
              </p>
            </button>
            <div
              v-if="!loadingContacts && !contacts.length"
              class="flex h-full flex-col items-center justify-center px-4 text-center"
            >
              <MessageCircle class="h-8 w-8 text-muted-foreground/40" />
              <p class="mt-3 text-sm text-muted-foreground">暂无入站消息</p>
            </div>
          </div>
        </aside>

        <section class="grid min-h-0 grid-rows-[auto_minmax(0,1fr)_auto]">
          <div class="flex min-h-12 items-center justify-between gap-3 border-b px-4 py-2">
            <div class="min-w-0">
              <div class="truncate font-mono text-xs">
                {{ activeContact?.userId || '请选择联系人' }}
              </div>
              <div v-if="activeContact" class="mt-0.5 text-xs text-muted-foreground">
                最近消息 {{ formatWechatTime(activeContact.lastActiveAt) }}
              </div>
            </div>
            <Badge v-if="activeContact" :variant="activeContact.canSend ? 'secondary' : 'outline'">
              {{ activeContact.canSend ? '可回复' : '24 小时窗口已过期' }}
            </Badge>
          </div>

          <div ref="messageViewport" class="min-h-0 space-y-3 overflow-y-auto bg-muted/10 p-4">
            <div v-if="loadingMessages" class="flex h-full items-center justify-center">
              <LoaderCircle class="h-7 w-7 animate-spin text-muted-foreground" />
            </div>
            <template v-else>
              <div
                v-for="message in orderedMessages"
                :key="message.id"
                class="flex"
                :class="message.direction === 'outbound' ? 'justify-end' : 'justify-start'"
              >
                <div
                  class="max-w-[82%] border px-3 py-2 text-sm"
                  :class="
                    message.direction === 'outbound'
                      ? 'border-primary bg-primary text-primary-foreground'
                      : 'bg-background'
                  "
                >
                  <p class="whitespace-pre-wrap break-words">{{ message.content }}</p>
                  <p
                    class="mt-1 text-[11px]"
                    :class="
                      message.direction === 'outbound'
                        ? 'text-primary-foreground/70'
                        : 'text-muted-foreground'
                    "
                  >
                    {{ formatWechatTime(message.occurredAt) }}
                  </p>
                </div>
              </div>
            </template>
            <div
              v-if="!loadingMessages && activeContact && !messages.length"
              class="flex h-full items-center justify-center text-sm text-muted-foreground"
            >
              暂无消息记录
            </div>
          </div>

          <div class="border-t p-3">
            <div class="flex items-end gap-2">
              <Textarea
                v-model="draft"
                class="min-h-20 resize-none"
                maxlength="4000"
                :disabled="!activeContact || !activeContact.canSend || bot?.status !== 'ONLINE'"
                :placeholder="
                  !activeContact
                    ? '请选择联系人'
                    : !activeContact.canSend
                      ? '该联系人会话已超过 24 小时'
                      : bot?.status !== 'ONLINE'
                        ? 'Bot 当前未连接'
                        : '输入回复内容'
                "
                @keydown="onDraftKeydown"
              />
              <Button
                size="icon"
                title="发送消息"
                :disabled="!canSend || sending"
                @click="sendMessage"
              >
                <LoaderCircle v-if="sending" class="h-4 w-4 animate-spin" />
                <Send v-else class="h-4 w-4" />
              </Button>
            </div>
          </div>
        </section>
      </div>
    </DialogFixedContent>
  </Dialog>
</template>
