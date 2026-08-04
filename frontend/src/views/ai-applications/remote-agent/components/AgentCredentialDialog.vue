<script setup lang="ts">
import { Copy, Download, KeyRound } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent, toast } from '@tabtab/ui'
import type { RemoteAgentCredential } from '@/api/ai-applications/remote-agent'
import { copyText } from '@/utils/clipboard'

defineProps<{ open: boolean; credential: RemoteAgentCredential | null }>()
const emit = defineEmits<{ close: [] }>()

async function copy(value: string, label: string) {
  try {
    await copyText(value)
    toast.success(`${label}已复制`)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '复制失败')
  }
}

function downloadConfig(content: string) {
  const url = URL.createObjectURL(new Blob([content], { type: 'application/yaml;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = 'remote-agent.yaml'
  link.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && emit('close')">
    <DialogFixedContent
      title="Agent 接入配置"
      description="注册 Token 仅在本次操作后显示。"
      class="sm:max-w-2xl"
    >
      <div v-if="credential" class="space-y-4">
        <div class="rounded-md border border-amber-500/40 bg-amber-500/5 px-4 py-3 text-sm">
          <div class="flex items-center gap-2 font-medium">
            <KeyRound class="h-4 w-4" />注册 Token
          </div>
          <div class="mt-2 flex items-center gap-2">
            <code class="min-w-0 flex-1 break-all rounded bg-muted px-2 py-1.5 text-xs">{{
              credential.registrationToken
            }}</code>
            <Button
              size="icon"
              variant="outline"
              title="复制 Token"
              @click="copy(credential.registrationToken, 'Token')"
              ><Copy class="h-4 w-4"
            /></Button>
          </div>
        </div>
        <div>
          <div class="mb-2 flex items-center justify-between">
            <span class="text-sm font-medium">remote-agent.yaml</span>
            <div class="flex gap-2">
              <Button
                size="icon"
                variant="outline"
                title="下载配置"
                @click="downloadConfig(credential.configYaml)"
                ><Download class="h-4 w-4"
              /></Button>
              <Button
                size="sm"
                variant="outline"
                class="gap-2"
                @click="copy(credential.configYaml, '配置')"
                ><Copy class="h-4 w-4" />复制配置</Button
              >
            </div>
          </div>
          <pre
            class="max-h-[45vh] overflow-auto rounded-md border bg-muted/30 p-4 text-xs leading-5"
          ><code>{{ credential.configYaml }}</code></pre>
        </div>
      </div>
      <template #footer><Button @click="emit('close')">完成</Button></template>
    </DialogFixedContent>
  </Dialog>
</template>
