import { requestClient } from '#/api/request';

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

export interface GetSendGroupListParams {
  page: number;
  size: number;
  search: string;
}

export const getMonitorSendGroupListApi = (data: GetSendGroupListParams) => {
  return requestClient.get(`/monitor/send_groups/list`, { params: data });
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
