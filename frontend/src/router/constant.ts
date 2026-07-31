import type { RouteRecordRaw } from 'vue-router'

/**
 * 常量路由 - 不需要权限验证的基础路由
 */
export const constantRoutes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: {
      titleKey: 'menu.login',
      title: 'Login',
      requiresAuth: false,
      hideInMenu: true,
      hideInTab: true,
    },
  },
  {
    path: '/401',
    name: '401',
    component: () => import('@/views/error/401.vue'),
    meta: {
      titleKey: 'menu.unauthorized',
      title: 'Unauthorized',
      hideInMenu: true,
      hideInTab: true,
    },
  },
  {
    path: '/403',
    name: '403',
    component: () => import('@/views/error/403.vue'),
    meta: {
      titleKey: 'menu.forbidden',
      title: 'Forbidden',
      hideInMenu: true,
      hideInTab: true,
    },
  },
  {
    path: '/404',
    name: '404',
    component: () => import('@/views/error/404.vue'),
    meta: {
      titleKey: 'menu.notFound',
      title: 'Page Not Found',
      hideInMenu: true,
      hideInTab: true,
    },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/error/404.vue'),
    meta: {
      titleKey: 'menu.notFound',
      title: 'Page Not Found',
      hideInMenu: true,
      hideInTab: true,
    },
  },
]

/**
 * 根路由 - 包含布局的父路由
 */
export const rootRoute: RouteRecordRaw = {
  path: '/',
  name: 'Root',
  component: () => import('@/layouts/DashboardLayout.vue'),
  meta: {
    titleKey: 'menu.home',
    title: 'Home',
    requiresAuth: true,
    hideInMenu: true,
  },
  children: [
    {
      path: '',
      name: 'Index',
      redirect: '/dashboard',
      meta: {
        titleKey: 'menu.index',
        title: 'Index',
        hideInMenu: true,
        hideInTab: true,
      },
    },
  ],
}
