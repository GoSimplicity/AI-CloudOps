import { requestClient } from '#/api/request';

// 工单状态枚举
export enum InstanceStatus {
  DRAFT = 0,      // 草稿
  PROCESSING = 1, // 处理中
  COMPLETED = 2,  // 已完成
  CANCELLED = 3,  // 已取消
  REJECTED = 4,   // 已拒绝
  PENDING = 5,    // 待处理
  OVERDUE = 6     // 已超时
}

// 优先级枚举
export enum Priority {
  LOW = 0,       // 低
  NORMAL = 1,    // 普通
  HIGH = 2,      // 高
  URGENT = 3,    // 紧急
  CRITICAL = 4   // 严重
}

// 工单实例请求类型
export interface CreateInstanceReq {
  title: string;
  template_id?: number;
  process_id: number;
  description?: string;
  priority?: Priority;
  category_id?: number;
  due_date?: string;
  tags?: string[];
  assignee_id?: number;
}

export interface UpdateInstanceReq {
  id: number;
  title: string;
  description?: string;
  priority?: Priority;
  category_id?: number;
  due_date?: string;
  tags?: string[];
}

export interface DeleteInstanceReq {
  id: number;
}

export interface DetailInstanceReq {
  id: number;
}

export interface ListInstanceReq {
  page?: number;
  size?: number;
  title?: string;
  status?: InstanceStatus;
  priority?: Priority;
  category_id?: number;
  creator_id?: number;
  assignee_id?: number;
  process_id?: number;
  template_id?: number;
  start_date?: string;
  end_date?: string;
  tags?: string[];
  overdue?: boolean;
}

export interface MyInstanceReq {
  page?: number;
  page_size?: number;
  type?: 'created' | 'assigned' | 'all';
  title?: string;
  status?: InstanceStatus;
  priority?: Priority;
  category_id?: number;
  process_id?: number;
  start_date?: string;
  end_date?: string;
}

export interface TransferInstanceReq {
  assignee_id: number;
  comment?: string;
}

export interface InstanceActionReq {
  instance_id: number;
  action: 'approve' | 'reject' | 'transfer' | 'revoke' | 'cancel';
  comment?: string;
  form_data?: Record<string, any>;
  assignee_id?: number;
  step_id: string;
}

export interface InstanceCommentReq {
  instance_id: number;
  content: string;
  parent_id?: number;
}

// 工单实例响应类型
export interface InstanceResp {
  id: number;
  title: string;
  template_id?: number;
  template?: any;
  process_id: number;
  process?: any;
  form_data: Record<string, any>;
  current_step: string;
  status: InstanceStatus;
  priority: Priority;
  category_id?: number;
  category?: any;
  creator_id: number;
  creator_name: string;
  description: string;
  assignee_id?: number;
  assignee_name?: string;
  completed_at?: string;
  due_date?: string;
  tags: string[];
  created_at: string;
  updated_at: string;
  flows?: InstanceFlowResp[];
  comments?: InstanceCommentResp[];
  attachments?: InstanceAttachmentResp[];
  next_steps?: string[];
  is_overdue: boolean;
  process_data?: Record<string, any>;
}

export interface InstanceItem {
  id: number;
  title: string;
  template_id?: number;
  template?: any;
  process_id: number;
  process?: any;
  current_step: string;
  form_data: any;
  status: InstanceStatus;
  priority: Priority;
  category_id?: number;
  category?: any;
  creator_id: number;
  creator_name: string;
  assignee_id?: number;
  assignee_name?: string;
  completed_at?: string;
  due_date?: string;
  tags: string[];
  created_at: string;
  updated_at: string;
  is_overdue: boolean;
}

// 工单流转记录类型
export interface InstanceFlowResp {
  id: number;
  instance_id: number;
  step_id: string;
  step_name: string;
  action: string;
  operator_id: number;
  operator_name: string;
  comment: string;
  form_data: Record<string, any>;
  duration?: number;
  from_step_id: string;
  to_step_id: string;
  created_at: string;
  updated_at: string;
}

// 工单评论类型
export interface InstanceCommentResp {
  id: number;
  instance_id: number;
  user_id: number;
  content: string;
  creator_id: number;
  creator_name: string;
  parent_id?: number;
  is_system: boolean;
  created_at: string;
  updated_at: string;
  children?: InstanceCommentResp[];
}

// 工单附件类型
export interface InstanceAttachmentResp {
  id: number;
  instance_id: number;
  file_name: string;
  file_size: number;
  file_path: string;
  file_type: string;
  uploader_id: number;
  uploader_name: string;
  created_at: string;
  updated_at: string;
  description: string;
}

// 附件相关请求类型
export interface UploadAttachmentReq {
  instance_id: number;
  description?: string;
}

export interface DeleteAttachmentReq {
  id: number;
}

// 创建工单
export async function createInstance(data: CreateInstanceReq) {
  return requestClient.post('/workorder/instance/create', data);
}

// 更新工单
export async function updateInstance(id: number, data: UpdateInstanceReq) {
  return requestClient.put(`/workorder/instance/update/${id}`, data);
}

// 删除工单
export async function deleteInstance(id: number) {
  return requestClient.delete(`/workorder/instance/delete/${id}`);
}

// 获取工单列表
export async function listInstance(data?: ListInstanceReq) {
  return requestClient.get('/workorder/instance/list', { params: data });
}

// 获取工单详情
export async function detailInstance(id: number) {
  return requestClient.get(`/workorder/instance/detail/${id}`);
}

// 转移工单
export async function transferInstance(id: number, data: TransferInstanceReq) {
  return requestClient.post(`/workorder/instance/transfer/${id}`, data);
}

// 获取我的工单
export async function getMyInstances(data?: MyInstanceReq) {
  return requestClient.get('/workorder/instance/my', { params: data });
}

// 获取超时工单
export async function getOverdueInstances(data?: { page?: number; page_size?: number }) {
  return requestClient.get('/workorder/instance/overdue', { params: data });
}

// 处理工单流程
export async function processInstanceFlow(id: number, data: InstanceActionReq) {
  return requestClient.post(`/workorder/instance/action/${id}`, data);
}

// 添加工单评论
export async function commentInstance(id: number, data: InstanceCommentReq) {
  return requestClient.post(`/workorder/instance/comment/${id}`, data);
}

// 获取工单评论
export async function getInstanceComments(id: number) {
  return requestClient.get(`/workorder/instance/comments/${id}`);
}

// 获取工单流转记录
export async function getInstanceFlows(id: number) {
  return requestClient.get(`/workorder/instance/flows/${id}`);
}

// 获取流程定义
export async function getProcessDefinition(processId: number) {
  return requestClient.get(`/workorder/instance/process/${processId}/definition`);
}