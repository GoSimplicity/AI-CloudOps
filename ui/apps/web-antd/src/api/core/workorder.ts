import { requestClient } from '#/api/request';
import type { Category } from '#/api/core/workorder_category'

// 表单设计相关类型
export interface ListFormDesignReq {
  page: number;
  size: number;
  category_id?: number;
  status?: number;
  search?: string;
}

export interface DetailFormDesignReq {
  id: number;
}

export interface PublishFormDesignReq {
  id: number;
}

export interface CloneFormDesignReq {
  id: number;
  name: string;
}

export interface FormFieldOption {
  label: string;
  value: any;
}

export interface FormFieldValidation {
  min_length?: number;
  max_length?: number;
  min?: number;
  max?: number;
  pattern?: string;
  message?: string;
}

export interface FormField {
  id: string;
  type: string;
  label: string;
  name: string;
  required: boolean;
  placeholder?: string;
  default_value?: any;
  options?: FormFieldOption[];
  validation?: FormFieldValidation;
  props?: Record<string, any>;
  sort_order?: number;
}

export interface FormSchema {
  fields: FormField[];
  layout?: string;
  style?: string;
}

export interface FormDesignReq {
  id?: number;
  name: string;
  description: string;
  schema: FormSchema;
  category_id?: number;
}

export interface FormDesign {
  id: number;
  name: string;
  description: string;
  schema: string;
  version: number;
  status: number;
  category_id?: number;
  creator_id: number;
  creator_name: string;
  created_at?: string;
  updated_at?: string;
}

export interface FormDesignResp {
  id: number;
  name: string;
  description: string;
  schema: FormSchema;
  version: number;
  status: number;
  category_id?: number;
  category?: Category;
  creator_id: number;
  creator_name: string;
  created_at: string;
  updated_at: string;
}

export interface FormDesignItem {
  id: number;
  name: string;
  description: string;
  version: number;
  status: number;
  category_id?: number;
  category?: Category;
  creator_id: number;
  creator_name: string;
  created_at: string;
  updated_at: string;
}

// 流程定义相关类型
export interface Step {
  step: string;
  role: string;
  action: string;
}

export interface Definition {
  steps: Step[];
}

export interface ProcessReq {
  id?: number;
  name: string;
  description: string;
  form_design_id: number;
  definition: Definition;
  version?: number;
  status?: number;
  category_id?: number;
  creator_id?: number;
  creator_name?: string;
}

export interface DeleteProcessReqReq {
  id: number;
}

export interface DetailProcessReqReq {
  id: number;
}

export interface ListProcessReq {
  page: number;
  size: number;
  status?: number;
  search?: string;
}

export interface PublishProcessReq {
  id: number;
}

export interface DetailProcessReq {
  id: number;
}
export interface DeleteProcessReq {
  id: number;
}

export interface CloneProcessReq {
  id: number;
  name: string;
}

export interface Process {
  id: number;
  name: string;
  description: string;
  form_design_id: number;
  definition: string;
  version: number;
  status: number;
  category_id?: number;
  creator_id: number;
  creator_name: string;
  created_at?: string;
  updated_at?: string;
}

// 工单模板相关类型
export interface DefaultValues {
  approver: string;
  deadline: string;
}

export interface DeleteTemplateReq {
  id: number;
}

export interface DetailTemplateReq {
  id: number;
}

export interface ListTemplateReq {
  page: number;
  size: number;
  status?: number;
  search?: string;
}

export interface TemplateReq {
  id?: number;
  name: string;
  description: string;
  process_id: number;
  default_values: DefaultValues;
  icon?: string;
  status?: number;
  sort_order?: number;
  category_id?: number;
  creator_id?: number;
  creator_name?: string;
}

export interface Template {
  id: number;
  name: string;
  description: string;
  process_id: number;
  default_values: string;
  icon?: string;
  status: number;
  sort_order: number;
  category_id?: number;
  creator_id: number;
  creator_name: string;
  created_at?: string;
  updated_at?: string;
}



// 工单统计相关类型
export interface WorkOrderStatistics {
  id: number;
  date: string;
  total_count: number;
  completed_count: number;
  processing_count: number;
  canceled_count: number;
  rejected_count: number;
  avg_process_time: number;
  category_stats?: string;
  user_stats?: string;
  created_at: string;
  updated_at: string;
}

// 用户工单处理绩效相关类型
export interface UserPerformance {
  id: number;
  user_id: number;
  user_name: string;
  department: string;
  date: string;
  assigned_count: number;
  completed_count: number;
  avg_response_time: number;
  avg_processing_time: number;
  satisfaction_score?: number;
  created_at: string;
  updated_at: string;
}

// 表单设计相关接口
export async function createFormDesign(data: FormDesignReq) {
  return requestClient.post('/workorder/form-design/create', data);
}

export async function updateFormDesign(data: FormDesignReq) {
  return requestClient.put(`/workorder/form-design/update/${data.id}`, data);
}

export async function deleteFormDesign(data: DetailFormDesignReq) {
  return requestClient.delete(`/workorder/form-design/delete/${data.id}`);
}

export async function listFormDesign(data: ListFormDesignReq) {
  return requestClient.get('/workorder/form-design/list', { params: data });
}

export async function detailFormDesign(data: DetailFormDesignReq) {
  return requestClient.get(`/workorder/form-design/detail/${data.id}`);
}

export async function publishFormDesign(data: PublishFormDesignReq) {
  return requestClient.post(`/workorder/form-design/publish/${data.id}`);
}

export async function cloneFormDesign(data: CloneFormDesignReq) {
  return requestClient.post(`/workorder/form-design/clone/${data.id}`, data);
}

export async function previewFormDesign(data: DetailFormDesignReq) {
  return requestClient.get(`/workorder/form-design/preview/${data.id}`);
}

// 流程定义相关接口
export async function createProcess(data: ProcessReq) {
  return requestClient.post('/workorder/process/create', data);
}

export async function updateProcess(data: ProcessReq) {
  return requestClient.post('/workorder/process/update', data);
}

export async function deleteProcess(data: DeleteProcessReqReq) {
  return requestClient.delete(`/workorder/process/delete/${data.id}`);
}

export async function listProcess(data: ListProcessReq) {
  return requestClient.post('/workorder/process/list', data);
}

export async function detailProcess(data: DetailProcessReqReq) {
  return requestClient.post('/workorder/process/detail', data);
}

export async function publishProcess(data: PublishProcessReq) {
  return requestClient.post('/workorder/process/publish', data);
}

export async function cloneProcess(data: CloneProcessReq) {
  return requestClient.post('/workorder/process/clone', data);
}

// 工单模板相关接口
export async function createTemplate(data: TemplateReq) {
  return requestClient.post('/workorder/template/create', data);
}

export async function updateTemplate(data: TemplateReq) {
  return requestClient.post('/workorder/template/update', data);
}

export async function deleteTemplate(data: DeleteTemplateReq) {
  return requestClient.post('/workorder/template/delete', data);
}

export async function listTemplate(data: ListTemplateReq) {
  return requestClient.post('/workorder/template/list', data);
}

export async function detailTemplate(data: DetailTemplateReq) {
  return requestClient.post('/workorder/template/detail', data);
}


export async function instanceStatistics(data: any) {
  return requestClient.post('/workorder/instance/statistics', data);
}

// 工单统计相关接口
export async function getStatisticsOverview() {
  return requestClient.post('/workorder/statistics/overview');
}

export async function getStatisticsTrend() {
  return requestClient.post('/workorder/statistics/trend');
}

export async function getStatisticsCategory() {
  return requestClient.post('/workorder/statistics/category');
}

export async function getStatisticsPerformance() {
  return requestClient.post('/workorder/statistics/performance');
}

export async function getStatisticsUser() {
  return requestClient.post('/workorder/statistics/user');
}
