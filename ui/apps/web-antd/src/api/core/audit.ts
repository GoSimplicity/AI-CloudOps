import { requestClient } from '#/api/request';

// 审计日志模型
export interface AuditLog {
  id: number;
  user_id: number;
  trace_id: string;
  ip_address: string;
  user_agent: string;
  http_method: string;
  endpoint: string;
  operation_type: string;
  target_type: string;
  target_id: string;
  status_code: number;
  request_body: any;
  response_body: any;
  duration: number;
  error_msg: string;
  created_at: string;
  updated_at: string;
}

// 审计日志批量写入
export interface AuditLogBatch {
  logs: AuditLog[];
}

// 创建审计日志请求
export interface CreateAuditLogRequest {
  user_id: number;
  trace_id: string;
  ip_address: string;
  user_agent: string;
  http_method: string;
  endpoint: string;
  operation_type: string;
  target_type: string;
  target_id: string;
  status_code: number;
  request_body: any;
  response_body: any;
  duration: number;
  error_msg: string;
}

// 审计日志列表查询参数
export interface ListAuditLogsRequest {
  page: number;
  size: number;
  search?: string;
  operation_type?: string;
  target_type?: string;
  status_code?: number;
  start_time?: number;
  end_time?: number;
}

// 审计日志搜索请求
export interface SearchAuditLogsRequest extends ListAuditLogsRequest {
  advanced?: AdvancedSearchOptions;
}

// 高级搜索选项
export interface AdvancedSearchOptions {
  ip_address_list?: string[];
  status_code_list?: number[];
  duration_min?: number;
  duration_max?: number;
  has_error?: boolean;
  endpoint_pattern?: string;
}

// 审计统计信息
export interface AuditStatistics {
  total_count: number;
  today_count: number;
  error_count: number;
  avg_duration: number;
  type_distribution: TypeDistributionItem[];
  status_distribution: StatusDistributionItem[];
  recent_activity: RecentActivityItem[];
  hourly_trend: HourlyTrendItem[];
}

// 操作类型分布项
export interface TypeDistributionItem {
  type: string;
  count: number;
}

// 状态码分布项
export interface StatusDistributionItem {
  status: number;
  count: number;
}

// 最近活动项
export interface RecentActivityItem {
  time: number;
  operation_type: string;
  user_id: number;
  username: string;
  target_type: string;
  status_code: number;
  duration: number;
}

// 小时趋势项
export interface HourlyTrendItem {
  hour: number;
  count: number;
}

// 批量删除请求
export interface BatchDeleteRequest {
  ids: number[];
}

// 归档审计日志请求
export interface ArchiveAuditLogsRequest {
  start_time: number;
  end_time: number;
}

// 审计类型信息
export interface AuditTypeInfo {
  type: string;
  description: string;
  category: string;
}

// 查询相关接口
export function listAuditLogsApi(data: ListAuditLogsRequest) {
  return requestClient.get('/audit/list', { params: data });
}

export function getAuditLogDetailApi(id: number) {
  return requestClient.get(`/audit/detail/${id}`);
}

export function searchAuditLogsApi(data: SearchAuditLogsRequest) {
  return requestClient.get('/audit/search', { params: data });
}

// 统计和分析接口
export function getAuditStatisticsApi() {
  return requestClient.get('/audit/statistics');
}

export function getAuditTypesApi() {
  return requestClient.get('/audit/types');
}

// 管理接口
export function deleteAuditLogApi(id: number) {
  return requestClient.delete(`/audit/${id}`);
}

export function batchDeleteLogsApi(data: BatchDeleteRequest) {
  return requestClient.post('/audit/batch-delete', data);
}

export function archiveAuditLogsApi(data: ArchiveAuditLogsRequest) {
  return requestClient.post('/audit/archive', data);
}

// 创建接口
export function createAuditLogApi(data: CreateAuditLogRequest) {
  return requestClient.post('/audit/create', data);
}

export function batchCreateAuditLogsApi(data: AuditLogBatch) {
  return requestClient.post('/audit/batch-create', data);
}
