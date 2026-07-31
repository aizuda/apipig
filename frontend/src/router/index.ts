import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { constantRoutes, rootRoute } from './constant'
import { setupRouterGuard } from './guard'

/**
 * 动态导入所有路由模块
 */
const modules = import.meta.glob('./modules/*.ts', { eager: true })

/**
 * 从模块中提取路由配置
 */
const moduleRoutes: RouteRecordRaw[] = []
for (const path in modules) {
  const module = modules[path] as { default: RouteRecordRaw[] }
  if (module?.default) {
    moduleRoutes.push(...module.default)
  }
}

/**
 * 对路由按 order 排序
 */
function sortRoutes(routes: RouteRecordRaw[]): RouteRecordRaw[] {
  return [...routes].sort((a, b) => {
    const orderA = (a.meta?.order as number | undefined) ?? 0
    const orderB = (b.meta?.order as number | undefined) ?? 0
    return orderA - orderB
  })
}

/**
 * 组装完整路由配置
 */
const indexRoute = rootRoute.children?.find((r) => r.path === '') || {
  path: '',
  name: 'Index',
  redirect: '/dashboard',
  meta: {
    title: '首页',
    hideInMenu: true,
    hideInTab: true,
  },
}
rootRoute.children = sortRoutes([indexRoute, ...moduleRoutes])

const routes: RouteRecordRaw[] = [...constantRoutes, rootRoute]

/**
 * 创建路由实例
 */
const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

/**
 * 设置路由守卫
 */
setupRouterGuard(router)

export default router
