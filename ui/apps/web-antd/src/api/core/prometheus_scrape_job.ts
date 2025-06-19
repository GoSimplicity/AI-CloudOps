import { requestClient } from '#/api/request';

export interface MonitorScrapeJobItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  name: string;
  user_id: number;
  enable: boolean;
  service_discovery_type: string;
  metrics_path: string;
  scheme: string;
  scrape_interval: number;
  scrape_timeout: number;
  pool_id: number;
  relabel_configs_yaml_string: string;
  refresh_interval: number;
  port: number;
  ip_address: string[];
  kube_config_file_path: string;
  tls_ca_file_path: string;
  tls_ca_content: string;
  bearer_token: string;
  bearer_token_file: string;
  kubernetes_sd_role: string;
  create_user_name: string;
}

export interface createScrapeJobReq {
  name: string;
  enable: boolean;
  service_discovery_type: string;
  metrics_path: string;
  scheme: string;
  scrape_interval: number;
  scrape_timeout: number;
  pool_id: number;
  relabel_configs_yaml_string: string;
  refresh_interval: number;
  port: number;
  ip_address: string;
  kube_config_file_path: string;
  tls_ca_file_path: string;
  tls_ca_content: string;
  bearer_token: string;
  bearer_token_file: string;
  kubernetes_sd_role: string;
}

export interface updateScrapeJobReq {
  id: number;
  name: string;
  enable: boolean;
  service_discovery_type: string;
  metrics_path: string;
  scheme: string;
  scrape_interval: number;
  scrape_timeout: number;
  pool_id: number;
  relabel_configs_yaml_string: string;
  refresh_interval: number;
  port: number;
  ip_address: string;
  kube_config_file_path: string;
  tls_ca_file_path: string;
  tls_ca_content: string;
  bearer_token: string;
  bearer_token_file: string;
  kubernetes_sd_role: string;
}

export interface GetScrapeJobListParams {
  page: number;
  size: number;
  search: string;
}

export const getMonitorScrapeJobListApi = (data: GetScrapeJobListParams) => {
  return requestClient.get(`/monitor/scrape_jobs/list`, { params: data });
};

export const getMonitorScrapeJobTotalApi = () => {
  return requestClient.get('/monitor/scrape_jobs/total');
};

export const createScrapeJobApi = (data: createScrapeJobReq) => {
  return requestClient.post('/monitor/scrape_jobs/create', data);
};

export const updateScrapeJobApi = (data: updateScrapeJobReq) => {
  return requestClient.post('/monitor/scrape_jobs/update', data);
};

export const deleteScrapeJobApi = (id: number) => {
  return requestClient.delete(`/monitor/scrape_jobs/${id}`);
};

export const getScrapeJobTotalApi = () => {
  return requestClient.get('/monitor/scrape_jobs/total');
};
