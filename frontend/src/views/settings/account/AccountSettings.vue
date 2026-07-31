<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  BadgeCheck,
  BriefcaseBusiness,
  CalendarDays,
  KeyRound,
  Mail,
  Phone,
  ShieldCheck,
  UserRound,
} from '@lucide/vue'
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
  Badge,
  Button,
  Card,
  CardContent,
  Input,
  Label,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  toast,
} from '@tabtab/ui'
import { systemApi, type SystemUser } from '@/api/system'
import AppPageHeader from '@/components/AppPageHeader.vue'
import { DEFAULT_AVATAR_URL, useUserStore } from '@/stores/user'

defineOptions({ name: 'SettingsAccount' })

const userStore = useUserStore()
const loading = ref(false)
const activeTab = ref('profile')
const userInfo = ref<SystemUser | null>(null)
const errorMessage = ref('')
const profile = reactive({
  realName: '',
  nickName: '',
  avatar: '',
  sex: 1,
  phone: '',
  email: '',
  jobNum: '',
})
const passwordForm = reactive({
  newPassword: '',
  confirmPassword: '',
})

const displayName = computed(
  () => userInfo.value?.nickName || userInfo.value?.realName || userInfo.value?.username || '用户',
)
const displayAvatar = computed(() => profile.avatar.trim() || DEFAULT_AVATAR_URL)
const avatarFallback = computed(() => Array.from(displayName.value)[0]?.toLocaleUpperCase() || 'U')

function errorText(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}

function assignProfile(user: SystemUser) {
  Object.assign(profile, {
    realName: user.realName || '',
    nickName: user.nickName || '',
    avatar: user.avatar || '',
    sex: user.sex || 1,
    phone: user.phone || '',
    email: user.email || '',
    jobNum: user.jobNum || '',
  })
}

async function loadProfile() {
  loading.value = true
  errorMessage.value = ''
  try {
    const user = await systemApi.userInfo()
    userInfo.value = user
    assignProfile(user)
    userStore.setUserProfile(user)
  } catch (error) {
    errorMessage.value = errorText(error, '个人资料加载失败')
    toast.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

async function saveProfile() {
  if (!userInfo.value?.id) return
  if (profile.phone && !/^1\d{10}$/.test(profile.phone)) {
    errorMessage.value = '请输入正确的 11 位手机号'
    return
  }
  if (profile.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(profile.email)) {
    errorMessage.value = '请输入正确的邮箱地址'
    return
  }
  loading.value = true
  errorMessage.value = ''
  try {
    const success = await systemApi.userUpdate({
      id: userInfo.value.id,
      realName: profile.realName.trim(),
      nickName: profile.nickName.trim(),
      avatar: profile.avatar.trim(),
      sex: profile.sex,
      phone: profile.phone.trim(),
      email: profile.email.trim(),
      jobNum: profile.jobNum.trim(),
    })
    if (!success) throw new Error('个人资料保存失败')
    toast.success('个人资料已更新')
    await loadProfile()
  } catch (error) {
    errorMessage.value = errorText(error, '个人资料保存失败')
    toast.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

async function changePassword() {
  if (!userInfo.value?.id) return
  if (passwordForm.newPassword.length < 6) {
    errorMessage.value = '新密码至少需要 6 个字符'
    return
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    errorMessage.value = '两次输入的密码不一致'
    return
  }
  loading.value = true
  errorMessage.value = ''
  try {
    const success = await systemApi.userResetPassword([userInfo.value.id], passwordForm.newPassword)
    if (!success) throw new Error('密码修改失败')
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
    toast.success('登录密码已修改')
    await loadProfile()
  } catch (error) {
    errorMessage.value = errorText(error, '密码修改失败')
    toast.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function formatTime(value?: number) {
  return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
}

onMounted(loadProfile)
</script>

<template>
  <div class="space-y-5">
    <AppPageHeader
      title="账户设置"
      description="维护个人资料、头像与登录密码。"
      :loading="loading"
      @refresh="loadProfile"
    />

    <div
      v-if="errorMessage"
      class="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ errorMessage }}
    </div>

    <div class="grid min-h-[560px] items-stretch gap-4 lg:grid-cols-[340px_minmax(0,1fr)]">
      <Card class="gap-0 py-0">
        <CardContent class="p-5">
          <h2 class="font-semibold">个人信息</h2>
          <div class="my-5 flex flex-col items-center border-y py-6 text-center">
            <Avatar class="size-20 border shadow-sm">
              <AvatarImage :src="displayAvatar" :alt="displayName" />
              <AvatarFallback class="text-2xl font-semibold">{{ avatarFallback }}</AvatarFallback>
            </Avatar>
            <strong class="mt-3 text-base">{{ displayName }}</strong>
            <span class="text-sm text-muted-foreground">{{ userInfo?.username || '-' }}</span>
            <Badge class="mt-2" :variant="userInfo?.status === 2 ? 'secondary' : 'default'">
              {{ userInfo?.status === 2 ? '已禁用' : '正常' }}
            </Badge>
          </div>
          <div class="divide-y text-sm">
            <div class="flex items-center justify-between gap-3 py-3">
              <span class="flex items-center gap-2 text-muted-foreground"
                ><UserRound class="size-4" />真实姓名</span
              >
              <span class="truncate font-medium">{{ userInfo?.realName || '-' }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 py-3">
              <span class="flex items-center gap-2 text-muted-foreground"
                ><BadgeCheck class="size-4" />用户性别</span
              >
              <span class="font-medium">{{ userInfo?.sex === 2 ? '女' : '男' }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 py-3">
              <span class="flex items-center gap-2 text-muted-foreground"
                ><Phone class="size-4" />手机号码</span
              >
              <span class="truncate font-medium">{{ userInfo?.phone || '-' }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 py-3">
              <span class="flex items-center gap-2 text-muted-foreground"
                ><Mail class="size-4" />用户邮箱</span
              >
              <span class="max-w-44 truncate font-medium">{{ userInfo?.email || '-' }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 py-3">
              <span class="flex items-center gap-2 text-muted-foreground"
                ><BriefcaseBusiness class="size-4" />工号</span
              >
              <span class="truncate font-medium">{{ userInfo?.jobNum || '-' }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 py-3">
              <span class="flex items-center gap-2 text-muted-foreground"
                ><CalendarDays class="size-4" />创建时间</span
              >
              <span class="text-right text-xs font-medium">{{
                formatTime(userInfo?.createdAt)
              }}</span>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card class="gap-0 py-0">
        <CardContent class="p-5">
          <Tabs v-model="activeTab" class="w-full">
            <TabsList class="grid w-full max-w-md grid-cols-2">
              <TabsTrigger value="profile"><UserRound class="mr-1.5 size-4" />基本资料</TabsTrigger>
              <TabsTrigger value="password"><KeyRound class="mr-1.5 size-4" />修改密码</TabsTrigger>
            </TabsList>

            <TabsContent value="profile" class="mt-6">
              <div class="max-w-2xl space-y-5">
                <div>
                  <h2 class="text-lg font-semibold">基本资料</h2>
                  <p class="text-sm text-muted-foreground">
                    更新后将同步显示在侧边栏和用户菜单中。
                  </p>
                </div>
                <div class="grid gap-4 sm:grid-cols-2">
                  <div class="space-y-1.5">
                    <Label for="account-username">登录账号</Label
                    ><Input
                      id="account-username"
                      :model-value="userInfo?.username || ''"
                      disabled
                    />
                  </div>
                  <div class="space-y-1.5">
                    <Label for="account-job-num">工号</Label
                    ><Input
                      id="account-job-num"
                      v-model="profile.jobNum"
                      placeholder="请输入工号"
                    />
                  </div>
                  <div class="space-y-1.5">
                    <Label for="account-nick-name">昵称</Label
                    ><Input
                      id="account-nick-name"
                      v-model="profile.nickName"
                      placeholder="请输入昵称"
                    />
                  </div>
                  <div class="space-y-1.5">
                    <Label for="account-real-name">真实姓名</Label
                    ><Input
                      id="account-real-name"
                      v-model="profile.realName"
                      placeholder="请输入真实姓名"
                    />
                  </div>
                  <div class="space-y-1.5">
                    <Label for="account-sex">性别</Label>
                    <select
                      id="account-sex"
                      v-model.number="profile.sex"
                      class="h-10 w-full rounded-md border bg-background px-3 text-sm"
                    >
                      <option :value="1">男</option>
                      <option :value="2">女</option>
                    </select>
                  </div>
                  <div class="space-y-1.5">
                    <Label for="account-phone">手机号</Label
                    ><Input id="account-phone" v-model="profile.phone" placeholder="请输入手机号" />
                  </div>
                  <div class="space-y-1.5 sm:col-span-2">
                    <Label for="account-email">邮箱</Label
                    ><Input
                      id="account-email"
                      v-model="profile.email"
                      type="email"
                      placeholder="请输入邮箱"
                    />
                  </div>
                  <div class="space-y-1.5 sm:col-span-2">
                    <Label for="account-avatar">头像地址</Label
                    ><Input
                      id="account-avatar"
                      v-model="profile.avatar"
                      placeholder="请输入可访问的图片 URL"
                    />
                    <p class="text-xs text-muted-foreground">
                      当前后端未提供文件上传接口，头像使用 URL 保存。
                    </p>
                  </div>
                </div>
                <div class="flex justify-end border-t pt-4">
                  <Button :disabled="loading" @click="saveProfile">保存资料</Button>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="password" class="mt-6">
              <div class="max-w-xl space-y-5">
                <div>
                  <h2 class="text-lg font-semibold">修改密码</h2>
                  <p class="text-sm text-muted-foreground">
                    设置新的登录密码，保存后下次登录生效。
                  </p>
                </div>
                <div class="rounded-lg border bg-muted/30 p-4 text-sm text-muted-foreground">
                  <div class="flex gap-3">
                    <ShieldCheck class="mt-0.5 size-5 shrink-0 text-primary" />
                    <div>
                      <p class="font-medium text-foreground">密码安全提示</p>
                      <p class="mt-1">建议至少使用 6 个字符，并组合字母、数字和特殊符号。</p>
                    </div>
                  </div>
                </div>
                <div class="space-y-4">
                  <div class="space-y-1.5">
                    <Label for="account-new-password">新密码</Label
                    ><Input
                      id="account-new-password"
                      v-model="passwordForm.newPassword"
                      type="password"
                      autocomplete="new-password"
                      placeholder="请输入新密码"
                    />
                  </div>
                  <div class="space-y-1.5">
                    <Label for="account-confirm-password">确认密码</Label
                    ><Input
                      id="account-confirm-password"
                      v-model="passwordForm.confirmPassword"
                      type="password"
                      autocomplete="new-password"
                      placeholder="请再次输入新密码"
                      @keyup.enter="changePassword"
                    />
                  </div>
                </div>
                <div class="flex justify-end border-t pt-4">
                  <Button
                    :disabled="
                      loading || !passwordForm.newPassword || !passwordForm.confirmPassword
                    "
                    @click="changePassword"
                    >修改密码</Button
                  >
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
