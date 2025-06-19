import { requestClient } from '#/api/request';

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
  labels_map: Record<string, string>;
  annotations_map: Record<string, string>;
}

export interface GetAlertEventsListParams {
  page: number;
  size: number;
  search: string;
}

export const getAlertEventsListApi = (data: GetAlertEventsListParams) => {
  return requestClient.get(`/monitor/alert_events/list`, { params: data });
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
