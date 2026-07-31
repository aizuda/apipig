<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { Button, Dialog, DialogFixedContent, Input, Label } from '@tabtab/ui'
import type { SystemRole, UserDetail, UserSaveParams } from '@/api/system'

const props = defineProps<{
  open: boolean
  loading: boolean
  errorMessage: string
  user: UserDetail | null
  roles: SystemRole[]
}>()

const emit = defineEmits<{
  close: []
  submit: [form: UserSaveParams]
}>()

const form = reactive<UserSaveParams>({
  username: '',
  password: '',
  realName: '',
  nickName: '',
  sex: 1,
  phone: '',
  email: '',
  jobNum: '',
  roleIds: [],
})

const isEditing = computed(() => Boolean(props.user?.id))

watch(
  () => [props.open, props.user] as const,
  ([open, user]) => {
    if (!open) return
    Object.assign(form, {
      id: user?.id,
      username: user?.username || '',
      password: '',
      realName: user?.realName || '',
      nickName: user?.nickName || '',
      sex: user?.sex || 1,
      phone: user?.phone || '',
      email: user?.email || '',
      jobNum: user?.jobNum || '',
      roleIds: [...(user?.roleIds || [])],
    })
  },
  { immediate: true },
)

function toggleRole(roleId: string, checked: boolean) {
  form.roleIds = checked
    ? Array.from(new Set([...form.roleIds, roleId]))
    : form.roleIds.filter((id) => id !== roleId)
}

function submit() {
  emit('submit', {
    ...form,
    username: form.username.trim(),
    password: form.password?.trim() || undefined,
    realName: form.realName?.trim(),
    nickName: form.nickName?.trim(),
    phone: form.phone?.trim(),
    email: form.email?.trim(),
    jobNum: form.jobNum?.trim(),
    roleIds: [...form.roleIds],
  })
}
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && !loading && emit('close')">
    <DialogFixedContent
      :title="isEditing ? '编辑用户' : '新增用户'"
      description="维护用户基础资料并分配系统角色。"
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
            <Label for="user-username">登录账号 <span class="text-destructive">*</span></Label>
            <Input id="user-username" v-model="form.username" :disabled="isEditing" />
          </div>
          <div class="space-y-1.5">
            <Label for="user-password"
              >登录密码 <span v-if="!isEditing" class="text-destructive">*</span></Label
            >
            <Input
              id="user-password"
              v-model="form.password"
              type="password"
              :placeholder="isEditing ? '留空表示不修改' : '请输入初始密码'"
            />
          </div>
          <div class="space-y-1.5">
            <Label for="user-real-name">真实姓名</Label
            ><Input id="user-real-name" v-model="form.realName" />
          </div>
          <div class="space-y-1.5">
            <Label for="user-nick-name">昵称</Label
            ><Input id="user-nick-name" v-model="form.nickName" />
          </div>
          <div class="space-y-1.5">
            <Label for="user-job-num">工号</Label><Input id="user-job-num" v-model="form.jobNum" />
          </div>
          <div class="space-y-1.5">
            <Label for="user-sex">性别</Label>
            <select
              id="user-sex"
              v-model.number="form.sex"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            >
              <option :value="1">男</option>
              <option :value="2">女</option>
            </select>
          </div>
          <div class="space-y-1.5">
            <Label for="user-phone">手机号</Label><Input id="user-phone" v-model="form.phone" />
          </div>
          <div class="space-y-1.5">
            <Label for="user-email">邮箱</Label
            ><Input id="user-email" v-model="form.email" type="email" />
          </div>
        </div>
        <div class="space-y-2">
          <Label>分配角色</Label>
          <div class="grid gap-2 rounded-lg border p-3 sm:grid-cols-2">
            <label
              v-for="role in roles"
              :key="role.id"
              class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 hover:bg-muted"
            >
              <input
                type="checkbox"
                class="size-4 accent-primary"
                :checked="Boolean(role.id && form.roleIds.includes(role.id))"
                @change="
                  role.id && toggleRole(role.id, ($event.target as HTMLInputElement).checked)
                "
              />
              <span class="text-sm">{{ role.name }}</span
              ><span class="text-xs text-muted-foreground">{{ role.alias }}</span>
            </label>
            <p v-if="roles.length === 0" class="text-sm text-muted-foreground">暂无可用角色</p>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="loading" @click="emit('close')">取消</Button>
        <Button :disabled="loading" @click="submit">{{
          isEditing ? '保存修改' : '创建用户'
        }}</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
