import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    component: BasicLayout,
    meta: {
      icon: 'lucide:layout-dashboard',
      order: -1,
      title: $t('page.dashboard.title'),
    },
    name: 'Dashboard',
    path: '/',
    children: [
      {
        name: 'Welcome',
        path: '/system_welcome',
        component: () => import('#/views/dashboard/SystemWelcome.vue'),
        meta: {
          affixTab: true,
          icon: 'lucide:area-chart',
          title: '欢迎页',
        },
      },
      {
        name: '用户管理',
        path: '/system_user',
        component: () => import('#/views/dashboard/SystemUser.vue'),
        meta: {
          icon: 'lucide:user',
          title: '用户管理',
        },
      },
      {
        name: '接口管理',
        path: '/system_api',
        component: () => import('#/views/dashboard/SystemApi.vue'),
        meta: {
          title: '接口管理',
          icon: 'lucide:zap',
        },
      },
      {
        name: '角色权限',
        path: '/system_role',
        component: () => import('#/views/dashboard/SystemRole.vue'),
        meta: {
          icon: 'lucide:users',
          title: '角色权限',
        },
      },
    ],
  },
];

export default routes;
