<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { KeyRound, Pencil, Plus, Search, Trash2 } from '@lucide/vue'
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
  Dialog,
  DialogFixedContent,
  Input,
  Label,
  toast,
} from '@tabtab/ui'
import {
  systemApi,
  type SystemRole,
  type SystemUser,
  type UserDetail,
  type UserSaveParams,
} from '@/api/system'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import UserEditorDialog from './UserEditorDialog.vue'
import { validateLoginPassword } from '@/utils/validation'

defineOptions({ name: 'SettingsUsers' })

const loading = ref(false)
const editorLoading = ref(false)
const errorMessage = ref('')
const editorErrorMessage = ref('')
const users = ref<SystemUser[]>([])
const roles = ref<SystemRole[]>([])
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)
const editorOpen = ref(false)
const editingUser = ref<UserDetail | null>(null)
const deleteTarget = ref<SystemUser | null>(null)
const statusTarget = ref<SystemUser | null>(null)
const resetTarget = ref<SystemUser | null>(null)
const resetPassword = ref('')
const filters = reactive({ username: '', realName: '', phone: '', status: 0 })

function errorText(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}

async function loadUsers() {
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await systemApi.userPage({
      page: currentPage.value,
      pageSize: pageSize.value,
      username: filters.username.trim() || undefined,
      realName: filters.realName.trim() || undefined,
      phone: filters.phone.trim() || undefined,
      status: filters.status || undefined,
    })
    users.value = result.records || []
    total.value = result.total || 0
    currentPage.value = result.page || currentPage.value
    pageSize.value = result.pageSize || pageSize.value
  } catch (error) {
    errorMessage.value = errorText(error, '用户数据加载失败')
    toast.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

async function loadRoles() {
  try {
    roles.value = await systemApi.roleList()
  } catch (error) {
    toast.error(errorText(error, '角色列表加载失败'))
  }
}

function search() {
  currentPage.value = 1
  loadUsers()
}

function resetFilters() {
  Object.assign(filters, { username: '', realName: '', phone: '', status: 0 })
  search()
}

function openCreate() {
  editingUser.value = null
  editorErrorMessage.value = ''
  editorOpen.value = true
}

function openResetPassword(user: SystemUser) {
  resetTarget.value = user
  resetPassword.value = ''
}

async function openEdit(user: SystemUser) {
  if (!user.id) return
  editorLoading.value = true
  editorErrorMessage.value = ''
  try {
    editingUser.value = await systemApi.userGet(user.id)
    editorOpen.value = true
  } catch (error) {
    toast.error(errorText(error, '用户详情加载失败'))
  } finally {
    editorLoading.value = false
  }
}

async function saveUser(form: UserSaveParams) {
  if (!form.username) {
    editorErrorMessage.value = '请输入登录账号'
    return
  }
  const password = form.password?.trim() || ''
  if (!form.id && !password) {
    editorErrorMessage.value = '请输入初始密码'
    return
  }
  if (password) {
    const passwordError = validateLoginPassword(password)
    if (passwordError) {
      editorErrorMessage.value = passwordError
      return
    }
  }
  editorLoading.value = true
  editorErrorMessage.value = ''
  try {
    if (!(await systemApi.userSave(form))) throw new Error('保存用户失败')
    toast.success(form.id ? '用户信息已更新' : '用户已创建')
    editorOpen.value = false
    await loadUsers()
  } catch (error) {
    editorErrorMessage.value = errorText(error, '保存用户失败')
    toast.error(editorErrorMessage.value)
  } finally {
    editorLoading.value = false
  }
}

async function confirmDelete() {
  if (!deleteTarget.value?.id) return
  loading.value = true
  try {
    if (!(await systemApi.userDelete([deleteTarget.value.id]))) throw new Error('删除用户失败')
    toast.success('用户已删除')
    deleteTarget.value = null
    if (users.value.length === 1 && currentPage.value > 1) currentPage.value -= 1
    await loadUsers()
  } catch (error) {
    toast.error(errorText(error, '删除用户失败'))
  } finally {
    loading.value = false
  }
}

async function confirmStatus() {
  const target = statusTarget.value
  if (!target?.id) return
  const targetId = target.id
  const status = target.status === 1 ? 2 : 1
  loading.value = true
  try {
    if (!(await systemApi.userUpdate({ id: targetId, status }))) throw new Error('状态更新失败')
    target.status = status
    statusTarget.value = null
    toast.success(`用户已${status === 1 ? '启用' : '禁用'}`)
  } catch (error) {
    toast.error(errorText(error, '状态更新失败'))
  } finally {
    loading.value = false
  }
}

async function confirmResetPassword() {
  if (!resetTarget.value?.id || !resetPassword.value.trim()) return
  const passwordError = validateLoginPassword(resetPassword.value.trim())
  if (passwordError) {
    toast.error(passwordError)
    return
  }
  editorLoading.value = true
  try {
    if (!(await systemApi.userResetPassword([resetTarget.value.id], resetPassword.value.trim())))
      throw new Error('密码重置失败')
    toast.success('密码已重置')
    resetTarget.value = null
    resetPassword.value = ''
  } catch (error) {
    toast.error(errorText(error, '密码重置失败'))
  } finally {
    editorLoading.value = false
  }
}

function changePage(page: number) {
  currentPage.value = page
  loadUsers()
}
function changePageSize(size: number) {
  pageSize.value = size
  currentPage.value = 1
  loadUsers()
}
function formatTime(value?: number) {
  return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
}

onMounted(() => Promise.all([loadUsers(), loadRoles()]))
</script>

<template>
  <div class="space-y-5">
    <AppPageHeader
      title="用户管理"
      description="维护系统用户、角色分配、启用状态与登录密码。"
      :loading="loading"
      @refresh="loadUsers"
    />

    <div
      class="flex flex-col gap-3 rounded-xl border bg-card p-3 xl:flex-row xl:items-center xl:justify-between"
    >
      <div class="grid w-full gap-2 sm:grid-cols-2 lg:grid-cols-4 xl:max-w-4xl">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            v-model="filters.username"
            class="h-9 pl-9"
            placeholder="登录账号"
            @keyup.enter="search"
          />
        </div>
        <Input
          v-model="filters.realName"
          class="h-9"
          placeholder="真实姓名"
          @keyup.enter="search"
        />
        <Input v-model="filters.phone" class="h-9" placeholder="手机号" @keyup.enter="search" />
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
        class="flex w-full flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center xl:w-auto"
      >
        <span class="whitespace-nowrap">{{ total }} 个用户</span>
        <div class="flex gap-2">
          <Button size="sm" variant="outline" @click="search">查询</Button>
          <Button size="sm" variant="ghost" @click="resetFilters">重置</Button>
          <Button size="sm" @click="openCreate"> <Plus class="mr-1.5 size-4" />新增用户 </Button>
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
        <div class="mobile-table-scroll" aria-label="用户数据表格，可横向滚动">
          <table class="w-full min-w-[980px] text-sm">
            <thead class="border-b bg-muted/40 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3 font-medium">用户</th>
                <th class="px-4 py-3 font-medium">联系方式</th>
                <th class="px-4 py-3 font-medium">工号</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 font-medium">最后登录</th>
                <th class="px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="user in users"
                :key="user.id"
                class="border-b last:border-0 hover:bg-muted/30"
              >
                <td class="px-4 py-3">
                  <div class="font-medium">
                    {{ user.realName || user.nickName || user.username }}
                  </div>
                  <div class="text-xs text-muted-foreground">{{ user.username }}</div>
                </td>
                <td class="px-4 py-3">
                  <div>{{ user.phone || '-' }}</div>
                  <div class="text-xs text-muted-foreground">{{ user.email || '-' }}</div>
                </td>
                <td class="px-4 py-3">{{ user.jobNum || '-' }}</td>
                <td class="px-4 py-3">
                  <button type="button" :disabled="loading" @click="statusTarget = user">
                    <Badge :variant="user.status === 1 ? 'default' : 'secondary'">{{
                      user.status === 1 ? '启用' : '禁用'
                    }}</Badge>
                  </button>
                </td>
                <td class="px-4 py-3 text-muted-foreground">{{ formatTime(user.loginTime) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end gap-1">
                    <Button variant="ghost" size="icon" title="编辑" @click="openEdit(user)"
                      ><Pencil class="size-4"
                    /></Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      title="重置密码"
                      @click="openResetPassword(user)"
                      ><KeyRound class="size-4"
                    /></Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      title="删除"
                      class="text-destructive hover:text-destructive"
                      @click="deleteTarget = user"
                      ><Trash2 class="size-4"
                    /></Button>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && users.length === 0">
                <td colspan="6" class="px-4 py-12 text-center text-muted-foreground">
                  暂无用户数据
                </td>
              </tr>
              <tr v-if="loading && users.length === 0">
                <td colspan="6" class="px-4 py-12 text-center text-muted-foreground">
                  正在加载用户数据...
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

    <UserEditorDialog
      :open="editorOpen"
      :loading="editorLoading"
      :error-message="editorErrorMessage"
      :user="editingUser"
      :roles="roles"
      @close="editorOpen = false"
      @submit="saveUser"
    />

    <Dialog
      :open="Boolean(resetTarget)"
      @update:open="(open) => !open && !editorLoading && (resetTarget = null)"
    >
      <DialogFixedContent
        title="重置用户密码"
        :description="`为 ${resetTarget?.username || ''} 设置新的登录密码。`"
        class="sm:max-w-md"
      >
        <div class="space-y-1.5">
          <Label for="reset-password">新密码</Label
          ><Input
            id="reset-password"
            v-model="resetPassword"
            type="password"
            placeholder="请输入新密码"
            @keyup.enter="confirmResetPassword"
          />
        </div>
        <template #footer
          ><Button variant="outline" :disabled="editorLoading" @click="resetTarget = null"
            >取消</Button
          ><Button :disabled="editorLoading || !resetPassword.trim()" @click="confirmResetPassword"
            >确认重置</Button
          ></template
        >
      </DialogFixedContent>
    </Dialog>

    <AlertDialog
      :open="Boolean(deleteTarget)"
      @update:open="(open) => !open && !loading && (deleteTarget = null)"
    >
      <AlertDialogContent
        ><AlertDialogHeader
          ><AlertDialogTitle>确认删除用户 {{ deleteTarget?.username }}？</AlertDialogTitle
          ><AlertDialogDescription
            >删除后无法恢复，用户关联角色也将失效。</AlertDialogDescription
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
            >确认{{ statusTarget?.status === 1 ? '禁用' : '启用' }}用户？</AlertDialogTitle
          ><AlertDialogDescription
            >用户“{{ statusTarget?.username }}”状态将立即更新。</AlertDialogDescription
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
