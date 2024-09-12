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
        path: '/welcome',
        component: () => import('#/views/dashboard/welcome/welcome.vue'),
        meta: {
          affixTab: true,
          icon: 'lucide:area-chart',
          title: $t('page.dashboard.welcome'),
        },
      },
      {
        name: '菜单管理',
        path: '/menus',
        component: () => import('#/views/dashboard/menus/menus.vue'),
        meta: {
          icon: 'lucide:menu',
          title: $t('page.dashboard.menus'),
        },
      },
      {
        name: '用户管理',
        path: '/users',
        component: () => import('#/views/dashboard/users/users.vue'),
        meta: {
          icon: 'lucide:user',
          title: $t('page.dashboard.users'),
        },
      },
      {
        name: '权限管理',
        path: '/roles',
        component: () => import('#/views/dashboard/roles/roles.vue'),
        meta: {
          icon: 'lucide:user',
          title: $t('page.dashboard.roles'),
        },
      },
      {
        name: '接口管理',
        path: '/apis',
        component: () => import('#/views/dashboard/apis/apis.vue'),
        meta: {
          title: $t('page.dashboard.apis'),
        },
      },
    ],
  },
];

export default routes;
