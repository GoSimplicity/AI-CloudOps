import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    component: BasicLayout,
    meta: {
      order: 1,
      title: $t('page.serviceTree.title'),
    },
    name: 'ServiceTree',
    path: '/tree',
    children: [
      {
        name: '服务树管理',
        path: '/system',
        component: () => import('#/views/servicetree/system/system.vue'),
        meta: {
          title: $t('page.serviceTree.system'),
        }
      },
    ],
  },
];

export default routes;
