<template>
  <div class="notification-management-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateNotification" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          <span class="btn-text">创建通知配置</span>
        </a-button>
        <div class="search-filters">
          <a-input-search 
            v-model:value="searchQuery" 
            placeholder="搜索通知配置..." 
            class="search-input"
            @search="handleSearch"
            @change="handleSearchChange"
            allow-clear 
          />
          <a-select 
            v-model:value="channelFilter" 
            placeholder="选择通知渠道" 
            class="channel-filter"
            @change="handleChannelChange"
            allow-clear
          >
            <a-select-option :value="undefined">全部渠道</a-select-option>
            <a-select-option :value="NotificationChannel.FEISHU">飞书</a-select-option>
            <a-select-option :value="NotificationChannel.EMAIL">邮箱</a-select-option>
            <a-select-option :value="NotificationChannel.DINGTALK">钉钉</a-select-option>
            <a-select-option :value="NotificationChannel.WECHAT">企业微信</a-select-option>
          </a-select>
          <a-select 
            v-model:value="statusFilter" 
            placeholder="状态" 
            class="status-filter"
            @change="handleStatusChange"
            allow-clear
          >
            <a-select-option :value="undefined">全部状态</a-select-option>
            <a-select-option :value="NotificationStatus.ENABLED">启用</a-select-option>
            <a-select-option :value="NotificationStatus.DISABLED">禁用</a-select-option>
          </a-select>
          <a-button @click="handleResetFilters" class="reset-btn">
            重置
          </a-button>
        </div>
      </div>
    </div>

    <div class="stats-row">
      <a-row :gutter="[16, 16]">
        <a-col :xs="12" :sm="12" :md="6" :lg="6">
          <a-card class="stats-card">
            <a-statistic title="总配置数" :value="stats.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <BellOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :xs="12" :sm="12" :md="6" :lg="6">
          <a-card class="stats-card">
            <a-statistic title="启用中" :value="stats.enabled" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :xs="12" :sm="12" :md="6" :lg="6">
          <a-card class="stats-card">
            <a-statistic title="禁用" :value="stats.disabled" :value-style="{ color: '#cf1322' }">
              <template #prefix>
                <StopOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :xs="12" :sm="12" :md="6" :lg="6">
          <a-card class="stats-card">
            <a-statistic title="今日发送" :value="stats.todaySent" :value-style="{ color: '#1890ff' }">
              <template #prefix>
                <SendOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <div class="table-container">
      <a-card>
        <a-table 
          :data-source="notifications" 
          :columns="columns" 
          :pagination="paginationConfig" 
          :loading="loading" 
          row-key="id"
          bordered
          :scroll="{ x: 1400 }"
          @change="handleTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'formName'">
              <div class="form-name-cell">
                <FormOutlined style="color: #1890ff; margin-right: 8px;" />
                <span class="form-name-text">{{ getFormName(record.formId) }}</span>
              </div>
            </template>

            <template v-if="column.key === 'channels'">
              <div class="channels-cell">
                <a-tag 
                  v-for="channel in record.channels" 
                  :key="channel"
                  :color="getChannelColor(channel)"
                  class="channel-tag"
                >
                  <component :is="getChannelIcon(channel)" style="margin-right: 4px;" />
                  {{ getChannelName(channel) }}
                </a-tag>
              </div>
            </template>

            <template v-if="column.key === 'recipients'">
              <div class="recipients-cell">
                <a-tooltip :title="record.recipients.join(', ')">
                  <span>{{ getRecipientsDisplay(record.recipients) }}</span>
                </a-tooltip>
              </div>
            </template>

            <template v-if="column.key === 'formUrl'">
              <div class="form-url-cell">
                <a-input 
                  :value="record.formUrl || generateFormUrl(record.formId)" 
                  readonly 
                  size="small"
                  class="url-input"
                />
                <a-button 
                  size="small" 
                  @click="copyFormUrl(record.formUrl || generateFormUrl(record.formId))"
                  style="margin-left: 8px;"
                >
                  <CopyOutlined />
                </a-button>
              </div>
            </template>

            <template v-if="column.key === 'status'">
              <a-switch 
                :checked="record.status === NotificationStatus.ENABLED"
                @change="(checked: boolean) => handleStatusToggle(record, checked)"
                :loading="record.statusLoading"
              />
            </template>

            <template v-if="column.key === 'sentCount'">
              <div class="sent-count-cell">
                <a-statistic 
                  :value="record.sentCount || 0" 
                  :value-style="{ fontSize: '14px' }"
                />
              </div>
            </template>

            <template v-if="column.key === 'lastSent'">
              <div class="date-info">
                <span class="date" v-if="record.lastSent">{{ formatDate(record.lastSent) }}</span>
                <span class="time" v-if="record.lastSent">{{ formatTime(record.lastSent) }}</span>
                <span v-else class="text-gray">未发送</span>
              </div>
            </template>

            <template v-if="column.key === 'createdAt'">
              <div class="date-info">
                <span class="date">{{ formatDate(record.createdAt) }}</span>
                <span class="time">{{ formatTime(record.createdAt) }}</span>
              </div>
            </template>

            <template v-if="column.key === 'action'">
              <div class="action-buttons">
                <a-button type="primary" size="small" @click="handleViewNotification(record)">
                  查看
                </a-button>
                <a-button type="default" size="small" @click="handleEditNotification(record)">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleMenuClick(e.key, record)">
                      <a-menu-item key="test">
                        <SendOutlined /> 测试发送
                      </a-menu-item>
                      <a-menu-item key="logs">
                        <FileTextOutlined /> 发送记录
                      </a-menu-item>
                      <a-menu-divider />
                      <a-menu-item key="duplicate">
                        <CopyOutlined /> 复制配置
                      </a-menu-item>
                      <a-menu-divider />
                      <a-menu-item key="delete" danger>删除</a-menu-item>
                    </a-menu>
                  </template>
                  <a-button size="small">
                    更多
                    <DownOutlined />
                  </a-button>
                </a-dropdown>
              </div>
            </template>
          </template>
        </a-table>
      </a-card>
    </div>

    <!-- 通知配置创建/编辑对话框 -->
    <a-modal 
      :open="notificationDialogVisible" 
      :title="notificationDialog.isEdit ? '编辑通知配置' : '创建通知配置'" 
      :width="notificationDialogWidth"
      @ok="saveNotification" 
      @cancel="closeNotificationDialog"
      :destroy-on-close="true"
      :confirm-loading="notificationDialog.saving"
      class="responsive-modal notification-config-modal"
    >
      <a-form ref="formRef" :model="notificationDialog.form" :rules="notificationRules" layout="vertical">
        <a-form-item label="关联表单" name="formId">
          <a-select 
            v-model:value="notificationDialog.form.formId" 
            placeholder="请选择要配置通知的表单"
            show-search
            :filter-option="filterFormOption"
            style="width: 100%"
            :options="publishedForms.map((form: any) => ({
              value: form.id,
              label: form.name,
              categoryName: form.categoryName
            }))"
            :loading="formsLoading"
            option-label-prop="label"
          >
            <template #option="{ categoryName, label }">
              <div class="form-option">
                <span class="form-name">{{ label }}</span>
                <span class="form-category">{{ categoryName }}</span>
              </div>
            </template>
          </a-select>
          <div style="margin-top: 8px; text-align: right;" v-if="formsPagination.total > formsPagination.pageSize">
            <a-pagination
              v-model:current="formsPagination.current"
              :pageSize="formsPagination.pageSize"
              :total="formsPagination.total"
              :showTotal="(total: number) => '共 ' + total + ' 条'"
              size="small"
              @change="onFormsPageChange"
              :showSizeChanger="false"
            />
          </div>
        </a-form-item>

        <a-form-item label="通知渠道" name="channels">
          <a-checkbox-group v-model:value="notificationDialog.form.channels" style="width: 100%;">
            <a-row :gutter="[16, 16]">
              <a-col :span="12">
                <a-checkbox :value="NotificationChannel.FEISHU">
                  <span class="channel-option">
                    <MessageOutlined style="color: #00b96b;" />
                    飞书
                  </span>
                </a-checkbox>
              </a-col>
              <a-col :span="12">
                <a-checkbox :value="NotificationChannel.EMAIL">
                  <span class="channel-option">
                    <MailOutlined style="color: #1890ff;" />
                    邮箱
                  </span>
                </a-checkbox>
              </a-col>
              <a-col :span="12">
                <a-checkbox :value="NotificationChannel.DINGTALK">
                  <span class="channel-option">
                    <PhoneOutlined style="color: #1677ff;" />
                    钉钉
                  </span>
                </a-checkbox>
              </a-col>
              <a-col :span="12">
                <a-checkbox :value="NotificationChannel.WECHAT">
                  <span class="channel-option">
                    <WechatOutlined style="color: #52c41a;" />
                    企业微信
                  </span>
                </a-checkbox>
              </a-col>
            </a-row>
          </a-checkbox-group>
        </a-form-item>

        <a-form-item label="通知对象" name="recipients">
          <a-select 
            v-model:value="notificationDialog.form.recipients" 
            mode="tags"
            placeholder="请输入邮箱地址、手机号或用户ID"
            style="width: 100%"
            :token-separators="[',', ';', ' ']"
            :loading="usersLoading"
            :options="suggestedUsers"
            :filter-option="false"
            @search="onUserSearch"
          >
            <template #option="{ label, value }">
              <div class="user-option">
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(label) }">
                  {{ getInitials(label) }}
                </a-avatar>
                <span style="margin-left: 8px;">{{ label }}</span>
              </div>
            </template>
          </a-select>
          <div style="margin-top: 8px; text-align: right;" v-if="usersPagination.total > usersPagination.pageSize">
            <a-pagination
              v-model:current="usersPagination.current"
              :pageSize="usersPagination.pageSize"
              :total="usersPagination.total"
              :showTotal="(total: number) => '共 ' + total + ' 条'"
              size="small"
              @change="onUsersPageChange"
              :showSizeChanger="false"
            />
          </div>
          <div class="recipients-help">
            <a-alert
              message="支持以下格式"
              description="邮箱：user@example.com | 手机号：13800138000 | 飞书用户ID：ou_xxx | 钉钉用户ID：xxx"
              type="info"
              show-icon
              banner
            />
          </div>
        </a-form-item>

        <a-form-item label="消息模板" name="messageTemplate">
          <a-textarea
            v-model:value="notificationDialog.form.messageTemplate"
            :rows="6"
            placeholder="请输入通知消息模板，支持变量：{formName}, {formUrl}, {senderName}, {currentTime}"
          />
          <div class="template-help">
            <a-alert
              message="可用变量"
              description="{formName} - 表单名称 | {formUrl} - 表单链接 | {senderName} - 发送人 | {currentTime} - 当前时间"
              type="info"
              show-icon
              banner
              style="margin-top: 8px;"
            />
          </div>
        </a-form-item>

        <a-form-item label="发送时机" name="triggerType">
          <a-radio-group v-model:value="notificationDialog.form.triggerType">
            <a-radio :value="NotificationTrigger.MANUAL">手动发送</a-radio>
            <a-radio :value="NotificationTrigger.IMMEDIATE">表单发布后立即发送</a-radio>
            <a-radio :value="NotificationTrigger.SCHEDULED">定时发送</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item 
          label="定时发送时间" 
          name="scheduledTime" 
          v-if="notificationDialog.form.triggerType === NotificationTrigger.SCHEDULED"
        >
          <a-date-picker
            v-model:value="notificationDialog.form.scheduledTime"
            show-time
            placeholder="请选择发送时间"
            style="width: 100%"
            :disabled-date="disabledDate"
          />
        </a-form-item>

        <a-form-item label="状态" name="status" v-if="notificationDialog.isEdit">
          <a-switch 
            :checked="notificationDialog.form.status === NotificationStatus.ENABLED"
            @change="(checked: boolean) => notificationDialog.form.status = checked ? NotificationStatus.ENABLED : NotificationStatus.DISABLED"
            checked-children="启用" 
            un-checked-children="禁用" 
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 通知详情对话框 -->
    <a-modal 
      :open="detailDialogVisible" 
      title="通知配置详情" 
      :width="detailDialogWidth"
      :footer="null" 
      @cancel="closeDetailDialog"
      class="detail-dialog"
    >
      <div v-if="detailDialog.notification" class="notification-details">
        <div class="detail-header">
          <h2>{{ getFormName(detailDialog.notification.formId) }} - 通知配置</h2>
          <a-switch 
            :checked="detailDialog.notification.status === NotificationStatus.ENABLED"
            @change="(checked: boolean) => handleStatusToggle(detailDialog.notification!, checked)"
          />
        </div>

        <a-descriptions bordered :column="1" :labelStyle="{ width: '120px' }">
          <a-descriptions-item label="配置ID">{{ detailDialog.notification.id }}</a-descriptions-item>
          <a-descriptions-item label="关联表单">
            <a href="#" @click="viewForm(detailDialog.notification.formId)">
              {{ getFormName(detailDialog.notification.formId) }}
            </a>
          </a-descriptions-item>
          <a-descriptions-item label="表单链接">
            <div class="form-url-display">
              <a-input :value="detailDialog.notification.formUrl || generateFormUrl(detailDialog.notification.formId)" readonly size="small" />
              <a-button size="small" @click="copyFormUrl(detailDialog.notification.formUrl || generateFormUrl(detailDialog.notification.formId))" style="margin-left: 8px;">
                <CopyOutlined />
              </a-button>
            </div>
          </a-descriptions-item>
          <a-descriptions-item label="通知渠道">
            <div class="channels-display">
              <a-tag 
                v-for="channel in detailDialog.notification.channels" 
                :key="channel"
                :color="getChannelColor(channel)"
              >
                <component :is="getChannelIcon(channel)" style="margin-right: 4px;" />
                {{ getChannelName(channel) }}
              </a-tag>
            </div>
          </a-descriptions-item>
          <a-descriptions-item label="通知对象">
            <div class="recipients-display">
              <a-tag v-for="recipient in detailDialog.notification.recipients" :key="recipient">
                {{ recipient }}
              </a-tag>
            </div>
          </a-descriptions-item>
          <a-descriptions-item label="发送时机">
            {{ getTriggerTypeName(detailDialog.notification.triggerType) }}
          </a-descriptions-item>
          <a-descriptions-item label="已发送次数">{{ detailDialog.notification.sentCount || 0 }}</a-descriptions-item>
          <a-descriptions-item label="最后发送时间">
            {{ detailDialog.notification.lastSent ? formatFullDateTime(detailDialog.notification.lastSent) : '未发送' }}
          </a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.notification.createdAt) }}</a-descriptions-item>
        </a-descriptions>

        <div class="message-template-preview">
          <h3>消息模板预览</h3>
          <div class="template-content">
            {{ getPreviewMessage(detailDialog.notification) }}
          </div>
        </div>

        <div class="detail-footer">
          <a-button @click="closeDetailDialog">关闭</a-button>
          <a-button type="default" @click="handleTestSend(detailDialog.notification)">测试发送</a-button>
          <a-button type="primary" @click="handleEditNotification(detailDialog.notification)">编辑</a-button>
        </div>
      </div>
    </a-modal>

    <!-- 发送记录对话框 -->
    <a-modal 
      :open="logsDialogVisible" 
      title="发送记录" 
      :width="logsDialogWidth"
      :footer="null" 
      @cancel="closeLogsDialog"
      class="logs-dialog"
    >
      <div class="send-logs">
        <a-table 
          :data-source="sendLogs" 
          :columns="logsColumns" 
          :pagination="logsPagination" 
          :loading="logsLoading" 
          row-key="id"
          size="small"
          @change="handleLogsTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'channel'">
              <a-tag :color="getChannelColor(record.channel)">
                <component :is="getChannelIcon(record.channel)" style="margin-right: 4px;" />
                {{ getChannelName(record.channel) }}
              </a-tag>
            </template>
            <template v-if="column.key === 'status'">
              <a-tag :color="record.status === 'success' ? 'green' : 'red'">
                {{ record.status === 'success' ? '成功' : '失败' }}
              </a-tag>
            </template>
            <template v-if="column.key === 'createdAt'">
              {{ formatFullDateTime(record.createdAt) }}
            </template>
          </template>
        </a-table>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, nextTick } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { debounce } from 'lodash-es';
import dayjs from 'dayjs';
import type { FormInstance } from 'ant-design-vue';
import {
  PlusOutlined,
  BellOutlined,
  CheckCircleOutlined,
  StopOutlined,
  SendOutlined,
  FormOutlined,
  DownOutlined,
  CopyOutlined,
  FileTextOutlined,
  MessageOutlined,
  MailOutlined,
  PhoneOutlined,
  WechatOutlined
} from '@ant-design/icons-vue';
import {
  NotificationStatus,
  NotificationChannel,
  type NotificationChannelType,
  NotificationTrigger,
  type NotificationTriggerType,
  type Notification,
  type NotificationLog,
  type CreateNotificationReq,
  type UpdateNotificationReq,
  type ListNotificationReq,
  type ListSendLogReq,
  getNotificationList,
  getNotificationDetail,
  createNotification,
  updateNotification,
  deleteNotification,
  updateNotificationStatus,
  getNotificationStats as fetchNotificationStats,
  getSendLogs,
  testSendNotification,
  duplicateNotification
} from '#/api/core/workorder_notification';
import { listFormDesign } from '#/api/core/workorder_form_design';
import { getUserList } from '#/api/core/user';

// 表单ref
const formRef = ref<FormInstance>();

// 响应式对话框宽度
const notificationDialogWidth = computed(() => {
  if (typeof window !== 'undefined') {
    const width = window.innerWidth;
    if (width < 768) return '95%';
    if (width < 1024) return '90%';
    return '800px';
  }
  return '800px';
});

const detailDialogWidth = computed(() => {
  if (typeof window !== 'undefined') {
    const width = window.innerWidth;
    if (width < 768) return '95%';
    if (width < 1024) return '90%';
    return '900px';
  }
  return '900px';
});

const logsDialogWidth = computed(() => {
  if (typeof window !== 'undefined') {
    const width = window.innerWidth;
    if (width < 768) return '95%';
    if (width < 1024) return '85%';
    return '1000px';
  }
  return '1000px';
});

// 列定义
const columns = [
  { title: '表单名称', dataIndex: 'formName', key: 'formName', width: 180, fixed: 'left' },
  { title: '通知渠道', dataIndex: 'channels', key: 'channels', width: 180 },
  { title: '通知对象', dataIndex: 'recipients', key: 'recipients', width: 160 },
  { title: '表单链接', dataIndex: 'formUrl', key: 'formUrl', width: 250 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80, align: 'center' as const },
  { title: '发送次数', dataIndex: 'sentCount', key: 'sentCount', width: 100, align: 'center' as const },
  { title: '最后发送', dataIndex: 'lastSent', key: 'lastSent', width: 140 },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 140 },
  { title: '操作', key: 'action', width: 180, align: 'center' as const, fixed: 'right' }
];

const logsColumns = [
  { title: '渠道', dataIndex: 'channel', key: 'channel', width: 100 },
  { title: '接收人', dataIndex: 'recipient', key: 'recipient', width: 150 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '发送时间', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
  { title: '错误信息', dataIndex: 'error', key: 'error', ellipsis: true }
];

// 状态数据
const loading = ref(false);
const logsLoading = ref(false);
const formsLoading = ref(false);
const usersLoading = ref(false);
const searchQuery = ref('');
const channelFilter = ref<NotificationChannelType | undefined>(undefined);
const statusFilter = ref<NotificationStatus | undefined>(undefined);
const notifications = ref<(Notification & { statusLoading?: boolean })[]>([]);
const publishedForms = ref<any[]>([]);
const suggestedUsers = ref<{label: string, value: any}[]>([]);
const sendLogs = ref<NotificationLog[]>([]);

// 分页配置
const paginationConfig = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number) => `共 ${total} 条记录`,
  size: 'default' as const
});

const logsPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  size: 'small' as const
});

// 统计数据
const stats = reactive({
  total: 0,
  enabled: 0,
  disabled: 0,
  todaySent: 0
});

// 对话框状态
const notificationDialogVisible = ref(false);
const detailDialogVisible = ref(false);
const logsDialogVisible = ref(false);

// 通知配置对话框数据
const notificationDialog = reactive({
  isEdit: false,
  saving: false,
  form: {
    id: undefined as number | undefined,
    formId: undefined as number | undefined,
    channels: [] as NotificationChannelType[],
    recipients: [] as string[],
    messageTemplate: '您好！\n\n表单"{formName}"已发布，请点击以下链接填写：\n{formUrl}\n\n发送人：{senderName}\n发送时间：{currentTime}',
    triggerType: NotificationTrigger.MANUAL as NotificationTriggerType,
    scheduledTime: undefined as any,
    status: NotificationStatus.ENABLED
  }
});

// 详情对话框数据
const detailDialog = reactive({
  notification: null as Notification | null
});

// 表单验证规则
const notificationRules = {
  formId: [
    { required: true, message: '请选择关联表单', trigger: 'change' }
  ],
  channels: [
    { required: true, type: 'array', min: 1, message: '请选择至少一个通知渠道', trigger: 'change' }
  ],
  recipients: [
    { required: true, type: 'array', min: 1, message: '请添加至少一个通知对象', trigger: 'change' }
  ],
  messageTemplate: [
    { required: true, message: '请输入消息模板', trigger: 'blur' }
  ]
};

// 防抖搜索
const debouncedSearch = debounce(() => {
  paginationConfig.current = 1;
  loadNotifications();
}, 500);

// 禁用过去日期
const disabledDate = (current: any) => {
  return current && current < dayjs().startOf('day');
};

// 获取表单名称
const getFormName = (formId?: number): string => {
  if (!formId) return '未知表单';
  const form = publishedForms.value.find(f => f.id === formId);
  return form?.name || '未知表单';
};

// 生成表单URL
const generateFormUrl = (formId?: number): string => {
  if (!formId) return '';
  return `${window.location.origin}/workorder/form/fill/${formId}`;
};

// 表单分页数据
const formsPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
});

// 用户分页数据
const usersPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
});

const userSearchQuery = ref('');

// 加载已发布表单数据
const loadPublishedForms = async (page = 1) => {
  formsLoading.value = true;
  try {
    const response = await listFormDesign({
      page: page,
      size: formsPagination.pageSize,
      status: 2
    });
    if (response) {
      publishedForms.value = response.items || [];
      formsPagination.total = response.total || 0;
      formsPagination.current = page;
    }
  } catch (error) {
    message.error('加载表单数据失败');
    console.error('Failed to load forms:', error);
  } finally {
    formsLoading.value = false;
  }
};

// 加载用户数据
const loadSuggestedUsers = async (page = 1, search = '') => {
  usersLoading.value = true;
  try {
    const response = await getUserList({
      page: page,
      size: usersPagination.pageSize,
      search: search,
      enable: 1
    });
    
    if (response && Array.isArray(response.items)) {
      suggestedUsers.value = response.items.map((user: any) => ({
        label: user.username || '未命名用户',
        value: user.id || ''
      })).filter((item: any) => item.value);
      usersPagination.total = response.total || 0;
      usersPagination.current = page;
    } else {
      suggestedUsers.value = [];
      usersPagination.total = 0;
    }
  } catch (error) {
    suggestedUsers.value = [];
    usersPagination.total = 0;
    console.error('Failed to load users:', error);
  } finally {
    usersLoading.value = false;
  }
};

// 表单分页处理函数
const onFormsPageChange = async (page: number, pageSize?: number) => {
  await nextTick();
  if (pageSize && pageSize !== formsPagination.pageSize) {
    formsPagination.pageSize = pageSize;
  }
  await loadPublishedForms(page);
};

// 用户分页处理函数  
const onUsersPageChange = async (page: number, pageSize?: number) => {
  await nextTick();
  if (pageSize && pageSize !== usersPagination.pageSize) {
    usersPagination.pageSize = pageSize;
  }
  await loadSuggestedUsers(page, userSearchQuery.value);
};

// 用户搜索处理函数
const onUserSearch = async (value: string) => {
  await nextTick();
  userSearchQuery.value = value || '';
  usersPagination.current = 1;
  await loadSuggestedUsers(1, userSearchQuery.value);
};

// 加载统计数据
const loadStats = async () => {
  try {
    const response = await fetchNotificationStats();
    if (response) {
      stats.enabled = response.enabled || 0;
      stats.disabled = response.disabled || 0;
      stats.todaySent = response.todaySent || 0;
      stats.total = stats.enabled + stats.disabled;
    }
  } catch (error) {
    console.error('Failed to load stats:', error);
  }
};

// 辅助方法
const getChannelColor = (channel: string): string => {
  const colorMap: Record<string, string> = {
    [NotificationChannel.FEISHU]: 'green',
    [NotificationChannel.EMAIL]: 'blue',
    [NotificationChannel.DINGTALK]: 'cyan',
    [NotificationChannel.WECHAT]: 'orange'
  };
  return colorMap[channel] || 'default';
};

const getChannelName = (channel: string): string => {
  const nameMap: Record<string, string> = {
    [NotificationChannel.FEISHU]: '飞书',
    [NotificationChannel.EMAIL]: '邮箱',
    [NotificationChannel.DINGTALK]: '钉钉',
    [NotificationChannel.WECHAT]: '企业微信'
  };
  return nameMap[channel] || channel;
};

const getChannelIcon = (channel: string) => {
  const iconMap: Record<string, any> = {
    [NotificationChannel.FEISHU]: MessageOutlined,
    [NotificationChannel.EMAIL]: MailOutlined,
    [NotificationChannel.DINGTALK]: PhoneOutlined,
    [NotificationChannel.WECHAT]: WechatOutlined
  };
  return iconMap[channel] || MessageOutlined;
};

const getTriggerTypeName = (type: string): string => {
  const typeMap: Record<string, string> = {
    [NotificationTrigger.MANUAL]: '手动发送',
    [NotificationTrigger.IMMEDIATE]: '表单发布后立即发送',
    [NotificationTrigger.SCHEDULED]: '定时发送'
  };
  return typeMap[type] || type;
};

const getRecipientsDisplay = (recipients: string[]): string => {
  if (recipients.length <= 2) {
    return recipients.join(', ');
  }
  return `${recipients.slice(0, 2).join(', ')} 等${recipients.length}人`;
};

const formatDate = (dateStr?: string): string => {
  if (!dateStr) return '';
  return dayjs(dateStr).format('YYYY-MM-DD');
};

const formatTime = (dateStr?: string): string => {
  if (!dateStr) return '';
  return dayjs(dateStr).format('HH:mm');
};

const formatFullDateTime = (dateStr?: string): string => {
  if (!dateStr) return '';
  return dayjs(dateStr).format('YYYY-MM-DD HH:mm:ss');
};

const getInitials = (name: string): string => {
  if (!name) return '';
  return name.slice(0, 2).toUpperCase();
};

const getAvatarColor = (name: string): string => {
  const colors = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'];
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return colors[Math.abs(hash) % colors.length]!;
};

const getPreviewMessage = (notification: Notification): string => {
  return notification.messageTemplate
    .replace('{formName}', getFormName(notification.formId))
    .replace('{formUrl}', notification.formUrl || generateFormUrl(notification.formId))
    .replace('{senderName}', '系统管理员')
    .replace('{currentTime}', dayjs().format('YYYY-MM-DD HH:mm:ss'));
};

const copyFormUrl = async (url: string) => {
  try {
    await navigator.clipboard.writeText(url);
    message.success('表单链接已复制到剪贴板');
  } catch (error) {
    message.error('复制失败，请手动复制');
  }
};

// 搜索和过滤
const handleSearch = (): void => {
  paginationConfig.current = 1;
  loadNotifications();
};

const handleSearchChange = (): void => {
  debouncedSearch();
};

const handleChannelChange = (): void => {
  paginationConfig.current = 1;
  loadNotifications();
};

const handleStatusChange = (): void => {
  paginationConfig.current = 1;
  loadNotifications();
};

const handleResetFilters = (): void => {
  searchQuery.value = '';
  channelFilter.value = undefined;
  statusFilter.value = undefined;
  paginationConfig.current = 1;
  loadNotifications();
  message.success('过滤条件已重置');
};

const handleTableChange = (pagination: any, filters: any, sorter: any): void => {
  paginationConfig.current = pagination.current;
  paginationConfig.pageSize = pagination.pageSize;
  loadNotifications();
};

// 表单过滤
const filterFormOption = (input: string, option: any) => {
  const formName = option.children?.props?.children?.[0]?.children || '';
  return formName.toLowerCase().indexOf(input.toLowerCase()) >= 0;
};

// 数据加载
const loadNotifications = async (): Promise<void> => {
  loading.value = true;
  
  try {
    const params: ListNotificationReq = {
      page: paginationConfig.current,
      size: paginationConfig.pageSize,
      search: searchQuery.value,
      channel: channelFilter.value,
      status: statusFilter.value
    };
    
    const response = await getNotificationList(params);
    if (response) {
      notifications.value = (response.items || []).map((item: any) => ({
        ...item,
        statusLoading: false
      }));
      paginationConfig.total = response.total || 0;
    }
  } catch (error) {
    message.error('加载通知配置失败');
    console.error('Failed to load notifications:', error);
  } finally {
    loading.value = false;
  }
};

// 处理发送记录表格分页变化
const handleLogsTableChange = (pagination: any): void => {
  logsPagination.current = pagination.current;
  logsPagination.pageSize = pagination.pageSize;
  if (detailDialog.notification?.id) {
    loadSendLogs(detailDialog.notification.id);
  }
};

// 加载发送记录
const loadSendLogs = async (notificationId: number): Promise<void> => {
  logsLoading.value = true;
  
  try {
    const params: ListSendLogReq = {
      page: logsPagination.current,
      size: logsPagination.pageSize,
      search: '',
      notificationId: notificationId
    };
    const response = await getSendLogs(params);
    if (response) {
      sendLogs.value = response.items || [];
      logsPagination.total = response.total || 0;
    }
  } catch (error) {
    message.error('加载发送记录失败');
    console.error('Failed to load send logs:', error);
  } finally {
    logsLoading.value = false;
  }
};

// 事件处理
const handleCreateNotification = async (): Promise<void> => {
  notificationDialog.isEdit = false;
  notificationDialog.saving = false;
  notificationDialog.form = {
    id: undefined,
    formId: undefined,
    channels: [],
    recipients: [],
    messageTemplate: '您好！\n\n表单"{formName}"已发布，请点击以下链接填写：\n{formUrl}\n\n发送人：{senderName}\n发送时间：{currentTime}',
    triggerType: NotificationTrigger.MANUAL,
    scheduledTime: undefined,
    status: NotificationStatus.ENABLED
  };
  
  // 重置分页并加载数据
  formsPagination.current = 1;
  usersPagination.current = 1;
  userSearchQuery.value = '';
  
  // 显示对话框
  notificationDialogVisible.value = true;
  
  // 在对话框显示后加载数据
  await nextTick();
  await Promise.all([
    loadPublishedForms(1),
    loadSuggestedUsers(1, '')
  ]);
};

const handleEditNotification = async (record: Notification): Promise<void> => {
  notificationDialog.isEdit = true;
  notificationDialog.saving = false;
  notificationDialog.form = { 
    id: record.id,
    formId: record.formId,
    channels: record.channels,
    recipients: record.recipients,
    messageTemplate: record.messageTemplate,
    triggerType: record.triggerType,
    scheduledTime: record.scheduledTime ? dayjs(record.scheduledTime) : undefined,
    status: record.status
  } as typeof notificationDialog.form;
  
  // 重置分页并加载数据
  formsPagination.current = 1;
  usersPagination.current = 1;
  userSearchQuery.value = '';
  
  // 关闭详情对话框并显示编辑对话框
  detailDialogVisible.value = false;
  notificationDialogVisible.value = true;
  
  // 在对话框显示后加载数据
  await nextTick();
  await Promise.all([
    loadPublishedForms(1),
    loadSuggestedUsers(1, '')
  ]);
};

const handleViewNotification = async (record: Notification): Promise<void> => {
  try {
    if (record.id) {
      const response = await getNotificationDetail(record.id);
      if (response) {
        detailDialog.notification = response;
        detailDialogVisible.value = true;
      }
    }
  } catch (error) {
    message.error('获取通知配置详情失败');
    console.error('Failed to get notification detail:', error);
  }
};

const handleStatusToggle = async (record: Notification & { statusLoading?: boolean }, checked: boolean): Promise<void> => {
  if (!record.id) {
    message.error('通知配置ID不存在');
    return;
  }

  record.statusLoading = true;
  const newStatus = checked ? NotificationStatus.ENABLED : NotificationStatus.DISABLED;
  
  try {
    await updateNotificationStatus(record.id, newStatus);
    record.status = newStatus;
    const statusText = newStatus === NotificationStatus.ENABLED ? '启用' : '禁用';
    message.success(`通知配置已${statusText}`);
    loadStats(); // 更新统计数据
  } catch (error) {
    message.error('更新状态失败');
    console.error('Failed to update status:', error);
  } finally {
    record.statusLoading = false;
  }
};

const handleMenuClick = (command: string, record: Notification): void => {
  switch (command) {
    case 'test':
      handleTestSend(record);
      break;
    case 'logs':
      handleViewLogs(record);
      break;
    case 'duplicate':
      handleDuplicateNotification(record);
      break;
    case 'delete':
      handleDeleteNotification(record);
      break;
  }
};

const handleTestSend = (record: Notification): void => {
  if (!record.id) {
    message.error('通知配置ID不存在');
    return;
  }
  
  Modal.confirm({
    title: '测试发送',
    content: `确定要向配置的接收人发送测试通知吗？`,
    okText: '发送',
    cancelText: '取消',
    onOk: async () => {
      try {
        const loadingMessage = message.loading('正在发送测试通知...', 0);
        await testSendNotification({ notificationId: record.id! });
        loadingMessage();
        message.success('测试通知发送成功');
      } catch (error) {
        message.error('测试通知发送失败');
        console.error('Failed to send test notification:', error);
      }
    }
  });
};

const handleViewLogs = (record: Notification): void => {
  if (record.id) {
    // 重置分页到第一页并加载数据
    logsPagination.current = 1;
    logsPagination.pageSize = 10;
    loadSendLogs(record.id);
    logsDialogVisible.value = true;
  } else {
    message.error('通知配置ID不存在');
  }
};

const handleDuplicateNotification = async (record: Notification): Promise<void> => {
  if (!record.id) {
    message.error('通知配置ID不存在');
    return;
  }
  
  try {
    const response = await duplicateNotification({ sourceId: record.id });
    if (response && response.id) {
      message.success('复制通知配置成功');
      loadNotifications();
      loadStats();
    }
  } catch (error) {
    message.error('复制配置失败');
    console.error('Failed to duplicate notification:', error);
  }
};

const handleDeleteNotification = (record: Notification): void => {
  if (!record.id) {
    message.error('通知配置ID不存在');
    return;
  }
  
  Modal.confirm({
    title: '删除确认',
    content: `确定要删除表单"${getFormName(record.formId)}"的通知配置吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try {
        await deleteNotification(record.id!);
        message.success('通知配置已删除');
        loadNotifications();
        loadStats(); // 更新统计数据
      } catch (error) {
        message.error('删除失败');
        console.error('Failed to delete notification:', error);
      }
    }
  });
};

const saveNotification = async (): Promise<void> => {
  try {
    await formRef.value?.validate();
  } catch (error) {
    return;
  }

  if (!notificationDialog.form.formId) {
    message.error('请选择关联表单');
    return;
  }
  
  if (notificationDialog.form.channels.length === 0) {
    message.error('请选择至少一个通知渠道');
    return;
  }
  
  if (notificationDialog.form.recipients.length === 0) {
    message.error('请添加至少一个通知对象');
    return;
  }

  notificationDialog.saving = true;
  
  try {
    const formData = {
      ...notificationDialog.form,
      scheduledTime: notificationDialog.form.scheduledTime ? 
        dayjs(notificationDialog.form.scheduledTime).format('YYYY-MM-DD HH:mm:ss') : 
        undefined,
      formUrl: generateFormUrl(notificationDialog.form.formId)
    };

    if (notificationDialog.isEdit) {
      // 更新通知配置
      if (!formData.id) {
        message.error('通知配置ID不存在');
        return;
      }
      
      const updateReq: UpdateNotificationReq = {
        id: formData.id,
        formId: formData.formId!,
        channels: formData.channels,
        recipients: formData.recipients,
        messageTemplate: formData.messageTemplate,
        triggerType: formData.triggerType,
        scheduledTime: formData.scheduledTime,
        status: formData.status,
        formUrl: formData.formUrl
      };
      
      await updateNotification(updateReq);
      message.success('通知配置已更新');
    } else {
      // 创建通知配置
      const createReq: CreateNotificationReq = {
        formId: formData.formId!,
        channels: formData.channels,
        recipients: formData.recipients,
        messageTemplate: formData.messageTemplate,
        triggerType: formData.triggerType,
        scheduledTime: formData.scheduledTime,
        formUrl: formData.formUrl
      };
      
      await createNotification(createReq);
      message.success('通知配置已创建');
    }
    
    notificationDialogVisible.value = false;
    loadNotifications();
    loadStats(); // 更新统计数据
  } catch (error) {
    message.error(notificationDialog.isEdit ? '更新失败' : '创建失败');
    console.error('Failed to save notification:', error);
  } finally {
    notificationDialog.saving = false;
  }
};

const viewForm = (formId?: number): void => {
  if (!formId) {
    message.error('表单ID不存在');
    return;
  }
  message.info(`跳转到表单详情页面 (ID: ${formId})`);
  // 这里应该跳转到表单详情页面
  // window.location.href = `/workorder/form/view/${formId}`;
};

// 对话框关闭
const closeNotificationDialog = (): void => {
  notificationDialogVisible.value = false;
  if (formRef.value) {
    formRef.value.resetFields();
  }
};

const closeDetailDialog = (): void => {
  detailDialogVisible.value = false;
  detailDialog.notification = null;
};

const closeLogsDialog = (): void => {
  logsDialogVisible.value = false;
  sendLogs.value = [];
};

// 生命周期
onMounted(async () => {
  await Promise.all([
    loadPublishedForms(1),
    loadSuggestedUsers(1, ''),
    loadNotifications(),
    loadStats()
  ]);
});
</script>

<style scoped>
.notification-management-container {
  padding: 12px;
  min-height: 100vh;
  background: #f5f5f5;
}

.page-header {
  margin-bottom: 20px;
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.btn-create {
  background: linear-gradient(135deg, #1890ff 0%);
  border: none;
  flex-shrink: 0;
  box-shadow: 0 2px 4px rgba(24, 144, 255, 0.3);
}

.btn-create:hover {
  background: linear-gradient(135deg, #40a9ff 0%);
  box-shadow: 0 4px 8px rgba(24, 144, 255, 0.4);
}

.search-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.search-input {
  width: 250px;
  min-width: 200px;
}

.channel-filter,
.status-filter {
  width: 120px;
  min-width: 100px;
}

.reset-btn {
  flex-shrink: 0;
}

.stats-row {
  margin-bottom: 20px;
}

.stats-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  height: 100%;
  transition: all 0.3s ease;
}

.stats-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.table-container {
  margin-bottom: 24px;
}

.table-container :deep(.ant-card) {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.form-name-cell {
  display: flex;
  align-items: center;
}

.form-name-text {
  font-weight: 500;
  word-break: break-all;
}

.channels-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.channel-tag {
  margin: 2px 0;
  border-radius: 4px;
  display: flex;
  align-items: center;
}

.recipients-cell {
  word-break: break-all;
}

.form-url-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.url-input {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.sent-count-cell {
  text-align: center;
}

.date-info {
  display: flex;
  flex-direction: column;
}

.date {
  font-weight: 500;
  font-size: 14px;
  color: #262626;
}

.time {
  font-size: 12px;
  color: #8c8c8c;
}

.text-gray {
  color: #999;
  font-style: italic;
}

.action-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
  flex-wrap: wrap;
}

/* 对话框样式 */
.form-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-name {
  font-weight: 500;
  color: #262626;
}

.form-category {
  font-size: 12px;
  color: #999;
  background: #f0f0f0;
  padding: 2px 6px;
  border-radius: 3px;
}

.channel-option {
  display: flex;
  align-items: center;
  gap: 6px;
}

.user-option {
  display: flex;
  align-items: center;
}

.recipients-help,
.template-help {
  margin-top: 8px;
}

.notification-config-modal :deep(.ant-modal-body) {
  max-height: 70vh;
  overflow-y: auto;
}

/* 详情对话框样式 */
.notification-details {
  margin-bottom: 20px;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 12px;
  padding: 16px;
  background: linear-gradient(135deg, #f6f9fc 0%, #ffffff 100%);
  border-radius: 8px;
  border: 1px solid #e8f4f8;
}

.detail-header h2 {
  margin: 0;
  font-size: 20px;
  color: #1f2937;
  word-break: break-all;
  font-weight: 600;
}

.form-url-display {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.form-url-display .ant-input {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.channels-display,
.recipients-display {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.message-template-preview {
  margin-top: 24px;
  border-top: 1px solid #f0f0f0;
  padding-top: 20px;
}

.message-template-preview h3 {
  margin-bottom: 12px;
  color: #1f2937;
  font-weight: 600;
}

.template-content {
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 16px;
  white-space: pre-line;
  font-family: system-ui, -apple-system, sans-serif;
  line-height: 1.6;
  color: #333;
  box-shadow: inset 0 1px 3px rgba(0, 0, 0, 0.05);
}

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-wrap: wrap;
  border-top: 1px solid #f0f0f0;
  padding-top: 16px;
}

/* 发送记录对话框样式 */
.send-logs {
  max-height: 500px;
  overflow-y: auto;
}

.logs-dialog :deep(.ant-table-tbody > tr:hover > td) {
  background: #f5f5f5;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .notification-management-container {
    padding: 8px;
  }
  
  .header-actions {
    flex-direction: column;
    align-items: stretch;
  }
  
  .search-filters {
    width: 100%;
  }
  
  .search-input,
  .channel-filter,
  .status-filter {
    width: 100%;
    min-width: auto;
  }
  
  .btn-text {
    display: none;
  }
  
  .btn-create {
    padding: 8px 12px;
    min-width: auto;
    justify-content: center;
  }
  
  .stats-card :deep(.ant-statistic-title) {
    font-size: 12px;
  }
  
  .stats-card :deep(.ant-statistic-content) {
    font-size: 16px;
  }
  
  .action-buttons {
    gap: 2px;
    flex-direction: column;
  }
  
  .action-buttons .ant-btn {
    padding: 4px 8px;
    font-size: 12px;
    min-width: 60px;
  }
  
  .form-url-cell {
    flex-direction: column;
    align-items: stretch;
    gap: 4px;
  }
  
  .channels-cell {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .detail-header {
    flex-direction: column;
    align-items: stretch;
    text-align: center;
  }
  
  .form-url-display {
    flex-direction: column;
    gap: 8px;
  }
  
  .detail-footer {
    justify-content: center;
  }
  
  .detail-footer .ant-btn {
    flex: 1;
    max-width: 120px;
  }
}

@media (max-width: 480px) {
  .channels-display,
  .recipients-display {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .template-content {
    font-size: 12px;
    padding: 12px;
  }
  
  .action-buttons .ant-btn {
    width: 100%;
  }
}

/* 响应式表格优化 */
.table-container :deep(.ant-table-wrapper) {
  overflow: auto;
  border-radius: 8px;
}

.table-container :deep(.ant-table-thead > tr > th) {
  white-space: nowrap;
  background: #fafafa;
  font-weight: 600;
}

.table-container :deep(.ant-table-tbody > tr > td) {
  word-break: break-word;
}

.table-container :deep(.ant-table-tbody > tr:hover > td) {
  background: #f8f9fa;
}

/* 对话框响应式优化 */
.responsive-modal :deep(.ant-modal) {
  max-width: calc(100vw - 16px);
  margin: 8px;
}

.responsive-modal :deep(.ant-modal-content) {
  border-radius: 8px;
  overflow: hidden;
}

@media (max-width: 768px) {
  .responsive-modal :deep(.ant-modal-body) {
    padding: 12px;
    max-height: calc(100vh - 120px);
    overflow-y: auto;
  }
  
  .responsive-modal :deep(.ant-modal-header) {
    padding: 12px 16px;
  }
  
  .responsive-modal :deep(.ant-modal-footer) {
    padding: 8px 16px;
  }
}

/* 加载状态优化 */
.table-container :deep(.ant-spin-nested-loading) {
  min-height: 200px;
}

/* 表单项样式优化 */
.notification-config-modal :deep(.ant-form-item-label) {
  font-weight: 600;
}

.notification-config-modal :deep(.ant-checkbox-wrapper) {
  margin-bottom: 8px;
}

/* 统计卡片动画 */
@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.stats-card {
  animation: fadeInUp 0.5s ease-out;
}

/* 表格加载优化 */
.table-container :deep(.ant-table-placeholder) {
  padding: 40px 20px;
}

.table-container :deep(.ant-empty-description) {
  color: #999;
}
</style>