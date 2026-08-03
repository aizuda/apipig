<script setup lang="ts">
import { ListChecks, MonitorCog } from '@lucide/vue'
import { Button } from '@tabtab/ui'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const items = [
  { label: 'Agent 节点', path: '/ai-applications/remote-agent/agents', icon: MonitorCog },
  { label: '远程任务', path: '/ai-applications/remote-agent/tasks', icon: ListChecks },
]
</script>

<template>
  <nav class="flex gap-1 border-b" aria-label="Remote Agent 导航">
    <Button
      v-for="item in items"
      :key="item.path"
      :variant="route.path.startsWith(item.path) ? 'secondary' : 'ghost'"
      class="rounded-b-none gap-2"
      @click="router.push(item.path)"
    >
      <component :is="item.icon" class="h-4 w-4" />
      {{ item.label }}
    </Button>
  </nav>
</template>
