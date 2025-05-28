import { requestClient } from '#/api/request';

// ==================== 流程定义相关类型 ====================

// 流程步骤定义
export interface ProcessStep {
  id: string; // 步骤ID
  name: string; // 步骤名称
  type: string; // 步骤类型
  roles: string[]; // 角色列表
  users: number[]; // 用户ID列表
  actions: string[]; // 可执行的动作
  conditions: ProcessCondition[]; // 条件列表
  time_limit?: number; // 时间限制(分钟)
  auto_assign: boolean; // 是否自动分配
  parallel: boolean; // 是否并行处理
  props: Record<string, any>; // 步骤属性
  position: ProcessPosition; // 步骤位置
}

// 流程条件
export interface ProcessCondition {
  field: string; // 字段名
  operator: string; // 操作符
  value: any; // 条件值
}

// 流程步骤位置
export interface ProcessPosition {
  x: number;
  y: number;
}

// 流程连接
export interface ProcessConnection {
  from: string; // 来源步骤ID
  to: string; // 目标步骤ID
  condition: string; // 条件表达式
  label: string; // 连接标签
}

// 流程变量
export interface ProcessVariable {
  name: string; // 变量名
  type: string; // 变量类型
  default_value: any; // 默认值
  description: string; // 变量描述
}

// 流程定义
export interface ProcessDefinition {
  steps: ProcessStep[];
  connections: ProcessConnection[];
  variables: ProcessVariable[];
}

// ==================== 请求结构 ====================

// 创建流程请求
export interface CreateProcessReq {
  name: string;
  description?: string;
  form_design_id?: number;
  definition: ProcessDefinition;
  category_id?: number;
}

// 更新流程请求
export interface UpdateProcessReq {
  id: number;
  name: string;
  description?: string;
  form_design_id: number;
  definition: ProcessDefinition;
  category_id?: number;
}

// 删除流程请求
export interface DeleteProcessReq {
  id: number;
}

// 流程详情请求
export interface DetailProcessReq {
  id: number;
}

// 流程列表请求
export interface ListProcessReq {
  page: number;
  size: number;
  name?: string;
  category_id?: number;
  form_design_id?: number;
  status?: number;
}

// 发布流程请求
export interface PublishProcessReq {
  id: number;
}

// 克隆流程请求
export interface CloneProcessReq {
  id: number;
  name: string;
}

// ==================== 响应结构 ====================

// 流程详情响应
export interface ProcessResp {
  id: number;
  name: string;
  description: string;
  form_design_id: number;
  form_design?: any; // 可替换为FormDesign类型
  definition: ProcessDefinition;
  version: number;
  status: number;
  category_id?: number;
  category?: any; // 可替换为Category类型
  creator_id: number;
  creator_name: string;
  created_at: string;
  updated_at: string;
}

// 流程验证响应
export interface ValidateProcessResp {
  is_valid: boolean;
  errors?: string[];
}

// 流程列表项
export interface ProcessItem {
  id: number;
  name: string;
  description: string;
  form_design_id: number;
  form_design?: any; // 可替换为FormDesign类型
  version: number;
  status: number;
  category_id?: number;
  category?: any; // 可替换为Category类型
  creator_id: number;
  creator_name: string;
  created_at: string;
  updated_at: string;
}

// ==================== API接口实现 ====================

// 创建流程
export async function createProcess(data: CreateProcessReq) {
  return requestClient.post('/workorder/process/create', data);
}

// 更新流程
export async function updateProcess(data: UpdateProcessReq) {
  return requestClient.put(`/workorder/process/update/${data.id}`, data);
}

// 删除流程
export async function deleteProcess(data: DeleteProcessReq) {
  return requestClient.delete(`/workorder/process/delete/${data.id}`);
}

// 获取流程详情
export async function detailProcess(data: DetailProcessReq) {
  return requestClient.get(`/workorder/process/detail/${data.id}`);
}

// 获取流程及关联信息
export async function getProcessWithRelations(data: DetailProcessReq) {
  return requestClient.get(`/workorder/process/relations/${data.id}`);
}

// 流程列表
export async function listProcess(params: ListProcessReq) {
  return requestClient.get('/workorder/process/list', { params });
}

// 发布流程
export async function publishProcess(data: PublishProcessReq) {
  return requestClient.post(`/workorder/process/publish/${data.id}`);
}

// 克隆流程
export async function cloneProcess(data: CloneProcessReq) {
  return requestClient.post(`/workorder/process/clone/${data.id}`, data);
}

// 校验流程
export async function validateProcess(id: number) {
  return requestClient.get(`/workorder/process/validate/${id}`);
}