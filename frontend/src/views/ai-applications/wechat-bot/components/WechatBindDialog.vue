<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { LoaderCircle, QrCode as QrCodeIcon, RefreshCw } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent } from '@tabtab/ui'
import QRCode from 'qrcode'

const props = defineProps<{
  open: boolean
  loading: boolean
  polling: boolean
  qrCode: string
  status: string
  rebind: boolean
}>()

const emit = defineEmits<{
  close: []
  restart: []
}>()

const qrCanvas = ref<HTMLCanvasElement | null>(null)
const qrRenderError = ref('')
let renderVersion = 0

async function renderQRCode(open: boolean, content: string) {
  const version = ++renderVersion
  qrRenderError.value = ''
  if (!open || !content) return
  await nextTick()
  if (version !== renderVersion || !qrCanvas.value) return
  try {
    await QRCode.toCanvas(qrCanvas.value, content, {
      width: 240,
      margin: 2,
      errorCorrectionLevel: 'M',
      color: { dark: '#000000', light: '#ffffff' },
    })
  } catch {
    if (version === renderVersion) qrRenderError.value = '二维码生成失败，请重新获取'
  }
}

watch(
  () => [props.open, props.qrCode] as const,
  ([open, content]) => void renderQRCode(open, content),
  { immediate: true },
)

function statusText() {
  if (props.loading) return '正在获取登录二维码...'
  if (props.status === 'SCANNED') return '已扫码，请在微信中确认'
  if (props.status === 'CONFIRMED') return '绑定成功，正在建立消息连接'
  if (props.status === 'EXPIRED') return '二维码已刷新，请重新扫码'
  if (props.polling) return '等待微信扫码'
  return '等待获取二维码'
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && emit('close')">
    <DialogFixedContent
      class="sm:max-w-md"
      :title="rebind ? '重新扫码绑定' : '绑定微信 Bot'"
      description="使用微信扫描二维码并在手机端确认。"
    >
      <div class="space-y-5">
        <div class="flex min-h-[292px] flex-col items-center justify-center border bg-muted/20 p-5">
          <div v-if="qrCode" class="relative h-60 w-60 bg-white">
            <canvas
              ref="qrCanvas"
              class="block h-60 w-60"
              role="img"
              aria-label="微信 Bot 登录二维码"
            />
            <div
              v-if="qrRenderError"
              class="absolute inset-0 flex items-center justify-center bg-white px-6 text-center text-sm text-destructive"
            >
              {{ qrRenderError }}
            </div>
          </div>
          <LoaderCircle v-else-if="loading" class="h-9 w-9 animate-spin text-muted-foreground" />
          <QrCodeIcon v-else class="h-12 w-12 text-muted-foreground/50" />
        </div>

        <div class="flex items-center justify-center gap-2 text-sm text-muted-foreground">
          <LoaderCircle v-if="polling" class="h-4 w-4 animate-spin" />
          <span>{{ statusText() }}</span>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="emit('close')">关闭</Button>
        <Button
          v-if="!polling && status !== 'CONFIRMED'"
          class="gap-2"
          :disabled="loading"
          @click="emit('restart')"
        >
          <RefreshCw class="h-4 w-4" />重新获取
        </Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
