import { requestClient } from '#/api/request';
  // 角色管理相关接口
  export interface Role {
    id: number;
    name: string; // 角色名称
    code: string; // 角色编码
    description: string; // 角色描述
    status: 0 | 1; // 状态 0禁用 1启用
    is_system: 0 | 1; // 是否系统角色 0否 1是
    apis?: any[]; // 关联API
    users?: any[]; // 关联用户
    created_at?: string;
    updated_at?: string;
  }

  export interface ListRolesReq {
    page: number; // 页码
    size: number; // 每页数量
    search?: string; // 搜索关键词
    status?: 0 | 1; // 状态筛选
  }

  export interface CreateRoleReq {
    name: string; // 角色名称
    code: string; // 角色编码
    description?: string; // 角色描述
    status: 0 | 1; // 状态
    api_ids?: number[]; // 关联的API ID列表
  }

  export interface UpdateRoleReq {
    id: number; // 角色ID
    name: string; // 角色名称
    code: string; // 角色编码
    description?: string; // 角色描述
    status: 0 | 1; // 状态
    api_ids?: number[]; // 关联的API ID列表
  }

  export interface DeleteRoleReq {
    id: number; // 角色ID
  }

  export interface AssignRoleApiReq {
    role_id: number; // 角色ID
    api_ids: number[]; // API ID列表
  }

  export interface RevokeRoleApiReq {
    role_id: number; // 角色ID
    api_ids: number[]; // API ID列表
  }

  export interface AssignRolesToUserReq {
    user_id: number; // 用户ID
    role_ids: number[]; // 角色ID列表
  }

  export interface RevokeRolesFromUserReq {
    user_id: number; // 用户ID
    role_ids: number[]; // 角色ID列表
  }

  export interface CheckUserPermissionReq {
    user_id: number; // 用户ID
    method: string; // 请求方法
    path: string; // 请求路径
  }

// 角色管理
export function listRolesApi(data: ListRolesReq) {
  return requestClient.post('/role/list', data);
}

export function createRoleApi(data: CreateRoleReq) {
  return requestClient.post('/role/create', data);
}

export function updateRoleApi(data: UpdateRoleReq) {
  return requestClient.post('/role/update', data);
}

export function deleteRoleApi(data: DeleteRoleReq) {
  return requestClient.post('/role/delete', data);
}

export function getRoleDetailApi(id: number) {
  return requestClient.get(`/role/detail/${id}`);
}

// 角色权限管理
export function assignApisToRoleApi(data: AssignRoleApiReq) {
  return requestClient.post('/role/assign-apis', data);
}

export function revokeApisFromRoleApi(data: RevokeRoleApiReq) {
  return requestClient.post('/role/revoke-apis', data);
}

export function getRoleApisApi(id: number) {
  return requestClient.get(`/role/apis/${id}`);
}

// 用户角色管理
export function assignRolesToUserApi(data: AssignRolesToUserReq) {
  return requestClient.post('/role/assign_users', data);
}

export function revokeRolesFromUserApi(data: RevokeRolesFromUserReq) {
  return requestClient.post('/role/revoke_users', data);
}

export function getRoleUsersApi(id: number) {
  return requestClient.get(`/role/users/${id}`);
}

export function getUserRolesApi(id: number) {
  return requestClient.get(`/role/user_roles/${id}`);
}

// 权限检查
export function checkUserPermissionApi(data: CheckUserPermissionReq) {
  return requestClient.post('/role/check_permission', data);
}

export function getUserPermissionsApi(id: number) {
  return requestClient.get(`/role/user_permissions/${id}`);
}
