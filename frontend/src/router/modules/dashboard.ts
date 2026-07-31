import type { RouteRecordRaw } from 'vue-router'
import { LayoutDashboard } from '@lucide/vue'

/**
 * 仪表盘路由模块
 */
const dashboardRoutes: RouteRecordRaw[] = [
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('@/views/Dashboard.vue'),
    meta: {
      titleKey: 'menu.dashboard',
      title: 'Gateway Dashboard',
      icon: LayoutDashboard,
      order: 1,
      keepAlive: true,
    },
  },
]

export default dashboardRoutes
