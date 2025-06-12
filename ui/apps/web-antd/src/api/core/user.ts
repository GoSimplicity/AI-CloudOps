import type { UserInfo } from '@vben/types';

import { requestClient } from '#/api/request';

/**
 * 获取用户信息
 */
type changePasswordReq = {
  username: string;
  password: string;
  newPassword: string;
  confirmPassword: string;
}

type RegisterParams = {
  username: string;
  password: string;
  confirmPassword: string;
  mobile: string;
  fei_shu_user_id: string;
  real_name: string;
  email: string;
  desc: string;
  home_path: string
};

type updateUserInfoReq = {
  user_id: number;
  real_name: string;
  desc: string;
  mobile: string; 
  fei_shu_user_id: string;
  account_type: number;
  email: string;
  home_path: string;
  enable: 0 | 1;
}

type WriteOffReq = {
  username: string;
  password: string;
}

export interface ListReq {
  page: number;
  size: number;
  search: string;
}

export async function getUserInfoApi() {
  return requestClient.get<UserInfo>('/user/profile');
}

export const getUserList = (data: ListReq) => {
  return requestClient.get('/user/list', { params: data });
};


export async function registerApi(data: RegisterParams) {
  return requestClient.post('/user/signup', data);
}

export async function changePassword(data: changePasswordReq) {
  return requestClient.post('/user/change_password', data);
}

export async function deleteUser(id: number) {
  return requestClient.delete(`/user/${id}`);
}

export async function updateUserInfo(data: updateUserInfoReq) {
  return requestClient.post('/user/profile/update', data);
}

export async function getUserDetailApi(id: number) {
  return requestClient.get(`/user/detail/${id}`);
}

export async function writeOffAccount(data: WriteOffReq) {
  return requestClient.post('/user/write_off', data);
}