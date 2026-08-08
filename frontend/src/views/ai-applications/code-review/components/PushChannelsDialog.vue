<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Bell, Check, Plus, Send, Trash2 } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent, Input, Label, toast } from '@tabtab/ui'
import { wechatBotApi, type WechatBot } from '@/api/ai-applications/wechat-bot'
import { codeReviewApi, type PushChannel, type PushChannelType } from '@/api/ai-applications/code-review'

const props = defineProps<{ open: boolean; projectId?: string; projectName: string; channels: PushChannel[] }>()
const emit = defineEmits<{ close: []; saved: [channels: PushChannel[]] }>()
const loading = ref(false)
const bots = ref<WechatBot[]>([])
const rows = ref<PushChannel[]>([])
const selected = ref(0)

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

watch(() => props.open, async (open) => {
  if (!open) return
  rows.value = props.channels.map((item) => ({ ...item, config: { ...item.config } }))
  selected.value = 0
  if (!bots.value.length) {
    try { bots.value = (await wechatBotApi.page({ page: 1, pageSize: 100, status: 'ONLINE' })).records }
    catch { /* optional channel */ }
  }
})

function addChannel() { rows.value.push(emptyChannel()); selected.value = rows.value.length - 1 }
function removeChannel(index: number) {
  rows.value.splice(index, 1)
  selected.value = Math.max(0, Math.min(selected.value, rows.value.length - 1))
}
function changeType(type: PushChannelType) {
  if (!current.value) return
  current.value.type = type
  current.value.name = typeLabels[type]
  current.value.config = type === 'wechat_bot' ? { botId: '', userId: '' } : { webhookUrl: '' }
}
async function testChannel() {
  if (!props.projectId || !current.value) return
  try { await codeReviewApi.testPushChannel(props.projectId, current.value); toast.success('测试消息已发送') }
  catch (error) { toast.error(error instanceof Error ? error.message : '测试发送失败') }
}
async function save() {
  if (!props.projectId) return
  loading.value = true
  try {
    const result = await codeReviewApi.savePushChannels(props.projectId, rows.value)
    emit('saved', result.channels); emit('close'); toast.success('推送渠道已保存')
  } catch (error) { toast.error(error instanceof Error ? error.message : '保存推送渠道失败') }
  finally { loading.value = false }
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && !loading && emit('close')">
    <DialogFixedContent title="推送渠道" :description="`${projectName} 的评审结果将发送到已启用渠道。`" class="sm:max-w-3xl">
      <div class="grid gap-5 md:grid-cols-[190px_1fr]">
        <div class="space-y-2">
          <div class="flex items-center justify-between text-sm font-medium"><span>已配置渠道</span><Button size="icon" variant="outline" aria-label="新增渠道" @click="addChannel"><Plus class="h-4 w-4" /></Button></div>
          <button v-for="(channel, index) in rows" :key="channel.id || index" class="flex w-full items-center gap-2 rounded-md border px-3 py-2 text-left text-sm" :class="index === selected ? 'border-primary bg-primary/5' : ''" @click="selected = index">
            <Bell class="h-4 w-4 shrink-0" /><span class="min-w-0 flex-1 truncate">{{ channel.name }}</span><span class="h-2 w-2 rounded-full" :class="channel.enabled ? 'bg-emerald-500' : 'bg-muted-foreground/40'" />
          </button>
          <p v-if="!rows.length" class="py-8 text-center text-xs text-muted-foreground">暂未配置渠道</p>
        </div>
        <div v-if="current" class="space-y-4 rounded-md border p-4">
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1.5"><Label>渠道类型</Label><select :value="current.type" class="h-10 w-full rounded-md border bg-background px-3 text-sm" @change="changeType(($event.target as HTMLSelectElement).value as PushChannelType)"><option v-for="(label, type) in typeLabels" :key="type" :value="type">{{ label }}</option></select></div>
            <div class="space-y-1.5"><Label>显示名称</Label><Input v-model="current.name" /></div>
          </div>
          <div v-if="current.type !== 'wechat_bot'" class="space-y-1.5"><Label>Webhook 地址</Label><Input v-model="current.config.webhookUrl" type="password" :placeholder="current.secretConfigured ? '已配置，留空保持原值' : '粘贴机器人 Webhook 地址'" /></div>
          <div v-if="current.type === 'dingtalk_robot'" class="space-y-1.5"><Label>加签密钥（可选）</Label><Input v-model="current.config.secret" type="password" :placeholder="current.secretConfigured ? '已配置，留空保持原值' : '未开启加签可留空'" /></div>
          <template v-else>
            <div class="space-y-1.5"><Label>关联 Bot</Label><select v-model="current.config.botId" class="h-10 w-full rounded-md border bg-background px-3 text-sm"><option value="">请选择在线 Bot</option><option v-for="bot in bots" :key="bot.id" :value="bot.id">{{ bot.name }} ({{ bot.status }})</option></select></div>
            <div class="space-y-1.5"><Label>接收人 User ID</Label><Input v-model="current.config.userId" placeholder="微信联系人 User ID" /></div>
          </template>
          <label class="flex items-center gap-2 text-sm"><input v-model="current.enabled" type="checkbox" />启用此渠道</label>
          <div class="flex justify-between border-t pt-4"><Button variant="ghost" class="gap-1.5 text-destructive" @click="removeChannel(selected)"><Trash2 class="h-4 w-4" />删除</Button><Button variant="outline" class="gap-1.5" :disabled="loading" @click="testChannel"><Send class="h-4 w-4" />发送测试</Button></div>
        </div>
        <div v-else class="rounded-md border border-dashed p-12 text-center text-sm text-muted-foreground">点击“新增渠道”开始配置</div>
      </div>
      <template #footer><Button variant="outline" :disabled="loading" @click="emit('close')">取消</Button><Button :disabled="loading" class="gap-1.5" @click="save"><Check class="h-4 w-4" />保存渠道</Button></template>
    </DialogFixedContent>
  </Dialog>
</template>
