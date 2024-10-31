import { requestClient } from '#/api/request';
import { interaction } from '@antv/g2plot/lib/adaptor/common';

export interface MonitorScrapePoolItem {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  name: string;
  prometheusInstances: string[];
  alertManagerInstances: string[];
  userId: number;
  scrapeInterval: number;
  scrapeTimeout: number;
  externalLabels: string[];
  supportAlert: number;
  supportRecord: number;
  remoteReadUrl: string;
  alertManagerUrl: string;
  ruleFilePath: string;
  recordFilePath: string;
  remoteWriteUrl: string;
  remoteTimeoutSeconds: number;
}

export interface GeneralRes {
  code: number;
  data: any;
  message: string;
  type: string;
}

export interface createMonitorScrapePoolReq {
  name: string;
  prometheusInstances: string[];
  alertManagerInstances: string[];
  scrapeInterval: number;
  scrapeTimeout: number;
  externalLabels: string[];
  supportAlert: number;
  supportRecord: number;
  remoteReadUrl: string;
  alertManagerUrl: string;
  ruleFilePath: string;
  recordFilePath: string;
  remoteWriteUrl: string;
  remoteTimeoutSeconds: number;
}

export interface updateMonitorScrapePoolReq {
  ID: number;
  name: string;
  prometheusInstances: string[];
  alertManagerInstances: string[];
  scrapeInterval: number;
  scrapeTimeout: number;
  externalLabels: string[];
  supportAlert: number;
  supportRecord: number;
  remoteReadUrl: string;
  alertManagerUrl: string;
  ruleFilePath: string;
  recordFilePath: string;
  remoteWriteUrl: string;
  remoteTimeoutSeconds: number;
}

export interface MonitorScrapeJobItem {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  name: string;
  userId: number;
  enable: number;
  serviceDiscoveryType: string;
  metricsPath: string;
  scheme: string;
  scrapeInterval: number;
  scrapeTimeout: number;
  poolId: number;
  refreshInterval: number;
  port: number;
  treeNodeIds: string[];
  key: string;
}

export interface createScrapeJobReq {
  name: string;
  enable: number;
  serviceDiscoveryType: string;
  metricsPath: string;
  scheme: string;
  scrapeInterval: number;
  scrapeTimeout: number;
  poolId: number | null;
  refreshInterval: number;
  port: number;
  treeNodeIds: string[];
}

export interface editScrapeJobReq {
  ID: number;
  name: string;
  enable: number;
  serviceDiscoveryType: string;
  metricsPath: string;
  scheme: string;
  scrapeInterval: number;
  scrapeTimeout: number;
  poolId: number | null;
  refreshInterval: number;
  port: number;
  treeNodeIds: string[];
}

export interface MonitorAlertPoolItem {
  name: string;
  userId: number;
  alertManagerInstances: string[];
  resolveTimeout: string;
  groupWait: string;
  groupInterval: string;
  repeatInterval: string;
  groupBy: string[];
  receiver: string;
  CreatedAt: string;
}

export interface createAlertManagerPoolReq {
  name: string;
  alertManagerInstances: string[];
  resolveTimeout: string;
  groupWait: string;
  groupInterval: string;
  repeatInterval: string;
  groupBy: string[];
  receiver: string;
}

export interface editAlertManagerPoolReq {
  ID: number;
  name: string;
  alertManagerInstances: string[];
  resolveTimeout: string;
  groupWait: string;
  groupInterval: string;
  repeatInterval: string;
  groupBy: string[];
  receiver: string;
}

export interface createAlertRuleReq {
  name: string;
  poolId: number;
  sendGroupId: number;
  treeNodeId: number;
  expr: string;
  severity: string;
  forTime: string;
  enable: number;
  labels: string[];
  annotations: string[];
}

export interface updateAlertRuleReq {
  ID: number;
  name: string;
  poolId: number;
  sendGroupId: number;
  treeNodeId: number;
  expr: string;
  severity: string;
  forTime: string;
  enable: number;
  labels: string[];
  annotations: string[];
}

export interface validateExprApiReq {
  promqlExpr: string;
}

export interface MonitorAlertEventItem {
  ID: number;
  alertName: string;
  fingerprint: string;
  status: string;
  sendGroupId: string;
  eventTimes: number;
  renLingUserId: string;
  labels: string[];
  createTime: string;
  silenceId: string;
}

export const getMonitorScrapePoolApi = () => {
  return requestClient.get<MonitorScrapePoolItem[]>('/monitor/scrape_pools');
};

export const createMonitorScrapePoolApi = (
  data: createMonitorScrapePoolReq,
) => {
  return requestClient.post('/monitor/scrape_pools/create', data);
};

export const deleteMonitorScrapePoolApi = (id: number) => {
  return requestClient.delete(`/monitor/scrape_pools/${id}`);
};

export const updateMonitorScrapePoolApi = (
  data: updateMonitorScrapePoolReq,
) => {
  return requestClient.post('/monitor/scrape_pools/update', data);
};

export const getMonitorScrapeJobApi = () => {
  return requestClient.get<MonitorScrapeJobItem[]>('/monitor/scrape_jobs');
};

export const createScrapeJobApi = (data: createScrapeJobReq) => {
  return requestClient.post('/monitor/scrape_jobs/create', data);
};

export const deleteScrapeJobApi = (id: number) => {
  return requestClient.delete(`/monitor/scrape_jobs/${id}`);
};

export const updateScrapeJobApi = (data: editScrapeJobReq) => {
  return requestClient.post('/monitor/scrape_jobs/update', data);
};

export const getAlertManagerPoolsApi = () => {
  return requestClient.get('/monitor/alertManager_pools/');
};

export const createAlertManagerPoolApi = (data: createAlertManagerPoolReq) => {
  return requestClient.post('/monitor/alertManager_pools/create', data);
};

export const updateAlertManagerPoolApi = (data: editAlertManagerPoolReq) => {
  return requestClient.post('/monitor/alertManager_pools/update', data);
};

export const deleteAlertManagerPoolApi = (id: number) => {
  return requestClient.delete(`/monitor/alertManager_pools/${id}`);
};

export const getAlertRulesApi = () => {
  return requestClient.get('/monitor/alert_rules');
};

export const createAlertRuleApi = (data: createAlertRuleReq) => {
  return requestClient.post('/monitor/alert_rules/create', data);
};

export const updateAlertRuleApi = (data: updateAlertRuleReq) => {
  return requestClient.post('/monitor/alert_rules/update', data);
};

export const deleteAlertRuleApi = (id: number) => {
  return requestClient.delete(`/monitor/alert_rules/${id}`);
};

export const validateExprApi = (data: validateExprApiReq) => {
  return requestClient.post('/monitor/alert_rules/promql_check', data);
};

export const getAlertEventsApi = () => {
  return requestClient.get('/monitor/alert_events');
};
export const silenceAlertApi = () => {
  return requestClient.get('/monitor/alert_events');
};
export const claimAlertApi = () => {
  return requestClient.get('/monitor/alert_events');
};
export const cancelSilenceAlertApi = () => {
  return requestClient.get('/monitor/alert_events');
};
export const silenceBatchApi = () => {
  return requestClient.get('/monitor/alert_events');
};
