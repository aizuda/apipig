import type { RouteRecordRaw } from 'vue-router'
import { Bot, GitPullRequest } from '@lucide/vue'

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
