import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';

const routes: RouteRecordRaw[] = [
  {
    component: BasicLayout,
    meta: {
      order: 2,
      title: "Promethues管理",
    },
    name: 'Prometheus',
    path: '/prometheus',
    children: [
      {
        name: 'MonitorScrapePool',
        path: '/monitor_pool',
        component: () =>
          import('#/views/promethues/MonitorScrapePool.vue'),
        meta: {
          title: "采集池",
          icon: 'lucide:database',
        },
      },
      {
        name: 'MonitorScrapeJob',
        path: '/monitor_job',
        component: () =>
          import('#/views/promethues/MonitorScrapeJob.vue'),
        meta: {
          title: "采集任务",
          icon: 'lucide:list-check',
        },
      },
      {
        name: 'MonitorAlert',
        path: '/monitor_alert',
        component: () =>
          import('#/views/promethues/MonitorAlert.vue'),
        meta: {
          title: "alert告警池",
          icon: 'lucide:alert-triangle',
        },
      },
      
      {
        name: 'MonitorAlertRule',
        path: '/monitor_alert_rule',
        component: () =>
          import('#/views/promethues/MonitorAlertRule.vue'),
        meta: {
          title: "告警规则",
          icon: 'lucide:badge-alert',
        },
      },
      {
        name: 'MonitorAlertEvent',
        path: '/monitor_alert_event',
        component: () =>
          import('#/views/promethues/MonitorAlertEvent.vue'),
        meta: {
          title: "告警事件",
          icon: 'lucide:bell-ring',
        },
      },
      {
        name: 'MonitorAlertRecord',
        path: '/monitor_alert_record',
        component: () =>
          import('#/views/promethues/MonitorAlertRecord.vue'),
        meta: {
          title: "预聚合",
          icon: 'lucide:box',
        },
      },
      {
        name: 'MonitorConfig',
        path: '/monitor_config',
        component: () =>
          import('#/views/promethues/MonitorConfig.vue'),
        meta: {
          title: "配置文件",
          icon: 'lucide:file-text',
        },
      },
      {
        name: 'MonitorOnDutyGroup',
        path: '/monitor_onduty_group',
        component: () =>
          import('#/views/promethues/MonitorOnDutyGroup.vue'),
        meta: {
          title: "值班组",
          icon: 'lucide:user-round-minus'
        },
      },
      {
        name: 'MonitorOnDutyGroup',
        path: '/monitor_onduty_group',
        component: () =>
          import('#/views/promethues/MonitorOnDutyGroup.vue'),
        meta: {
          title: "排班表",
          icon: 'lucide:user-round-minus'
        },
      },
      {
        name: 'MonitorSend',
        path: '/monitor_send',
        component: () =>
          import('#/views/promethues/MonitorSend.vue'),
        meta: {
          title: "发送组",
          icon: 'lucide:send-horizontal'
        },
      },
    ],
  },
];

export default routes;
