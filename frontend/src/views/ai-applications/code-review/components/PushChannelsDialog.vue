<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Bell, Check, Eye, EyeOff, LoaderCircle, Plus, Send, Trash2 } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent, Input, Label, toast } from '@tabtab/ui'
import { wechatBotApi, type WechatBot, type WechatContact } from '@/api/ai-applications/wechat-bot'
import {
  codeReviewApi,
  type PushChannel,
  type PushChannelType,
} from '@/api/ai-applications/code-review'
import WechatBotSelect from '../../wechat-bot/components/WechatBotSelect.vue'

const props = defineProps<{
  open: boolean
  projectId?: string
  projectName: string
  channels: PushChannel[]
}>()
const emit = defineEmits<{ close: []; saved: [channels: PushChannel[]] }>()
const loading = ref(false)
const detailsLoading = ref(false)
const webhookVisible = ref(false)
const secretVisible = ref(false)
const contacts = ref<WechatContact[]>([])
const contactsLoading = ref(false)
const rows = ref<PushChannel[]>([])
const selected = ref(0)
let contactsRequestId = 0
let detailsRequestId = 0

const typeLabels: Record<PushChannelType, string> = {
  wecom_robot: '企业微信机器人',
  dingtalk_robot: '钉钉机器人',
  wechat_bot: '微信 Bot',
}
const emptyChannel = (type: PushChannelType = 'wecom_robot'): PushChannel => ({
  type,
  name: typeLabels[type],
  enabled: true,
  config: type === 'wechat_bot' ? { botId: '', userId: '' } : { webhookUrl: '' },
})
const current = computed(() => rows.value[selected.value])

watch(
  () => props.open,
  async (open) => {
    const requestId = ++detailsRequestId
    hideSecrets()
    if (!open) {
      detailsLoading.value = false
      return
    }
    rows.value = props.channels.map((item) => ({ ...item, config: { ...item.config } }))
    selected.value = 0
    if (props.projectId) {
      detailsLoading.value = true
      try {
        const channels = await codeReviewApi.getPushChannels(props.projectId)
        if (requestId !== detailsRequestId || !props.open) return
        rows.value = channels.map((item) => ({ ...item, config: { ...item.config } }))
      } catch (error) {
        if (requestId !== detailsRequestId || !props.open) return
        toast.error(error instanceof Error ? error.message : '加载推送渠道失败')
      } finally {
        if (requestId === detailsRequestId) detailsLoading.value = false
      }
    }
    await loadContacts(current.value?.type === 'wechat_bot' ? current.value.config.botId : '')
  },
)

function hideSecrets() {
  webhookVisible.value = false
  secretVisible.value = false
}

function addChannel() {
  rows.value.push(emptyChannel())
  selected.value = rows.value.length - 1
  contacts.value = []
  hideSecrets()
}
function removeChannel(index: number) {
  rows.value.splice(index, 1)
  selected.value = Math.max(0, Math.min(selected.value, rows.value.length - 1))
  hideSecrets()
  void loadContacts(current.value?.type === 'wechat_bot' ? current.value.config.botId : '')
}
function changeType(type: PushChannelType) {
  if (!current.value) return
  current.value.type = type
  current.value.name = typeLabels[type]
  current.value.config = type === 'wechat_bot' ? { botId: '', userId: '' } : { webhookUrl: '' }
  contacts.value = []
  hideSecrets()
}

function selectChannel(index: number) {
  selected.value = index
  hideSecrets()
  const channel = current.value
  void loadContacts(channel?.type === 'wechat_bot' ? channel.config.botId : '')
}

function selectBot(bot: WechatBot) {
  if (!current.value) return
  current.value.config.botId = bot.id
  current.value.config.userId = ''
  void loadContacts(bot.id)
}

async function loadContacts(botId = '') {
  const requestId = ++contactsRequestId
  contacts.value = []
  if (!botId) {
    contactsLoading.value = false
    return
  }
  contactsLoading.value = true
  try {
    const result = await wechatBotApi.contacts(botId)
    if (requestId !== contactsRequestId) return
    contacts.value = result.filter((contact) => contact.canSend)
  } catch (error) {
    if (requestId !== contactsRequestId) return
    toast.error(error instanceof Error ? error.message : '加载微信会话失败')
  } finally {
    if (requestId === contactsRequestId) contactsLoading.value = false
  }
}
async function testChannel() {
  if (!props.projectId || !current.value) return
  try {
    await codeReviewApi.testPushChannel(props.projectId, current.value)
    toast.success('测试消息已发送')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '测试发送失败')
  }
}
async function save() {
  if (!props.projectId) return
  loading.value = true
  try {
    const result = await codeReviewApi.savePushChannels(props.projectId, rows.value)
    emit('saved', result.channels)
    emit('close')
    toast.success('推送渠道已保存')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '保存推送渠道失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && !loading && emit('close')">
    <DialogFixedContent
      title="推送渠道"
      :description="`${projectName} 的评审结果将发送到已启用渠道。`"
      class="sm:max-w-3xl"
    >
      <div
        v-if="detailsLoading"
        class="flex min-h-64 items-center justify-center gap-2 text-sm text-muted-foreground"
      >
        <LoaderCircle class="h-4 w-4 animate-spin" />正在加载渠道配置
      </div>
      <div v-else class="grid gap-5 md:grid-cols-[190px_1fr]">
        <div class="space-y-2">
          <div class="flex items-center justify-between text-sm font-medium">
            <span>已配置渠道</span
            ><Button size="icon" variant="outline" aria-label="新增渠道" @click="addChannel"
              ><Plus class="h-4 w-4"
            /></Button>
          </div>
          <button
            v-for="(channel, index) in rows"
            :key="channel.id || index"
            class="flex w-full items-center gap-2 rounded-md border px-3 py-2 text-left text-sm"
            :class="index === selected ? 'border-primary bg-primary/5' : ''"
            @click="selectChannel(index)"
          >
            <Bell class="h-4 w-4 shrink-0" /><span class="min-w-0 flex-1 truncate">{{
              channel.name
            }}</span
            ><span
              class="h-2 w-2 rounded-full"
              :class="channel.enabled ? 'bg-emerald-500' : 'bg-muted-foreground/40'"
            />
          </button>
          <p v-if="!rows.length" class="py-8 text-center text-xs text-muted-foreground">
            暂未配置渠道
          </p>
        </div>
        <div v-if="current" class="space-y-4 rounded-md border p-4">
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label>渠道类型</Label
              ><select
                :value="current.type"
                class="h-10 w-full rounded-md border bg-background px-3 text-sm"
                @change="changeType(($event.target as HTMLSelectElement).value as PushChannelType)"
              >
                <option v-for="(label, type) in typeLabels" :key="type" :value="type">
                  {{ label }}
                </option>
              </select>
            </div>
            <div class="space-y-1.5"><Label>显示名称</Label><Input v-model="current.name" /></div>
          </div>
          <div v-if="current.type !== 'wechat_bot'" class="space-y-1.5">
            <Label>Webhook 地址</Label>
            <div class="relative">
              <Input
                v-model="current.config.webhookUrl"
                :type="webhookVisible ? 'text' : 'password'"
                class="pr-10"
                autocomplete="new-password"
                placeholder="粘贴机器人 Webhook 地址"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                :aria-label="webhookVisible ? '隐藏 Webhook 地址' : '显示 Webhook 地址'"
                @click="webhookVisible = !webhookVisible"
              >
                <Eye v-if="!webhookVisible" class="h-4 w-4" /><EyeOff v-else class="h-4 w-4" />
              </button>
            </div>
          </div>
          <div v-if="current.type === 'dingtalk_robot'" class="space-y-1.5">
            <Label>加签密钥（可选）</Label>
            <div class="relative">
              <Input
                v-model="current.config.secret"
                :type="secretVisible ? 'text' : 'password'"
                class="pr-10"
                autocomplete="new-password"
                placeholder="未开启加签可留空"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                :aria-label="secretVisible ? '隐藏加签密钥' : '显示加签密钥'"
                @click="secretVisible = !secretVisible"
              >
                <Eye v-if="!secretVisible" class="h-4 w-4" /><EyeOff v-else class="h-4 w-4" />
              </button>
            </div>
          </div>
          <template v-if="current.type === 'wechat_bot'">
            <div class="space-y-1.5">
              <Label>关联 Bot</Label>
              <WechatBotSelect
                v-model="current.config.botId"
                :disabled="loading"
                @select="selectBot"
              />
            </div>
            <div class="space-y-1.5">
              <Label>可用会话</Label>
              <select
                v-model="current.config.userId"
                class="h-10 w-full rounded-md border bg-background px-3 text-sm"
                :disabled="loading || contactsLoading || !current.config.botId"
              >
                <option value="">
                  {{ contactsLoading ? '正在加载会话' : '默认使用最新会话' }}
                </option>
                <option v-for="contact in contacts" :key="contact.id" :value="contact.userId">
                  {{ contact.userId }} · {{ contact.lastMessage || '暂无消息' }}
                </option>
              </select>
            </div>
          </template>
          <label class="flex items-center gap-2 text-sm"
            ><input v-model="current.enabled" type="checkbox" />启用此渠道</label
          >
          <div class="flex justify-between border-t pt-4">
            <Button
              variant="ghost"
              class="gap-1.5 text-destructive"
              @click="removeChannel(selected)"
              ><Trash2 class="h-4 w-4" />删除</Button
            ><Button variant="outline" class="gap-1.5" :disabled="loading" @click="testChannel"
              ><Send class="h-4 w-4" />发送测试</Button
            >
          </div>
        </div>
        <div
          v-else
          class="rounded-md border border-dashed p-12 text-center text-sm text-muted-foreground"
        >
          点击“新增渠道”开始配置
        </div>
      </div>
      <template #footer
        ><Button variant="outline" :disabled="loading" @click="emit('close')">取消</Button
        ><Button :disabled="loading || detailsLoading" class="gap-1.5" @click="save"
          ><Check class="h-4 w-4" />保存渠道</Button
        ></template
      >
    </DialogFixedContent>
  </Dialog>
</template>
