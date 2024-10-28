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
          icon: 'material-symbols:overview'
        },
      },
      {
        name: '服务树节点管理',
        path: '/tree_node_manager',
        component: () =>
          import('#/views/servicetree/TreeNodeManager.vue'),
        meta: {
          title: "服务树节点管理",
          icon: 'fluent-mdl2:task-manager'
        },
      },
      {
        name: 'ECS管理',
        path: '/ecs_resource_operation',
        component: () =>
          import('#/views/servicetree/ECSResourceOperation.vue'),
        meta: {
          title: "ECS管理",
          icon: 'mdi:cloud-cog-outline'
        },
      },
      // {
      //   name: 'RDS管理',
      //   path: '/rds_resource_operation',
      //   component: () =>
      //     import('#/views/servicetree/RDSResourceOperation.vue'),
      //   meta: {
      //     title: "RDS管理",
      //   },
      // },
      // {
      //   name: 'ELB管理',
      //   path: '/elb_resource_operation',
      //   component: () =>
      //     import('#/views/servicetree/ELBResourceOperation.vue'),
      //   meta: {
      //     title: "ELB管理",
      //   },
      // },
    ],
  },
];

export default routes;
