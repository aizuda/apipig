import { onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { toast } from '@tabtab/ui'
import type { GatewayPageParams, PageResult } from '@/api/ai-gateway'

interface GatewayResourceRecord {
  id?: string
  name: string
  status: number
}

interface GatewayResourcePageOptions<T extends GatewayResourceRecord> {
  fetchPage: (params: GatewayPageParams) => Promise<PageResult<T>>
  deleteRecord: (id: string) => Promise<boolean>
  changeStatus: (id: string, status: number) => Promise<boolean>
}

export function useGatewayResourcePage<T extends GatewayResourceRecord>(
  options: GatewayResourcePageOptions<T>,
) {
  const loading = ref(false)
  const errorMessage = ref('')
  const editorErrorMessage = ref('')
  const editorOpen = ref(false)
  const searchQuery = ref('')
  const records = ref<T[]>([]) as Ref<T[]>
  const currentPage = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const deleteTarget = ref<{ id: string; name: string } | null>(null)
  const statusTarget = ref<{ item: T; targetStatus: 1 | 2 } | null>(null)
  const statusChangingKey = ref('')
  let loadSequence = 0
  let searchTimer: ReturnType<typeof setTimeout> | undefined

  async function loadData() {
    const sequence = ++loadSequence
    loading.value = true
    errorMessage.value = ''
    try {
      const page = await options.fetchPage({
        page: currentPage.value,
        pageSize: pageSize.value,
        keyword: searchQuery.value.trim() || undefined,
      })
      if (sequence !== loadSequence) return
      records.value = page.records
      total.value = page.total
      currentPage.value = page.page || currentPage.value
      pageSize.value = page.pageSize || pageSize.value
    } catch (error) {
      if (sequence !== loadSequence) return
      errorMessage.value = error instanceof Error ? error.message : '数据加载失败'
      toast.error(errorMessage.value)
    } finally {
      if (sequence === loadSequence) loading.value = false
    }
  }

  async function run(action: () => Promise<void>, successMessage?: string) {
    loading.value = true
    if (editorOpen.value) editorErrorMessage.value = ''
    else errorMessage.value = ''
    try {
      await action()
      if (successMessage) toast.success(successMessage)
    } catch (error) {
      const message = error instanceof Error ? error.message : '操作失败'
      if (editorOpen.value) editorErrorMessage.value = message
      else errorMessage.value = message
      toast.error(message)
    } finally {
      loading.value = false
    }
  }

  function changePage(page: number) {
    const totalPages = Math.max(1, Math.ceil(total.value / pageSize.value))
    if (page < 1 || page > totalPages || page === currentPage.value || loading.value) return
    currentPage.value = page
    loadData()
  }

  function changePageSize(size: number) {
    if (loading.value || size === pageSize.value) return
    pageSize.value = size
    currentPage.value = 1
    loadData()
  }

  function requestDelete(id: string | undefined, name: string) {
    if (id) deleteTarget.value = { id, name }
  }

  async function confirmDelete() {
    const target = deleteTarget.value
    if (!target) return
    await run(async () => {
      const success = await options.deleteRecord(target.id)
      if (!success) throw new Error('删除失败')
      deleteTarget.value = null
      if (records.value.length === 1 && currentPage.value > 1) currentPage.value -= 1
      await loadData()
    }, target.name + ' 已删除')
  }

  function requestStatusChange(item: T) {
    if (!item.id || statusChangingKey.value) return
    statusTarget.value = { item, targetStatus: item.status === 1 ? 2 : 1 }
  }

  async function confirmStatusChange() {
    const target = statusTarget.value
    if (!target?.item.id || statusChangingKey.value) return
    statusChangingKey.value = target.item.id
    errorMessage.value = ''
    try {
      const success = await options.changeStatus(target.item.id, target.targetStatus)
      if (!success) throw new Error('状态切换失败')
      target.item.status = target.targetStatus
      statusTarget.value = null
      toast.success(target.item.name + ' 已' + statusLabel(target.targetStatus))
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : '状态切换失败'
      toast.error(errorMessage.value)
    } finally {
      statusChangingKey.value = ''
    }
  }

  function closeEditor() {
    if (!loading.value) {
      editorOpen.value = false
      editorErrorMessage.value = ''
    }
  }

  function statusLabel(status: number) {
    return status === 1 ? '启用' : '禁用'
  }

  function nextStatus(status: number): 1 | 2 {
    return status === 1 ? 2 : 1
  }

  function statusVariant(status: number) {
    return status === 1 ? 'default' : 'secondary'
  }

  watch(
    searchQuery,
    () => {
      if (searchTimer) clearTimeout(searchTimer)
      searchTimer = setTimeout(() => {
        currentPage.value = 1
        loadData()
      }, 300)
    },
    { flush: 'sync' },
  )

  onBeforeUnmount(() => {
    if (searchTimer) clearTimeout(searchTimer)
    loadSequence += 1
  })

  loadData()

  return {
    loading,
    errorMessage,
    editorErrorMessage,
    editorOpen,
    searchQuery,
    records,
    currentPage,
    pageSize,
    total,
    deleteTarget,
    statusTarget,
    statusChangingKey,
    loadData,
    run,
    changePage,
    changePageSize,
    requestDelete,
    confirmDelete,
    requestStatusChange,
    confirmStatusChange,
    closeEditor,
    statusLabel,
    nextStatus,
    statusVariant,
  }
}
