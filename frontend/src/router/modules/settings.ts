import type { RouteRecordRaw } from 'vue-router'
import {
  Settings,
  Users,
  Shield,
  ListTree,
  Bell,
  UserCircle,
  ShieldCheck,
  UserCog,
} from '@lucide/vue'

/**
 * 系统设置路由模块
 */
const settingsRoutes: RouteRecordRaw[] = [
  {
    path: '/settings',
    name: 'Settings',
    redirect: '/settings/users',
    meta: {
      titleKey: 'menu.settings',
      title: 'Settings',
      icon: Settings,
      order: 4,
    },
    children: [
      {
        path: '/settings/users',
        name: 'SettingsUsers',
        component: () => import('@/views/settings/users/UserManagement.vue'),
        meta: {
          titleKey: 'menu.users',
          title: 'Users',
          icon: Users,
          order: 1,
          keepAlive: true,
        },
      },
      {
        path: '/settings/users/detail/:id',
        name: 'SettingsUserDetail',
        component: () => import('@/views/settings/users/UserDetail.vue'),
        meta: {
          titleKey: 'settings.userDetail',
          title: 'User Detail',
          icon: UserCircle,
          order: 2,
          keepAlive: true,
          hideInMenu: true,
        },
      },
      {
        path: '/settings/roles',
        name: 'SettingsRoles',
        component: () => import('@/views/settings/roles/RoleManagement.vue'),
        meta: {
          titleKey: 'menu.roles',
          title: 'Roles',
          icon: Shield,
          order: 2,
          keepAlive: true,
        },
      },
      {
        path: '/settings/roles/detail/:id',
        name: 'SettingsRoleDetail',
        component: () => import('@/views/settings/roles/RoleDetail.vue'),
        meta: {
          titleKey: 'settings.roleDetail',
          title: 'Role Detail',
          icon: ShieldCheck,
          order: 3,
          keepAlive: true,
          hideInMenu: true,
        },
      },
      {
        path: '/settings/menus',
        name: 'SettingsMenus',
        component: () => import('@/views/settings/menus/MenuManagement.vue'),
        meta: {
          titleKey: 'menu.permissions',
          title: 'Menu Management',
          icon: ListTree,
          order: 3,
          keepAlive: true,
        },
      },
      {
        path: '/settings/permissions',
        name: 'SettingsPermissionsLegacy',
        redirect: '/settings/menus',
        meta: {
          title: 'Legacy Permissions Redirect',
          hideInMenu: true,
          hideInTab: true,
        },
      },
      {
        path: '/settings/account',
        name: 'SettingsAccount',
        component: () => import('@/views/settings/account/AccountSettings.vue'),
        meta: {
          titleKey: 'menu.account',
          title: 'Account Settings',
          icon: UserCog,
          order: 4,
          keepAlive: true,
        },
      },
      {
        path: '/settings/messages',
        name: 'MyMessages',
        component: () => import('@/views/settings/messages/MyMessages.vue'),
        meta: {
          titleKey: 'menu.myMessages',
          title: 'My Messages',
          icon: Bell,
          order: 5,
          keepAlive: true,
        },
      },
    ],
  },
]

export default settingsRoutes
