<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { ArrowDown, ArrowUp, Tags, Trash2 } from '@lucide/vue'
import { Badge, Button, Dialog, DialogFixedContent, Input, Label } from '@tabtab/ui'
import type { AccessTokenTag } from '@/api/ai-gateway'
import GatewayResourceConfirmDialogs from '../components/GatewayResourceConfirmDialogs.vue'

const props = defineProps<{
  open: boolean
  loading: boolean
  errorMessage: string
  form: { name: string; remark: string }
  tags: AccessTokenTag[]
  editingId: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  reset: []
  save: []
  move: [id: string | undefined, direction: -1 | 1]
  edit: [tag: AccessTokenTag]
  delete: [id: string | undefined]
}>()

const tokenTagManagerOpen = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value),
})
const tokenTagLoading = computed(() => props.loading)
const tokenTagErrorMessage = computed(() => props.errorMessage)
const tokenTagForm = toRef(props, 'form')
const sortedTokenTags = toRef(props, 'tags')
const editingTokenTagId = computed(() => props.editingId)
const pendingDeleteTokenTag = ref<AccessTokenTag | null>(null)

function resetTokenTagForm() {
  emit('reset')
}
function saveTokenTag() {
  emit('save')
}
function moveTokenTag(id: string | undefined, direction: -1 | 1) {
  emit('move', id, direction)
}
function editTokenTag(tag: AccessTokenTag) {
  emit('edit', tag)
}
function requestDeleteTokenTag(tag: AccessTokenTag) {
  pendingDeleteTokenTag.value = tag
}
function closeDeleteTokenTagConfirm() {
  pendingDeleteTokenTag.value = null
}
function confirmDeleteTokenTag() {
  const id = pendingDeleteTokenTag.value?.id
  pendingDeleteTokenTag.value = null
  if (id) emit('delete', id)
}
function handleTokenTagManagerOpenChange(open: boolean) {
  if (open) return
  pendingDeleteTokenTag.value = null
  resetTokenTagForm()
}
</script>

<template>
  <Dialog v-model:open="tokenTagManagerOpen" @update:open="handleTokenTagManagerOpenChange">
    <DialogFixedContent
      class="sm:max-w-2xl"
      title="API 密钥标签管理"
      description="维护 API 密钥使用的标签，支持自定义排序与备注。"
    >
      <div class="space-y-4">
        <div
          v-if="tokenTagErrorMessage"
          class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ tokenTagErrorMessage }}
        </div>
        <div
          class="grid gap-3 rounded-xl border bg-muted/20 p-3 sm:grid-cols-[minmax(0,180px)_minmax(0,1fr)_auto] sm:items-end"
        >
          <div class="space-y-1.5">
            <Label for="api-key-tag-name">标签名称</Label>
            <Input
              id="api-key-tag-name"
              v-model="tokenTagForm.name"
              maxlength="30"
              placeholder="例如：生产环境"
              @keydown.enter.prevent="saveTokenTag"
            />
          </div>
          <div class="space-y-1.5">
            <Label for="api-key-tag-remark">备注</Label>
            <Input
              id="api-key-tag-remark"
              v-model="tokenTagForm.remark"
              maxlength="120"
              placeholder="说明标签用途或使用范围"
              @keydown.enter.prevent="saveTokenTag"
            />
          </div>
          <div class="flex gap-2">
            <Button
              v-if="editingTokenTagId"
              variant="outline"
              class="flex-1 sm:flex-none"
              @click="resetTokenTagForm"
            >
              取消
            </Button>
            <Button class="flex-1 sm:flex-none" :disabled="tokenTagLoading" @click="saveTokenTag">
              {{ editingTokenTagId ? '保存' : '新增' }}
            </Button>
          </div>
        </div>

        <div class="overflow-hidden rounded-xl border">
          <div class="flex items-center justify-between border-b bg-muted/40 px-4 py-2.5">
            <div class="text-sm font-medium">标签列表</div>
            <div class="text-xs text-muted-foreground">共 {{ sortedTokenTags.length }} 个</div>
          </div>
          <div
            v-if="tokenTagLoading && sortedTokenTags.length === 0"
            class="px-4 py-10 text-center text-sm text-muted-foreground"
          >
            标签加载中...
          </div>
          <div
            v-else-if="sortedTokenTags.length === 0"
            class="flex flex-col items-center justify-center px-4 py-10 text-center"
          >
            <Tags class="mb-3 h-8 w-8 text-muted-foreground/50" />
            <p class="text-sm font-medium">暂无标签</p>
            <p class="mt-1 text-xs text-muted-foreground">在上方填写名称和备注后新增。</p>
          </div>
          <div v-else class="max-h-[45vh] divide-y overflow-y-auto">
            <div
              v-for="(tag, index) in sortedTokenTags"
              :key="tag.id"
              class="flex flex-col gap-3 p-3 sm:flex-row sm:items-center"
            >
              <div class="flex min-w-0 flex-1 items-start gap-3">
                <Badge variant="outline" class="mt-0.5 shrink-0 tabular-nums">
                  {{ index + 1 }}
                </Badge>
                <div class="min-w-0">
                  <div class="font-medium">{{ tag.name }}</div>
                  <p class="mt-1 break-words text-sm text-muted-foreground">
                    {{ tag.remark || '暂无备注' }}
                  </p>
                </div>
              </div>
              <div class="flex items-center justify-end gap-1">
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8"
                  :disabled="tokenTagLoading || index === 0"
                  :aria-label="`${tag.name}上移`"
                  @click="moveTokenTag(tag.id, -1)"
                >
                  <ArrowUp class="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8"
                  :disabled="tokenTagLoading || index === sortedTokenTags.length - 1"
                  :aria-label="`${tag.name}下移`"
                  @click="moveTokenTag(tag.id, 1)"
                >
                  <ArrowDown class="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  :disabled="tokenTagLoading"
                  @click="editTokenTag(tag)"
                  >编辑</Button
                >
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8"
                  :disabled="tokenTagLoading"
                  :aria-label="`删除${tag.name}`"
                  @click="requestDeleteTokenTag(tag)"
                >
                  <Trash2 class="h-4 w-4 text-destructive" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="tokenTagManagerOpen = false">关闭</Button>
      </template>
    </DialogFixedContent>
  </Dialog>

  <GatewayResourceConfirmDialogs
    :delete-target="pendingDeleteTokenTag"
    :loading="tokenTagLoading"
    delete-description="删除后将同时清除该标签与 API 密钥的关联数据，且无法恢复。"
    @close-delete="closeDeleteTokenTagConfirm"
    @confirm-delete="confirmDeleteTokenTag"
  />
</template>
