import { requestClient } from '#/api/request';

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
}

export const deleteScrapeJobApi = (id: number) => {
  return requestClient.delete(`/monitor/scrape_jobs/${id}`);
}

export const updateScrapeJobApi = (data: editScrapeJobReq) => {
  return requestClient.post('/monitor/scrape_jobs/update', data);
}