import { requestClient } from '#/api/request';

export interface ChartItem {
  name: string;
  value: number;
}

export interface User {
  id: number;
  name: string;
  realName: string;
  roles: string[];
  userId: number;
  username: string
}

export interface TreeNode {
  ID: number;
  title: string;
  pId: number;
  level: number;
  isLeaf: number;
  desc: string;
  ops_admins: User[];
  rd_admins: User[];
  rd_members: User[];
  bind_ecs: ResourceEcs[];
  bind_elb: ResourceElb[];
  bind_rds: ResourceRds[];
  children?: TreeNode[];
  key: string;
  label: string;
  value: number;
  ops_admin_users: User[];
  rd_admin_users: User[];
  rd_member_users: User[];
  ecsNum: number;
  elbNum: number;
  rdsNum: number;
  nodeNum: number;
  leafNodeNum: number;
  ecsCpuTotal: number;
  elbBandWithTotal: number;
  ecsMemoryTotal: number;
  ecsDiskTotal: number;
}

export interface ResourceEcs {
  ID: number;
  osType: string;
  vmType: number;
  instanceType: string;
  cpu: number;
  memory: number;
  disk: number;
  osName: string;
  imageId: string;
  hostname: string;
  networkInterfaces: string[];
  diskIds: string[];
  startTime: string;
  autoReleaseTime: string;
  lastInvokedTime: string;
}

export interface ResourceElb {
  ID: number;
  loadBalancerType: string;
  bandwidthCapacity: number;
  addressType: string;
  dnsName: string;
  bandwidthPackageId: string;
  crossZoneEnabled: boolean;
}

export interface ResourceRds {
  ID: number;
  engine: string;
  dbInstanceNetType: string;
  dbInstanceClass: string;
  dbInstanceType: string;
  engineVersion: string;
  masterInstanceId: string;
  dbInstanceStatus: string;
  replicateId: string;
}


export interface BindResourceReq {
  nodeId: number;
  resource_ids: number[];
}

export interface GeneralRes {
  code: number;
  data: any;
  message: string;
  type: string;
}

export interface CreateTreeNodeReq {
  title: string,
  desc: string,
  pId: number,
  isLeaf: number,
  level: number
}

export interface updateTreeNodeReq {
  ID: number,
  title: string,
  desc: string,
  ops_admins: User[],
  rd_admins: User[],
  rd_members: User[],
}

export async function getAllTreeNodes() {
  return requestClient.get<TreeNode[]>('/tree/listTreeNode');
}

export async function createTreeNode(data: CreateTreeNodeReq) {
  return requestClient.post<GeneralRes>('/tree/createTreeNode', data);
}

export async function updateTreeNode(data: updateTreeNodeReq) {
  return requestClient.post<GeneralRes>('/tree/updateTreeNode', data);
}

export async function deleteTreeNode(id: number) {
  return requestClient.delete<GeneralRes>(`/tree/deleteTreeNode/${id}`);
}