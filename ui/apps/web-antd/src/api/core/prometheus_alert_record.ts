import { requestClient } from '#/api/request';

export interface AlertRecordItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  name: string;
  user_id: number;
  pool_id: number;
  enable: boolean;
  for_time: string;
  expr: string;
  labels: string[];
  annotations: string[];
  ip_address: string;
  port: number;
  pool_name: string;
  send_group_name: string;
  create_user_name: string;
}

export interface createAlertManagerRecordReq {
  name: string;
  pool_id?: number | null;
  enable: boolean;
  for_time: string;
  expr: string;
  labels: string[];
  annotations: string[];
}

export interface updateAlertManagerRecordReq {
  id: number;
  name: string;
  pool_id?: number | null;
  enable: boolean;
  for_time: string;
  expr: string;
  labels: string[];
  annotations: string[];
}

export interface GetRecordRulesListParams {
  page: number;
  size: number;
  search: string;
}

export const getRecordRulesListApi = (data: GetRecordRulesListParams) => {
  return requestClient.get(`/monitor/record_rules/list`, { params: data });
};

export const getRecordRulesTotalApi = () => {
  return requestClient.get('/monitor/record_rules/total');
};

export const createRecordRuleApi = (data: createAlertManagerRecordReq) => {
  return requestClient.post('/monitor/record_rules/create', data);
};

export const updateRecordRuleApi = (data: updateAlertManagerRecordReq) => {
  return requestClient.post('/monitor/record_rules/update', data);
};

export const deleteRecordRuleApi = (id: number) => {
  return requestClient.delete(`/monitor/record_rules/${id}`);
};

export const getRecordRuleTotalApi = () => {
  return requestClient.get('/monitor/record_rules/total');
};
