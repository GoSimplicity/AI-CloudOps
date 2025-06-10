import { requestClient } from '#/api/request';

export interface ResourceEcs {
  id: number;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
  instance_name: string;
  instance_id: string;
  cloud_provider: string;
  region_id: string;
  zone_id: string;
  vpc_id: string;
  status: string;
  imageName: string;
  creation_time: string;
  environment: string;
  instance_charge_type: string;
  description: string;
  tags: string[];
  security_group_ids: string[];
  private_ip_address: string[];
  public_ip_address: string[];

  // 资源创建和管理标志
  create_by_order: boolean;
  last_sync_time: string;
  tree_node_id: number;
  cpu: number;
  memory: number;
  instanceType: string;
  imageId: string;
  ipAddr: string;
  port: number;
  hostname: string;
  password: string;
  key: string;
  authMode: string; // password或key
  osType: string;
  vmType: number;
  osName: string;
  disk: number;
  networkInterfaces: string[];
  diskIds: string[];
  startTime: string;
  autoReleaseTime: string;
  ecsTreeNodes: any[];
}

export interface CreateEcsResourceReq {
  periodUnit: string; // Month 月 Year 年
  period: number;
  autoRenew: boolean; // 是否自动续费
  instanceChargeType: string; // 付费类型
  spotStrategy: string; // NoSpot 默认值 表示正常按量付费 SpotAsPriceGo 表示自动竞价
  spotDuration: number; // 竞价时长
  systemDiskSize: number; // 系统盘大小
  dataDiskSize: number; // 数据盘大小
  dataDiskCategory: string; // 数据盘类型
  dryRun: boolean; // 是否仅预览而不创建
  tags: string[];
}

export interface UpdateEcsResourceReq {
  id: number;
  provider: string;
  region: string;
  instanceId: string;
  instanceName: string;
  description: string;
  tags: string[];
  securityGroupIds: string[];
  hostname: string;
  password: string;
  treeNodeId: number;
  environment: string;
  ipAddr: string;
  port: number;
  authMode: string;
  key: string;
}

export interface ListEcsResourceReq {
  page: number;
  size: number;
  provider: string;
  region: string;
}

// ResourceECSListResp ECS资源列表响应
export interface ResourceECSListResp {
  total: number;
  data: ResourceEcs[];
}

// ResourceECSDetailResp ECS资源详情响应
export interface ResourceECSDetailResp {
  data: ResourceEcs;
}

// StartEcsReq ECS启动请求
export interface StartEcsReq {
  provider: string;
  region: string;
  instanceId: string;
}

// StopEcsReq ECS停止请求
export interface StopEcsReq {
  provider: string;
  region: string;
  instanceId: string;
}

// RestartEcsReq ECS重启请求
export interface RestartEcsReq {
  provider: string;
  region: string;
  instanceId: string;
}

// DeleteEcsReq ECS删除请求
export interface DeleteEcsReq {
  provider: string;
  region: string;
  instanceId: string;
}

// GetEcsDetailReq 获取ECS详情请求
export interface GetEcsDetailReq {
  provider: string;
  region: string;
  instanceId: string;
}

// ListInstanceOptionsReq 实例选项列表请求
export interface ListInstanceOptionsReq {
  provider: string;
  payType?: string;
  region?: string;
  zone?: string;
  instanceType?: string;
  imageId?: string;
  systemDiskCategory?: string;
  dataDiskCategory?: string;
  pageNumber?: number;
  pageSize?: number;
}

export interface ListInstanceOptionsResp {
  dataDiskCategory: string;
  systemDiskCategory: string;
  instanceType: string;
  region: string;
  zone: string;
  payType: string;
  valid: boolean;
  cpu: number;
  memory: number;
  imageId: string;
  osName: string;
  osType: string;
  architecture: string;
}

// ListVpcResourcesReq VPC资源列表查询参数
export interface ListVpcResourcesReq {
  pageNumber: number;
  pageSize: number;
  provider: string;
  region?: string;
}

// CreateVpcResourceReq VPC创建参数
export interface CreateVpcResourceReq {
  provider: string;
  region: string;
  zoneId: string;
  vpcName: string;
  description?: string;
  cidrBlock: string;
  vSwitchName: string;
  vSwitchCidrBlock: string;
  dryRun?: boolean;
  tags?: Record<string, string>;
}

// GetVpcDetailReq 获取VPC详情请求
export interface GetVpcDetailReq {
  provider: string;
  region: string;
  vpcId: string;
}

// DeleteVpcReq VPC删除请求
export interface DeleteVpcReq {
  provider: string;
  region: string;
  vpcId: string;
}

// ListVpcsReq VPC列表请求
export interface ListVpcsReq {
  provider: string;
  region: string;
}

// ListSecurityGroupsReq 安全组列表查询参数
export interface ListSecurityGroupsReq {
  pageNumber?: number;
  pageSize?: number;
  provider: string;
  region?: string;
}

// GetSecurityGroupDetailReq 获取安全组详情请求
export interface GetSecurityGroupDetailReq {
  provider: string;
  region: string;
  securityGroupId: string;
}

// DeleteSecurityGroupReq 删除安全组请求
export interface DeleteSecurityGroupReq {
  provider: string;
  region: string;
  securityGroupId: string;
}

// SecurityGroupRule 安全组规则
export interface SecurityGroupRule {
  id?: number;
  securityGroupId?: number;
  ipProtocol: string;
  portRange: string;
  direction: string;
  policy: string;
  priority: number;
  sourceCidrIp?: string;
  destCidrIp?: string;
  sourceGroupId?: string;
  destGroupId?: string;
  description?: string;
}

// CreateSecurityGroupReq 创建安全组请求
export interface CreateSecurityGroupReq {
  provider: string;
  region: string;
  securityGroupName: string;
  description?: string;
  vpcId: string;
  securityGroupType?: string;
  resourceGroupId?: string;
  treeNodeId?: number;
  securityGroupRules?: SecurityGroupRule[];
  tags?: Record<string, string>;
}

// AddSecurityGroupRuleReq 添加安全组规则请求
export interface AddSecurityGroupRuleReq {
  provider: string;
  region: string;
  securityGroupId: string;
  rule: SecurityGroupRule;
}

// RemoveSecurityGroupRuleReq 删除安全组规则请求
export interface RemoveSecurityGroupRuleReq {
  provider: string;
  region: string;
  securityGroupId: string;
  ruleId: number;
}

// ListSecurityGroupRulesReq 获取安全组规则列表请求
export interface ListSecurityGroupRulesReq {
  provider: string;
  region: string;
}

// 树节点管理
export interface TreeNode {
  id: number;
  name: string;
  parentId: number;
  level: number;
  description: string;
  creatorId: number;
  creatorName: string;
  status: string;
  isLeaf: boolean;
  childCount?: number;
  resourceCount?: number;
  parentName?: string;
  adminUsers?: string[];
  memberUsers?: string[];
  children?: TreeNode[];
  createdAt?: string;
  updatedAt?: string;
}

export interface TreeNodeCreateReq {
  name: string;
  parentId: number;
  creatorId: number;
  description: string;
  isLeaf: boolean;
  status?: string;
}

export interface TreeNodeUpdateReq {
  id: number;
  name: string;
  parentId: number;
  description: string;
  isLeaf: boolean;
  status?: string;
}

export interface TreeNodeMemberReq {
  nodeId: number;
  userId: number;
  type: string; // admin 或 member
}

export interface TreeNodeResourceBindReq {
  nodeId: number;
  resourceType: string;
  resourceIds: string[];
}

export interface TreeNodeResourceUnbindReq {
  nodeId: number;
  resourceId: string;
  resourceType: string;
}

export interface TreeNodeResp {
  id: number;
  name: string;
  parentId: number;
  level: number;
  description: string;
  creatorId: number;
  status: string;
  parentName: string;
  childCount: number;
  isLeaf: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface TreeNodeDetailReq {
  id: number;
}

export interface TreeNodeDetailResp extends TreeNodeResp {
  adminUsers: string[];
  memberUsers: string[];
  resourceCount: number;
}

export interface TreeStatisticsResp {
  totalNodes: number;
  totalResources: number;
  totalAdmins: number;
  totalMembers: number;
  activeNodes: number;
  inactiveNodes: number;
}

export interface TreeNodeResourceResp {
  id: number;
  resourceId: string;
  resourceType: string;
  resourceName: string;
  resourceStatus: string;
  resourceCreateTime: string;
  resourceUpdateTime: string;
  resourceDeleteTime: string;
}

export interface TreeNodeListReq {
  level?: number;
  status?: string;
}

export interface TreeNodeListResp {
  id: number;
  created_at: string;
  updated_at: string;
  name: string;
  parentId: number;
  level: number;
  creatorId: number;
  status: string;
  children?: TreeNodeListResp[];
  isLeaf: boolean;
}

export interface TreeNodeDeleteReq {
  id: number;
}


export function getVpcResourceList(req: ListVpcResourcesReq) {
  return requestClient.post('/tree/vpc/list', req);
}

export function createVpcResource(req: CreateVpcResourceReq) {
  return requestClient.post('/tree/vpc/create', req);
}

export function getVpcResourceDetail(req: GetVpcDetailReq) {
  return requestClient.post('/tree/vpc/detail', req);
}

export function deleteVpcResource(req: DeleteVpcReq) {
  return requestClient.delete('/tree/vpc/delete', { data: req });
}

export function getEcsResourceList(req: ListEcsResourceReq) {
  return requestClient.get('/tree/ecs/list', { params: req });
}

export function getEcsResourceDetail(req: GetEcsDetailReq) {
  return requestClient.get(`/tree/ecs/detail/${req.instanceId}`, { params: req });
}

export function createEcsResource(req: CreateEcsResourceReq) {
  return requestClient.post('/tree/ecs/create', req);
}

export function updateEcsResource(req: UpdateEcsResourceReq) {
  return requestClient.put('/tree/ecs/update', req);
}

export function startEcsResource(req: StartEcsReq) {
  return requestClient.post('/tree/ecs/start', req);
}

export function stopEcsResource(req: StopEcsReq) {
  return requestClient.post('/tree/ecs/stop', req);
}

export function restartEcsResource(req: RestartEcsReq) {
  return requestClient.post('/tree/ecs/restart', req);
}

export function deleteEcsResource(req: DeleteEcsReq) {
  return requestClient.delete(`/tree/ecs/delete/${req.instanceId}`, { data: req });
}

export function getInstanceOptions(req: ListInstanceOptionsReq) {
  return requestClient.post('/tree/ecs/instance_options', req);
}

export function createSecurityGroup(req: CreateSecurityGroupReq) {
  return requestClient.post('/tree/security_group/create', req);
}

export function deleteSecurityGroup(req: DeleteSecurityGroupReq) {
  return requestClient.delete('/tree/security_group/delete', { data: req });
}

export function listSecurityGroups(req: ListSecurityGroupsReq) {
  return requestClient.post('/tree/security_group/list', req);
}

export function getSecurityGroupDetail(req: GetSecurityGroupDetailReq) {
  return requestClient.post('/tree/security_group/detail', req);
}

export interface TreeNode {
  id: number;
  name: string;
  parentId: number;
  level: number;
  description: string;
  creatorId: number;
  status: string;
  isLeaf: boolean;
  childCount?: number;
  resourceCount?: number;
  parentName?: string;
  adminUsers?: string[];
  memberUsers?: string[];
  children?: TreeNode[];
  createdAt?: string;
  updatedAt?: string;
}

export interface TreeNodeCreateReq {
  name: string;
  parentId: number;
  creatorId: number;
  description: string;
  isLeaf: boolean;
  status?: string;
}

export interface TreeNodeUpdateReq {
  id: number;
  name: string;
  parentId: number;
  description: string;
  status?: string;
}

export interface TreeNodeMemberReq {
  nodeId: number;
  userId: number;
  type: string; // admin 或 member
}

export interface TreeNodeResp {
  id: number;
  name: string;
  parentId: number;
  level: number;
  description: string;
  creatorId: number;
  status: string;
  parentName: string;
  childCount: number;
  isLeaf: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface TreeNodeDetailReq {
  id: number;
}

export interface TreeNodeDetailResp extends TreeNodeResp {
  adminUsers: string[];
  memberUsers: string[];
  resourceCount: number;
}

export interface TreeStatisticsResp {
  totalNodes: number;
  totalResources: number;
  totalAdmins: number;
  totalMembers: number;
  activeNodes: number;
  inactiveNodes: number;
}

export interface TreeNodeListReq {
  level?: number;
  status?: string;
}

export interface TreeNodeListResp {
  id: number;
  created_at: string;
  updated_at: string;
  name: string;
  parentId: number;
  level: number;
  creatorId: number;
  status: string;
  children?: TreeNodeListResp[];
  isLeaf: boolean;
}

export interface TreeNodeDeleteReq {
  id: number;
}

// 树结构相关接口
export function getTreeList(req: TreeNodeListReq) {
  return requestClient.get('/tree/node/list', { params: req });
}

export function getNodeDetail(id: number) {
  return requestClient.get(`/tree/node/detail/${id}`);
}

export function getTreeStatistics() {
  return requestClient.get('/tree/node/statistics');
}

export function createNode(req: TreeNodeCreateReq) {
  return requestClient.post('/tree/node/create', req);
}

export function updateNode(req: TreeNodeUpdateReq) {
  return requestClient.post('/tree/node/update', req);
}

export function deleteNode(id: number) {
  return requestClient.delete(`/tree/node/delete/${id}`);
}

export function addNodeMember(req: TreeNodeMemberReq) {
  return requestClient.post('/tree/node/member/add', req);
}

export function removeNodeMember(req: TreeNodeMemberReq) {
  return requestClient.post('/tree/node/member/remove', req);
}
