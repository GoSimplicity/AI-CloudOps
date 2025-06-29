import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';

const routes: RouteRecordRaw[] = [
  {
    component: BasicLayout,
    meta: {
      icon: 'lucide:ticket',
      order: -1,
      title: '工单管理',
    },
    name: 'WorkOrder',
    path: '/workorder',
    children: [
      {
        name: '表单设计',
        path: '/form_design',
        component: () => import('#/views/workorder/FormDesign.vue'),
        meta: {
          icon: 'lucide:pencil',
          title: '表单设计',
        },
      },
      {
        name: '表单管理',
        path: '/form_management',
        component: () => import('#/views/workorder/FormManagement.vue'),
        meta: {
          icon: 'lucide:file-text',
          title: '表单管理',
        },
      },
      {
        name: '分类管理',
        path: '/category',
        component: () => import('#/views/workorder/Category.vue'),
        meta: {
          icon: 'lucide:folder',
          title: '分类管理',
        },
      },
      {
        name: '流程管理',
        path: '/process',
        component: () => import('#/views/workorder/Process.vue'),
        meta: {
          icon: 'lucide:git-branch',
          title: '流程管理',
        },
      },
      {
        name: '工单实例',
        path: '/instance',
        component: () => import('#/views/workorder/Instance.vue'),
        meta: {
          icon: 'lucide:ticket',
          title: '工单实例',
        },
      },
      {
        name: '通知配置',
        path: '/notification',
        component: () => import('#/views/workorder/Notification.vue'),
        meta: {
          icon: 'lucide:bell',
          title: '通知配置',
        },
      },
      {
        name: '统计分析',
        path: '/statistics',
        component: () => import('#/views/workorder/Statistics.vue'),
        meta: {
          icon: 'lucide:bar-chart',
          title: '统计分析',
        },
      },
      {
        name: '模板管理',
        path: '/template',
        component: () => import('#/views/workorder/Template.vue'),
        meta: {
          icon: 'lucide:copy',
          title: '模板管理',
        },
      },
    ],
  },
];

export default routes;
