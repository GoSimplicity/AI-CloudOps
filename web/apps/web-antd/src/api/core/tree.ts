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
  username: string;
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
  instanceName: string;
  vmType: number;
  vendor: string;
  CreatedAt: string;
  ipAddr: string;
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
  isBound?: boolean;
  boundNodeId?: number;
  description?: string;
  tags: string[];

  // aliyun
  name: string;
  region: string;
  instance_name: string;
  instance_availability_zone: string;
  instance_type: string;
  system_disk_category: string;
  system_disk_name: string;
  system_disk_description: string;
  image_id: string;
  internet_max_bandwidth_out: number;
  vpc_name: string;
  cidr_block: string;
  vswitch_cidr: string;
  zone_id: string;
  security_group_name: string;
  security_group_description: string;
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
  title: string;
  desc: string;
  pId: number;
  isLeaf: number;
  level: number;
}

export interface updateTreeNodeReq {
  ID: number;
  title: string;
  desc: string;
  ops_admins: User[];
  rd_admins: User[];
  rd_members: User[];
}

export interface CreateECSResourceReq {
  instanceName: string;
  vendor: string;
  description: string;
  tags: string[];
  ipAddr: string;
  osName: string;
  hostname: string;
}

export interface EditECSResourceReq {
  ID: number;
  instanceName: string;
  vendor: string;
  description: string;
  tags: string[];
  ipAddr: string;
  osName: string;
  hostname: string;
}

export interface createAliECSResourcesReq {
  name: string;
  region: string;
  instance: {
    instance_availability_zone: string;
    instance_type: string;
    system_disk_category: string;
    system_disk_name: string;
    system_disk_description: string;
    image_id: string;
    instance_name: string;
    internet_max_bandwidth_out: number;
  };
  vpc: {
    vpc_name: string;
    cidr_block: string;
    vswitch_cidr: string;
    zone_id: string;
  };
  security: {
    security_group_name: string;
    security_group_description: string;
  };
  // 其他通用字段
  instanceName: string;
  description: string;
  tags: string[];
  vendor: string;
  hostname: string;
  ipAddr: string;
  osName: string;
}

export interface OtherEcsResourceReq {
  ID: number;
  name: string;
  description: string;
  region: string;
  instance_name: string;
  instance_availability_zone: string;
  instance_type: string;
  system_disk_category: string;
  system_disk_name: string;
  system_disk_description: string;
  image_id: string;
  internet_max_bandwidth_out: number;
  vpc_name: string;
  cidr_block: string;
  vswitch_cidr: string;
  zone_id: string;
  security_group_name: string;
  security_group_description: string;
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

export async function getAllECSResources() {
  return requestClient.get<ResourceEcs[]>('/tree/getEcsList');
}
export async function getAllELBResources() {
  return requestClient.get<ResourceElb[]>('/tree/getElbList');
}
export async function getAllRDSResources() {
  return requestClient.get<ResourceRds[]>('/tree/getRdsList');
}

export async function createECSResources(data: CreateECSResourceReq) {
  return requestClient.post<GeneralRes>('/tree/createEcsResource', data);
}

export async function deleteECSResources(id: number) {
  return requestClient.delete<GeneralRes>(`/tree/deleteEcsResource/${id}`);
}

export async function editECSResources(data: EditECSResourceReq) {
  return requestClient.post<GeneralRes>('/tree/updateEcsResource', data);
}

export async function bindECSResources(data: BindResourceReq) {
  return requestClient.post<GeneralRes>('/tree/bindEcs', data);
}

export async function unbindECSResources(data: BindResourceReq) {
  return requestClient.post<GeneralRes>('/tree/unBindEcs', data);
}

export async function createAliECSResources(data: createAliECSResourcesReq) {
  return requestClient.post<GeneralRes>('/tree/createAliResource', data);
}

export async function editOtherECSResources(data: OtherEcsResourceReq) {
  return requestClient.post<GeneralRes>('/tree/updateAliResource', data);
}

export async function deleteOtherECSResources(id: number) {
  return requestClient.delete<GeneralRes>(`/tree/deleteAliResource/${id}`);
}
