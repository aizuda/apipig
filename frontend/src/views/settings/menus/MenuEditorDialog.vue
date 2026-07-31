<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { Button, Dialog, DialogFixedContent, Input, Label, Textarea } from '@tabtab/ui'
import type { ResourceSaveParams, ResourceTreeNode } from '@/api/system'
import MenuIcon from '@/components/menu/MenuIcon.vue'
import {
  menuIconCategories,
  menuIconOptions,
  type MenuIconCategory,
} from '@/components/menu/menu-icons'

const props = defineProps<{
  open: boolean
  loading: boolean
  errorMessage: string
  menu: ResourceTreeNode | null
  menus: ResourceTreeNode[]
  parentId: string
}>()

const emit = defineEmits<{
  close: []
  submit: [form: ResourceSaveParams]
}>()

const form = reactive<ResourceSaveParams>({
  pid: '1',
  title: '',
  alias: '',
  type: 1,
  code: '',
  redirect: '',
  path: '',
  icon: '',
  status: 1,
  sort: 0,
  component: '',
  color: '',
  hidden: false,
  parentRoute: '',
  keepAlive: false,
  query: '{}',
})
const iconKeyword = ref('')
const iconCategory = ref<'all' | MenuIconCategory>('all')

const isEditing = computed(() => Boolean(props.menu?.id))
const excludedParentIds = computed(() =>
  props.menu ? new Set(descendantIds(props.menu)) : new Set<string>(),
)
const parentOptions = computed(() => [
  { id: '1', title: '根目录', depth: 0 },
  ...flattenMenus(props.menus),
])
const filteredIconOptions = computed(() => {
  const keyword = iconKeyword.value.trim().toLowerCase()
  return menuIconOptions.filter(
    (option) =>
      (iconCategory.value === 'all' || option.category === iconCategory.value) &&
      (!keyword ||
        option.label.toLowerCase().includes(keyword) ||
        option.value.toLowerCase().includes(keyword) ||
        option.keywords?.toLowerCase().includes(keyword)),
  )
})

watch(
  () => [props.open, props.menu, props.parentId] as const,
  ([open, menu, parentId]) => {
    if (!open) return
    iconKeyword.value = ''
    iconCategory.value = 'all'
    Object.assign(form, {
      id: menu?.id,
      pid: menu?.pid || parentId || '1',
      title: menu?.title || '',
      alias: menu?.alias || '',
      type: menu?.type || 1,
      code: menu?.code || '',
      redirect: menu?.redirect || '',
      path: menu?.path || '',
      icon: menu?.icon || '',
      status: menu?.status || 1,
      sort: menu?.sort || 0,
      component: menu?.component || '',
      color: menu?.color || '',
      hidden: menu?.hidden || false,
      parentRoute: menu?.parentRoute || '',
      keepAlive: menu?.keepAlive || false,
      query: menu?.query || '{}',
    })
  },
  { immediate: true },
)

function flattenMenus(
  nodes: ResourceTreeNode[],
  depth = 0,
): Array<{ id: string; title: string; depth: number }> {
  return nodes.flatMap((node) => [
    ...(node.id ? [{ id: node.id, title: node.title, depth }] : []),
    ...flattenMenus(node.children || [], depth + 1),
  ])
}

function descendantIds(node: ResourceTreeNode): string[] {
  return [node.id, ...(node.children || []).flatMap(descendantIds)].filter(Boolean) as string[]
}

function submit() {
  const type = Number(form.type)
  emit('submit', {
    ...form,
    type,
    title: form.title.trim(),
    alias: form.alias?.trim(),
    code: form.code?.trim(),
    redirect: form.redirect?.trim(),
    path: form.path.trim(),
    icon: form.icon?.trim(),
    component: form.component?.trim() || (type === 2 || type === 3 ? 'iframe-page' : ''),
    color: form.color?.trim(),
    parentRoute: form.parentRoute?.trim(),
    query: form.query?.trim() || '{}',
  })
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && !loading && emit('close')">
    <DialogFixedContent
      :title="isEditing ? '编辑菜单' : '新增菜单'"
      description="配置菜单路由、组件、展示状态和层级关系。"
      class="sm:max-w-3xl"
    >
      <div class="space-y-4">
        <div
          v-if="errorMessage"
          class="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
        >
          {{ errorMessage }}
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="menu-parent">上级菜单</Label>
            <select
              id="menu-parent"
              v-model="form.pid"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            >
              <option
                v-for="option in parentOptions"
                :key="option.id"
                :value="option.id"
                :disabled="excludedParentIds.has(option.id)"
              >
                {{ `${'　'.repeat(option.depth)}${option.title}` }}
              </option>
            </select>
          </div>
          <div class="space-y-1.5">
            <Label for="menu-type">菜单类型</Label>
            <select
              id="menu-type"
              v-model.number="form.type"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            >
              <option :value="1">普通菜单</option>
              <option :value="2">内嵌页面</option>
              <option :value="3">外部链接</option>
            </select>
          </div>
          <div class="space-y-1.5">
            <Label for="menu-title">菜单名称 <span class="text-destructive">*</span></Label>
            <Input id="menu-title" v-model="form.title" placeholder="例如 用户管理" />
          </div>
          <div class="space-y-1.5">
            <Label for="menu-alias">路由别名</Label>
            <Input id="menu-alias" v-model="form.alias" placeholder="例如 setting_user" />
          </div>
          <div class="space-y-1.5">
            <Label for="menu-path">路由地址 <span class="text-destructive">*</span></Label>
            <Input id="menu-path" v-model="form.path" placeholder="例如 /settings/users" />
          </div>
          <div class="space-y-1.5">
            <Label for="menu-component">组件路径</Label>
            <Input
              id="menu-component"
              v-model="form.component"
              :placeholder="form.type === 1 ? '例如 settings/users/index' : '默认 iframe-page'"
            />
          </div>
          <div class="space-y-1.5">
            <Label for="menu-redirect">重定向／跳转地址</Label>
            <Input
              id="menu-redirect"
              v-model="form.redirect"
              placeholder="内嵌页或外链填写完整 URL"
            />
          </div>
          <div class="space-y-1.5">
            <Label for="menu-parent-route">上级路由名称</Label>
            <Input id="menu-parent-route" v-model="form.parentRoute" placeholder="可选" />
          </div>
          <div class="space-y-1.5">
            <Label for="menu-code">权限编码</Label>
            <Input id="menu-code" v-model="form.code" placeholder="可选" />
          </div>
          <div class="space-y-1.5">
            <Label for="menu-status">状态</Label>
            <select
              id="menu-status"
              v-model.number="form.status"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            >
              <option :value="1">启用</option>
              <option :value="2">禁用</option>
            </select>
          </div>
          <div class="space-y-1.5">
            <Label for="menu-sort">排序</Label>
            <Input id="menu-sort" v-model.number="form.sort" type="number" min="0" />
          </div>
          <div class="space-y-2 sm:col-span-2">
            <Label for="menu-icon">菜单图标</Label>
            <div class="flex items-center gap-2">
              <span
                class="flex size-10 shrink-0 items-center justify-center rounded-md border bg-muted/30"
                :style="{ color: form.color || undefined }"
              >
                <MenuIcon :icon="form.icon" class="size-5" />
              </span>
              <Input
                id="menu-icon"
                v-model="form.icon"
                placeholder="选择下方图标，或输入 lucide:图标名"
              />
            </div>
            <div class="rounded-md border bg-muted/10 p-2">
              <div class="mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <span class="text-xs text-muted-foreground">
                  共 {{ menuIconOptions.length }} 个图标，当前显示
                  {{ filteredIconOptions.length }} 个
                </span>
                <Input
                  v-model="iconKeyword"
                  class="h-8 w-full sm:w-52"
                  placeholder="搜索名称或 lucide 值"
                  aria-label="搜索菜单图标"
                />
              </div>
              <div class="mb-2 flex gap-1 overflow-x-auto pb-1">
                <Button
                  v-for="category in menuIconCategories"
                  :key="category.value"
                  type="button"
                  size="sm"
                  :variant="iconCategory === category.value ? 'secondary' : 'ghost'"
                  class="h-7 shrink-0 px-2.5 text-xs"
                  @click="iconCategory = category.value"
                >
                  {{ category.label }}
                </Button>
              </div>
              <div
                class="grid max-h-56 grid-cols-4 gap-1 overflow-y-auto sm:grid-cols-6 lg:grid-cols-8"
              >
                <button
                  v-for="option in filteredIconOptions"
                  :key="option.value"
                  type="button"
                  class="flex min-h-16 flex-col items-center justify-center gap-1 rounded-md border px-1 py-2 text-xs transition-colors hover:bg-muted"
                  :class="
                    form.icon === option.value ? 'border-primary bg-primary/10 text-primary' : ''
                  "
                  :title="option.value"
                  @click="form.icon = option.value"
                >
                  <MenuIcon :icon="option.value" class="size-5" />
                  <span class="max-w-full truncate">{{ option.label }}</span>
                </button>
                <div
                  v-if="filteredIconOptions.length === 0"
                  class="col-span-full py-6 text-center text-xs text-muted-foreground"
                >
                  未找到匹配图标
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="flex cursor-pointer items-center gap-2 rounded-md border p-3 text-sm">
            <input v-model="form.keepAlive" type="checkbox" class="size-4 accent-primary" />缓存页面
          </label>
          <label class="flex cursor-pointer items-center gap-2 rounded-md border p-3 text-sm">
            <input v-model="form.hidden" type="checkbox" class="size-4 accent-primary" />隐藏菜单
          </label>
        </div>
        <div class="space-y-1.5">
          <Label for="menu-query">路由参数</Label>
          <Textarea id="menu-query" v-model="form.query" placeholder='例如 {"tab":"profile"}' />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="loading" @click="emit('close')">取消</Button>
        <Button :disabled="loading" @click="submit">{{
          isEditing ? '保存修改' : '创建菜单'
        }}</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
