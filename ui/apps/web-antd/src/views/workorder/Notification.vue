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
              <a-select-option value="feishu">飞书</a-select-option>
              <a-select-option value="email">邮箱</a-select-option>
              <a-select-option value="dingtalk">钉钉</a-select-option>
              <a-select-option value="wechat">企业微信</a-select-option>
            </a-select>
            <a-select 
              v-model:value="statusFilter" 
              placeholder="状态" 
              class="status-filter"
              @change="handleStatusChange"
              allow-clear
            >
              <a-select-option :value="undefined">全部状态</a-select-option>
              <a-select-option :value="1">启用</a-select-option>
              <a-select-option :value="0">禁用</a-select-option>
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
                  <span class="form-name-text">{{ record.formName }}</span>
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
                    :value="record.formUrl" 
                    readonly 
                    size="small"
                    class="url-input"
                  />
                  <a-button 
                    size="small" 
                    @click="copyFormUrl(record.formUrl)"
                    style="margin-left: 8px;"
                  >
                    <CopyOutlined />
                  </a-button>
                </div>
              </template>
  
              <template v-if="column.key === 'status'">
                <a-switch 
                  v-model:checked="record.status" 
                  :checked-value="1" 
                  :un-checked-value="0"
                  @change="handleStatusToggle(record)"
                />
              </template>
  
              <template v-if="column.key === 'sentCount'">
                <div class="sent-count-cell">
                  <a-statistic 
                    :value="record.sentCount" 
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
            >
              <a-select-option v-for="form in publishedForms" :key="form.id" :value="form.id">
                <div class="form-option">
                  <span class="form-name">{{ form.name }}</span>
                  <span class="form-category">{{ form.categoryName }}</span>
                </div>
              </a-select-option>
            </a-select>
          </a-form-item>
  
          <a-form-item label="通知渠道" name="channels">
            <a-checkbox-group v-model:value="notificationDialog.form.channels" style="width: 100%;">
              <a-row :gutter="[16, 16]">
                <a-col :span="12">
                  <a-checkbox value="feishu">
                    <span class="channel-option">
                      <MessageOutlined style="color: #00b96b;" />
                      飞书
                    </span>
                  </a-checkbox>
                </a-col>
                <a-col :span="12">
                  <a-checkbox value="email">
                    <span class="channel-option">
                      <MailOutlined style="color: #1890ff;" />
                      邮箱
                    </span>
                  </a-checkbox>
                </a-col>
                <a-col :span="12">
                  <a-checkbox value="dingtalk">
                    <span class="channel-option">
                      <PhoneOutlined style="color: #1677ff;" />
                      钉钉
                    </span>
                  </a-checkbox>
                </a-col>
                <a-col :span="12">
                  <a-checkbox value="wechat">
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
            >
              <a-select-option v-for="user in suggestedUsers" :key="user.value" :value="user.value">
                <div class="user-option">
                  <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(user.label) }">
                    {{ getInitials(user.label) }}
                  </a-avatar>
                  <span style="margin-left: 8px;">{{ user.label }}</span>
                </div>
              </a-select-option>
            </a-select>
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
              <a-radio value="manual">手动发送</a-radio>
              <a-radio value="immediate">表单发布后立即发送</a-radio>
              <a-radio value="scheduled">定时发送</a-radio>
            </a-radio-group>
          </a-form-item>
  
          <a-form-item 
            label="定时发送时间" 
            name="scheduledTime" 
            v-if="notificationDialog.form.triggerType === 'scheduled'"
          >
            <a-date-picker
              v-model:value="notificationDialog.form.scheduledTime"
              show-time
              placeholder="请选择发送时间"
              style="width: 100%"
            />
          </a-form-item>
  
          <a-form-item label="状态" name="status" v-if="notificationDialog.isEdit">
            <a-switch 
              v-model:checked="notificationDialog.form.status" 
              :checked-value="1" 
              :un-checked-value="0"
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
            <h2>{{ detailDialog.notification.formName }} - 通知配置</h2>
            <a-switch 
              v-model:checked="detailDialog.notification.status" 
              :checked-value="1" 
              :un-checked-value="0"
              @change="handleStatusToggle(detailDialog.notification)"
            />
          </div>
  
          <a-descriptions bordered :column="1" :labelStyle="{ width: '120px' }">
            <a-descriptions-item label="配置ID">{{ detailDialog.notification.id }}</a-descriptions-item>
            <a-descriptions-item label="关联表单">
              <a href="#" @click="viewForm(detailDialog.notification.formId)">
                {{ detailDialog.notification.formName }}
              </a>
            </a-descriptions-item>
            <a-descriptions-item label="表单链接">
              <div class="form-url-display">
                <a-input :value="detailDialog.notification.formUrl" readonly size="small" />
                <a-button size="small" @click="copyFormUrl(detailDialog.notification.formUrl)" style="margin-left: 8px;">
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
            <a-descriptions-item label="已发送次数">{{ detailDialog.notification.sentCount }}</a-descriptions-item>
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
              <template v-if="column.key === 'sentAt'">
                {{ formatFullDateTime(record.sentAt) }}
              </template>
            </template>
          </a-table>
        </div>
      </a-modal>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, reactive, onMounted, computed } from 'vue';
  import { message, Modal } from 'ant-design-vue';
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
    { title: '发送时间', dataIndex: 'sentAt', key: 'sentAt', width: 180 },
    { title: '错误信息', dataIndex: 'error', key: 'error', ellipsis: true }
  ];
  
  // 状态数据
  const loading = ref(false);
  const logsLoading = ref(false);
  const searchQuery = ref('');
  const channelFilter = ref<string | undefined>(undefined);
  const statusFilter = ref<number | undefined>(undefined);
  const notifications = ref<any[]>([]);
  const publishedForms = ref<any[]>([]);
  const suggestedUsers = ref<any[]>([]);
  const sendLogs = ref<any[]>([]);
  
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
    form: {
      id: undefined as number | undefined,
      formId: undefined as number | undefined,
      channels: [] as string[],
      recipients: [] as string[],
      messageTemplate: '您好！\n\n表单"{formName}"已发布，请点击以下链接填写：\n{formUrl}\n\n发送人：{senderName}\n发送时间：{currentTime}',
      triggerType: 'manual' as string,
      scheduledTime: undefined,
      status: 1 as number
    }
  });
  
  // 详情对话框数据
  const detailDialog = reactive({
    notification: null as any
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
  
  // 假数据
  const initializeData = () => {
    // 已发布的表单数据
    publishedForms.value = [
      { id: 1, name: '用户注册表单', categoryName: '用户管理' },
      { id: 2, name: '意见反馈表单', categoryName: '客户服务' },
      { id: 3, name: '设备申请表单', categoryName: '资产管理' },
      { id: 4, name: '请假申请表单', categoryName: '人事管理' },
      { id: 5, name: '会议室预约表单', categoryName: '行政管理' }
    ];
  
    // 建议用户数据
    suggestedUsers.value = [
      { label: '张三', value: 'zhangsan@company.com' },
      { label: '李四', value: 'lisi@company.com' },
      { label: '王五', value: 'wangwu@company.com' },
      { label: '赵六', value: '13800138000' },
      { label: '钱七', value: 'ou_7dab8a3d3cdcc9da365777c7ad535d62' }
    ];
  
    // 通知配置数据
    notifications.value = [
      {
        id: 1,
        formId: 1,
        formName: '用户注册表单',
        channels: ['feishu', 'email'],
        recipients: ['zhangsan@company.com', 'lisi@company.com'],
        formUrl: 'https://form.company.com/register?token=abc123',
        status: 1,
        sentCount: 25,
        lastSent: '2024-06-28T10:30:00',
        createdAt: '2024-06-25T09:00:00',
        messageTemplate: '您好！\n\n用户注册表单已发布，请点击以下链接填写：\n{formUrl}\n\n发送人：{senderName}\n发送时间：{currentTime}',
        triggerType: 'immediate'
      },
      {
        id: 2,
        formId: 2,
        formName: '意见反馈表单',
        channels: ['email', 'dingtalk'],
        recipients: ['feedback@company.com', '13800138001'],
        formUrl: 'https://form.company.com/feedback?token=def456',
        status: 1,
        sentCount: 12,
        lastSent: '2024-06-27T14:20:00',
        createdAt: '2024-06-20T16:30:00',
        messageTemplate: '亲爱的用户：\n\n我们需要您的宝贵意见，请填写反馈表单：\n{formUrl}\n\n谢谢配合！',
        triggerType: 'manual'
      },
      {
        id: 3,
        formId: 3,
        formName: '设备申请表单',
        channels: ['wechat'],
        recipients: ['equipment@company.com'],
        formUrl: 'https://form.company.com/equipment?token=ghi789',
        status: 0,
        sentCount: 8,
        lastSent: '2024-06-26T11:15:00',
        createdAt: '2024-06-22T10:45:00',
        messageTemplate: '设备申请表单已开放，请及时填写：{formUrl}',
        triggerType: 'scheduled'
      },
      {
        id: 4,
        formId: 4,
        formName: '请假申请表单',
        channels: ['feishu', 'email', 'dingtalk'],
        recipients: ['hr@company.com', 'manager@company.com'],
        formUrl: 'https://form.company.com/leave?token=jkl012',
        status: 1,
        sentCount: 45,
        lastSent: '2024-06-29T08:00:00',
        createdAt: '2024-06-15T14:20:00',
        messageTemplate: '请假申请表单通知：\n{formUrl}\n请在需要时填写申请。',
        triggerType: 'immediate'
      },
      {
        id: 5,
        formId: 5,
        formName: '会议室预约表单',
        channels: ['email'],
        recipients: ['admin@company.com', 'meeting@company.com'],
        formUrl: 'https://form.company.com/meeting?token=mno345',
        status: 1,
        sentCount: 18,
        lastSent: '2024-06-28T16:45:00',
        createdAt: '2024-06-18T11:30:00',
        messageTemplate: '会议室预约系统已上线：{formUrl}',
        triggerType: 'manual'
      }
    ];
  
    // 更新统计数据
    stats.total = notifications.value.length;
    stats.enabled = notifications.value.filter(n => n.status === 1).length;
    stats.disabled = notifications.value.filter(n => n.status === 0).length;
    stats.todaySent = 35; // 今日发送总数
  
    paginationConfig.total = notifications.value.length;
  };
  
  // 发送记录假数据
  const initializeSendLogs = (notificationId: number) => {
    sendLogs.value = [
      {
        id: 1,
        channel: 'feishu',
        recipient: 'zhangsan@company.com',
        status: 'success',
        sentAt: '2024-06-29T10:30:00',
        error: null
      },
      {
        id: 2,
        channel: 'email',
        recipient: 'lisi@company.com',
        status: 'success',
        sentAt: '2024-06-29T10:30:05',
        error: null
      },
      {
        id: 3,
        channel: 'dingtalk',
        recipient: '13800138001',
        status: 'failed',
        sentAt: '2024-06-29T10:30:10',
        error: '用户不存在或未关注企业应用'
      }
    ];
    logsPagination.total = sendLogs.value.length;
  };
  
  // 辅助方法
  const getChannelColor = (channel: string): string => {
    const colorMap: Record<string, string> = {
      feishu: 'green',
      email: 'blue',
      dingtalk: 'cyan',
      wechat: 'orange'
    };
    return colorMap[channel] || 'default';
  };
  
  const getChannelName = (channel: string): string => {
    const nameMap: Record<string, string> = {
      feishu: '飞书',
      email: '邮箱',
      dingtalk: '钉钉',
      wechat: '企业微信'
    };
    return nameMap[channel] || channel;
  };
  
  const getChannelIcon = (channel: string) => {
    const iconMap: Record<string, any> = {
      feishu: MessageOutlined,
      email: MailOutlined,
      dingtalk: PhoneOutlined,
      wechat: WechatOutlined
    };
    return iconMap[channel] || MessageOutlined;
  };
  
  const getTriggerTypeName = (type: string): string => {
    const typeMap: Record<string, string> = {
      manual: '手动发送',
      immediate: '表单发布后立即发送',
      scheduled: '定时发送'
    };
    return typeMap[type] || type;
  };
  
  const getRecipientsDisplay = (recipients: string[]): string => {
    if (recipients.length <= 2) {
      return recipients.join(', ');
    }
    return `${recipients.slice(0, 2).join(', ')} 等${recipients.length}人`;
  };
  
  const formatDate = (dateStr: string): string => {
    if (!dateStr) return '';
    return new Date(dateStr).toLocaleDateString('zh-CN');
  };
  
  const formatTime = (dateStr: string): string => {
    if (!dateStr) return '';
    return new Date(dateStr).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
  };
  
  const formatFullDateTime = (dateStr: string): string => {
    if (!dateStr) return '';
    return new Date(dateStr).toLocaleString('zh-CN');
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
  
  const getPreviewMessage = (notification: any): string => {
    return notification.messageTemplate
      .replace('{formName}', notification.formName)
      .replace('{formUrl}', notification.formUrl)
      .replace('{senderName}', '系统管理员')
      .replace('{currentTime}', new Date().toLocaleString('zh-CN'));
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
    // 防抖处理
    setTimeout(() => {
      paginationConfig.current = 1;
      loadNotifications();
    }, 500);
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
  
  const handleTableChange = (pagination: any): void => {
    paginationConfig.current = pagination.current;
    paginationConfig.pageSize = pagination.pageSize;
    loadNotifications();
  };
  
  // 表单过滤
  const filterFormOption = (input: string, option: any) => {
    return option.children.props.children[0].children.toLowerCase().indexOf(input.toLowerCase()) >= 0;
  };
  
  // 数据加载
  const loadNotifications = (): void => {
    loading.value = true;
    
    // 模拟API调用
    setTimeout(() => {
      let filteredData = [...notifications.value];
      
      // 搜索过滤
      if (searchQuery.value) {
        filteredData = filteredData.filter(item => 
          item.formName.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
          item.recipients.some((r: string) => r.toLowerCase().includes(searchQuery.value.toLowerCase()))
        );
      }
      
      // 渠道过滤
      if (channelFilter.value) {
        filteredData = filteredData.filter(item => 
          item.channels.includes(channelFilter.value)
        );
      }
      
      // 状态过滤
      if (statusFilter.value !== undefined) {
        filteredData = filteredData.filter(item => item.status === statusFilter.value);
      }
      
      // 分页处理
      const start = (paginationConfig.current - 1) * paginationConfig.pageSize;
      const end = start + paginationConfig.pageSize;
      notifications.value = filteredData.slice(start, end);
      paginationConfig.total = filteredData.length;
      
      loading.value = false;
    }, 300);
  };
  
  // 事件处理
  const handleCreateNotification = (): void => {
    notificationDialog.isEdit = false;
    notificationDialog.form = {
      id: undefined,
      formId: undefined,
      channels: [],
      recipients: [],
      messageTemplate: '您好！\n\n表单"{formName}"已发布，请点击以下链接填写：\n{formUrl}\n\n发送人：{senderName}\n发送时间：{currentTime}',
      triggerType: 'manual',
      scheduledTime: undefined,
      status: 1
    };
    notificationDialogVisible.value = true;
  };
  
  const handleEditNotification = (record: any): void => {
    notificationDialog.isEdit = true;
    notificationDialog.form = { ...record };
    notificationDialogVisible.value = true;
    detailDialogVisible.value = false;
  };
  
  const handleViewNotification = (record: any): void => {
    detailDialog.notification = record;
    detailDialogVisible.value = true;
  };
  
  const handleStatusToggle = (record: any): void => {
    const status = record.status === 1 ? '启用' : '禁用';
    message.success(`通知配置已${status}`);
    // 这里应该调用API更新状态
  };
  
  const handleMenuClick = (command: string, record: any): void => {
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
  
  const handleTestSend = (record: any): void => {
    Modal.confirm({
      title: '测试发送',
      content: `确定要向配置的接收人发送测试通知吗？`,
      okText: '发送',
      cancelText: '取消',
      onOk() {
        message.loading('正在发送测试通知...', 2);
        setTimeout(() => {
          message.success('测试通知发送成功');
        }, 2000);
      }
    });
  };
  
  const handleViewLogs = (record: any): void => {
    initializeSendLogs(record.id);
    logsDialogVisible.value = true;
  };
  
  const handleDuplicateNotification = (record: any): void => {
    const newRecord = { 
      ...record, 
      id: undefined, 
      formName: record.formName + ' - 副本',
      sentCount: 0,
      lastSent: null,
      createdAt: new Date().toISOString()
    };
    notificationDialog.isEdit = false;
    notificationDialog.form = newRecord;
    notificationDialogVisible.value = true;
  };
  
  const handleDeleteNotification = (record: any): void => {
    Modal.confirm({
      title: '删除确认',
      content: `确定要删除表单"${record.formName}"的通知配置吗？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk() {
        message.success('通知配置已删除');
        loadNotifications();
      }
    });
  };
  
  const saveNotification = (): void => {
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
    
    const formName = publishedForms.value.find(f => f.id === notificationDialog.form.formId)?.name || '';
    const action = notificationDialog.isEdit ? '更新' : '创建';
    
    message.success(`通知配置已${action}成功`);
    notificationDialogVisible.value = false;
    loadNotifications();
  };
  
  const viewForm = (formId: number): void => {
    message.info(`跳转到表单详情页面 (ID: ${formId})`);
    // 这里应该跳转到表单详情页面
  };
  
  // 对话框关闭
  const closeNotificationDialog = (): void => {
    notificationDialogVisible.value = false;
  };
  
  const closeDetailDialog = (): void => {
    detailDialogVisible.value = false;
  };
  
  const closeLogsDialog = (): void => {
    logsDialogVisible.value = false;
    sendLogs.value = [];
  };
  
  // 生命周期
  onMounted(() => {
    initializeData();
    loadNotifications();
  });
  </script>
  
  <style scoped>
  .notification-management-container {
    padding: 12px;
    min-height: 100vh;
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
  }
  
  .table-container {
    margin-bottom: 24px;
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
    font-family: monospace;
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
  }
  
  .form-category {
    font-size: 12px;
    color: #999;
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
  }
  
  .detail-header h2 {
    margin: 0;
    font-size: 20px;
    color: #1f2937;
    word-break: break-all;
  }
  
  .form-url-display {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
  }
  
  .form-url-display .ant-input {
    font-family: monospace;
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
  }
  
  .message-template-preview h3 {
    margin-bottom: 12px;
    color: #1f2937;
  }
  
  .template-content {
    background: #f5f5f5;
    border: 1px solid #d9d9d9;
    border-radius: 6px;
    padding: 16px;
    white-space: pre-line;
    font-family: sans-serif;
    line-height: 1.5;
  }
  
  .detail-footer {
    margin-top: 24px;
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    flex-wrap: wrap;
  }
  
  /* 发送记录对话框样式 */
  .send-logs {
    max-height: 400px;
    overflow-y: auto;
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
      padding: 4px 8px;
      min-width: auto;
    }
    
    .stats-card :deep(.ant-statistic-title) {
      font-size: 12px;
    }
    
    .stats-card :deep(.ant-statistic-content) {
      font-size: 16px;
    }
    
    .action-buttons {
      gap: 2px;
    }
    
    .action-buttons .ant-btn {
      padding: 0 4px;
      font-size: 12px;
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
  }
  
  /* 响应式表格优化 */
  .table-container :deep(.ant-table-wrapper) {
    overflow: auto;
  }
  
  .table-container :deep(.ant-table-thead > tr > th) {
    white-space: nowrap;
  }
  
  .table-container :deep(.ant-table-tbody > tr > td) {
    word-break: break-word;
  }
  
  /* 对话框响应式优化 */
  .responsive-modal :deep(.ant-modal) {
    max-width: calc(100vw - 16px);
    margin: 8px;
  }
  
  @media (max-width: 768px) {
    .responsive-modal :deep(.ant-modal-body) {
      padding: 12px;
      max-height: calc(100vh - 120px);
      overflow-y: auto;
    }
  }
  </style>