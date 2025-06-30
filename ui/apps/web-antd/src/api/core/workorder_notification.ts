import { requestClient } from '#/api/request';

// 通知状态常量
export enum NotificationStatus {
  DISABLED = 0, // 禁用
  ENABLED = 1   // 启用
}

// 通知渠道类型
export const NotificationChannel = {
  FEISHU: 'feishu',     // 飞书
  EMAIL: 'email',       // 邮箱
  DINGTALK: 'dingtalk', // 钉钉
  WECHAT: 'wechat'      // 企业微信
} as const;

export type NotificationChannelType = typeof NotificationChannel[keyof typeof NotificationChannel];

// 触发类型
export const NotificationTrigger = {
  MANUAL: 'manual',       // 手动发送
  IMMEDIATE: 'immediate', // 表单发布后立即发送
  SCHEDULED: 'scheduled'  // 定时发送
} as const;

export type NotificationTriggerType = typeof NotificationTrigger[keyof typeof NotificationTrigger];

// 通知配置模型
export interface Notification {
  id?: number;
  formId: number;
  channels: NotificationChannelType[];
  recipients: string[];
  messageTemplate: string;
  triggerType: NotificationTriggerType;
  scheduledTime?: string;
  status: NotificationStatus;
  sentCount?: number;
  lastSent?: string;
  formUrl?: string;
  creatorId?: number;
  creatorName?: string;
  createdAt?: string;
  updatedAt?: string;
}

// 通知发送记录
export interface NotificationLog {
  id?: number;
  notificationId: number;
  channel: NotificationChannelType;
  recipient: string;
  status: 'success' | 'failed';
  error?: string;
  content?: string;
  senderId: number;
  senderName?: string;
  createdAt?: string;
}

// 创建通知配置请求
export interface CreateNotificationReq {
  formId: number;
  channels: NotificationChannelType[];
  recipients: string[];
  messageTemplate: string;
  triggerType: NotificationTriggerType;
  scheduledTime?: string;
  formUrl?: string;
}

// 更新通知配置请求
export interface UpdateNotificationReq {
  id: number;
  formId: number;
  channels: NotificationChannelType[];
  recipients: string[];
  messageTemplate: string;
  triggerType: NotificationTriggerType;
  scheduledTime?: string;
  status?: NotificationStatus;
  formUrl?: string;
}

// 删除通知配置请求
export interface DeleteNotificationReq {
  id: number;
}

// 查询通知配置列表请求
export interface ListNotificationReq {
  page: number;
  size: number;
  search: string;
  formId?: number;
  channel?: NotificationChannelType;
  status?: NotificationStatus;
}

// 获取通知配置详情请求
export interface DetailNotificationReq {
  id: number;
}

// 更新通知配置状态请求
export interface UpdateStatusReq {
  id: number;
  status: NotificationStatus;
}

// 测试发送通知请求
export interface TestSendReq {
  notificationId: number;
}

// 复制通知配置请求
export interface DuplicateNotificationReq {
  sourceId: number;
  rename?: boolean;
}

// 查询发送记录请求
export interface ListSendLogReq {
  page: number;
  size: number;
  search: string;
  notificationId: number;
  channel?: NotificationChannelType;
  status?: NotificationStatus;
}

// 通知统计数据
export interface NotificationStats {
  enabled: number;   // 启用状态数量
  disabled: number;  // 禁用状态数量
  todaySent: number; // 今日发送数量
}

// 列表查询响应
export interface NotificationListResponse {
  list: Notification[];
  total: number;
}

// 发送记录查询响应
export interface SendLogListResponse {
  list: NotificationLog[];
  total: number;
}

// 获取通知配置列表
export function getNotificationList(params: ListNotificationReq) {
  return requestClient.get('/workorder/notification/list', {
    params
  });
}

// 获取通知配置详情
export function getNotificationDetail(id: number) {
  return requestClient.get(`/workorder/notification/detail/${id}`);
}

// 创建通知配置
export function createNotification(data: CreateNotificationReq) {
  return requestClient.post('/workorder/notification/create', data);
}

// 更新通知配置
export function updateNotification(data: UpdateNotificationReq) {
  return requestClient.put(`/workorder/notification/update/${data.id}`, data);
}

// 删除通知配置
export function deleteNotification(id: number) {
  return requestClient.delete(`/workorder/notification/delete/${id}`);
}

// 更新通知配置状态
export function updateNotificationStatus(id: number, status: NotificationStatus) {
  return requestClient.put(`/workorder/notification/status/${id}`, { status });
}

// 获取通知统计信息
export function getNotificationStats() {
  return requestClient.get('/workorder/notification/statistics');
}

// 获取发送记录
export function getSendLogs(params: ListSendLogReq) {
  return requestClient.get('/workorder/notification/logs', {
    params
  });
}

// 测试发送通知
export function testSendNotification(data: TestSendReq) {
  return requestClient.post('/workorder/notification/test/send', data);
}

// 复制通知配置
export function duplicateNotification(data: DuplicateNotificationReq) {
  return requestClient.post('/workorder/notification/duplicate', data);
}
