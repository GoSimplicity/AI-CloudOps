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
        name: '服务树概览',
        path: '/tree_overview',
        component: () =>
          import('#/views/servicetree/tree_overview/TreeOverview.vue'),
        meta: {
          title: $t('page.serviceTree.overview'),
        },
      },
    ],
  },
];

export default routes;
