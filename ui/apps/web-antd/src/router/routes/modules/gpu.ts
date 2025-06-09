import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';

const routes: RouteRecordRaw[] = [
  {
    component: BasicLayout,
    meta: {
      icon: 'lucide:cpu',
      order: 50,
      title: 'GPU训练管理',
    },
    name: 'GPUTraining',
    path: '/gpu',
    children: [
      {
        name: 'VolcanoJobs',
        path: '/jobs',
        component: () => import('#/views/gpu/VolcanoJobs.vue'),
        meta: {
          icon: 'lucide:play-circle',
          title: '训练作业',
        },
      },
      {
        name: 'JobQueues',
        path: '/queues',
        component: () => import('#/views/gpu/JobQueues.vue'),
        meta: {
          icon: 'lucide:layers',
          title: '作业队列',
        },
      },
      {
        name: 'JobTemplates',
        path: '/templates',
        component: () => import('#/views/gpu/JobTemplates.vue'),
        meta: {
          icon: 'lucide:file-code',
          title: '作业模板',
        },
      },
      {
        name: 'WorkflowManagement',
        path: '/workflows',
        component: () => import('#/views/gpu/Workflows.vue'),
        meta: {
          icon: 'lucide:git-branch',
          title: '工作流管理',
        },
      },
      {
        name: 'GPUTopology',
        path: '/topology',
        component: () => import('#/views/gpu/GPUTopology.vue'),
        meta: {
          icon: 'lucide:network',
          title: 'GPU拓扑',
        },
      },
      {
        name: 'ModelRegistry',
        path: '/models',
        component: () => import('#/views/gpu/ModelRegistry.vue'),
        meta: {
          icon: 'lucide:brain',
          title: '模型注册',
        },
      },
      {
        name: 'DataManagement',
        path: '/data',
        component: () => import('#/views/gpu/DataManagement.vue'),
        meta: {
          icon: 'lucide:database',
          title: '数据管理',
        },
      },
      {
        name: 'ExperimentTracking',
        path: '/experiments',
        component: () => import('#/views/gpu/ExperimentTracking.vue'),
        meta: {
          icon: 'lucide:flask-conical',
          title: '实验跟踪',
        },
      },
      {
        name: 'NotebookServices',
        path: '/notebooks',
        component: () => import('#/views/gpu/NotebookServices.vue'),
        meta: {
          icon: 'lucide:notebook-pen',
          title: 'Notebook服务',
        },
      },
      {
        name: 'JobMonitoring',
        path: '/monitoring',
        component: () => import('#/views/gpu/JobMonitoring.vue'),
        meta: {
          icon: 'lucide:activity',
          title: '作业监控',
        },
      },
    ],
  },
];

export default routes;