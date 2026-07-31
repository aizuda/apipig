<script setup lang="ts">
import { computed, toRef } from 'vue'
import { Button, Dialog, DialogFixedContent, Input, Label, Textarea } from '@tabtab/ui'
import type { AccessToken } from '@/api/ai-gateway'
import type { ReviewProject } from '@/api/ai-applications/code-review'

const props = defineProps<{
  open: boolean
  loading: boolean
  form: ReviewProject
  tokens: AccessToken[]
}>()

const emit = defineEmits<{
  close: []
  submit: []
}>()

const projectForm = toRef(props, 'form')
const availableModels = computed(() => {
  const token = props.tokens.find((item) => item.id === projectForm.value.accessTokenId)
  return (token?.models || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
})
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && !loading && emit('close')">
    <DialogFixedContent
      :title="projectForm.id ? '编辑评审项目' : '新增评审项目'"
      description="配置代码仓库、触发事件和 AI 评审策略。敏感凭据不会在编辑时回显。"
      class="sm:max-w-4xl"
    >
      <div class="space-y-6">
        <section class="space-y-3">
          <div>
            <h3 class="text-sm font-semibold">基础信息</h3>
            <p class="text-xs text-muted-foreground">用于识别仓库及确定需要监听的代码分支。</p>
          </div>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label for="review-project-name"
                >项目名称 <span class="text-destructive">*</span></Label
              >
              <Input
                id="review-project-name"
                v-model="projectForm.name"
                placeholder="例如 订单服务"
              />
            </div>
            <div class="space-y-1.5">
              <Label for="review-project-provider">Git 平台</Label>
              <select
                id="review-project-provider"
                v-model="projectForm.provider"
                class="h-10 w-full rounded-md border bg-background px-3 text-sm"
              >
                <option value="github">GitHub</option>
                <option value="gitlab">GitLab</option>
                <option value="gitee">Gitee</option>
              </select>
            </div>
            <div class="space-y-1.5 sm:col-span-2">
              <Label for="review-repository-url"
                >HTTPS 仓库地址 <span class="text-destructive">*</span></Label
              >
              <Input
                id="review-repository-url"
                v-model="projectForm.repositoryUrl"
                placeholder="https://github.com/org/repo.git"
              />
            </div>
            <div class="space-y-1.5">
              <Label for="review-default-branch">默认分支</Label>
              <Input
                id="review-default-branch"
                v-model="projectForm.defaultBranch"
                placeholder="main"
              />
            </div>
            <div class="space-y-1.5">
              <Label for="review-branch-pattern">监听分支</Label>
              <Input
                id="review-branch-pattern"
                v-model="projectForm.branchPattern"
                placeholder="main,release/*；留空表示全部"
              />
            </div>
          </div>
        </section>

        <section class="space-y-3 border-t pt-5">
          <div>
            <h3 class="text-sm font-semibold">凭据与模型</h3>
            <p class="text-xs text-muted-foreground">
              仓库 Token 仅私有仓库需要；AI 密钥来自网关的 API 密钥配置。
            </p>
          </div>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label for="review-repository-token">仓库访问 Token</Label>
              <Input
                id="review-repository-token"
                v-model="projectForm.repositoryToken"
                type="password"
                :placeholder="projectForm.id ? '留空表示不修改' : '公开仓库可留空'"
              />
            </div>
            <div class="space-y-1.5">
              <Label for="review-access-token"
                >AI API 密钥 <span class="text-destructive">*</span></Label
              >
              <select
                id="review-access-token"
                v-model="projectForm.accessTokenId"
                class="h-10 w-full rounded-md border bg-background px-3 text-sm"
              >
                <option value="">请选择 API 密钥</option>
                <option v-for="token in tokens" :key="token.id" :value="token.id">
                  {{ token.name }}
                </option>
              </select>
            </div>
            <div class="space-y-1.5 sm:col-span-2">
              <Label for="review-model">评审模型 <span class="text-destructive">*</span></Label>
              <Input
                id="review-model"
                v-model="projectForm.model"
                list="review-model-options"
                placeholder="选择或输入已授权模型"
              />
              <datalist id="review-model-options">
                <option v-for="model in availableModels" :key="model" :value="model" />
              </datalist>
            </div>
          </div>
        </section>

        <section class="space-y-3 border-t pt-5">
          <div>
            <h3 class="text-sm font-semibold">触发与评审策略</h3>
            <p class="text-xs text-muted-foreground">
              控制触发事件、忽略文件及项目特有的评审要求。
            </p>
          </div>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label for="review-ignore-patterns">忽略文件</Label>
              <Input
                id="review-ignore-patterns"
                v-model="projectForm.ignorePatterns"
                placeholder="例如 vendor/**,*.lock,dist/**"
              />
            </div>
            <div class="space-y-1.5">
              <Label for="review-project-status">项目状态</Label>
              <select
                id="review-project-status"
                v-model.number="projectForm.status"
                class="h-10 w-full rounded-md border bg-background px-3 text-sm"
              >
                <option :value="1">启用</option>
                <option :value="2">禁用</option>
              </select>
            </div>
            <div class="space-y-1.5 sm:col-span-2">
              <Label for="review-prompt">项目评审规则</Label>
              <Textarea
                id="review-prompt"
                v-model="projectForm.reviewPrompt"
                rows="4"
                placeholder="例如：重点检查数据库事务、向后兼容和权限边界"
              />
            </div>
            <div class="space-y-1.5 sm:col-span-2">
              <Label for="review-remark">备注</Label>
              <Input
                id="review-remark"
                v-model="projectForm.remark"
                placeholder="记录负责人或维护说明"
              />
            </div>
          </div>
          <div class="flex flex-wrap gap-4 rounded-lg border bg-muted/20 p-3 text-sm">
            <label class="flex items-center gap-2">
              <input v-model="projectForm.pushEnabled" type="checkbox" />监听 Push
            </label>
            <label class="flex items-center gap-2">
              <input v-model="projectForm.pullRequestEnabled" type="checkbox" />监听 Pull/Merge
              Request
            </label>
            <label v-if="projectForm.id" class="flex items-center gap-2">
              <input v-model="projectForm.rotateWebhookSecret" type="checkbox" />保存时轮换 WebHook
              Secret
            </label>
          </div>
        </section>
      </div>

      <template #footer>
        <Button variant="outline" :disabled="loading" @click="emit('close')">取消</Button>
        <Button :disabled="loading" @click="emit('submit')">
          {{ loading ? '保存中...' : projectForm.id ? '保存修改' : '创建项目' }}
        </Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
