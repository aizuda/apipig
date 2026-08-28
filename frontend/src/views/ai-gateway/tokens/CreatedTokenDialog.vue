<script setup lang="ts">
import { computed } from 'vue'
import { Check, Copy } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent } from '@tabtab/ui'

const props = withDefaults(
  defineProps<{
    token: string
    copied: boolean
    mode?: 'create' | 'reset'
  }>(),
  { mode: 'create' },
)

const emit = defineEmits<{
  close: []
  copy: [token: string]
}>()

const displayedToken = computed({
  get: () => props.token,
  set: (value) => {
    if (!value) emit('close')
  },
})
const dialogTitle = computed(() =>
  props.mode === 'reset' ? 'API 密钥重置成功' : 'API 密钥创建成功',
)
const dialogDescription = computed(() =>
  props.mode === 'reset'
    ? '旧密钥已失效。出于安全考虑，新密钥只展示这一次，关闭弹窗前请复制并保存到安全位置。'
    : '出于安全考虑，完整密钥只展示这一次。关闭弹窗前请复制并保存到安全位置。',
)

function copyCreatedToken(token: string) {
  emit('copy', token)
}
</script>

<template>
  <Dialog
    :open="Boolean(displayedToken)"
    @update:open="
      (open) => {
        if (!open) displayedToken = ''
      }
    "
  >
    <DialogFixedContent :title="dialogTitle" :description="dialogDescription" class="sm:max-w-lg">
      <div class="rounded-lg border border-amber-300 bg-amber-50 p-4 text-amber-950">
        <code class="block break-all rounded bg-white/80 p-3 text-sm">{{ displayedToken }}</code>
      </div>
      <template #footer>
        <Button variant="outline" @click="displayedToken = ''">我已保存</Button>
        <Button @click="copyCreatedToken(displayedToken)">
          <Check v-if="copied" class="mr-1.5 h-4 w-4" />
          <Copy v-else class="mr-1.5 h-4 w-4" />
          {{ copied ? '已复制' : '复制 API 密钥' }}
        </Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
