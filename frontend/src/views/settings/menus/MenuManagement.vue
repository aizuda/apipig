<script setup lang="ts">
import { computed, onMounted, ref, type Component } from 'vue'
import {
  ChevronDown,
  ChevronRight,
  ExternalLink,
  FolderTree,
  MonitorUp,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Trash2,
} from '@lucide/vue'
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
import { systemApi, type ResourceSaveParams, type ResourceTreeNode } from '@/api/system'
import AppPageHeader from '@/components/AppPageHeader.vue'
import MenuIcon from '@/components/menu/MenuIcon.vue'
import MenuEditorDialog from './MenuEditorDialog.vue'

defineOptions({ name: 'SettingsMenus' })

interface VisibleMenu {
  menu: ResourceTreeNode
  depth: number
}

const loading = ref(false)
const editorLoading = ref(false)
const errorMessage = ref('')
const editorErrorMessage = ref('')
const menus = ref<ResourceTreeNode[]>([])
const keyword = ref('')
const selectedId = ref('root')
const expandedIds = ref<Set<string>>(new Set())
const editorOpen = ref(false)
const editingMenu = ref<ResourceTreeNode | null>(null)
const createParentId = ref('1')
const deleteTarget = ref<ResourceTreeNode | null>(null)
const statusTarget = ref<ResourceTreeNode | null>(null)

const filteredMenus = computed(() => filterMenus(menus.value, keyword.value.trim().toLowerCase()))
const visibleMenus = computed(() => flattenVisible(filteredMenus.value))
const selectedMenu = computed(() => findMenu(menus.value, selectedId.value))

function errorText(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}

function findMenu(nodes: ResourceTreeNode[], id: string): ResourceTreeNode | null {
  for (const menu of nodes) {
    if (menu.id === id) return menu
    const found = findMenu(menu.children || [], id)
    if (found) return found
  }
  return null
}

function filterMenus(nodes: ResourceTreeNode[], query: string): ResourceTreeNode[] {
  if (!query) return nodes
  return nodes.flatMap((menu) => {
    const children = filterMenus(menu.children || [], query)
    const matched = [menu.title, menu.alias, menu.path].some((value) =>
      value?.toLowerCase().includes(query),
    )
    return matched || children.length > 0 ? [{ ...menu, children }] : []
  })
}

function flattenVisible(nodes: ResourceTreeNode[], depth = 0): VisibleMenu[] {
  return nodes.flatMap((menu) => {
    const row = { menu, depth }
    const showChildren =
      Boolean(keyword.value.trim()) || Boolean(menu.id && expandedIds.value.has(menu.id))
    return showChildren ? [row, ...flattenVisible(menu.children || [], depth + 1)] : [row]
  })
}

function toggleExpanded(id?: string) {
  if (!id) return
  const next = new Set(expandedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedIds.value = next
}

function expandParents(
  nodes: ResourceTreeNode[],
  targetId: string,
  parents: string[] = [],
): boolean {
  for (const menu of nodes) {
    if (menu.id === targetId) {
      expandedIds.value = new Set([...expandedIds.value, ...parents])
      return true
    }
    if (expandParents(menu.children || [], targetId, menu.id ? [...parents, menu.id] : parents))
      return true
  }
  return false
}

async function loadMenus() {
  loading.value = true
  errorMessage.value = ''
  try {
    menus.value = await systemApi.resourceTree()
    if (selectedId.value !== 'root' && !findMenu(menus.value, selectedId.value))
      selectedId.value = 'root'
  } catch (error) {
    errorMessage.value = errorText(error, '菜单数据加载失败')
    toast.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function selectMenu(menu: ResourceTreeNode) {
  if (!menu.id) return
  selectedId.value = menu.id
}

function openCreate(parentId = '1') {
  editingMenu.value = null
  createParentId.value = parentId
  editorErrorMessage.value = ''
  editorOpen.value = true
}

async function openEdit(menu: ResourceTreeNode) {
  if (!menu.id) return
  editorLoading.value = true
  editorErrorMessage.value = ''
  try {
    const detail = await systemApi.resourceGet(menu.id)
    editingMenu.value = { ...detail, children: menu.children }
    editorOpen.value = true
  } catch (error) {
    toast.error(errorText(error, '菜单详情加载失败'))
  } finally {
    editorLoading.value = false
  }
}

async function saveMenu(form: ResourceSaveParams) {
  if (!form.title || !form.path) {
    editorErrorMessage.value = '请填写菜单名称和路由地址'
    return
  }
  if ((form.type === 2 || form.type === 3) && !form.redirect) {
    editorErrorMessage.value = '内嵌页面和外部链接必须填写跳转地址'
    return
  }
  if (form.query) {
    try {
      JSON.parse(form.query)
    } catch {
      editorErrorMessage.value = '路由参数必须是有效的 JSON'
      return
    }
  }
  editorLoading.value = true
  editorErrorMessage.value = ''
  try {
    const success = form.id
      ? await systemApi.resourceUpdate({ ...form, id: form.id })
      : await systemApi.resourceCreate(form)
    if (!success) throw new Error('保存菜单失败')
    const selected = form.id
    toast.success(form.id ? '菜单信息已更新' : '菜单已创建')
    editorOpen.value = false
    await loadMenus()
    if (selected) {
      selectedId.value = selected
      expandParents(menus.value, selected)
    }
  } catch (error) {
    editorErrorMessage.value = errorText(error, '保存菜单失败')
    toast.error(editorErrorMessage.value)
  } finally {
    editorLoading.value = false
  }
}

async function confirmDelete() {
  if (!deleteTarget.value?.id) return
  loading.value = true
  try {
    if (!(await systemApi.resourceDelete([deleteTarget.value.id]))) throw new Error('删除菜单失败')
    toast.success('菜单已删除')
    deleteTarget.value = null
    selectedId.value = 'root'
    await loadMenus()
  } catch (error) {
    toast.error(errorText(error, '删除菜单失败'))
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
    if (!(await systemApi.resourceUpdate({ id: targetId, status }))) throw new Error('状态更新失败')
    target.status = status
    statusTarget.value = null
    toast.success(`菜单已${status === 1 ? '启用' : '禁用'}`)
  } catch (error) {
    toast.error(errorText(error, '状态更新失败'))
  } finally {
    loading.value = false
  }
}

function typeLabel(type: number) {
  return type === 2 ? '内嵌页' : type === 3 ? '外链' : '菜单'
}

function menuTypeFallback(type: number): Component {
  if (type === 2) return MonitorUp
  if (type === 3) return ExternalLink
  return FolderTree
}

function formatTime(value?: number) {
  return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
}

onMounted(loadMenus)
</script>

<template>
  <div class="space-y-5">
    <AppPageHeader
      title="菜单管理"
      description="维护系统菜单层级、路由组件、展示状态和跳转配置。"
      :loading="loading"
      @refresh="loadMenus"
    />

    <div
      v-if="errorMessage"
      class="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ errorMessage }}
    </div>

    <div class="grid min-h-[620px] gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
      <Card class="gap-0 overflow-hidden py-0">
        <div class="flex items-center justify-between border-b px-4 py-3">
          <div>
            <h2 class="font-semibold">菜单列表</h2>
            <p class="text-xs text-muted-foreground">共 {{ visibleMenus.length }} 个可见节点</p>
          </div>
          <div class="flex gap-1">
            <Button variant="ghost" size="icon" title="新增一级菜单" @click="openCreate('1')">
              <Plus class="size-4" />
            </Button>
            <Button variant="ghost" size="icon" title="刷新" :disabled="loading" @click="loadMenus">
              <RefreshCw class="size-4" :class="{ 'animate-spin': loading }" />
            </Button>
          </div>
        </div>
        <CardContent class="p-0">
          <div class="border-b p-3">
            <div class="relative">
              <Search
                class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
              />
              <Input v-model="keyword" class="h-9 pl-9" placeholder="搜索菜单名称、别名或路由" />
            </div>
          </div>
          <div class="max-h-[560px] min-h-[480px] overflow-y-auto p-2">
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm hover:bg-muted"
              :class="selectedId === 'root' ? 'bg-muted font-medium' : ''"
              @click="selectedId = 'root'"
            >
              <span class="flex size-6 items-center justify-center"
                ><FolderTree class="size-4"
              /></span>
              <span class="flex-1">根目录</span>
              <Button
                variant="ghost"
                size="icon"
                class="size-7"
                title="新增一级菜单"
                @click.stop="openCreate('1')"
              >
                <Plus class="size-3.5" />
              </Button>
            </button>

            <button
              v-for="item in visibleMenus"
              :key="item.menu.id"
              type="button"
              class="group flex w-full items-center gap-1 rounded-md py-1.5 pr-1 text-left text-sm hover:bg-muted"
              :class="selectedId === item.menu.id ? 'bg-muted font-medium' : ''"
              :style="{ paddingLeft: `${item.depth * 20 + 8}px` }"
              @click="selectMenu(item.menu)"
            >
              <span class="flex size-5 shrink-0 items-center justify-center">
                <button
                  v-if="item.menu.children?.length"
                  type="button"
                  class="rounded p-0.5 hover:bg-background"
                  @click.stop="toggleExpanded(item.menu.id)"
                >
                  <ChevronDown
                    v-if="item.menu.id && expandedIds.has(item.menu.id)"
                    class="size-3.5"
                  />
                  <ChevronRight v-else class="size-3.5" />
                </button>
              </span>
              <MenuIcon
                :icon="item.menu.icon"
                :fallback="menuTypeFallback(item.menu.type)"
                class="size-3.5 shrink-0 text-muted-foreground"
                :style="{ color: item.menu.color || undefined }"
              />
              <span class="min-w-0 flex-1 truncate">{{ item.menu.title }}</span>
              <Badge
                v-if="item.menu.status !== 1"
                variant="secondary"
                class="px-1.5 py-0 text-[10px]"
                >禁用</Badge
              >
              <Button
                v-if="item.menu.type === 1"
                variant="ghost"
                size="icon"
                class="size-7 opacity-0 group-hover:opacity-100"
                title="新增子菜单"
                @click.stop="item.menu.id && openCreate(item.menu.id)"
              >
                <Plus class="size-3.5" />
              </Button>
            </button>

            <div
              v-if="!loading && visibleMenus.length === 0"
              class="px-4 py-12 text-center text-sm text-muted-foreground"
            >
              暂无匹配菜单
            </div>
          </div>
        </CardContent>
      </Card>

      <Card class="gap-0 overflow-hidden py-0">
        <template v-if="selectedMenu">
          <div
            class="flex flex-col gap-3 border-b px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <h2 class="truncate text-lg font-semibold">{{ selectedMenu.title }}</h2>
                <Badge variant="outline">{{ typeLabel(selectedMenu.type) }}</Badge>
              </div>
              <p class="mt-0.5 truncate text-sm text-muted-foreground">
                {{ selectedMenu.path || '未配置路由地址' }}
              </p>
            </div>
            <div class="flex gap-2">
              <Button
                v-if="selectedMenu.type === 1"
                size="sm"
                variant="outline"
                @click="selectedMenu.id && openCreate(selectedMenu.id)"
              >
                <Plus class="mr-1.5 size-4" />新增子菜单
              </Button>
              <Button size="sm" variant="outline" @click="openEdit(selectedMenu)">
                <Pencil class="mr-1.5 size-4" />编辑
              </Button>
              <Button size="sm" variant="destructive" @click="deleteTarget = selectedMenu">
                <Trash2 class="mr-1.5 size-4" />删除
              </Button>
            </div>
          </div>
          <CardContent class="space-y-5 p-5">
            <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <div class="rounded-lg border p-3">
                <div class="text-xs text-muted-foreground">路由别名</div>
                <div class="mt-1 break-all text-sm font-medium">
                  {{ selectedMenu.alias || '-' }}
                </div>
              </div>
              <div class="rounded-lg border p-3">
                <div class="text-xs text-muted-foreground">组件路径</div>
                <div class="mt-1 break-all text-sm font-medium">
                  {{ selectedMenu.component || '-' }}
                </div>
              </div>
              <div class="rounded-lg border p-3">
                <div class="text-xs text-muted-foreground">图标</div>
                <div class="mt-1 flex items-center gap-2 text-sm font-medium">
                  <MenuIcon
                    :icon="selectedMenu.icon"
                    :fallback="menuTypeFallback(selectedMenu.type)"
                    class="size-4 shrink-0"
                    :style="{ color: selectedMenu.color || undefined }"
                  />
                  <span class="break-all">{{ selectedMenu.icon || '未配置' }}</span>
                </div>
              </div>
              <div class="rounded-lg border p-3">
                <div class="text-xs text-muted-foreground">排序</div>
                <div class="mt-1 text-sm font-medium">{{ selectedMenu.sort || 0 }}</div>
              </div>
              <div class="rounded-lg border p-3">
                <div class="text-xs text-muted-foreground">状态</div>
                <Button
                  variant="ghost"
                  size="sm"
                  class="mt-1 h-auto rounded-full p-0"
                  @click="statusTarget = selectedMenu"
                >
                  <Badge :variant="selectedMenu.status === 1 ? 'default' : 'secondary'">{{
                    selectedMenu.status === 1 ? '启用' : '禁用'
                  }}</Badge>
                </Button>
              </div>
              <div class="rounded-lg border p-3">
                <div class="text-xs text-muted-foreground">页面设置</div>
                <div class="mt-1 text-sm font-medium">
                  {{ selectedMenu.keepAlive ? '缓存' : '不缓存' }} ·
                  {{ selectedMenu.hidden ? '隐藏菜单' : '显示菜单' }}
                </div>
              </div>
            </div>
            <div class="grid gap-4 lg:grid-cols-2">
              <div class="space-y-3 rounded-lg border p-4">
                <h3 class="font-medium">路由配置</h3>
                <dl class="grid gap-3 text-sm">
                  <div>
                    <dt class="text-xs text-muted-foreground">路由地址</dt>
                    <dd class="mt-1 break-all">{{ selectedMenu.path || '-' }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-muted-foreground">重定向／跳转地址</dt>
                    <dd class="mt-1 break-all">{{ selectedMenu.redirect || '-' }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-muted-foreground">上级路由名称</dt>
                    <dd class="mt-1 break-all">{{ selectedMenu.parentRoute || '-' }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-muted-foreground">路由参数</dt>
                    <dd class="mt-1 break-all font-mono text-xs">
                      {{ selectedMenu.query || '{}' }}
                    </dd>
                  </div>
                </dl>
              </div>
              <div class="space-y-3 rounded-lg border p-4">
                <h3 class="font-medium">基础信息</h3>
                <dl class="grid gap-3 text-sm">
                  <div>
                    <dt class="text-xs text-muted-foreground">菜单 ID</dt>
                    <dd class="mt-1 break-all font-mono text-xs">{{ selectedMenu.id }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-muted-foreground">上级 ID</dt>
                    <dd class="mt-1 break-all font-mono text-xs">{{ selectedMenu.pid }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-muted-foreground">权限编码</dt>
                    <dd class="mt-1 break-all">{{ selectedMenu.code || '-' }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-muted-foreground">更新时间</dt>
                    <dd class="mt-1">
                      {{ formatTime(selectedMenu.updatedAt || selectedMenu.createdAt) }}
                    </dd>
                  </div>
                </dl>
              </div>
            </div>
          </CardContent>
        </template>
        <div
          v-else
          class="flex min-h-[620px] flex-col items-center justify-center px-6 text-center"
        >
          <div class="flex size-14 items-center justify-center rounded-full bg-muted">
            <FolderTree class="size-7 text-muted-foreground" />
          </div>
          <h2 class="mt-4 text-lg font-semibold">请选择一个菜单</h2>
          <p class="mt-1 max-w-md text-sm text-muted-foreground">
            从左侧菜单树选择节点查看详情，或直接新增一级菜单。
          </p>
          <Button class="mt-4" size="sm" @click="openCreate('1')"
            ><Plus class="mr-1.5 size-4" />新增一级菜单</Button
          >
        </div>
      </Card>
    </div>

    <MenuEditorDialog
      :open="editorOpen"
      :loading="editorLoading"
      :error-message="editorErrorMessage"
      :menu="editingMenu"
      :menus="menus"
      :parent-id="createParentId"
      @close="editorOpen = false"
      @submit="saveMenu"
    />

    <AlertDialog
      :open="Boolean(deleteTarget)"
      @update:open="(open) => !open && !loading && (deleteTarget = null)"
    >
      <AlertDialogContent
        ><AlertDialogHeader
          ><AlertDialogTitle>确认删除菜单 {{ deleteTarget?.title }}？</AlertDialogTitle
          ><AlertDialogDescription
            >存在子菜单、角色关联或接口权限关联时，后台将拒绝删除。</AlertDialogDescription
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
            >确认{{ statusTarget?.status === 1 ? '禁用' : '启用' }}菜单？</AlertDialogTitle
          ><AlertDialogDescription
            >菜单“{{
              statusTarget?.title
            }}”状态将立即更新，并影响关联角色的菜单访问。</AlertDialogDescription
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
