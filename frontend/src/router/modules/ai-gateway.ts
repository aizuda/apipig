import type { RouteRecordRaw } from 'vue-router'
import { Activity, Bot, Cable, KeyRound, Network, Route, Users } from '@lucide/vue'

const aiGatewayRoutes: RouteRecordRaw[] = [
  {
    path: '/ai-gateway',
    name: 'AiGateway',
    redirect: '/ai-gateway/overview',
    meta: {
      titleKey: 'menu.aiGateway',
      title: 'Gateway Console',
      icon: Bot,
      order: 2,
    },
    children: [
      {
        path: '/ai-gateway/overview',
        name: 'AiGatewayOverview',
        component: () => import('@/views/ai-gateway/overview/Overview.vue'),
        meta: {
          titleKey: 'menu.aiGatewayOverview',
          title: 'Operations Overview',
          icon: Bot,
          order: 1,
        },
      },
      {
        path: '/ai-gateway/providers',
        name: 'AiGatewayProviders',
        component: () => import('@/views/ai-gateway/providers/ProviderManagement.vue'),
        meta: {
          titleKey: 'menu.providers',
          title: 'Provider Access',
          icon: Network,
          order: 2,
        },
      },
      {
        path: '/ai-gateway/channels',
        name: 'AiGatewayChannels',
        component: () => import('@/views/ai-gateway/channels/ChannelManagement.vue'),
        meta: {
          titleKey: 'menu.channels',
          title: 'Routing Channels',
          icon: Route,
          order: 3,
        },
      },
      {
        path: '/ai-gateway/channel-accounts',
        name: 'AiGatewayChannelAccounts',
        component: () => import('@/views/ai-gateway/channel-accounts/ChannelAccountManagement.vue'),
        meta: {
          titleKey: 'menu.channelAccounts',
          title: 'Channel Accounts',
          icon: Users,
          order: 4,
        },
      },
      {
        path: '/ai-gateway/tokens',
        name: 'AiGatewayTokens',
        component: () => import('@/views/ai-gateway/tokens/AccessTokenManagement.vue'),
        meta: {
          titleKey: 'menu.accessTokens',
          title: 'Access Tokens',
          icon: KeyRound,
          order: 5,
        },
      },
      {
        path: '/ai-gateway/proxies',
        name: 'AiGatewayProxies',
        component: () => import('@/views/ai-gateway/proxies/ProxyManagement.vue'),
        meta: {
          titleKey: 'menu.proxyPool',
          title: 'Egress Proxies',
          icon: Cable,
          order: 6,
        },
      },
      {
        path: '/ai-gateway/logs',
        name: 'AiGatewayLogs',
        component: () => import('@/views/ai-gateway/calllogs/CallLogsManagement.vue'),
        meta: {
          titleKey: 'menu.callLogs',
          title: 'Request Logs',
          icon: Activity,
          order: 7,
        },
      },
    ],
  },
]

export default aiGatewayRoutes
