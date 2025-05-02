import { requestClient } from '#/api/request';

export namespace SystemApi {
  // API管理相关接口
  export interface ListApisReq {
    page_number: number; // 页码
    page_size: number; // 每页数量
  }

  export interface CreateApiReq {
    name: string; // API名称
    path: string; // API路径
    method: number; // 请求方法
    description?: string; // API描述
    version?: string; // API版本
    category?: number; // API分类
    is_public: 0 | 1; // 是否公开
  }

  export interface UpdateApiReq {
    id: number; // API ID
    user_id: number; // 用户ID
    name: string; // API名称
    path: string; // API路径
    method: number; // 请求方法
    description?: string; // API描述
    version?: string; // API版本
    category?: number; // API分类
    is_public: 0 | 1; // 是否公开
  }

  export interface DeleteApiReq {
    id: number; // API ID
  }

  // 角色管理相关接口
  export interface ListRolesReq {
    page_number: number; // 页码
    page_size: number; // 每页数量
  }

  export interface Role {
    name: string; // 角色名称
    domain: string; // 域ID
    path: string; // 路径
    method: string; // 方法
  }

  export interface CreateRoleReq {
    name: string; // 角色名称
    domain: string; // 域ID
    path: string; // 路径
    method: string; // 方法
  }

  export interface UpdateRoleReq {
    new_role: Role; // 新角色信息
    old_role: Role; // 旧角色信息
  }

  export interface DeleteRoleReq {
    name: string; // 角色名称
    domain: string; // 域ID
    path: string; // 路径
    method: string; // 方法
  }

  export interface ListUserRolesReq {
    page_number: number; // 页码
    page_size: number; // 每页数量
  }

  export interface UpdateUserRoleReq {
    user_id: number; // 用户ID
    api_ids?: number[]; // API ID列表
    role_ids: number[]; // 角色ID列表
  }
}

// API管理
export function listApisApi(data: SystemApi.ListApisReq) {
  return requestClient.post('/apis/list', data);
}

export function createApiApi(data: SystemApi.CreateApiReq) {
  return requestClient.post('/apis/create', data);
}

export function updateApiApi(data: SystemApi.UpdateApiReq) {
  return requestClient.post('/apis/update', data);
}

export function deleteApiApi(id: string) {
  return requestClient.delete(`/apis/${id}`);
}

// 角色管理
export function listRolesApi(data: SystemApi.ListRolesReq) {
  return requestClient.post('/roles/list', data);
}

export function createRoleApi(data: SystemApi.CreateRoleReq) {
  return requestClient.post('/roles/create', data);
}

export function updateRoleApi(data: SystemApi.UpdateRoleReq) {
  return requestClient.post('/roles/update', data);
}

export function deleteRoleApi(data: SystemApi.DeleteRoleReq) {
  return requestClient.post('/roles/delete', data);
}

// 获取用户角色
export function getUserRolesApi(data: SystemApi.ListUserRolesReq) {
  return requestClient.post('/roles/user/roles', data);
}

// 更新用户角色
export function updateUserRoleApi(data: SystemApi.UpdateUserRoleReq) {
  return requestClient.post('/permissions/user/assign', data);
}
