import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';

const routes: RouteRecordRaw[] = [
  {
    component: BasicLayout,
    meta: {
      icon: 'lucide:ticket',
      order: 99,
      title: 'AIops',
    },
    name: 'AIops',
    path: '/aiops',
    children: [
      {
        name: '根因分析',
        path: '/root_cause',
        component: () => import('#/views/aiops/RootCause.vue'),
        meta: {
          icon: 'lucide:pencil',
          title: '根因分析',
        },
      },
      {
        name: '告警预测',
        path: '/alarm_prediction',
        component: () => import('#/views/aiops/AlarmPrediction.vue'),
        meta: {
          icon: 'lucide:git-branch',
          title: '告警预测',
        },
      },
      {
        name: '故障自动修复',
        path: '/fault_auto_repair',
        component: () => import('#/views/aiops/FaultAutoRepair.vue'),
        meta: {
          icon: 'lucide:copy',
          title: '故障自动修复',
        },
      },
    ],
  },
];

export default routes;
