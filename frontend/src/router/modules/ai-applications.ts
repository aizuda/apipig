import type { RouteRecordRaw } from 'vue-router'
import { Bot, GitPullRequest, ListChecks, MonitorCog } from '@lucide/vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/ai-applications',
    name: 'AiApplications',
    redirect: '/ai-applications/code-review',
    meta: { title: 'AI 应用', icon: Bot, order: 3 },
    children: [
      {
        path: '/ai-applications/code-review',
        name: 'CodeReview',
        redirect: '/ai-applications/code-review/projects',
        meta: { title: 'AI 代码评审', icon: GitPullRequest, order: 1 },
      },
      {
        path: '/ai-applications/remote-agent',
        name: 'RemoteAgent',
        redirect: '/ai-applications/remote-agent/agents',
        meta: { titleKey: 'menu.remoteAgent', title: 'Remote Agent', icon: MonitorCog, order: 2 },
      },
      {
        path: '/ai-applications/remote-agent/agents',
        name: 'RemoteAgentAgents',
        component: () => import('@/views/ai-applications/remote-agent/AgentManagement.vue'),
        meta: {
          titleKey: 'menu.remoteAgentAgents',
          title: 'Agent 节点',
          icon: MonitorCog,
          order: 3,
          hideInMenu: true,
        },
      },
      {
        path: '/ai-applications/remote-agent/agents/:id',
        name: 'RemoteAgentAgentDetail',
        component: () => import('@/views/ai-applications/remote-agent/AgentDetail.vue'),
        meta: { title: 'Agent 详情', icon: MonitorCog, order: 4, hideInMenu: true },
      },
      {
        path: '/ai-applications/remote-agent/tasks',
        name: 'RemoteAgentTasks',
        component: () => import('@/views/ai-applications/remote-agent/TaskManagement.vue'),
        meta: {
          titleKey: 'menu.remoteAgentTasks',
          title: '远程任务',
          icon: ListChecks,
          order: 5,
          hideInMenu: true,
        },
      },
      {
        path: '/ai-applications/remote-agent/tasks/:id',
        name: 'RemoteAgentTaskDetail',
        component: () => import('@/views/ai-applications/remote-agent/TaskDetail.vue'),
        meta: { title: '任务详情', icon: ListChecks, order: 6, hideInMenu: true },
      },
      {
        path: '/ai-applications/code-review/projects',
        name: 'CodeReviewProjects',
        component: () => import('@/views/ai-applications/code-review/CodeReviewProjects.vue'),
        meta: {
          titleKey: 'menu.codeReviewProjects',
          title: '代码评审项目',
          icon: GitPullRequest,
          order: 1,
          hideInMenu: true,
        },
      },
      {
        path: '/ai-applications/code-review/tasks',
        name: 'CodeReviewTasks',
        component: () => import('@/views/ai-applications/code-review/CodeReviewTasks.vue'),
        meta: {
          titleKey: 'menu.codeReviewTasks',
          title: '评审任务',
          icon: GitPullRequest,
          order: 2,
          hideInMenu: true,
        },
      },
    ],
  },
]

export default routes
