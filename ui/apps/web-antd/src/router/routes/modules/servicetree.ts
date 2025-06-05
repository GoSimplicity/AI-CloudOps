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
        component: () => import('#/views/servicetree/TreeOverview.vue'),
        meta: {
          title: '服务树概览',
          icon: 'material-symbols:overview',
        },
      },
      {
        name: '服务树节点管理',
        path: '/tree_node_manager',
        component: () => import('#/views/servicetree/TreeNodeManager.vue'),
        meta: {
          title: '服务树节点管理',
          icon: 'fluent-mdl2:task-manager',
        },
      },
      {
        name: '资源管理',
        path: '/resource_management',
        component: () => import('#/views/servicetree/ResourceManagement.vue'),
        meta: {
          title: '资源管理',
          icon: 'mdi:cloud-cog-outline',
        },
      },
      {
        name: '本地资源管理',
        path: '/local_resource_management',
        component: () => import('#/views/servicetree/LocalResourceManagement.vue'),
        meta: {
          title: '本地资源管理',
          icon: 'mdi:server-network',
        },
      },
      {
        name: '云厂商管理',
        path: '/cloud_provider_management',
        component: () =>
          import('#/views/servicetree/CloudProviderManagement.vue'),
        meta: {
          title: '云厂商管理',
          icon: 'mdi:cloud-outline',
        },
      },
      {
        name: 'TerminalConnect',
        path: '/terminal_connect',
        component: () => import('#/views/servicetree/TerminalConnect.vue'),
        meta: {
          hideInMenu: true,
          title: '终端连接',
        },
      },
    ],
  },
];

export default routes;
