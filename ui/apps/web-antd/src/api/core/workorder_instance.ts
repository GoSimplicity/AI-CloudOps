import { requestClient } from '#/api/request';

// 工单实例请求类型
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

export interface UpdateInstanceReq {
  title: string;
  form_data: Record<string, any>;
  description?: string;
  priority?: number;
  category_id?: number;
  due_date?: string;
  tags?: string[];
}

export interface ListInstanceReq {
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

export interface MyInstanceReq  {
  type?: 'created' | 'assigned' | 'all';
  title?: string;
  status?: number;
  priority?: number;
  category_id?: number;
  process_id?: number;
  start_date?: string;
  end_date?: string;
}

export interface InstanceFlowReq {
  action: 'approve' | 'reject' | 'transfer' | 'revoke' | 'cancel';
  comment?: string;
  form_data?: Record<string, any>;
  assignee_id?: number;
  step_id: string;
}

export interface InstanceCommentReq {
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
  status: number;
  priority: number;
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

// ================= API接口 =================

// 创建工单实例
export async function createInstance(data: CreateInstanceReq) {
  return requestClient.post('/api/workorder/instance/create', data);
}

// 更新工单实例
export async function updateInstance(id: number, data: UpdateInstanceReq) {
  return requestClient.put(`/api/workorder/instance/update/${id}`, data);
}

// 删除工单实例
export async function deleteInstance(id: number) {
  return requestClient.delete(`/api/workorder/instance/delete/${id}`);
}

// 获取工单实例详情
export async function detailInstance(id: number) {
  return requestClient.get(`/api/workorder/instance/detail/${id}`);
}

// 列表查询工单实例
export async function listInstance(params: ListInstanceReq) {
  return requestClient.get('/api/workorder/instance/list', { params });
}

// 批量更新工单状态
export async function batchUpdateInstanceStatus(data: { ids: number[]; status: number }) {
  return requestClient.put('/api/workorder/instance/batch/status', data);
}

// 获取我的工单
export async function myInstance(params: MyInstanceReq) {
  return requestClient.get('/api/workorder/instance/my', { params });
}

// 获取逾期工单
export async function overdueInstance(params?: ListInstanceReq) {
  return requestClient.get('/api/workorder/instance/overdue', { params });
}

// 工单流程操作
export async function actionInstance(id: number, data: InstanceFlowReq) {
  return requestClient.post(`/api/workorder/instance/action/${id}`, data);
}

// 工单转交
export async function transferInstance(id: number, data: { to_user_id: number; comment?: string }) {
  return requestClient.post(`/api/workorder/instance/transfer/${id}`, data);
}

// 添加工单评论
export async function commentInstance(id: number, data: InstanceCommentReq) {
  return requestClient.post(`/api/workorder/instance/comment/${id}`, data);
}

// 获取工单评论
export async function getInstanceComments(id: number) {
  return requestClient.get(`/api/workorder/instance/comments/${id}`);
}

// 获取工单流程流转记录
export async function getInstanceFlows(id: number) {
  return requestClient.get(`/api/workorder/instance/flows/${id}`);
}

// 获取流程定义
export async function getProcessDefinition(pid: number) {
  return requestClient.get(`/api/workorder/instance/process/${pid}/definition`);
}

// ================= 工单附件相关接口 =================

// 上传附件
export async function uploadAttachment(id: number, data: FormData) {
  return requestClient.post(`/api/workorder/instance/attachment/${id}`, data, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
}

// 删除单个附件
export async function deleteAttachment(id: number, aid: number) {
  return requestClient.delete(`/api/workorder/instance/${id}/attachment/${aid}`);
}

// 获取工单附件列表
export async function getInstanceAttachments(id: number) {
  return requestClient.get(`/api/workorder/instance/attachments/${id}`);
}

// 批量删除附件
export async function batchDeleteAttachments(id: number, data: { attachment_ids: number[] }) {
  return requestClient.delete(`/api/workorder/instance/attachments/batch/${id}`, { data });
}
