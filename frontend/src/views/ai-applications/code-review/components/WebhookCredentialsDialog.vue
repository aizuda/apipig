<script setup lang="ts">
import { Button, Dialog, DialogFixedContent, toast } from '@tabtab/ui'

defineProps<{
  open: boolean
  projectName: string
  webhookUrl: string
  webhookSecret: string
}>()

const emit = defineEmits<{
  close: []
}>()

async function copy(value: string) {
  if (!value) return
  await navigator.clipboard.writeText(value)
  toast.success('已复制到剪贴板')
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && emit('close')">
    <DialogFixedContent
      title="WebHook 接入信息"
      :description="`将以下信息配置到 ${projectName || 'Git 平台'} 的 WebHook 设置中。`"
      class="sm:max-w-2xl"
    >
      <div class="space-y-4">
        <div class="rounded-lg border bg-muted/20 p-4">
          <div class="mb-2 flex items-center justify-between gap-3">
            <span class="text-sm font-medium">Payload URL</span>
            <Button size="sm" variant="outline" @click="copy(webhookUrl)">复制 URL</Button>
          </div>
          <code class="block break-all rounded bg-background px-3 py-2 text-xs leading-5">{{
            webhookUrl
          }}</code>
        </div>
        <div v-if="webhookSecret" class="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4">
          <div class="mb-2 flex items-center justify-between gap-3">
            <span class="text-sm font-medium">WebHook Secret</span>
            <Button size="sm" variant="outline" @click="copy(webhookSecret)">复制 Secret</Button>
          </div>
          <code class="block break-all rounded bg-background px-3 py-2 text-xs leading-5">{{
            webhookSecret
          }}</code>
          <p class="mt-2 text-xs text-amber-700 dark:text-amber-300">
            Secret 仅在创建或轮换后显示一次，请立即保存到 Git 平台。
          </p>
        </div>
        <div v-else class="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
          当前仅可查看 WebHook URL。如需新 Secret，请编辑项目并勾选“轮换 WebHook Secret”。
        </div>
        <div class="rounded-lg bg-muted/30 p-4 text-sm leading-6 text-muted-foreground">
          GitHub 请选择 Push 和 Pull requests；GitLab/Gitee 请选择 Push 和 Merge/Pull Request 事件。
        </div>
      </div>
      <template #footer>
        <Button @click="emit('close')">完成</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
