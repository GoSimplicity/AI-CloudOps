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
  tree_node_ids: string[];
  kube_config_file_path: string;
  tls_ca_file_path: string;
  tls_ca_content: string;
  bearer_token: string;
  bearer_token_file: string;
  kubernetes_sd_role: string;
  tree_node_names: string[];
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
  tree_node_ids: string[];
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
  tree_node_ids: string[];
  kube_config_file_path: string;
  tls_ca_file_path: string;
  tls_ca_content: string;
  bearer_token: string;
  bearer_token_file: string;
  kubernetes_sd_role: string;
}

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

export interface AlertRuleItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  name: string;
  user_id: number;
  pool_id?: number | null;
  send_group_id?: number | null;
  tree_node_id?: number | null;
  enable: boolean;
  expr: string;
  severity: string;
  grafana_link: string;
  for_time: string;
  labels: string[];
  annotations: string[];
  node_path: string;
  tree_node_names: string[];
  pool_name: string;
  send_group_name: string;
  create_user_name: string;
}

export interface createAlertRuleReq {
  name: string;
  pool_id?: number | null;
  send_group_id?: number | null;
  tree_node_id?: number | null;
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
  tree_node_id?: number | null;
  enable: boolean;
  expr: string;
  severity: string;
  grafana_link: string;
  for_time: string;
  labels: string[];
  annotations: string[];
}

export interface MonitorAlertEventItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  alert_name: string;
  fingerprint: string;
  status: string;
  rule_id: number;
  send_group_id: number;
  event_times: number;
  silence_id: string;
  ren_ling_user_id: number;
  labels: string[];
  alert_rule_name: string;
  send_group_name: string;
  alert: any;
  send_group: any;
  ren_ling_user: any;
  rule: any;
}

export interface AlertRecordItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  name: string;
  user_id: number;
  pool_id: number;
  tree_node_id: number;
  enable: boolean;
  for_time: string;
  expr: string;
  labels: string[];
  annotations: string[];
  node_path: string;
  tree_node_names: string[];
  pool_name: string;
  send_group_name: string;
  create_user_name: string;
}

export interface createAlertManagerRecordReq {
  name: string;
  pool_id?: number | null;
  tree_node_id?: number | null;
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
  tree_node_id?: number | null;
  enable: boolean;
  for_time: string;
  expr: string;
  labels: string[];
  annotations: string[];
}

export interface SendGroupItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  name: string;
  name_zh: string;
  enable: boolean;
  user_id: number;
  pool_id: number;
  on_duty_group_id: number;
  static_receive_users: any[];
  fei_shu_qun_robot_token: string;
  repeat_interval: string;
  send_resolved: boolean;
  notify_methods: string[];
  need_upgrade: boolean;
  first_upgrade_users: any[];
  upgrade_minutes: number;
  second_upgrade_users: any[];
  tree_node_names: string[];
  static_receive_user_names: string[];
  first_user_names: string[];
  second_user_names: string[];
  pool_name: string;
  on_duty_group_name: string;
  create_user_name: string;
}

export interface createSendGroupReq {
  name: string;
  name_zh: string;
  enable: boolean;
  pool_id: number;
  on_duty_group_id: number;
  static_receive_users: any[];
  fei_shu_qun_robot_token: string;
  repeat_interval: string;
  send_resolved: boolean;
  notify_methods: string[];
  need_upgrade: boolean;
  first_upgrade_users: any[];
  upgrade_minutes: number;
  second_upgrade_users: any[];
}

export interface updateSendGroupReq {
  id: number;
  name: string;
  name_zh: string;
  enable: boolean;
  pool_id: number;
  on_duty_group_id: number;
  static_receive_users: any[];
  fei_shu_qun_robot_token: string;
  repeat_interval: string;
  send_resolved: boolean;
  notify_methods: string[];
  need_upgrade: boolean;
  first_upgrade_users: any[];
  upgrade_minutes: number;
  second_upgrade_users: any[];
}

export interface OnDutyGroupItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  name: string;
  user_id: number;
  members: any[];
  shift_days: number;
  yesterday_normal_duty_user_id: number;
  today_duty_user: any;
  user_names: string[];
  create_user_name: string;
}

export interface OnDutyGroupChangeItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  on_duty_group_id: number;
  user_id: number;
  date: string;
  origin_user_id: number;
  on_duty_user_id: number;
  target_user_name: string;
  origin_user_name: string;
  pool_name: string;
  create_user_name: string;
}

export interface OnDutyGroupHistoryItem {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at: number;
  on_duty_group_id: number;
  date_string: string;
  on_duty_user_id: number;
  origin_user_id: number;
  on_duty_user_name: string;
  origin_user_name: string;
  pool_name: string;
  create_user_name: string;
}

export interface createOnDutyReq {
  name: string;
  shift_days: number;
  user_names: string[];
}

export interface createOnDutychangeReq {
  on_duty_group_id: number;
  date: string;
  origin_user_id: number;
  on_duty_user_id: number;
}

export interface updateOnDutyReq {
  id: number;
  on_duty_group_id: number;
  date: string;
  origin_user_id: number;
  on_duty_user_id: number;
}

export interface getOnDutyFuturePlan {
  id: number;
  start_time: string;
  end_time: string;
}

export interface validateExprApiReq {
  promql_expr: string;
}

export const getMonitorScrapePoolListApi = (
  page: number,
  size: number,
  search: string,
) => {
  return requestClient.get<MonitorScrapePoolItem[]>(
    `/monitor/scrape_pools/list?page=${page}&size=${size}&search=${search}`,
  );
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

export const getMonitorScrapeJobListApi = (
  page: number,
  size: number,
  search: string,
) => {
  return requestClient.get<MonitorScrapeJobItem[]>(
    `/monitor/scrape_jobs/list?page=${page}&size=${size}&search=${search}`,
  );
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

export const getAlertManagerPoolListApi = (
  page: number,
  size: number,
  search: string,
) => {
  return requestClient.get<MonitorAlertPoolItem[]>(
    `/monitor/alertManager_pools/list?page=${page}&size=${size}&search=${search}`,
  );
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

export const getAlertManagerPoolTotalApi = () => {
  return requestClient.get('/monitor/alertManager_pools/total');
};

export const getAlertRulesListApi = (
  page: number,
  size: number,
  search: string,
) => {
  return requestClient.get<AlertRuleItem[]>(
    `/monitor/alert_rules/list?page=${page}&size=${size}&search=${search}`,
  );
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

export const getAlertEventsListApi = (
  page: number,
  size: number,
  search: string,
) => {
  return requestClient.get<MonitorAlertEventItem[]>(
    `/monitor/alert_events/list?page=${page}&size=${size}&search=${search}`,
  );
};

export const getAlertEventsTotalApi = () => {
  return requestClient.get('/monitor/alert_events/total');
};

export const getRecordRulesListApi = (
  page: number,
  size: number,
  search: string,
) => {
  return requestClient.get<AlertRecordItem[]>(
    `/monitor/record_rules/list?page=${page}&size=${size}&search=${search}`,
  );
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

export const getMonitorSendGroupListApi = (
  page: number,
  size: number,
  search: string,
) => {
  return requestClient.get<SendGroupItem[]>(
    `/monitor/send_groups/list?page=${page}&size=${size}&search=${search}`,
  );
};

export const getAllMonitorSendGroupApi = () => {
  return requestClient.get('/monitor/send_groups/all');
};

export const getMonitorSendGroupTotalApi = () => {
  return requestClient.get('/monitor/send_groups/total');
};

export const createMonitorSendGroupApi = (data: createSendGroupReq) => {
  return requestClient.post('/monitor/send_groups/create', data);
};

export const updateMonitorSendGroupApi = (data: updateSendGroupReq) => {
  return requestClient.post('/monitor/send_groups/update', data);
};

export const deleteMonitorSendGroupApi = (id: number) => {
  return requestClient.delete(`/monitor/send_groups/${id}`);
};

export const getSendGroupTotalApi = () => {
  return requestClient.get('/monitor/send_groups/total');
};

export const silenceAlertApi = (id: number) => {
  return requestClient.get(`/monitor/alert_events/silence/${id}`);
};

export const claimAlertApi = (id: number) => {
  return requestClient.get(`/monitor/alert_events/claim/${id}`);
};

export const cancelSilenceAlertApi = (id: number) => {
  return requestClient.get(`/monitor/alert_events/cancel_silence/${id}`);
};

export const silenceBatchApi = (ids: number[]) => {
  return requestClient.post('/monitor/alert_events/silence_batch', ids);
};

export const getOnDutyListApi = (
  page: number,
  size: number,
  search: string,
) => {
  return requestClient.get<OnDutyGroupItem[]>(
    `/monitor/onDuty_groups/list?page=${page}&size=${size}&search=${search}`,
  );
};

export const getAllOnDutyGroupApi = () => {
  return requestClient.get('/monitor/onDuty_groups/all');
};

export const getOnDutyTotalApi = () => {
  return requestClient.get('/monitor/onDuty_groups/total');
};

export const getOnDutyApi = (id: number) => {
  return requestClient.get(`/monitor/onDuty_groups/${id}`);
};

export const createOnDutyApi = (data: createOnDutyReq) => {
  return requestClient.post('/monitor/onDuty_groups/create', data);
};

export const updateOnDutyApi = (data: any) => {
  return requestClient.post('/monitor/onDuty_groups/update', data);
};

export const deleteOnDutyApi = (id: number) => {
  return requestClient.delete(`/monitor/onDuty_groups/${id}`);
};

export const getOnDutyFuturePlanApi = (data: getOnDutyFuturePlan) => {
  return requestClient.get(
    `/monitor/onDuty_groups/future_plan?id=${data.id}&start_time=${data.start_time}&end_time=${data.end_time}`,
  );
};

export const createOnDutyChangeApi = (data: createOnDutychangeReq) => {
  return requestClient.post('/monitor/onDuty_groups/changes', data);
};
