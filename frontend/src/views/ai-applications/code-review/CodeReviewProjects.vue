<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Bell, ExternalLink, Pencil, Plus, Search, Trash2, Webhook } from '@lucide/vue'
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
import { aiGatewayApi, type AccessToken } from '@/api/ai-gateway'
import { codeReviewApi, type ReviewProject } from '@/api/ai-applications/code-review'
import { apiUrl } from '@/api/request'
import AppPageHeader from '@/components/AppPageHeader.vue'
import AppPagination from '@/components/AppPagination.vue'
import CodeReviewSectionNav from './components/CodeReviewSectionNav.vue'
import ProjectEditorDialog from './components/ProjectEditorDialog.vue'
import WebhookCredentialsDialog from './components/WebhookCredentialsDialog.vue'
import PushChannelsDialog from './components/PushChannelsDialog.vue'

defineOptions({ name: 'CodeReviewProjects' })

const loading = ref(false)
const saving = ref(false)
const projects = ref<ReviewProject[]>([])
const tokens = ref<AccessToken[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')
const provider = ref('')
const status = ref(0)
const editorOpen = ref(false)
const deleteTarget = ref<ReviewProject | null>(null)
const webhookDialog = reactive({ open: false, projectName: '', url: '', secret: '' })
const pushDialog = reactive({ open: false, project: null as ReviewProject | null })

const emptyProject = (): ReviewProject => ({
  name: '',
  provider: 'github',
  repositoryUrl: '',
  defaultBranch: 'main',
  accessTokenId: '',
  model: '',
  reviewPrompt: '',
  branchPattern: '',
  ignorePatterns: '',
  pushEnabled: true,
  pullRequestEnabled: true,
  status: 1,
  remark: '',
  repositoryToken: '',
  webhookSecret: '',
})
const form = reactive<ReviewProject>(emptyProject())

async function loadProjects() {
  loading.value = true
  try {
    const result = await codeReviewApi.projectPage({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      provider: provider.value || undefined,
      status: status.value || undefined,
    })
    projects.value = result.records
    total.value = result.total
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载项目失败')
  } finally {
    loading.value = false
  }
}

async function loadTokens() {
  try {
    const result = await aiGatewayApi.tokenPage({ page: 1, pageSize: 100 })
    tokens.value = result.records.filter((item) => item.status === 1)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '加载 API 密钥失败')
  }
}

function openCreate() {
  Object.assign(form, emptyProject())
  editorOpen.value = true
}

function openEdit(project: ReviewProject) {
  Object.assign(form, emptyProject(), project, {
    repositoryToken: '',
    webhookSecret: '',
    rotateWebhookSecret: false,
  })
  editorOpen.value = true
}

async function saveProject() {
  if (
    !form.name.trim() ||
    !form.repositoryUrl.trim() ||
    !form.accessTokenId ||
    !form.model.trim()
  ) {
    toast.error('请填写项目名称、仓库地址、API 密钥和评审模型')
    return
  }
  saving.value = true
  try {
    const result = await codeReviewApi.saveProject({ ...form })
    editorOpen.value = false
    webhookDialog.open = true
    webhookDialog.projectName = result.project.name
    webhookDialog.url = result.webhookUrl
    webhookDialog.secret = result.webhookSecret || ''
    toast.success(form.id ? '项目配置已更新' : '评审项目已创建')
    await loadProjects()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '保存项目失败')
  } finally {
    saving.value = false
  }
}

function showWebhook(project: ReviewProject) {
  webhookDialog.open = true
  webhookDialog.projectName = project.name
  webhookDialog.url = apiUrl(
    `/apps/code-review/webhook/${encodeURIComponent(project.webhookKey || '')}`,
  )
  webhookDialog.secret = ''
}

function showPushChannels(project: ReviewProject) {
  pushDialog.project = project
  pushDialog.open = true
}

function updatePushChannels(channels: ReviewProject['pushChannels']) {
  if (pushDialog.project) pushDialog.project.pushChannels = channels
}

async function confirmDelete() {
  if (!deleteTarget.value?.id) return
  loading.value = true
  try {
    await codeReviewApi.deleteProject(deleteTarget.value.id)
    toast.success('项目已删除')
    deleteTarget.value = null
    if (projects.value.length === 1 && page.value > 1) page.value -= 1
    await loadProjects()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '删除项目失败')
  } finally {
    loading.value = false
  }
}

function searchProjects() {
  page.value = 1
  void loadProjects()
}

function resetFilters() {
  keyword.value = ''
  provider.value = ''
  status.value = 0
  searchProjects()
}

function changePage(value: number) {
  page.value = value
  void loadProjects()
}

function changePageSize(value: number) {
  pageSize.value = value
  page.value = 1
  void loadProjects()
}

function providerLabel(value: string) {
  return value === 'github'
    ? 'GitHub'
    : value === 'gitlab'
      ? 'GitLab'
      : value === 'gitee'
        ? 'Gitee'
        : value
}

onMounted(async () => {
  await Promise.all([loadProjects(), loadTokens()])
})
</script>

<template>
  <div class="space-y-6">
    <AppPageHeader
      title="代码评审项目"
      description="一个项目对应一个代码仓库和一套独立的 AI 评审策略。"
      :loading="loading"
      @refresh="loadProjects"
    >
      <template #actions>
        <Button size="sm" class="gap-2" @click="openCreate"
          ><Plus class="h-4 w-4" />新增项目</Button
        >
      </template>
    </AppPageHeader>

    <CodeReviewSectionNav />

    <Card>
      <CardContent class="p-0">
        <form
          class="flex flex-col gap-3 border-b p-4 lg:flex-row lg:items-center"
          @submit.prevent="searchProjects"
        >
          <div class="relative min-w-0 flex-1">
            <Search
              class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            />
            <Input v-model="keyword" class="pl-9" placeholder="搜索项目名称或仓库地址" />
          </div>
          <select
            v-model="provider"
            class="h-10 rounded-md border bg-background px-3 text-sm lg:w-36"
          >
            <option value="">全部平台</option>
            <option value="github">GitHub</option>
            <option value="gitlab">GitLab</option>
            <option value="gitee">Gitee</option>
          </select>
          <select
            v-model.number="status"
            class="h-10 rounded-md border bg-background px-3 text-sm lg:w-32"
          >
            <option :value="0">全部状态</option>
            <option :value="1">启用</option>
            <option :value="2">禁用</option>
          </select>
          <div class="flex gap-2">
            <Button type="submit" variant="outline" :disabled="loading">查询</Button>
            <Button type="button" variant="ghost" :disabled="loading" @click="resetFilters"
              >重置</Button
            >
          </div>
        </form>

        <div v-if="projects.length" class="overflow-x-auto">
          <table class="w-full min-w-[900px] text-sm">
            <thead class="border-b bg-muted/20 text-left text-muted-foreground">
              <tr>
                <th class="px-4 py-3 font-medium">项目与仓库</th>
                <th class="px-4 py-3 font-medium">平台 / 分支</th>
                <th class="px-4 py-3 font-medium">模型</th>
                <th class="px-4 py-3 font-medium">触发事件</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="project in projects"
                :key="project.id"
                class="border-b last:border-0 hover:bg-muted/20"
              >
                <td class="px-4 py-4">
                  <div class="font-medium">{{ project.name }}</div>
                  <a
                    :href="project.repositoryUrl"
                    target="_blank"
                    rel="noreferrer"
                    class="mt-1 inline-flex max-w-[340px] items-center gap-1 truncate text-xs text-muted-foreground hover:text-foreground"
                  >
                    {{ project.repositoryUrl }}<ExternalLink class="h-3 w-3 shrink-0" />
                  </a>
                </td>
                <td class="px-4 py-4">
                  <div>{{ providerLabel(project.provider) }}</div>
                  <div class="mt-1 text-xs text-muted-foreground">
                    {{ project.defaultBranch || 'main' }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <div>{{ project.model }}</div>
                  <div class="mt-1 max-w-52 truncate text-xs text-muted-foreground">
                    {{ project.remark || '未填写备注' }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <div class="flex flex-wrap gap-1">
                    <Badge v-if="project.pushEnabled" variant="outline">Push</Badge>
                    <Badge v-if="project.pullRequestEnabled" variant="outline">Pull/Merge</Badge>
                    <span
                      v-if="!project.pushEnabled && !project.pullRequestEnabled"
                      class="text-muted-foreground"
                      >未启用</span
                    >
                  </div>
                </td>
                <td class="px-4 py-4">
                  <Badge :variant="project.status === 1 ? 'default' : 'secondary'">
                    {{ project.status === 1 ? '启用' : '禁用' }}
                  </Badge>
                </td>
                <td class="px-4 py-4 text-right">
                  <Button variant="ghost" size="sm" class="gap-1.5" @click="showPushChannels(project)">
                    <Bell class="h-4 w-4" />推送渠道
                  </Button>
                  <Button variant="ghost" size="sm" class="gap-1.5" @click="showWebhook(project)">
                    <Webhook class="h-4 w-4" />WebHook
                  </Button>
                  <Button variant="ghost" size="sm" class="gap-1.5" @click="openEdit(project)">
                    <Pencil class="h-4 w-4" />编辑
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label="删除项目"
                    @click="deleteTarget = project"
                  >
                    <Trash2 class="h-4 w-4 text-destructive" />
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else-if="!loading" class="px-6 py-16 text-center">
          <Webhook class="mx-auto h-10 w-10 text-muted-foreground/50" />
          <h3 class="mt-4 font-medium">暂无评审项目</h3>
          <p class="mt-1 text-sm text-muted-foreground">
            创建项目后即可获得 WebHook URL 和 Secret。
          </p>
          <Button class="mt-4" @click="openCreate">创建第一个项目</Button>
        </div>

        <AppPagination
          :total="total"
          :page="page"
          :page-size="pageSize"
          :loading="loading"
          @change-page="changePage"
          @change-page-size="changePageSize"
        />
      </CardContent>
    </Card>

    <ProjectEditorDialog
      :open="editorOpen"
      :loading="saving"
      :form="form"
      :tokens="tokens"
      @close="editorOpen = false"
      @submit="saveProject"
    />
    <WebhookCredentialsDialog
      :open="webhookDialog.open"
      :project-name="webhookDialog.projectName"
      :webhook-url="webhookDialog.url"
      :webhook-secret="webhookDialog.secret"
      @close="webhookDialog.open = false"
    />
    <PushChannelsDialog
      :open="pushDialog.open"
      :project-id="pushDialog.project?.id"
      :project-name="pushDialog.project?.name || ''"
      :channels="pushDialog.project?.pushChannels || []"
      @close="pushDialog.open = false"
      @saved="updatePushChannels"
    />

    <AlertDialog
      :open="Boolean(deleteTarget)"
      @update:open="(value) => !value && !loading && (deleteTarget = null)"
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除“{{ deleteTarget?.name }}”？</AlertDialogTitle>
          <AlertDialogDescription>
            删除后该项目将不再接收 WebHook；存在等待或执行中任务时，后端会阻止删除。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="loading" @click="deleteTarget = null"
            >取消</AlertDialogCancel
          >
          <Button variant="destructive" :disabled="loading" @click="confirmDelete">确认删除</Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
