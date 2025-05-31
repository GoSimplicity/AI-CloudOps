import { requestClient } from '#/api/request';

// ==================== 接口请求参数类型定义 ====================

// 创建模板请求
export interface CreateTemplateReq {
  name: string;
  description?: string;
  process_id: number;
  default_values?: any;
  icon?: string;
  category_id?: number;
  sort_order?: number;
}

// 更新模板请求
export interface UpdateTemplateReq {
  id: number;
  name: string;
  description?: string;
  process_id: number;
  default_values?: any;
  icon?: string;
  category_id?: number;
  sort_order?: number;
  status?: 0 | 1;
}

// 克隆模板请求
export interface CloneTemplateReq {
  id: number;
  name: string;
}

// 模板列表请求
export interface ListTemplateReq {
  page?: number;
  size?: number;
  name?: string;
  category_id?: number;
  process_id?: number;
  status?: 0 | 1;
}

// 模板详情请求
export interface DetailTemplateReq {
  id: number;
}

// ==================== 接口响应类型定义 ====================

// 模板实体
export interface TemplateItem {
  id: number;
  name: string;
  description: string;
  process_id: number;
  default_values: string;
  icon?: string;
  status: 0 | 1;
  sort_order: number;
  category_id?: number;
  creator_id: number;
  creator_name: string;
  process?: any; 
  category?: any; 
  created_at: string;
  updated_at: string;
}

// ==================== API接口实现 ====================

// 创建模板
export async function createTemplate(data: CreateTemplateReq) {
  return requestClient.post('/workorder/template/create', data);
}

// 更新模板
export async function updateTemplate(data: UpdateTemplateReq) {
  return requestClient.put(`/workorder/template/update/${data.id}`, data);
}

// 删除模板
export async function deleteTemplate(id: number) {
  return requestClient.delete(`/workorder/template/delete/${id}`);
}

// 获取模板详情
export async function detailTemplate(id: number) {
  return requestClient.get(`/workorder/template/detail/${id}`);
}

// 模板列表
export async function listTemplate(params: ListTemplateReq) {
  return requestClient.get('/workorder/template/list', { params });
}

// 启用模板
export async function enableTemplate(id: number) {
  return requestClient.put(`/workorder/template/enable/${id}`);
}

// 禁用模板
export async function disableTemplate(id: number) {
  return requestClient.put(`/workorder/template/disable/${id}`);
}

// 克隆模板
export async function cloneTemplate(data: CloneTemplateReq) {
  return requestClient.post(`/workorder/template/clone/${data.id}`, { name: data.name });
}
