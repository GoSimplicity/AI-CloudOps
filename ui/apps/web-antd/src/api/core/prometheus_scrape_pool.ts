import { requestClient } from '#/api/request';

export interface MonitorScrapePoolItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  name: string;
  prometheus_instances: string[];
  alert_manager_instances: string[];
  user_id: number;
  scrape_interval: number;
  scrape_timeout: number;
  remote_timeout_seconds: number;
  support_alert: boolean;
  support_record: boolean;
  external_labels: string[];
  remote_write_url: string;
  remote_read_url: string;
  alert_manager_url: string;
  rule_file_path: string;
  record_file_path: string;
  create_user_name: string;
}

export interface createMonitorScrapePoolReq {
  name: string;
  prometheus_instances: string[];
  alert_manager_instances: string[];
  scrape_interval: number;
  scrape_timeout: number;
  remote_timeout_seconds: number;
  support_alert: boolean;
  support_record: boolean;
  external_labels: string[];
  remote_write_url: string;
  remote_read_url: string;
  alert_manager_url: string;
  rule_file_path: string;
  record_file_path: string;
}

export interface updateMonitorScrapePoolReq {
  id: number;
  name: string;
  prometheus_instances: string[];
  alert_manager_instances: string[];
  scrape_interval: number;
  scrape_timeout: number;
  remote_timeout_seconds: number;
  support_alert: boolean;
  support_record: boolean;
  external_labels: string[];
  remote_write_url: string;
  remote_read_url: string;
  alert_manager_url: string;
  rule_file_path: string;
  record_file_path: string;
}

export interface GetScrapePoolListParams {
  page: number;
  size: number;
  search: string;
}

export const getMonitorScrapePoolListApi = (data: GetScrapePoolListParams) => {
  return requestClient.get(`/monitor/scrape_pools/list`, { params: data });
};

export const getAllMonitorScrapePoolApi = () => {
  return requestClient.get('/monitor/scrape_pools/all');
};

export const createMonitorScrapePoolApi = (
  data: createMonitorScrapePoolReq,
) => {
  return requestClient.post('/monitor/scrape_pools/create', data);
};

export const updateMonitorScrapePoolApi = (
  data: updateMonitorScrapePoolReq,
) => {
  return requestClient.post('/monitor/scrape_pools/update', data);
};

export const deleteMonitorScrapePoolApi = (id: number) => {
  return requestClient.delete(`/monitor/scrape_pools/${id}`);
};

export const getMonitorScrapePoolTotalApi = () => {
  return requestClient.get('/monitor/scrape_pools/total');
};
