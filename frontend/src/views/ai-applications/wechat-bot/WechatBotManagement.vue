<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  LoaderCircle,
  MessageCircle,
  Pencil,
  Plus,
  Power,
  QrCode,
  RefreshCw,
  Search,
  Trash2,
} from '@lucide/vue'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  Card,
  CardContent,
  Dialog,
  DialogFixedContent,
  Input,
  Label,
  toast,
} from '@tabtab/ui'
import {
  wechatBotApi,
  type WechatBot,
  type WechatBotStatus,
} from '@/api/ai-applications/wechat-bot'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import WechatBindDialog from './components/WechatBindDialog.vue'
import WechatMessageConsole from './components/WechatMessageConsole.vue'
import { formatWechatTime, wechatStatusLabel, wechatStatusVariant } from './presentation'

defineOptions({ name: 'WechatBot' })

const loading = ref(false)
const refreshing = ref(false)
const bots = ref<WechatBot[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const status = ref<WechatBotStatus | ''>('')

const bindOpen = ref(false)
const bindLoading = ref(false)
const bindPolling = ref(false)
const bindName = ref('')
const bindQRCode = ref('')
const bindStatus = ref('')
const bindSessionID = ref('')
const bindTarget = ref<WechatBot | null>(null)
let bindGeneration = 0

const renameTarget = ref<WechatBot | null>(null)
const renameName = ref('')
const renaming = ref(false)
const deleteTarget = ref<WechatBot | null>(null)
const deleting = ref(false)
const actionID = ref('')
const consoleBot = ref<WechatBot | null>(null)
let refreshTimer: number | undefined

const onlineCount = computed(() => bots.value.filter((item) => item.status === 'ONLINE').length)
const expiredCount = computed(
  () => bots.value.filter((item) => item.status === 'SESSION_EXPIRED').length,
)

async function loadBots(silent = false) {
  if (silent && (loading.value || refreshing.value)) return
  if (silent) refreshing.value = true
  else loading.value = true
  try {
    const result = await wechatBotApi.page({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      status: status.value || undefined,
    })
    bots.value = result.records || []
    total.value = result.total || 0
    if (consoleBot.value) {
      consoleBot.value = bots.value.find((item) => item.id === consoleBot.value?.id) || null
    }
  } catch (error) {
    if (!silent) toast.error(error instanceof Error ? error.message : '加载微信 Bot 失败')
  } finally {
    if (silent) refreshing.value = false
    else loading.value = false
  }
}

function wait(milliseconds: number) {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds))
}

function openBind(bot: WechatBot | null = null) {
  bindTarget.value = bot
  bindName.value = bot?.name || ''
  bindOpen.value = true
  void startBinding()
}

async function startBinding() {
  const generation = ++bindGeneration
  bindLoading.value = true
  bindPolling.value = false
  bindQRCode.value = ''
  bindStatus.value = ''
  bindSessionID.value = ''
  try {
    const result = await wechatBotApi.startBind({
      name: bindName.value.trim() || undefined,
      botId: bindTarget.value?.id,
    })
    if (generation !== bindGeneration || !bindOpen.value) return
    bindSessionID.value = result.sessionId
    bindQRCode.value = result.qrCode
    bindStatus.value = 'WAITING'
    bindPolling.value = true
    void pollBinding(generation)
  } catch (error) {
    if (generation === bindGeneration) {
      toast.error(error instanceof Error ? error.message : '获取登录二维码失败')
    }
  } finally {
    if (generation === bindGeneration) bindLoading.value = false
  }
}

async function pollBinding(generation: number) {
  while (generation === bindGeneration && bindOpen.value && bindSessionID.value) {
    try {
      const result = await wechatBotApi.pollBind(bindSessionID.value)
      if (generation !== bindGeneration || !bindOpen.value) return
      bindStatus.value = result.status
      if (result.qrCode) bindQRCode.value = result.qrCode
      if (result.status === 'CONFIRMED') {
        bindPolling.value = false
        toast.success(bindTarget.value ? '微信 Bot 已重新绑定' : '微信 Bot 已绑定')
        await loadBots()
        await wait(700)
        closeBind()
        return
      }
      await wait(800)
    } catch (error) {
      if (generation === bindGeneration) {
        bindPolling.value = false
        toast.error(error instanceof Error ? error.message : '查询扫码状态失败')
      }
      return
    }
  }
}

function closeBind() {
  bindGeneration += 1
  bindOpen.value = false
  bindLoading.value = false
  bindPolling.value = false
  bindSessionID.value = ''
}

function openRename(bot: WechatBot) {
  renameTarget.value = bot
  renameName.value = bot.name
}

async function confirmRename() {
  if (!renameTarget.value || !renameName.value.trim() || renaming.value) return
  renaming.value = true
  try {
    await wechatBotApi.rename(renameTarget.value.id, renameName.value.trim())
    renameTarget.value = null
    toast.success('Bot 名称已更新')
    await loadBots()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '重命名失败')
  } finally {
    renaming.value = false
  }
}

async function reconnect(bot: WechatBot) {
  if (actionID.value) return
  actionID.value = bot.id
  try {
    await wechatBotApi.reconnect(bot.id)
    toast.success('正在重新建立连接')
    await loadBots(true)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '重连失败')
  } finally {
    actionID.value = ''
  }
}

async function toggleEnabled(bot: WechatBot) {
  if (actionID.value) return
  actionID.value = bot.id
  try {
    await wechatBotApi.setEnabled(bot.id, !bot.enabled)
    toast.success(bot.enabled ? '微信 Bot 已停用' : '正在启用并连接')
    await loadBots()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '更新状态失败')
  } finally {
    actionID.value = ''
  }
}

async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) return
  deleting.value = true
  try {
    await wechatBotApi.delete(deleteTarget.value.id)
    deleteTarget.value = null
    toast.success('微信 Bot 及其本地消息记录已删除')
    if (bots.value.length === 1 && page.value > 1) page.value -= 1
    await loadBots()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '删除失败')
  } finally {
    deleting.value = false
  }
}

function searchBots() {
  page.value = 1
  void loadBots()
}

function resetFilters() {
  keyword.value = ''
  status.value = ''
  searchBots()
}

onMounted(() => {
  void loadBots()
  refreshTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') void loadBots(true)
  }, 5000)
})

onBeforeUnmount(() => {
  window.clearInterval(refreshTimer)
  closeBind()
})
</script>

<template>
  <div class="space-y-6">
    <AppPageHeader
      title="微信 Bot"
      description="管理 iLink 微信 Bot 账户、连接状态和最近会话。"
      :loading="loading"
      @refresh="loadBots()"
    >
      <template #actions>
        <Button size="sm" class="gap-2" @click="openBind()">
          <Plus class="h-4 w-4" />扫码绑定
        </Button>
      </template>
    </AppPageHeader>

    <div class="grid grid-cols-3 divide-x border-y py-3 text-center">
      <div>
        <div class="text-xl font-semibold tabular-nums">{{ total }}</div>
        <div class="text-xs text-muted-foreground">账户总数</div>
      </div>
      <div>
        <div class="text-xl font-semibold tabular-nums">{{ onlineCount }}</div>
        <div class="text-xs text-muted-foreground">本页在线</div>
      </div>
      <div>
        <div class="text-xl font-semibold tabular-nums">{{ expiredCount }}</div>
        <div class="text-xs text-muted-foreground">本页会话过期</div>
      </div>
    </div>

    <Card class="gap-0 overflow-hidden py-0">
      <CardContent class="p-0">
        <form
          class="flex flex-col gap-3 border-b p-4 lg:flex-row lg:items-center"
          @submit.prevent="searchBots"
        >
          <div class="relative min-w-0 flex-1">
            <Search
              class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            />
            <Input v-model="keyword" class="pl-9" placeholder="搜索名称、Bot ID 或 iLink User ID" />
          </div>
          <select
            v-model="status"
            class="h-10 rounded-md border bg-background px-3 text-sm lg:w-40"
          >
            <option value="">全部状态</option>
            <option value="ONLINE">在线</option>
            <option value="CONNECTING">连接中</option>
            <option value="OFFLINE">离线</option>
            <option value="SESSION_EXPIRED">会话过期</option>
            <option value="ERROR">连接异常</option>
            <option value="DISABLED">已停用</option>
          </select>
          <div class="flex gap-2">
            <Button type="submit" variant="outline" :disabled="loading">查询</Button>
            <Button type="button" variant="ghost" :disabled="loading" @click="resetFilters"
              >重置</Button
            >
          </div>
        </form>

        <div v-if="bots.length" class="mobile-table-scroll" aria-label="微信 Bot 账户表格">
          <table class="w-full min-w-[1020px] text-sm">
            <thead class="border-b bg-muted/20 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3 font-medium">Bot 账户</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 font-medium">消息</th>
                <th class="px-4 py-3 font-medium">连接时间</th>
                <th class="px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="bot in bots"
                :key="bot.id"
                class="border-b last:border-0 hover:bg-muted/20"
              >
                <td class="px-4 py-4">
                  <div class="font-medium">{{ bot.name }}</div>
                  <div class="mt-1 max-w-[320px] truncate font-mono text-xs text-muted-foreground">
                    {{ bot.botId }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <Badge :variant="wechatStatusVariant(bot.status)">{{
                    wechatStatusLabel(bot.status)
                  }}</Badge>
                  <p
                    v-if="bot.lastError"
                    class="mt-1 max-w-[280px] truncate text-xs text-destructive"
                    :title="bot.lastError"
                  >
                    {{ bot.lastError }}
                  </p>
                </td>
                <td class="px-4 py-4">
                  <div>{{ bot.messageCount }} 条</div>
                  <div class="mt-1 text-xs text-muted-foreground">
                    {{ formatWechatTime(bot.lastMessageAt) }}
                  </div>
                </td>
                <td class="px-4 py-4 text-muted-foreground">
                  {{ formatWechatTime(bot.connectedAt) }}
                </td>
                <td class="px-4 py-4 text-right">
                  <div class="flex justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      title="消息控制台"
                      @click="consoleBot = bot"
                    >
                      <MessageCircle class="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      title="重新连接"
                      :disabled="!bot.enabled || actionID === bot.id"
                      @click="reconnect(bot)"
                    >
                      <LoaderCircle v-if="actionID === bot.id" class="h-4 w-4 animate-spin" />
                      <RefreshCw v-else class="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" title="重新扫码" @click="openBind(bot)">
                      <QrCode class="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" title="重命名" @click="openRename(bot)">
                      <Pencil class="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      :title="bot.enabled ? '停用' : '启用'"
                      :disabled="actionID === bot.id"
                      @click="toggleEnabled(bot)"
                    >
                      <Power class="h-4 w-4" :class="{ 'text-muted-foreground': !bot.enabled }" />
                    </Button>
                    <Button variant="ghost" size="icon" title="删除" @click="deleteTarget = bot">
                      <Trash2 class="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else-if="!loading" class="px-6 py-16 text-center">
          <MessageCircle class="mx-auto h-10 w-10 text-muted-foreground/50" />
          <h3 class="mt-4 font-medium">暂无微信 Bot</h3>
          <Button class="mt-4 gap-2" @click="openBind()"><Plus class="h-4 w-4" />扫码绑定</Button>
        </div>
        <div v-else class="px-6 py-16 text-center text-sm text-muted-foreground">
          正在加载账户...
        </div>

        <AppPagination
          :total="total"
          :page="page"
          :page-size="pageSize"
          :loading="loading"
          @change-page="
            (value) => {
              page = value
              loadBots()
            }
          "
          @change-page-size="
            (value) => {
              pageSize = value
              page = 1
              loadBots()
            }
          "
        />
      </CardContent>
    </Card>

    <WechatBindDialog
      :open="bindOpen"
      :loading="bindLoading"
      :polling="bindPolling"
      :qr-code="bindQRCode"
      :status="bindStatus"
      :rebind="Boolean(bindTarget)"
      @close="closeBind"
      @restart="startBinding"
    />

    <WechatMessageConsole
      :open="Boolean(consoleBot)"
      :bot="consoleBot"
      @close="consoleBot = null"
    />

    <Dialog
      :open="Boolean(renameTarget)"
      @update:open="(value) => !value && !renaming && (renameTarget = null)"
    >
      <DialogFixedContent class="sm:max-w-md" title="重命名微信 Bot">
        <div class="space-y-1.5">
          <Label for="wechat-bot-rename">显示名称</Label>
          <Input
            id="wechat-bot-rename"
            v-model="renameName"
            maxlength="100"
            autofocus
            @keydown.enter.prevent="confirmRename"
          />
        </div>
        <template #footer>
          <Button variant="outline" :disabled="renaming" @click="renameTarget = null">取消</Button>
          <Button :disabled="renaming || !renameName.trim()" @click="confirmRename">保存</Button>
        </template>
      </DialogFixedContent>
    </Dialog>

    <AlertDialog
      :open="Boolean(deleteTarget)"
      @update:open="(value) => !value && !deleting && (deleteTarget = null)"
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除“{{ deleteTarget?.name }}”？</AlertDialogTitle>
          <AlertDialogDescription>
            该操作会断开连接，并删除此 Bot 的联系人上下文和本地消息记录。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="deleting">取消</AlertDialogCancel>
          <Button variant="destructive" :disabled="deleting" @click="confirmDelete"
            >确认删除</Button
          >
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
