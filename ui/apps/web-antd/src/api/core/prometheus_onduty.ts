import { requestClient } from '#/api/request';

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

export interface GetOnDutyListParams {
  page: number;
  size: number;
  search: string;
}

export const getOnDutyListApi = (data: GetOnDutyListParams) => {
  return requestClient.get(`/monitor/onDuty_groups/list`, { params: data });
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
