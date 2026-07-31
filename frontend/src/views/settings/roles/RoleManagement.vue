<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Pencil, Plus, Search, Trash2 } from '@lucide/vue'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  Card,
  CardContent,
  Input,
  toast,
} from '@tabtab/ui'
import {
  systemApi,
  type ResourceTreeNode,
  type RoleDetail,
  type RoleSaveParams,
  type SystemRole,
} from '@/api/system'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import RoleEditorDialog from './RoleEditorDialog.vue'

defineOptions({ name: 'SettingsRoles' })

const loading = ref(false)
const editorLoading = ref(false)
const errorMessage = ref('')
const editorErrorMessage = ref('')
const roles = ref<SystemRole[]>([])
const resources = ref<ResourceTreeNode[]>([])
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)
const editorOpen = ref(false)
const editingRole = ref<RoleDetail | null>(null)
const deleteTarget = ref<SystemRole | null>(null)
const statusTarget = ref<SystemRole | null>(null)
const filters = reactive({ name: '', alias: '', status: 0 })

function errorText(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}

async function loadRoles() {
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await systemApi.rolePage({
      page: currentPage.value,
      pageSize: pageSize.value,
      name: filters.name.trim() || undefined,
      alias: filters.alias.trim() || undefined,
      status: filters.status || undefined,
    })
    roles.value = result.records || []
    total.value = result.total || 0
    currentPage.value = result.page || currentPage.value
    pageSize.value = result.pageSize || pageSize.value
  } catch (error) {
    errorMessage.value = errorText(error, '角色数据加载失败')
    toast.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

async function loadResources() {
  try {
    resources.value = await systemApi.resourceTree()
  } catch (error) {
    toast.error(errorText(error, '菜单权限加载失败'))
  }
}

function search() {
  currentPage.value = 1
  loadRoles()
}
function resetFilters() {
  Object.assign(filters, { name: '', alias: '', status: 0 })
  search()
}

function openCreate() {
  editingRole.value = null
  editorErrorMessage.value = ''
  editorOpen.value = true
}

async function openEdit(role: SystemRole) {
  if (!role.id) return
  editorLoading.value = true
  editorErrorMessage.value = ''
  try {
    editingRole.value = await systemApi.roleGet(role.id)
    editorOpen.value = true
  } catch (error) {
    toast.error(errorText(error, '角色详情加载失败'))
  } finally {
    editorLoading.value = false
  }
}

async function saveRole(form: RoleSaveParams) {
  if (!form.name || !form.alias) {
    editorErrorMessage.value = '请填写角色名称和别名'
    return
  }
  editorLoading.value = true
  editorErrorMessage.value = ''
  try {
    if (!(await systemApi.roleSave(form))) throw new Error('保存角色失败')
    toast.success(form.id ? '角色信息已更新' : '角色已创建')
    editorOpen.value = false
    await loadRoles()
  } catch (error) {
    editorErrorMessage.value = errorText(error, '保存角色失败')
    toast.error(editorErrorMessage.value)
  } finally {
    editorLoading.value = false
  }
}

async function confirmDelete() {
  if (!deleteTarget.value?.id) return
  loading.value = true
  try {
    if (!(await systemApi.roleDelete([deleteTarget.value.id]))) throw new Error('删除角色失败')
    toast.success('角色已删除')
    deleteTarget.value = null
    if (roles.value.length === 1 && currentPage.value > 1) currentPage.value -= 1
    await loadRoles()
  } catch (error) {
    toast.error(errorText(error, '删除角色失败'))
  } finally {
    loading.value = false
  }
}

async function confirmStatus() {
  if (!statusTarget.value?.id) return
  const target = statusTarget.value
  const status = target.status === 1 ? 2 : 1
  loading.value = true
  try {
    if (!(await systemApi.roleUpdate({ id: target.id, status }))) throw new Error('状态更新失败')
    target.status = status
    statusTarget.value = null
    toast.success(`角色已${status === 1 ? '启用' : '禁用'}`)
  } catch (error) {
    toast.error(errorText(error, '状态更新失败'))
  } finally {
    loading.value = false
  }
}

function changePage(page: number) {
  currentPage.value = page
  loadRoles()
}
function changePageSize(size: number) {
  pageSize.value = size
  currentPage.value = 1
  loadRoles()
}
function formatTime(value?: number) {
  return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
}

onMounted(() => Promise.all([loadRoles(), loadResources()]))
</script>

<template>
  <div class="space-y-5">
    <AppPageHeader
      title="角色管理"
      description="维护系统角色、启用状态及菜单访问权限。"
      :loading="loading"
      @refresh="loadRoles"
    />

    <div
      class="flex flex-col gap-3 rounded-xl border bg-card p-3 lg:flex-row lg:items-center lg:justify-between"
    >
      <div class="grid w-full gap-2 sm:grid-cols-3 lg:max-w-2xl">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            v-model="filters.name"
            class="h-9 pl-9"
            placeholder="角色名称"
            @keyup.enter="search"
          />
        </div>
        <Input v-model="filters.alias" class="h-9" placeholder="角色别名" @keyup.enter="search" />
        <select
          v-model.number="filters.status"
          class="h-9 rounded-md border bg-background px-3 text-sm"
        >
          <option :value="0">全部状态</option>
          <option :value="1">启用</option>
          <option :value="2">禁用</option>
        </select>
      </div>
      <div
        class="flex w-full flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center lg:w-auto"
      >
        <span class="whitespace-nowrap">{{ total }} 个角色</span>
        <div class="flex gap-2">
          <Button size="sm" variant="outline" @click="search">查询</Button>
          <Button size="sm" variant="ghost" @click="resetFilters">重置</Button>
          <Button size="sm" @click="openCreate"> <Plus class="mr-1.5 size-4" />新增角色 </Button>
        </div>
      </div>
    </div>

    <div
      v-if="errorMessage"
      class="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ errorMessage }}
    </div>

    <Card class="gap-0 overflow-hidden py-0">
      <CardContent class="p-0">
        <div class="mobile-table-scroll" aria-label="角色数据表格，可横向滚动">
          <table class="w-full min-w-[820px] text-sm">
            <thead class="border-b bg-muted/40 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3 font-medium">角色名称</th>
                <th class="px-4 py-3 font-medium">别名</th>
                <th class="px-4 py-3 font-medium">排序</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 font-medium">更新时间</th>
                <th class="px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="role in roles"
                :key="role.id"
                class="border-b last:border-0 hover:bg-muted/30"
              >
                <td class="px-4 py-3">
                  <div class="font-medium">{{ role.name }}</div>
                  <div class="max-w-72 truncate text-xs text-muted-foreground">
                    {{ role.remark || '暂无备注' }}
                  </div>
                </td>
                <td class="px-4 py-3 font-mono text-xs">{{ role.alias }}</td>
                <td class="px-4 py-3">{{ role.sort }}</td>
                <td class="px-4 py-3">
                  <button type="button" :disabled="loading" @click="statusTarget = role">
                    <Badge :variant="role.status === 1 ? 'default' : 'secondary'">{{
                      role.status === 1 ? '启用' : '禁用'
                    }}</Badge>
                  </button>
                </td>
                <td class="px-4 py-3 text-muted-foreground">
                  {{ formatTime(role.updatedAt || role.createdAt) }}
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end gap-1">
                    <Button variant="ghost" size="icon" title="编辑" @click="openEdit(role)"
                      ><Pencil class="size-4" /></Button
                    ><Button
                      variant="ghost"
                      size="icon"
                      title="删除"
                      class="text-destructive hover:text-destructive"
                      @click="deleteTarget = role"
                      ><Trash2 class="size-4"
                    /></Button>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && roles.length === 0">
                <td colspan="6" class="px-4 py-12 text-center text-muted-foreground">
                  暂无角色数据
                </td>
              </tr>
              <tr v-if="loading && roles.length === 0">
                <td colspan="6" class="px-4 py-12 text-center text-muted-foreground">
                  正在加载角色数据...
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <AppPagination
          :total="total"
          :page="currentPage"
          :page-size="pageSize"
          :loading="loading"
          @change-page="changePage"
          @change-page-size="changePageSize"
        />
      </CardContent>
    </Card>

    <RoleEditorDialog
      :open="editorOpen"
      :loading="editorLoading"
      :error-message="editorErrorMessage"
      :role="editingRole"
      :resources="resources"
      @close="editorOpen = false"
      @submit="saveRole"
    />

    <AlertDialog
      :open="Boolean(deleteTarget)"
      @update:open="(open) => !open && !loading && (deleteTarget = null)"
    >
      <AlertDialogContent
        ><AlertDialogHeader
          ><AlertDialogTitle>确认删除角色 {{ deleteTarget?.name }}？</AlertDialogTitle
          ><AlertDialogDescription
            >删除后无法恢复；若角色仍有关联用户，后台可能拒绝本次操作。</AlertDialogDescription
          ></AlertDialogHeader
        ><AlertDialogFooter
          ><AlertDialogCancel :disabled="loading" @click="deleteTarget = null"
            >取消</AlertDialogCancel
          ><Button variant="destructive" :disabled="loading" @click="confirmDelete"
            >确认删除</Button
          ></AlertDialogFooter
        ></AlertDialogContent
      >
    </AlertDialog>

    <AlertDialog
      :open="Boolean(statusTarget)"
      @update:open="(open) => !open && !loading && (statusTarget = null)"
    >
      <AlertDialogContent
        ><AlertDialogHeader
          ><AlertDialogTitle
            >确认{{ statusTarget?.status === 1 ? '禁用' : '启用' }}角色？</AlertDialogTitle
          ><AlertDialogDescription
            >角色“{{
              statusTarget?.name
            }}”状态将立即更新，并影响关联用户权限。</AlertDialogDescription
          ></AlertDialogHeader
        ><AlertDialogFooter
          ><AlertDialogCancel :disabled="loading" @click="statusTarget = null"
            >取消</AlertDialogCancel
          ><Button :disabled="loading" @click="confirmStatus">确认操作</Button></AlertDialogFooter
        ></AlertDialogContent
      >
    </AlertDialog>
  </div>
</template>
