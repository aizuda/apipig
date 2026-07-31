<script setup lang="ts">
import { FolderKanban, ListChecks } from '@lucide/vue'
import { Button } from '@tabtab/ui'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const items = [
  {
    label: '项目配置',
    description: '管理仓库、模型和 WebHook',
    path: '/ai-applications/code-review/projects',
    icon: FolderKanban,
  },
  {
    label: '评审任务',
    description: '查看执行状态和评审报告',
    path: '/ai-applications/code-review/tasks',
    icon: ListChecks,
  },
]
</script>

<template>
  <nav
    class="grid gap-2 rounded-xl border bg-muted/20 p-2 sm:grid-cols-2"
    aria-label="代码评审导航"
  >
    <Button
      v-for="item in items"
      :key="item.path"
      :variant="route.path === item.path ? 'secondary' : 'ghost'"
      class="h-auto justify-start gap-3 px-4 py-3 text-left"
      @click="router.push(item.path)"
    >
      <component :is="item.icon" class="h-5 w-5 shrink-0" />
      <span class="min-w-0">
        <span class="block font-medium">{{ item.label }}</span>
        <span class="block truncate text-xs font-normal text-muted-foreground">{{
          item.description
        }}</span>
      </span>
    </Button>
  </nav>
</template>
