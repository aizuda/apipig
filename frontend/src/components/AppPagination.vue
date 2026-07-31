<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from '@lucide/vue'
import { Button } from '@tabtab/ui'

const props = withDefaults(
  defineProps<{
    total: number
    page: number
    pageSize: number
    loading?: boolean
    pageSizeOptions?: number[]
    showPageSize?: boolean
    showTotal?: boolean
  }>(),
  {
    loading: false,
    pageSizeOptions: () => [10, 20, 50],
    showPageSize: true,
    showTotal: true,
  },
)

const emit = defineEmits<{
  changePage: [page: number]
  changePageSize: [pageSize: number]
}>()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const pageNumbers = computed(() => {
  const start = Math.max(1, Math.min(props.page - 2, totalPages.value - 4))
  const end = Math.min(totalPages.value, start + 4)
  return Array.from({ length: end - start + 1 }, (_, index) => start + index)
})

function updatePageSize(event: Event) {
  emit('changePageSize', Number((event.target as HTMLSelectElement).value))
}
</script>

<template>
  <div
    v-if="total > 0"
    class="flex flex-col gap-3 border-t px-3 py-3 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-4"
  >
    <div v-if="showTotal || showPageSize" class="flex items-center gap-2">
      <span v-if="showTotal">共 {{ total }} 条</span>
      <select
        v-if="showPageSize"
        :value="pageSize"
        class="h-8 rounded-md border bg-background px-2 text-sm text-foreground"
        :disabled="loading"
        @change="updatePageSize"
      >
        <option v-for="option in pageSizeOptions" :key="option" :value="option">
          {{ option }} 条/页
        </option>
      </select>
    </div>
    <div class="flex items-center justify-between gap-1 sm:hidden">
      <Button
        variant="outline"
        size="sm"
        class="min-w-20"
        :disabled="loading || page <= 1"
        @click="emit('changePage', page - 1)"
      >
        <ChevronLeft class="mr-1 h-4 w-4" />上一页
      </Button>
      <span class="px-2 tabular-nums">{{ page }} / {{ totalPages }}</span>
      <Button
        variant="outline"
        size="sm"
        class="min-w-20"
        :disabled="loading || page >= totalPages"
        @click="emit('changePage', page + 1)"
      >
        下一页<ChevronRight class="ml-1 h-4 w-4" />
      </Button>
    </div>
    <div class="hidden items-center gap-1 sm:flex">
      <Button
        variant="ghost"
        size="icon"
        class="h-8 w-8"
        :disabled="loading || page <= 1"
        @click="emit('changePage', page - 1)"
      >
        <ChevronLeft class="h-4 w-4" />
      </Button>
      <Button
        v-for="pageNumber in pageNumbers"
        :key="pageNumber"
        :variant="pageNumber === page ? 'outline' : 'ghost'"
        size="icon"
        class="h-8 w-8"
        :disabled="loading"
        @click="emit('changePage', pageNumber)"
      >
        {{ pageNumber }}
      </Button>
      <Button
        variant="ghost"
        size="icon"
        class="h-8 w-8"
        :disabled="loading || page >= totalPages"
        @click="emit('changePage', page + 1)"
      >
        <ChevronRight class="h-4 w-4" />
      </Button>
    </div>
  </div>
</template>
