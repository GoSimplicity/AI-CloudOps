import { requestClient } from '#/api/request';

export interface ChartItem {
  name: string;
  value: number;
}

export interface User {
  id: number;
  name: string;
}

export interface TreeNode {
  ID: number;
  title: string;
  pId: number;
  level: number;
  isLeaf: number;
  description: string;
  opsAdmins: User[];
  rdAdmins: User[];
  rdMembers: User[];
  bindEcs: ResourceEcs[];
  bindElb: ResourceElb[];
  bindRds: ResourceRds[];
  children?: TreeNode[];
  key: string;
  label: string;
  value: number;
  opsAdminUsers: string[];
  rdAdminUsers: string[];
  rdMemberUsers: string[];
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
  id: number;
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
  id: number;
  loadBalancerType: string;
  bandwidthCapacity: number;
  addressType: string;
  dnsName: string;
  bandwidthPackageId: string;
  crossZoneEnabled: boolean;
}

export interface ResourceRds {
  id: number;
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
  description: string,
  pId: number,
  isLeaf: number,
  level: number
}

export async function getAllTreeNodes() {
  return requestClient.get<TreeNode[]>('/tree/listTreeNode');
}

export async function createTreeNode(data: CreateTreeNodeReq) {
  return requestClient.post<GeneralRes>('/tree/createTreeNode', data);
}
