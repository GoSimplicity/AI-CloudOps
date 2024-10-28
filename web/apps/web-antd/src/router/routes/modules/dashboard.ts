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
        component: () => import('#/views/dashboard/system_welcome/SystemWelcome.vue'),
        meta: {
          affixTab: true,
          icon: 'lucide:area-chart',
          title: $t('page.dashboard.welcome'),
        },
      },
      {
        name: '菜单管理',
        path: '/system_menu',
        component: () => import('#/views/dashboard/system_menu/SystemMenu.vue'),
        meta: {
          icon: 'lucide:menu',
          title: $t('page.dashboard.menus'),
        },
      },
      {
        name: '用户管理',
        path: '/system_user',
        component: () => import('#/views/dashboard/system_user/SystemUser.vue'),
        meta: {
          icon: 'lucide:user',
          title: $t('page.dashboard.users'),
        },
      },
      {
        name: '权限管理',
        path: '/system_role',
        component: () => import('#/views/dashboard/system_role/SystemRole.vue'),
        meta: {
          icon: 'lucide:user',
          title: $t('page.dashboard.roles'),
        },
      },
      {
        name: '接口管理',
        path: '/system_api',
        component: () => import('#/views/dashboard/system_api/SystemApi.vue'),
        meta: {
          title: $t('page.dashboard.apis'),
          icon: 'lucide:zap',
        },
      },
    ],
  },
];

export default routes;
