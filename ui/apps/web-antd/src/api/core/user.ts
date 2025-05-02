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
  home_path: string;
  enable: number;
}

export async function getUserInfoApi() {
  return requestClient.get<UserInfo>('/user/profile');
}

export async function getAllUsers() {
  return requestClient.get('/user/list');
}

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
