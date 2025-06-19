import { requestClient } from '#/api/request';

export interface AlertRuleItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  name: string;
  user_id: number;
  pool_id?: number | null;
  ip_address: string;
  send_group_id?: number | null;
  enable: boolean;
  expr: string;
  severity: string;
  grafana_link: string;
  for_time: string;
  labels: string[];
  annotations: string[];
  node_path: string;
  pool_name: string;
  send_group_name: string;
  create_user_name: string;
}

export interface createAlertRuleReq {
  name: string;
  pool_id?: number | null;
  send_group_id?: number | null;
  enable: boolean;
  expr: string;
  severity: string;
  grafana_link: string;
  for_time: string;
  labels: string[];
  annotations: string[];
}

export interface updateAlertRuleReq {
  id: number;
  name: string;
  pool_id?: number | null;
  send_group_id?: number | null;
  enable: boolean;
  expr: string;
  severity: string;
  grafana_link: string;
  for_time: string;
  labels: string[];
  annotations: string[];
}

export interface validateExprApiReq {
  promql_expr: string;
}

export interface GetAlertRulesListParams {
  page: number;
  size: number;
  search: string;
}

export const getAlertRulesListApi = (data: GetAlertRulesListParams) => {
  return requestClient.get(`/monitor/alert_rules/list`, { params: data });
};

export const getMonitorAlertRuleTotalApi = () => {
  return requestClient.get('/monitor/alert_rules/total');
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

export const getAlertRuleTotalApi = () => {
  return requestClient.get('/monitor/alert_rules/total');
};

export const validateExprApi = (data: validateExprApiReq) => {
  return requestClient.post('/monitor/alert_rules/promql_check', data);
};
