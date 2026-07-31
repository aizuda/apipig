<script setup lang="ts">
import { computed } from 'vue'
import { Check, Copy } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent } from '@tabtab/ui'

const props = defineProps<{
  token: string
  copied: boolean
}>()

const emit = defineEmits<{
  close: []
  copy: [token: string]
}>()

const createdToken = computed({
  get: () => props.token,
  set: (value) => {
    if (!value) emit('close')
  },
})
const copiedTokenId = computed(() => (props.copied ? 'created' : ''))

function copyCreatedToken(token: string) {
  emit('copy', token)
}
</script>

<template>
  <Dialog
    :open="Boolean(createdToken)"
    @update:open="
      (open) => {
        if (!open) createdToken = ''
      }
    "
  >
    <DialogFixedContent
      title="API 密钥创建成功"
      description="出于安全考虑，完整密钥只展示这一次。关闭弹窗前请复制并保存到安全位置。"
      class="sm:max-w-lg"
    >
      <div class="rounded-lg border border-amber-300 bg-amber-50 p-4 text-amber-950">
        <code class="block break-all rounded bg-white/80 p-3 text-sm">{{ createdToken }}</code>
      </div>
      <template #footer>
        <Button variant="outline" @click="createdToken = ''">我已保存</Button>
        <Button @click="copyCreatedToken(createdToken)">
          <Check v-if="copiedTokenId === 'created'" class="mr-1.5 h-4 w-4" />
          <Copy v-else class="mr-1.5 h-4 w-4" />
          {{ copiedTokenId === 'created' ? '已复制' : '复制 API 密钥' }}
        </Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
