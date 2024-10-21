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
          import('#/views/servicetree/TreeOverview.vue'),
        meta: {
          title: "服务树概览",
        },
      },
      {
        name: '服务树节点管理',
        path: '/tree_node_manager',
        component: () =>
          import('#/views/servicetree/TreeNodeManager.vue'),
        meta: {
          title: "服务树节点管理",
        },
      },
      {
        name: '资产管理',
        path: '/resource_operation',
        component: () =>
          import('#/views/servicetree/ResourceOperation.vue'),
        meta: {
          title: "资产管理",
        },
      },
    ],
  },
];

export default routes;
