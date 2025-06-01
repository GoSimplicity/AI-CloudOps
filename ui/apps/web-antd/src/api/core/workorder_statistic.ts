import { requestClient } from "#/api/request";

// ==================== 统计请求结构 ====================

// 统一的统计请求
export interface StatsReq {
  start_date?: string;
  end_date?: string;
  dimension?: 'day' | 'week' | 'month'; // 趋势统计用
  category_id?: number; // 分类筛选
  user_id?: number; // 用户筛选
  status?: string; // 状态筛选
  priority?: string; // 优先级筛选
  top?: number; // 排行榜数量
  sort_by?: 'count' | 'completion_rate' | 'avg_process_time'; // 排序字段
}

// ==================== 统计响应结构 ====================

// 概览统计
export interface OverviewStats {
  total_count: number; // 总工单数
  completed_count: number; // 已完成
  processing_count: number; // 处理中
  pending_count: number; // 待处理
  overdue_count: number; // 超时
  completion_rate: number; // 完成率
  avg_process_time: number; // 平均处理时间(小时)
  avg_response_time: number; // 平均响应时间(小时)
  today_created: number; // 今日创建
  today_completed: number; // 今日完成
}

// 趋势统计
export interface TrendStats {
  dates: string[]; // 日期列表
  created_counts: number[]; // 创建数量
  completed_counts: number[]; // 完成数量
  completion_rates: number[]; // 完成率
  avg_process_times: number[]; // 平均处理时间
}

// 分类统计
export interface CategoryStats {
  category_id: number; // 分类ID
  category_name: string; // 分类名称
  count: number; // 数量
  percentage: number; // 百分比
  completion_rate: number; // 完成率
  avg_process_time: number; // 平均处理时间
}

// 用户统计
export interface UserStats {
  user_id: number; // 用户ID
  user_name: string; // 用户名
  assigned_count: number; // 分配数量
  completed_count: number; // 完成数量
  pending_count: number; // 待处理数量
  completion_rate: number; // 完成率
  avg_response_time: number; // 平均响应时间
  avg_processing_time: number; // 平均处理时间
  overdue_count: number; // 超时数量
}

// 模板统计
export interface TemplateStats {
  template_id: number; // 模板ID
  template_name: string; // 模板名称
  category_name: string; // 分类名称
  count: number; // 使用数量
  percentage: number; // 百分比
  completion_rate: number; // 完成率
  avg_processing_time: number; // 平均处理时间
}

// 状态分布
export interface StatusDistribution {
  status: string; // 状态
  count: number; // 数量
  percentage: number; // 百分比
}

// 优先级分布
export interface PriorityDistribution {
  priority: string; // 优先级
  count: number; // 数量
  percentage: number; // 百分比
}

// 获取工单概览统计
export function getWorkorderOverview(params: StatsReq) {
  return requestClient.get<OverviewStats>('/workorder/statistics/overview', {
    params,
  });
}

// 获取工单趋势统计
export function getWorkorderTrend(params: StatsReq) {
  return requestClient.get<TrendStats>('/workorder/statistics/trend', {
    params,
  });
}

// 获取工单分类统计
export function getWorkorderCategoryStats(params: StatsReq) {
  return requestClient.get<CategoryStats[]>('/workorder/statistics/category', {
    params,
  });
}

// 获取工单用户统计
export function getWorkorderUserStats(params: StatsReq) {
  return requestClient.get<UserStats[]>('/workorder/statistics/user', {
    params,
  });
}

// 获取工单模板统计
export function getWorkorderTemplateStats(params: StatsReq) {
  return requestClient.get<TemplateStats[]>('/workorder/statistics/template', {
    params,
  });
}

// 获取工单状态分布
export function getWorkorderStatusDistribution(params: StatsReq) {
  return requestClient.get<StatusDistribution[]>('/workorder/statistics/status', {
    params,
  });
}

// 获取工单优先级分布
export function getWorkorderPriorityDistribution(params: StatsReq) {
  return requestClient.get<PriorityDistribution[]>('/workorder/statistics/priority', {
    params,
  });
}
