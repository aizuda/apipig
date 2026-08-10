<script setup lang="ts">
import { Copy, LoaderCircle, RefreshCw } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent, toast } from '@tabtab/ui'
import type { WechatWebhookCredentials } from '@/api/ai-applications/wechat-bot'
import { copyText } from '@/utils/clipboard'

defineProps<{
  open: boolean
  botName: string
  credentials: WechatWebhookCredentials | null
  loading: boolean
}>()

const emit = defineEmits<{
  close: []
  rotate: []
}>()

const defaultRequestBody = JSON.stringify({ content: '消息内容' }, null, 2)

async function copy(value: string, label: string) {
  if (!value) return
  try {
    await copyText(value)
    toast.success(`${label}已复制`)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '复制失败')
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && emit('close')">
    <DialogFixedContent title="Webhook 接入凭据" :description="botName" class="sm:max-w-2xl">
      <div v-if="loading" class="flex h-44 items-center justify-center text-muted-foreground">
        <LoaderCircle class="h-5 w-5 animate-spin" />
      </div>
      <div v-else-if="credentials" class="divide-y border-y">
        <div class="py-4">
          <div class="mb-2 flex items-center justify-between gap-3">
            <span class="text-sm font-medium">Webhook URL</span>
            <Button
              size="icon"
              variant="outline"
              title="复制 URL"
              @click="copy(credentials.webhookUrl, 'URL')"
            >
              <Copy class="h-4 w-4" />
            </Button>
          </div>
          <code class="block break-all bg-muted/40 px-3 py-2 text-xs leading-5">{{
            credentials.webhookUrl
          }}</code>
        </div>

        <div class="py-4">
          <div class="mb-2 flex items-center justify-between gap-3">
            <span class="text-sm font-medium">签名密钥</span>
            <Button
              v-if="credentials.webhookSecret"
              size="icon"
              variant="outline"
              title="复制签名密钥"
              @click="copy(credentials.webhookSecret, '签名密钥')"
            >
              <Copy class="h-4 w-4" />
            </Button>
          </div>
          <code
            v-if="credentials.webhookSecret"
            class="block break-all bg-muted/40 px-3 py-2 text-xs leading-5"
            >{{ credentials.webhookSecret }}</code
          >
          <p v-else class="text-sm text-muted-foreground">密钥已隐藏，可轮换后重新获取。</p>
          <p
            v-if="credentials.webhookSecret"
            class="mt-2 text-xs text-amber-700 dark:text-amber-300"
          >
            密钥仅显示一次，请只保存在调用方的安全配置中，不要放入请求头。
          </p>
        </div>

        <div class="py-4">
          <div class="mb-2 flex items-center justify-between gap-3">
            <span class="text-sm font-medium">请求体（默认最新会话）</span>
            <Button
              size="icon"
              variant="outline"
              title="复制请求体"
              @click="copy(defaultRequestBody, '请求体')"
            >
              <Copy class="h-4 w-4" />
            </Button>
          </div>
          <pre
            class="overflow-auto bg-muted/40 px-3 py-2 text-xs leading-5"
          ><code>{{ defaultRequestBody }}</code></pre>
          <p class="mt-2 text-xs text-muted-foreground">
            请求头需同时提供 <code>X-Webhook-Timestamp</code> 和
            <code>X-Webhook-Signature</code>；签名为
            <code>Base64(HMAC-SHA256(timestamp + "\n" + secret))</code>。 可传
            <code>userId</code> 指定仍在有效期内的联系人会话。
          </p>
        </div>
      </div>

      <template #footer>
        <Button variant="outline" :disabled="loading" class="gap-2" @click="emit('rotate')">
          <RefreshCw class="h-4 w-4" />轮换密钥
        </Button>
        <Button @click="emit('close')">完成</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
