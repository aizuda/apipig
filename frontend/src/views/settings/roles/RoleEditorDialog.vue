<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { Button, Dialog, DialogFixedContent, Input, Label, Textarea } from '@tabtab/ui'
import type { ResourceTreeNode, RoleDetail, RoleSaveParams } from '@/api/system'

const props = defineProps<{
  open: boolean
  loading: boolean
  errorMessage: string
  role: RoleDetail | null
  resources: ResourceTreeNode[]
}>()

const emit = defineEmits<{
  close: []
  submit: [form: RoleSaveParams]
}>()

const form = reactive<RoleSaveParams>({
  name: '',
  alias: '',
  remark: '',
  status: 1,
  sort: 0,
  resourceIds: [],
})
const isEditing = computed(() => Boolean(props.role?.id))
const flatResources = computed(() => flattenResources(props.resources))

watch(
  () => [props.open, props.role] as const,
  ([open, role]) => {
    if (!open) return
    Object.assign(form, {
      id: role?.id,
      name: role?.name || '',
      alias: role?.alias || '',
      remark: role?.remark || '',
      status: role?.status || 1,
      sort: role?.sort || 0,
      resourceIds: [...(role?.resourceIds || [])],
    })
  },
  { immediate: true },
)

function descendantIds(node: ResourceTreeNode): string[] {
  return [node.id, ...(node.children || []).flatMap(descendantIds)].filter(Boolean) as string[]
}

function flattenResources(
  nodes: ResourceTreeNode[],
  depth = 0,
): Array<{ node: ResourceTreeNode; depth: number }> {
  return nodes.flatMap((node) => [
    { node, depth },
    ...flattenResources(node.children || [], depth + 1),
  ])
}

function ancestorIds(
  targetId: string,
  nodes: ResourceTreeNode[],
  parents: string[] = [],
): string[] {
  for (const node of nodes) {
    if (node.id === targetId) return parents
    const found = ancestorIds(
      targetId,
      node.children || [],
      node.id ? [...parents, node.id] : parents,
    )
    if (found.length > 0) return found
  }
  return []
}

function toggleResource(node: ResourceTreeNode, checked: boolean) {
  const ids = descendantIds(node)
  form.resourceIds = checked
    ? Array.from(
        new Set([
          ...form.resourceIds,
          ...ids,
          ...(node.id ? ancestorIds(node.id, props.resources) : []),
        ]),
      )
    : form.resourceIds.filter((id) => !ids.includes(id))
}

function submit() {
  emit('submit', {
    ...form,
    name: form.name.trim(),
    alias: form.alias.trim(),
    remark: form.remark?.trim(),
    resourceIds: [...form.resourceIds],
  })
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && !loading && emit('close')">
    <DialogFixedContent
      :title="isEditing ? '编辑角色' : '新增角色'"
      description="配置角色基本信息与可访问菜单权限。"
      class="sm:max-w-2xl"
    >
      <div class="space-y-4">
        <div
          v-if="errorMessage"
          class="rounded-md border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {{ errorMessage }}
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="role-name">角色名称 <span class="text-destructive">*</span></Label
            ><Input id="role-name" v-model="form.name" placeholder="例如 系统管理员" />
          </div>
          <div class="space-y-1.5">
            <Label for="role-alias">角色别名 <span class="text-destructive">*</span></Label
            ><Input id="role-alias" v-model="form.alias" placeholder="例如 admin" />
          </div>
          <div class="space-y-1.5">
            <Label for="role-status">状态</Label>
            <select
              id="role-status"
              v-model.number="form.status"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            >
              <option :value="1">启用</option>
              <option :value="2">禁用</option>
            </select>
          </div>
          <div class="space-y-1.5">
            <Label for="role-sort">排序</Label
            ><Input id="role-sort" v-model.number="form.sort" type="number" min="0" />
          </div>
        </div>
        <div class="space-y-1.5">
          <Label for="role-remark">备注</Label
          ><Textarea id="role-remark" v-model="form.remark" placeholder="填写角色职责或使用范围" />
        </div>
        <div class="space-y-2">
          <Label>菜单权限</Label>
          <div class="max-h-72 overflow-y-auto rounded-lg border p-3">
            <label
              v-for="item in flatResources"
              :key="item.node.id"
              class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 hover:bg-muted"
              :style="{ paddingLeft: `${item.depth * 24 + 8}px` }"
            >
              <input
                type="checkbox"
                class="size-4 accent-primary"
                :checked="Boolean(item.node.id && form.resourceIds.includes(item.node.id))"
                @change="toggleResource(item.node, ($event.target as HTMLInputElement).checked)"
              />
              <span :class="item.depth === 0 ? 'font-medium' : 'text-sm'">{{
                item.node.title
              }}</span>
              <span v-if="item.node.path" class="text-xs text-muted-foreground">{{
                item.node.path
              }}</span>
            </label>
            <p v-if="resources.length === 0" class="text-sm text-muted-foreground">暂无菜单资源</p>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="loading" @click="emit('close')">取消</Button>
        <Button :disabled="loading" @click="submit">{{
          isEditing ? '保存修改' : '创建角色'
        }}</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
