<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Tags } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent } from '@tabtab/ui'
import type { AccessTokenTag } from '@/api/ai-gateway'

const props = defineProps<{
  open: boolean
  loading: boolean
  tokenName: string
  tags: AccessTokenTag[]
  selectedTagIds: string[]
}>()

const emit = defineEmits<{
  close: []
  save: [tagIds: string[]]
}>()

const selectedTagIds = ref<string[]>([])
const labels = {
  title: '\u7f16\u8f91 API \u5bc6\u94a5\u6807\u7b7e',
  available: '\u53ef\u9009\u6807\u7b7e',
  selected: '\u5df2\u9009\u62e9',
  countUnit: '\u4e2a',
  empty:
    '\u6682\u65e0\u53ef\u9009\u6807\u7b7e\uff0c\u8bf7\u5148\u901a\u8fc7\u201c\u6807\u7b7e\u7ba1\u7406\u201d\u65b0\u589e\u6807\u7b7e\u3002',
  noRemark: '\u6682\u65e0\u5907\u6ce8',
  cancel: '\u53d6\u6d88',
  saving: '\u4fdd\u5b58\u4e2d...',
  save: '\u4fdd\u5b58\u6807\u7b7e',
}
const description = computed(
  () =>
    '\u4e3a\u201c' +
    props.tokenName +
    '\u201d\u9009\u62e9\u5173\u8054\u6807\u7b7e\uff0c\u4fdd\u5b58\u540e\u5c06\u7acb\u5373\u5237\u65b0\u5f53\u524d\u5206\u9875\u5217\u8868\u3002',
)
const selectableTags = computed(() =>
  props.tags
    .filter((tag): tag is AccessTokenTag & { id: string } => Boolean(tag.id))
    .sort((left, right) => left.sort - right.sort),
)

watch(
  [() => props.open, () => props.selectedTagIds],
  ([open, tagIds]) => {
    if (open) selectedTagIds.value = [...tagIds]
  },
  { deep: true, immediate: true },
)

function closeDialog() {
  if (!props.loading) emit('close')
}

function saveTags() {
  emit('save', [...selectedTagIds.value])
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && closeDialog()">
    <DialogFixedContent :title="labels.title" :description="description" class="sm:max-w-lg">
      <div class="space-y-3">
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2 text-sm font-medium">
            <Tags class="h-4 w-4 text-primary" />
            {{ labels.available }}
          </div>
          <span class="text-xs text-muted-foreground">
            {{ labels.selected }} {{ selectedTagIds.length }} {{ labels.countUnit }}
          </span>
        </div>

        <div
          v-if="selectableTags.length === 0"
          class="rounded-md border border-dashed p-5 text-center text-sm text-muted-foreground"
        >
          {{ labels.empty }}
        </div>

        <div v-else class="grid max-h-80 gap-2 overflow-y-auto pr-1 sm:grid-cols-2">
          <label
            v-for="tag in selectableTags"
            :key="tag.id"
            class="flex min-w-0 cursor-pointer items-start gap-2 rounded-lg border p-3 transition-colors hover:bg-muted/40"
            :class="{
              'border-primary/50 bg-primary/5': selectedTagIds.includes(tag.id),
            }"
          >
            <input
              v-model="selectedTagIds"
              type="checkbox"
              :value="tag.id"
              class="mt-0.5 h-4 w-4 shrink-0 cursor-pointer accent-primary"
            />
            <span class="min-w-0">
              <span class="block truncate text-sm font-medium" :title="tag.name">{{
                tag.name
              }}</span>
              <span class="mt-0.5 block break-words text-xs text-muted-foreground">
                {{ tag.remark || labels.noRemark }}
              </span>
            </span>
          </label>
        </div>
      </div>

      <template #footer>
        <Button variant="outline" :disabled="loading" @click="closeDialog">
          {{ labels.cancel }}
        </Button>
        <Button :disabled="loading" @click="saveTags">
          {{ loading ? labels.saving : labels.save }}
        </Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
