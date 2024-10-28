import type { RouteRecordStringComponent } from '@vben/types';

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

export const getMonitorScrapePoolApi = () => {
  return requestClient.get<MonitorScrapePoolItem[]>('/monitor/scrape_pools');
};

export const createMonitorScrapePoolApi = (data: createMonitorScrapePoolReq) => {
  return requestClient.post('/monitor/scrape_pools/create', data);
};

export const deleteMonitorScrapePoolApi = (id: number) => {
  return requestClient.delete(`/monitor/scrape_pools/${id}`);
};

export const updateMonitorScrapePoolApi = (data: updateMonitorScrapePoolReq) => {
  return requestClient.post('/monitor/scrape_pools/update', data);
};