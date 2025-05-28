import { requestClient } from '#/api/request';

// 工单详情请求
export interface DetailInstanceReq {
  id: number;
}

// 工单列表请求
export interface ListInstanceReq {
  page: number;
  size: number;
  title?: string;
  status?: number;
  priority?: number;
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

// 创建工单请求
export interface CreateInstanceReq {
  title: string;
  template_id?: number;
  process_id: number;
  form_data: Record<string, any>;
  description?: string;
  priority?: number;
  category_id?: number;
  due_date?: string;
  tags?: string[];
  assignee_id?: number;
}

// 更新工单请求
export interface UpdateInstanceReq {
  id: number;
  title: string;
  form_data: Record<string, any>;
  description?: string;
  priority?: number;
  category_id?: number;
  due_date?: string;
  tags?: string[];
}

// 工单实例响应
export interface Instance {
  id: number;
  title: string;
  template_id?: number;
  template?: any;
  process_id: number;
  process?: any;
  form_data: Record<string, any>;
  current_step: string;
  status: number;
  priority: number;
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
  flows?: InstanceFlow[];
  comments?: InstanceComment[];
  attachments?: InstanceAttachment[];
  next_steps?: string[];
  is_overdue: boolean;
  process_data?: Record<string, any>;
}

// 我的工单请求
export interface MyInstanceReq {
  page: number;
  size: number;
  type?: 'created' | 'assigned' | 'all';
  title?: string;
  status?: number;
  priority?: number;
  category_id?: number;
  process_id?: number;
  start_date?: string;
  end_date?: string;
}

// ================= 工单流转记录相关类型 =================

export interface InstanceFlowReq {
  instance_id: number;
  action: 'approve' | 'reject' | 'transfer' | 'revoke' | 'cancel';
  comment?: string;
  form_data?: Record<string, any>;
  assignee_id?: number;
  step_id: string;
}

export interface InstanceFlow {
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

// ================= 工单评论相关类型 =================

export interface InstanceCommentReq {
  instance_id: number;
  content: string;
  parent_id?: number;
}

export interface InstanceComment {
  id: number;
  instance_id: number;
  content: string;
  creator_id: number;
  creator_name: string;
  parent_id?: number;
  is_system: boolean;
  created_at: string;
  updated_at: string;
  children?: InstanceComment[];
}

// ================= 工单附件相关类型 =================

export interface InstanceAttachment {
  id: number;
  instance_id: number;
  file_name: string;
  file_size: number;
  file_path: string;
  file_type: string;
  uploader_id: number;
  uploader_name: string;
  created_at: string;
  description: string;
  updated_at: string;
}

// 附件上传请求
export interface UploadAttachmentReq {
  instance_id: number;
  description?: string;
}

// 删除附件请求
export interface DeleteAttachmentReq {
  id: number;
}

// 创建工单实例
export async function createInstance(data: CreateInstanceReq) {
  return requestClient.post('/workorder/instance', data);
}

// 更新工单实例
export async function updateInstance(id: number, data: UpdateInstanceReq) {
  return requestClient.put(`/workorder/instance/${id}`, data);
}

// 删除工单实例
export async function deleteInstance(id: number) {
  return requestClient.delete(`/workorder/instance/${id}`);
}

// 获取工单实例详情
export async function detailInstance(id: number) {
  return requestClient.get(`/workorder/instance/${id}`);
}

// 列表查询工单实例
export async function listInstance(params: ListInstanceReq) {
  return requestClient.get('/workorder/instance', { params });
}

// 批量更新工单状态
export async function batchUpdateInstanceStatus(data: { ids: number[]; status: number }) {
  return requestClient.put('/workorder/instance/batch/status', data);
}

// 获取我的工单
export async function myInstance(params: MyInstanceReq) {
  return requestClient.get('/workorder/instance/my', { params });
}

// 获取逾期工单
export async function overdueInstance(params: any) {
  return requestClient.get('/workorder/instance/overdue', { params });
}

// 工单流程操作
export async function actionInstance(id: number, data: InstanceFlowReq) {
  return requestClient.post(`/workorder/instance/${id}/action`, data);
}

// 工单转交
export async function transferInstance(id: number, data: InstanceFlowReq) {
  return requestClient.post(`/workorder/instance/${id}/transfer`, data);
}

// 添加工单评论
export async function commentInstance(id: number, data: InstanceCommentReq) {
  return requestClient.post(`/workorder/instance/${id}/comment`, data);
}

// 获取工单评论
export async function getInstanceComments(id: number) {
  return requestClient.get(`/workorder/instance/${id}/comments`);
}

// 获取工单流程流转记录
export async function getInstanceFlows(id: number) {
  return requestClient.get(`/workorder/instance/${id}/flows`);
}

// 获取流程定义
export async function getProcessDefinition(pid: number) {
  return requestClient.get(`/workorder/instance/process/${pid}/definition`);
}

// ================= 工单附件相关接口 =================

// 上传附件
export async function uploadAttachment(id: number, data: FormData) {
  return requestClient.post(`/workorder/instance/${id}/attachment`, data, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
}

// 删除单个附件
export async function deleteAttachment(id: number, aid: number) {
  return requestClient.delete(`/workorder/instance/${id}/attachment/${aid}`);
}

// 获取工单附件列表
export async function getInstanceAttachments(id: number) {
  return requestClient.get(`/workorder/instance/${id}/attachments`);
}

// 批量删除附件
export async function batchDeleteAttachments(id: number, data: { ids: number[] }) {
  return requestClient.delete(`/workorder/instance/${id}/attachments/batch`, { data });
}
