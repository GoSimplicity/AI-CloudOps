import { requestClient } from '#/api/request';

// 表单设计相关类型
export interface ListFormDesignReq {
  page: number;
  page_size: number;
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
  name: string;
  id: number;
}

export interface Field {
  type: string;
  label: string;
  field: string;
  required: boolean;
}

export interface Schema {
  fields: Field[];
}

export interface FormDesignReq {
  id?: number;
  name: string;
  description: string;
  schema: Schema;
  version?: number;
  status?: number;
  category_id?: number;
  creator_id?: number;
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
  page_size: number;
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
  page_size: number;
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

// 工单实例相关类型
export interface DetailInstanceReq {
  id: number;
}

export interface ListInstanceReq {
  page: number;
  page_size: number;
  status?: number;
  keyword?: string;
  date_range?: string[];
  creator_id?: number;
  assignee_id?: number;
  workflow_id?: number;
}

export interface FormData {
  approved_days: number;
  reason: string;
  date_range: string[];
  type: string;
}

export interface CreateInstanceReq {
  title: string;
  workflow_id: number;
  form_data: FormData;
  description?: string;
  priority?: number;
  category_id?: number;
}

export interface UpdateInstanceReq {
  id: number;
  title: string;
  form_data: FormData;
  description?: string;
  priority?: number;
  category_id?: number;
}

export interface Instance {
  id: number;
  title: string;
  workflow_id: number;
  form_data: string;
  description?: string;
  current_step: string;
  current_role: string;
  status: number;
  priority: number;
  category_id?: number;
  creator_id: number;
  creator_name: string;
  assignee_id?: number;
  assignee_name?: string;
  completed_at?: string;
  due_date?: string;
  created_at?: string;
  updated_at?: string;
  flows?: InstanceFlow[];
  comments?: InstanceComment[];
}

// 工单流转记录相关类型
export interface InstanceFlowReq {
  instance_id: number;
  action: string;
  comment?: string;
  form_data?: FormData;
}

export interface InstanceFlow {
  id: number;
  instance_id: number;
  step: string;
  action: string;
  operator_id: number;
  operator_name: string;
  comment?: string;
  form_data?: string;
  created_at?: string;
  updated_at?: string;
}

// 工单评论相关类型
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
  parent_id: number;
  created_at?: string;
  updated_at?: string;
}

// 工单分类相关类型
export interface Category {
  id: number;
  name: string;
  parent_id?: number;
  icon?: string;
  sort_order: number;
  status: number;
  description?: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
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
  return requestClient.post('/workorder/form_design/create', data);
}

export async function updateFormDesign(data: FormDesignReq) {
  return requestClient.post('/workorder/form_design/update', data);
}

export async function deleteFormDesign(data: DetailFormDesignReq) {
  return requestClient.post('/workorder/form_design/delete', data);
}

export async function listFormDesign(data: ListFormDesignReq) {
  return requestClient.post('/workorder/form_design/list', data);
}

export async function detailFormDesign(data: DetailFormDesignReq) {
  return requestClient.post('/workorder/form_design/detail', data);
}

export async function publishFormDesign(data: PublishFormDesignReq) {
  return requestClient.post('/workorder/form_design/publish', data);
}

export async function cloneFormDesign(data: CloneFormDesignReq) {
  return requestClient.post('/workorder/form_design/clone', data);
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

// 工单实例相关接口
export async function createInstance(data: CreateInstanceReq) {
  return requestClient.post('/workorder/instance/create', data);
}

export async function updateInstance(data: UpdateInstanceReq) {
  return requestClient.post('/workorder/instance/update', data);
}

export async function approveInstance(data: InstanceFlowReq) {
  return requestClient.post('/workorder/instance/approve', data);
}

export async function actionInstance(data: InstanceFlowReq) {
  return requestClient.post('/workorder/instance/action', data);
}

export async function commentInstance(data: InstanceCommentReq) {
  return requestClient.post('/workorder/instance/comment', data);
}

export async function transferInstance(data: InstanceFlowReq) {
  return requestClient.post('/workorder/instance/transfer', data);
}

export async function listInstance(data: ListInstanceReq) {
  return requestClient.post('/workorder/instance/list', data);
}

export async function detailInstance(data: DetailInstanceReq) {
  return requestClient.post('/workorder/instance/detail', data);
}

export async function deleteInstance(data: DetailInstanceReq) {
  return requestClient.delete(`/workorder/instance/delete/${data.id}`);
}

export async function myInstance(data: ListInstanceReq) {
  return requestClient.post('/workorder/instance/my', data);
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
