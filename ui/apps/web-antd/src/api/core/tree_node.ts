import { requestClient } from '#/api/request';

// 树节点相关类型定义

// 树节点基本信息
export interface TreeNode {
  id: number;
  name: string;
  parentId: number;
  level: number;
  description: string;
  creatorId: number;
  status: string;
  isLeaf: boolean;
  createdAt: string;
  updatedAt: string;
}

// 树节点详细信息
export interface TreeNodeDetail {
  id: number;
  name: string;
  parentId: number;
  level: number;
  description: string;
  creatorId: number;
  status: string;
  isLeaf: boolean;
  createdAt: string;
  updatedAt: string;
  creatorName: string;
  parentName: string;
  childCount: number;
  adminUsers: string[];
  memberUsers: string[];
  resourceCount: number;
}

// 树节点列表项
export interface TreeNodeListItem {
  id: number;
  created_at: string;
  updated_at: string;
  name: string;
  parentId: number;
  level: number;
  description: string;
  creatorId: number;
  status: string;
  children: TreeNodeListItem[];
  isLeaf: boolean;
}

// 树统计信息
export interface TreeStatistics {
  totalNodes: number;     // 节点总数
  totalResources: number; // 资源总数
  totalAdmins: number;    // 管理员总数
  totalMembers: number;   // 成员总数
  activeNodes: number;    // 活跃节点数
  inactiveNodes: number;  // 非活跃节点数
}

// 节点资源信息
export interface TreeNodeResource {
  id: number;                 // 关联ID
  resourceId: string;         // 资源ID
  resourceType: string;       // 资源类型
  resourceName: string;       // 资源名称
  resourceStatus: string;     // 资源状态
  resourceCreateTime: string; // 资源创建时间
  resourceUpdateTime: string; // 资源更新时间
  resourceDeleteTime: string; // 资源删除时间
}

// 请求参数类型
export interface GetTreeListParams {
  level?: number;
  status?: 'active' | 'inactive' | 'deleted';
}

export interface CreateNodeParams {
  name: string;
  parentId?: number;
  description?: string;
  isLeaf?: boolean;
  status?: 'active' | 'inactive';
}

export interface UpdateNodeParams {
  name: string;
  parentId?: number;
  description?: string;
  status?: 'active' | 'inactive';
}

export interface UpdateNodeStatusParams {
  status: 'active' | 'inactive';
}

export interface MoveNodeParams {
  newParentId: number;
}

export interface GetNodeMembersParams {
  type?: 'admin' | 'member';
}

export interface AddNodeMemberParams {
  nodeId: number;
  userId: number;
  memberType: 'admin' | 'member';
}

export interface RemoveNodeMemberParams {
  nodeId: number;
  userId: number;
  memberType: 'admin' | 'member';
}

export interface BatchAddNodeMembersParams {
  nodeId: number;
  userIds: number[];
  memberType: 'admin' | 'member';
}

export interface BindResourceParams {
  nodeId: number;
  resourceType: string;
  resourceIds: string[];
}

export interface UnbindResourceParams {
  nodeId: number;
  resourceId: string;
  resourceType: string;
}

// API接口
// 获取树节点列表
export const getTreeList = (params?: GetTreeListParams) => {
  return requestClient.get('/tree/node/list', { params });
};

// 获取节点详情
export const getNodeDetail = (id: number) => {
  return requestClient.get(`/tree/node/detail/${id}`);
};

// 获取子节点列表
export const getChildNodes = (id: number) => {
  return requestClient.get(`/tree/node/children/${id}`);
};

// 获取树统计信息
export const getTreeStatistics = () => {
  return requestClient.get('/tree/node/statistics');
};

// 创建节点
export const createNode = (data: CreateNodeParams) => {
  return requestClient.post('/tree/node/create', data );
};

// 更新节点
export const updateNode = (id: number, data: UpdateNodeParams) => {
  return requestClient.put(`/tree/node/update/${id}`, data);
};

// 删除节点
export const deleteNode = (id: number) => {
  return requestClient.delete(`/tree/node/delete/${id}`, { data: { id: id } });
};

// 移动节点
export const moveNode = (id: number, data: MoveNodeParams) => {
  return requestClient.put(`/tree/node/move/${id}`, data);
};

// 更新节点状态
export const updateNodeStatus = (id: number, data: UpdateNodeStatusParams) => {
  return requestClient.put(`/tree/node/status/${id}`, data);
};

// 获取节点成员
export const getNodeMembers = (id: number, params?: GetNodeMembersParams) => {
  return requestClient.get(`/tree/node/members/${id}`, { params });
};

// 添加节点成员
export const addNodeMember = (data: AddNodeMemberParams) => {
  return requestClient.post('/tree/node/member/add', data);
};

// 移除节点成员
export const removeNodeMember = (data: RemoveNodeMemberParams) => {
  return requestClient.delete(`/tree/node/member/remove/${data.nodeId}`, { data: data });
};

// 获取节点资源
export const getNodeResources = (id: number) => {
  return requestClient.get(`/tree/node/resources/${id}`);
};

// 绑定资源
export const bindResource = (data: BindResourceParams) => {
  return requestClient.post('/tree/node/resource/bind', data );
};

// 解绑资源
export const unbindResource = (data: UnbindResourceParams) => {
  return requestClient.delete('/tree/node/resource/unbind', { data });
};
