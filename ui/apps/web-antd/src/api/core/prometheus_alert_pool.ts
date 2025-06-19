import { requestClient } from '#/api/request';

export interface MonitorAlertPoolItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  name: string;
  alert_manager_instances: string[];
  user_id: number;
  resolve_timeout: string;
  group_wait: string;
  group_interval: string;
  repeat_interval: string;
  group_by: string[];
  receiver: string;
  create_user_name: string;
  data_length: number;
}

export interface createAlertManagerPoolReq {
  name: string;
  alert_manager_instances: string[];
  resolve_timeout: string;
  group_wait: string;
  group_interval: string;
  repeat_interval: string;
  group_by: string[];
  receiver: string;
}

export interface updateAlertManagerPoolReq {
  id: number;
  name: string;
  alert_manager_instances: string[];
  resolve_timeout: string;
  group_wait: string;
  group_interval: string;
  repeat_interval: string;
  group_by: string[];
  receiver: string;
}

export interface GetAlertManagerPoolListParams {   
  page: number;
  size: number;
  search: string;
}

export const getAlertManagerPoolListApi = (data: GetAlertManagerPoolListParams) => {
  return requestClient.get(`/monitor/alertManager_pools/list`, { params: data });
};

export const getAllAlertManagerPoolApi = () => {
  return requestClient.get('/monitor/alertManager_pools/all');
};

export const createAlertManagerPoolApi = (data: createAlertManagerPoolReq) => {
  return requestClient.post('/monitor/alertManager_pools/create', data);
};

export const updateAlertManagerPoolApi = (data: updateAlertManagerPoolReq) => {
  return requestClient.post('/monitor/alertManager_pools/update', data);
};

export const deleteAlertManagerPoolApi = (id: number) => {
  return requestClient.delete(`/monitor/alertManager_pools/${id}`);
};

