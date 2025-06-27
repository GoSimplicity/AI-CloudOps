import type { UserInfo } from '@vben/types';

import { requestClient } from '#/api/request';

/**
 * 获取用户信息
 */
export interface ChangePasswordReq {
  user_id: number;
  username: string;
  password: string;
  new_password: string;
  confirm_password: string;
}

export interface UserSignUpReq {
  username: string;
  password: string;
  mobile: string;
  real_name: string;
  fei_shu_user_id?: string;
  desc?: string;
  account_type: 1 | 2; // 1普通用户 2服务账号
  home_path?: string;
  enable?: 1 | 2; // 1正常 2冻结
}

export interface UpdateProfileReq {
  id: number;
  real_name: string;
  desc?: string;
  avatar?: string;
  mobile: string;
  email?: string;
  fei_shu_user_id?: string;
  account_type: 1 | 2;
  home_path?: string;
  enable?: 1 | 2;
}

export interface WriteOffReq {
  username: string;
  password: string;
}

export interface GetUserListReq {
  page: number;
  size: number;
  search: string;
  enable?: number;
  account_type?: number;
}

export async function getUserInfoApi() {
  return requestClient.get<UserInfo>('/user/profile');
}

export const getUserList = (data: GetUserListReq) => {
  return requestClient.get('/user/list', { params: data });
};

export async function registerApi(data: UserSignUpReq) {
  return requestClient.post('/user/signup', data);
}

export async function changePassword(data: ChangePasswordReq) {
  return requestClient.post('/user/change_password', data);
}

export async function deleteUser(id: number) {
  return requestClient.delete(`/user/${id}`);
}

export async function updateUserInfo(data: UpdateProfileReq) {
  return requestClient.post('/user/profile/update', data);
}

export async function getUserDetailApi(id: number) {
  return requestClient.get(`/user/detail/${id}`);
}

export async function writeOffAccount(data: WriteOffReq) {
  return requestClient.post('/user/write_off', data);
}

export async function getUserStatistics() {
  return requestClient.get('/user/statistics');
}