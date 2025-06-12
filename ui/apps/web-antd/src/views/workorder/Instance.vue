<template>
  <div class="form-instance-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateInstance" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建新工单
        </a-button>
        <a-input-search v-model:value="searchQuery" placeholder="搜索工单..." style="width: 250px" @search="handleSearch"
          allow-clear />
        <a-select v-model:value="statusFilter" placeholder="状态" style="width: 120px" @change="handleStatusChange">
          <a-select-option :value="null">全部</a-select-option>
          <a-select-option :value="0">草稿</a-select-option>
          <a-select-option :value="1">处理中</a-select-option>
          <a-select-option :value="2">已完成</a-select-option>
          <a-select-option :value="3">已取消</a-select-option>
          <a-select-option :value="4">已拒绝</a-select-option>
          <a-select-option :value="5">待处理</a-select-option>
          <a-select-option :value="6">已超时</a-select-option>
        </a-select>
        <a-range-picker 
          v-model:value="dateRange" 
          style="width: 240px" 
          @change="handleDateRangeChange" 
          :allowClear="true"
          :placeholder="['开始日期', '结束日期']"
        />
        <a-button @click="fetchMyInstances('all')">我的工单</a-button>
        <a-button @click="fetchOverdueInstances">超时工单</a-button>
      </div>
    </div>

    <div class="stats-row">
      <a-row :gutter="16">
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="总工单数" :value="statistics.total_count" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <FileOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="处理中" :value="statistics.processing_count" :value-style="{ color: '#1890ff' }">
              <template #prefix>
                <ClockCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已完成" :value="statistics.completed_count" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已拒绝" :value="statistics.rejected_count" :value-style="{ color: '#f5222d' }">
              <template #prefix>
                <CloseCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <div class="table-container">
      <a-card>
        <a-table :data-source="instances" :columns="columns" :pagination="false" :loading="loading"
          row-key="id" bordered>
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'title'">
              <div class="form-name-cell">
                <div class="form-badge" :class="getStatusClass(record.status)"></div>
                <div>
                  <div class="form-name-text">{{ record.title }}</div>
                  <div class="instance-id">#{{ record.id }}</div>
                </div>
              </div>
            </template>

            <template v-if="column.key === 'status'">
              <a-tag :color="getStatusColor(record.status)">
                {{ getStatusText(record.status) }}
              </a-tag>
            </template>

            <template v-if="column.key === 'priority'">
              <a-tag :color="getPriorityColor(record.priority)">
                {{ getPriorityText(record.priority) }}
              </a-tag>
            </template>

            <template v-if="column.key === 'creator'">
              <div class="creator-info">
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.creator_name) }">
                  {{ getInitials(record.creator_name) }}
                </a-avatar>
                <span class="creator-name">{{ record.creator_name }}</span>
              </div>
            </template>

            <template v-if="column.key === 'assignee'">
              <div class="creator-info" v-if="record.assignee_name">
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.assignee_name) }">
                  {{ getInitials(record.assignee_name) }}
                </a-avatar>
                <span class="creator-name">{{ record.assignee_name }}</span>
              </div>
              <span v-else>-</span>
            </template>

            <template v-if="column.key === 'created_at'">
              <div class="date-info">
                <span class="date">{{ formatDate(record.created_at) }}</span>
                <span class="time">{{ formatTime(record.created_at) }}</span>
              </div>
            </template>

            <template v-if="column.key === 'action'">
              <div class="action-buttons">
                <a-button type="primary" size="small" @click="handleViewInstance(record)">
                  查看
                </a-button>
                <a-button type="default" size="small" @click="handleEditInstance(record)"
                  :disabled="record.status !== 0">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleCommand(e.key, record)">
                      <a-menu-item key="approve" v-if="record.status === 1">批准</a-menu-item>
                      <a-menu-item key="reject" v-if="record.status === 1">拒绝</a-menu-item>
                      <a-menu-item key="transfer" v-if="record.status === 1">转交</a-menu-item>
                      <a-menu-item key="cancel" v-if="[0, 1, 5].includes(record.status)">取消</a-menu-item>
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

        <div class="pagination-container">
          <a-pagination 
            v-model:current="currentPage" 
            :total="totalItems" 
            :pageSize="pageSize"
            :pageSizeOptions="['10', '20', '50', '100']" 
            :showSizeChanger="true" 
            @change="handleCurrentChange"
            @showSizeChange="handleSizeChange" 
            :showTotal="(total: number) => `共 ${total} 条`" 
          />
        </div>
      </a-card>
    </div>

    <!-- 工单实例详情对话框 -->
    <a-modal v-model:visible="detailDialog.visible" title="工单详情" width="70%" :footer="null" class="detail-dialog">
      <div v-if="detailDialog.instance" class="instance-details">
        <div class="detail-header">
          <h2>{{ detailDialog.instance.title }}</h2>
          <a-tag :color="getStatusColor(detailDialog.instance.status)">
            {{ getStatusText(detailDialog.instance.status) }}
          </a-tag>
          <a-tag :color="getPriorityColor(detailDialog.instance.priority)">
            {{ getPriorityText(detailDialog.instance.priority) }}
          </a-tag>
          <a-tag v-if="detailDialog.instance.is_overdue" color="red">已逾期</a-tag>
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="工单ID">{{ detailDialog.instance.id }}</a-descriptions-item>
          <a-descriptions-item label="流程ID">{{ detailDialog.instance.process_id }}</a-descriptions-item>
          <a-descriptions-item label="当前节点">{{ detailDialog.instance.current_step }}</a-descriptions-item>
          <a-descriptions-item label="创建人">{{ detailDialog.instance.creator_name }}</a-descriptions-item>
          <a-descriptions-item label="提交时间">{{ formatFullDateTime(detailDialog.instance.created_at || '') }}</a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.assignee_name" label="处理人">
            {{ detailDialog.instance.assignee_name }}
          </a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.completed_at" label="完成时间">
            {{ formatFullDateTime(detailDialog.instance.completed_at || '') }}
          </a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.due_date" label="截止时间">
            {{ formatFullDateTime(detailDialog.instance.due_date || '') }}
          </a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.tags?.length" label="标签">
            <a-tag v-for="tag in detailDialog.instance.tags" :key="tag" color="blue">{{ tag }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.description" label="描述" :span="2">
            {{ detailDialog.instance.description }}
          </a-descriptions-item>
        </a-descriptions>

        <div class="form-data-preview">
          <h3>表单数据</h3>
          <a-collapse>
            <a-collapse-panel key="1" header="表单内容">
              <a-form layout="vertical">
                <template v-if="displayFormData && Object.keys(displayFormData).length > 0">
                  <a-form-item v-for="(value, field) in displayFormData" :key="field" :label="getFieldLabel(field)">
                    <a-input v-if="!Array.isArray(value)" v-model:value="displayFormData[field]" :disabled="true" />
                    <span v-else>{{ value.join(', ') }}</span>
                  </a-form-item>
                </template>
                <template v-else-if="formFieldDefinitions && formFieldDefinitions.length > 0">
                  <a-form-item v-for="field in formFieldDefinitions" :key="field.id" :label="field.label">
                    <a-input :value="getFieldValue(field)" :disabled="true" />
                  </a-form-item>
                </template>
                <a-alert v-else type="info" message="暂无表单数据" />
              </a-form>
            </a-collapse-panel>
          </a-collapse>
        </div>

        <div v-if="instanceFlows && instanceFlows.length > 0" class="flow-records">
          <h3>流转记录</h3>
          <a-timeline>
            <a-timeline-item v-for="flow in instanceFlows" :key="flow.id" :color="getFlowColor(flow.action)">
              <div class="flow-item">
                <div class="flow-header">
                  <span class="flow-node">{{ flow.step_name }}</span>
                  <span class="flow-action">{{ getFlowActionText(flow.action) }}</span>
                  <span class="flow-time">{{ formatFullDateTime(flow.created_at || '') }}</span>
                </div>
                <div class="flow-operator">
                  操作人: {{ flow.operator_name }}
                </div>
                <div class="flow-comment" v-if="flow.comment">
                  备注: {{ flow.comment }}
                </div>
                <div class="flow-duration" v-if="flow.duration">
                  处理时长: {{ formatDuration(flow.duration) }}
                </div>
              </div>
            </a-timeline-item>
          </a-timeline>
        </div>

        <div v-if="instanceComments && instanceComments.length > 0" class="comments-section">
          <h3>评论</h3>
          <div class="comment-list">
            <div v-for="comment in instanceComments" :key="comment.id" class="comment-item">
              <div class="comment-header">
                <a-avatar :style="{ backgroundColor: getAvatarColor(comment.creator_name) }">
                  {{ getInitials(comment.creator_name) }}
                </a-avatar>
                <div class="comment-info">
                  <div class="comment-author">{{ comment.creator_name }}</div>
                  <div class="comment-time">{{ formatFullDateTime(comment.created_at || '') }}</div>
                  <a-tag v-if="comment.is_system" color="blue" size="small">系统</a-tag>
                </div>
              </div>
              <div class="comment-content">{{ comment.content }}</div>
            </div>
          </div>
        </div>

        <div v-if="instanceAttachments && instanceAttachments.length > 0" class="attachments-section">
          <h3>附件</h3>
          <div class="attachment-list">
            <div v-for="attachment in instanceAttachments" :key="attachment.id" class="attachment-item">
              <div class="attachment-info">
                <span class="attachment-name">{{ attachment.file_name }}</span>
                <span class="attachment-size">{{ formatFileSize(attachment.file_size) }}</span>
                <span class="attachment-uploader">上传者: {{ attachment.uploader_name }}</span>
              </div>
              <div class="attachment-actions">
                <a-button size="small" type="link" @click="downloadAttachment(attachment)">下载</a-button>
                <a-button size="small" type="link" danger @click="deleteAttachmentConfirm(attachment.id)">删除</a-button>
              </div>
            </div>
          </div>
        </div>

        <div v-if="detailDialog.instance.status === 1" class="action-area">
          <a-divider orientation="left">工单处理</a-divider>
          <a-textarea v-model:value="processingComment" :rows="3" placeholder="请输入处理意见..." />
          <div class="action-buttons mt-16">
            <a-button type="primary" @click="processInstance(detailDialog.instance, 'approve')">
              批准
            </a-button>
            <a-button danger @click="processInstance(detailDialog.instance, 'reject')">
              拒绝
            </a-button>
            <a-button @click="showTransferDialog">
              转交
            </a-button>
          </div>
        </div>

        <div class="action-area" v-if="detailDialog.instance.status !== 0">
          <a-divider orientation="left">添加评论</a-divider>
          <a-textarea v-model:value="newComment" :rows="3" placeholder="请输入评论..." />
          <div class="action-buttons mt-16">
            <a-button type="primary" @click="addComment">
              提交评论
            </a-button>
          </div>
        </div>

        <div class="action-area">
          <a-divider orientation="left">上传附件</a-divider>
          <a-upload
            :before-upload="beforeUpload"
            :file-list="uploadFileList"
            @change="handleUploadChange"
            multiple
          >
            <a-button>
              <UploadOutlined />
              选择文件
            </a-button>
          </a-upload>
          <a-button type="primary" @click="uploadInstanceAttachment" :disabled="uploadFileList.length === 0" class="mt-8">
            上传附件
          </a-button>
        </div>

        <div class="detail-footer">
          <a-button @click="detailDialog.visible = false">关闭</a-button>
          <a-button v-if="detailDialog.instance.status === 0" type="primary"
            @click="handleEditInstance(detailDialog.instance)">
            编辑
          </a-button>
        </div>
      </div>
    </a-modal>

    <!-- 创建/编辑工单实例对话框 -->
    <a-modal 
      v-model:visible="instanceDialog.visible" 
      :title="instanceDialog.isEdit ? '编辑工单' : '创建工单'" 
      width="760px"
      @ok="saveInstance" 
      :destroy-on-close="false"
    >
      <!-- 步骤1：选择流程 -->
      <div v-if="!selectedProcess && !instanceDialog.isEdit" class="process-selection">
        <a-form-item label="标题" required>
          <a-input v-model:value="newInstance.title" placeholder="请输入工单标题" />
        </a-form-item>
        
        <a-form-item label="选择流程" required>
          <a-select 
            v-model:value="newInstance.process_id" 
            placeholder="请选择流程" 
            style="width: 100%" 
            @change="handleSelectProcess"
          >
            <a-select-option v-for="process in processes" :key="process.id" :value="process.id">
              {{ process.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item label="模板">
          <a-select 
            v-model:value="newInstance.template_id" 
            placeholder="请选择模板(可选)" 
            style="width: 100%" 
            allow-clear
          >
            <a-select-option v-for="template in templates" :key="template.id" :value="template.id">
              {{ template.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item label="分类">
          <a-select 
            v-model:value="newInstance.category_id" 
            placeholder="请选择分类(可选)" 
            style="width: 100%" 
            allow-clear
          >
            <a-select-option v-for="category in categories" :key="category.id" :value="category.id">
              {{ category.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item label="优先级" required>
          <a-select v-model:value="newInstance.priority" placeholder="请选择优先级" style="width: 100%">
            <a-select-option :value="0">低</a-select-option>
            <a-select-option :value="1">普通</a-select-option>
            <a-select-option :value="2">高</a-select-option>
            <a-select-option :value="3">紧急</a-select-option>
            <a-select-option :value="4">严重</a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item label="指定处理人">
          <a-select 
            v-model:value="newInstance.assignee_id" 
            placeholder="请选择处理人(可选)" 
            style="width: 100%" 
            allow-clear
          >
            <a-select-option v-for="user in users" :key="user.id" :value="user.id">
              {{ user.username }}
            </a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item label="截止日期">
          <a-date-picker 
            v-model:value="dueDate" 
            style="width: 100%" 
            :show-time="{ format: 'HH:mm:ss' }"
            format="YYYY-MM-DD HH:mm:ss"
          />
        </a-form-item>

        <a-form-item label="标签">
          <a-select
            v-model:value="newInstance.tags"
            mode="tags"
            style="width: 100%"
            placeholder="输入标签"
            :token-separators="[',']"
          />
        </a-form-item>

        <a-form-item label="描述">
          <a-textarea v-model:value="newInstance.description" :rows="3" placeholder="请输入工单描述..." />
        </a-form-item>
      </div>

      <!-- 步骤2：填写表单字段 -->
      <div v-if="selectedProcess || instanceDialog.isEdit" class="instance-form">
        <div class="instance-form-header">
          <a-button 
            v-if="!instanceDialog.isEdit" 
            @click="backToProcessSelection" 
            type="default"
            class="back-button"
          >
            <template #icon><ArrowLeftOutlined /></template>
            返回
          </a-button>

          <div class="instance-form-title">
            <template v-if="!instanceDialog.isEdit">
              <h3>{{ selectedProcess?.name }}</h3>
              <p>{{ selectedProcess?.description }}</p>
            </template>
            
            <template v-else>
              <h3>编辑: {{ instanceDialog.instance?.title }}</h3>
            </template>
          </div>
        </div>

        <a-form layout="vertical">
          <template v-if="formFields.length > 0">
            <a-form-item 
              v-for="field in formFields" 
              :key="field.field" 
              :label="field.label" 
              :name="field.field"
              :rules="[{ required: field.required, message: `请输入${field.label}!` }]"
            >
              <!-- 文本框 -->
              <a-input 
                v-if="field.type === 'text'" 
                v-model:value="formDataValues[field.field]"
                :placeholder="field.placeholder || `请输入${field.label}`" 
              />

              <!-- 数字输入 -->
              <a-input-number 
                v-else-if="field.type === 'number'" 
                v-model:value="formDataValues[field.field]"
                style="width: 100%" 
                :placeholder="field.placeholder || `请输入${field.label}`" 
              />

              <!-- 日期选择器 -->
              <a-date-picker 
                v-else-if="field.type === 'date'" 
                v-model:value="formDataValues[field.field]"
                style="width: 100%" 
                :placeholder="field.placeholder || `请选择${field.label}`" 
              />

              <!-- 下拉选择 -->
              <a-select 
                v-else-if="field.type === 'select'" 
                v-model:value="formDataValues[field.field]" 
                style="width: 100%"
                :placeholder="field.placeholder || `请选择${field.label}`"
              >
                <a-select-option v-for="option in field.options" :key="option" :value="option">
                  {{ option }}
                </a-select-option>
              </a-select>

              <!-- 复选框 -->
              <a-checkbox 
                v-else-if="field.type === 'checkbox'" 
                v-model:checked="formDataValues[field.field]"
              >
                {{ field.label }}
              </a-checkbox>

              <!-- 单选框组 -->
              <a-radio-group 
                v-else-if="field.type === 'radio'" 
                v-model:value="formDataValues[field.field]"
              >
                <a-radio v-for="option in field.options" :key="option" :value="option">
                  {{ option }}
                </a-radio>
              </a-radio-group>

              <!-- 多行文本 -->
              <a-textarea 
                v-else-if="field.type === 'textarea'" 
                v-model:value="formDataValues[field.field]" 
                :rows="3"
                :placeholder="field.placeholder || `请输入${field.label}`" 
              />
            </a-form-item>
          </template>
          <a-alert v-else type="warning" message="未找到表单字段定义" />
        </a-form>
      </div>
    </a-modal>

    <!-- 转交对话框 -->
    <a-modal
      v-model:visible="transferDialog.visible"
      title="工单转交"
      @ok="confirmTransfer"
      okText="转交"
      cancelText="取消"
    >
      <a-form layout="vertical">
        <a-form-item label="转交给" required>
          <a-select v-model:value="transferDialog.assigneeId" placeholder="请选择处理人">
            <a-select-option v-for="user in users" :key="user.id" :value="user.id">
              {{ user.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="转交说明">
          <a-textarea v-model:value="transferDialog.comment" :rows="3" placeholder="请输入转交说明..." />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 删除确认对话框 -->
    <a-modal
      v-model:visible="deleteDialog.visible"
      title="删除工单" 
      @ok="confirmDelete"
      okText="删除"
      okType="danger"
      cancelText="取消"
    >
      <p>确认要删除此工单吗？此操作不可恢复。</p>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue';
import { message } from 'ant-design-vue';
import { 
  PlusOutlined, 
  FileOutlined, 
  CheckCircleOutlined, 
  ClockCircleOutlined,
  CloseCircleOutlined,
  DownOutlined,
  ArrowLeftOutlined,
  UploadOutlined
} from '@ant-design/icons-vue';
import dayjs from 'dayjs';
import {
  listInstance,
  detailInstance,
  createInstance,
  updateInstance,
  deleteInstance,
  processInstanceFlow,
  transferInstance,
  commentInstance,
  getInstanceComments,
  getInstanceFlows,
  getMyInstances,
  getOverdueInstances,
  getProcessDefinition,
  type ListInstanceReq,
  type MyInstanceReq,
  type InstanceResp,
  type InstanceItem,
  type CreateInstanceReq,
  type UpdateInstanceReq,
  type InstanceActionReq,
  type InstanceCommentReq,
  type InstanceFlowResp,
  type InstanceCommentResp,
  type InstanceAttachmentResp,
  type TransferInstanceReq,
  InstanceStatus,
  Priority
} from '#/api/core/workorder_instance';

import { listProcess } from '#/api/core/workorder_process';
import { listCategory } from '#/api/core/workorder_category';
import { getUserList } from '#/api/core/user';

// 定义类型
interface Process {
  id: number;
  name: string;
  description?: string;
  version: number;
}

interface Template {
  id: number;
  name: string;
  description?: string;
}

interface Category {
  id: number;
  name: string;
}

interface User {
  id: number;
  name: string;
  username: string;
}

interface Field {
  field: string;
  label: string;
  type: string;
  required: boolean;
  options?: string[];
  placeholder?: string;
}

interface WorkOrderStatistics {
  total_count: number;
  completed_count: number;
  processing_count: number;
  canceled_count: number;
  rejected_count: number;
}

// 状态数据
const loading = ref(false);
const searchQuery = ref('');
const statusFilter = ref<InstanceStatus | null>(null);
const currentPage = ref(1);
const pageSize = ref(10);
const totalItems = ref(0);
const dateRange = ref<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
const processingComment = ref('');
const newComment = ref('');
const dueDate = ref<dayjs.Dayjs | null>(null);
const uploadFileList = ref<any[]>([]);

// 数据源
const instances = ref<InstanceItem[]>([]);
const processes = ref<Process[]>([]);
const templates = ref<Template[]>([]);
const categories = ref<Category[]>([]);
const users = ref<User[]>([]);
const instanceFlows = ref<InstanceFlowResp[]>([]);
const instanceComments = ref<InstanceCommentResp[]>([]);
const instanceAttachments = ref<InstanceAttachmentResp[]>([]);
const statistics = ref<WorkOrderStatistics>({
  total_count: 0,
  completed_count: 0,
  processing_count: 0,
  canceled_count: 0,
  rejected_count: 0
});

// 表单字段和数据
const formFields = ref<Field[]>([]);
const formDataValues = reactive<Record<string, any>>({});
const displayFormData = ref<Record<string, any> | null>(null);
const formFieldDefinitions = ref<any[]>([]);

// 对话框状态
const detailDialog = reactive({
  visible: false,
  instance: null as InstanceResp | null
});

const instanceDialog = reactive({
  visible: false,
  isEdit: false,
  instance: null as InstanceResp | null,
});

const transferDialog = reactive({
  visible: false,
  instanceId: 0,
  assigneeId: null as number | null,
  comment: ''
});

const deleteDialog = reactive({
  visible: false,
  instanceId: 0
});

// 新工单实例
const newInstance = reactive<CreateInstanceReq>({
  title: '',
  process_id: 0,
  priority: Priority.NORMAL,
  tags: []
});

// 选择的流程
const selectedProcess = ref<Process | null>(null);

// 列定义
const columns = [
  {
    title: '工单标题',
    dataIndex: 'title',
    key: 'title',
    width: 250,
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100,
    align: 'center',
  },
  {
    title: '优先级',
    dataIndex: 'priority',
    key: 'priority',
    width: 100,
    align: 'center',
  },
  {
    title: '创建人',
    dataIndex: 'creator_name',
    key: 'creator',
    width: 120,
  },
  {
    title: '处理人',
    dataIndex: 'assignee_name',
    key: 'assignee',
    width: 120,
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: 180,
  },
  {
    title: '操作',
    key: 'action',
    width: 200,
    align: 'center',
  },
];

// 监听分页和搜索条件的变化
watch(
  [currentPage, pageSize, searchQuery, statusFilter, dateRange],
  () => {
    fetchInstances();
  },
  { deep: true }
);

// 方法
const fetchInstances = async () => {
  try {
    loading.value = true;
    const params: ListInstanceReq = {
      page: currentPage.value,
      size: pageSize.value,
    };

    if (searchQuery.value) {
      params.title = searchQuery.value;
    }

    if (statusFilter.value !== null) {
      params.status = statusFilter.value;
    }

    if (dateRange.value) {
      params.start_date = dateRange.value[0].format('YYYY-MM-DD');
      params.end_date = dateRange.value[1].format('YYYY-MM-DD');
    }

    const response = await listInstance(params);

    if (response && typeof response === 'object') {
      if (Array.isArray(response.items)) {
        instances.value = response.items;
        totalItems.value = typeof response.total === 'number' ? response.total : response.items.length;
      } else if (Array.isArray(response)) {
        instances.value = response;
        totalItems.value = response.length;
      } else {
        instances.value = [];
        totalItems.value = 0;
      }
    } else {
      instances.value = [];
      totalItems.value = 0;
    }

    calculateStatistics();
  } catch (error) {
    message.error('获取工单列表失败');
    console.error('Failed to fetch instances:', error);
    instances.value = [];
    totalItems.value = 0;
  } finally {
    loading.value = false;
  }
};

// 获取我的工单
const fetchMyInstances = async (type: 'created' | 'assigned' | 'all' = 'all') => {
  try {
    loading.value = true;
    const params: MyInstanceReq = {
      page: currentPage.value,
      page_size: pageSize.value,
      type: type
    };

    if (searchQuery.value) {
      params.title = searchQuery.value;
    }

    if (statusFilter.value !== null) {
      params.status = statusFilter.value;
    }

    if (dateRange.value) {
      params.start_date = dateRange.value[0].format('YYYY-MM-DD');
      params.end_date = dateRange.value[1].format('YYYY-MM-DD');
    }

    const response = await getMyInstances(params);
    
    if (response && Array.isArray(response)) {
      instances.value = response;
      totalItems.value = response.length;
    } else if (response && typeof response === 'object' && Array.isArray(response.data)) {
      instances.value = response.data;
      totalItems.value = response.total || response.data.length;
    } else {
      instances.value = [];
      totalItems.value = 0;
    }
    
    calculateStatistics();
  } catch (error) {
    message.error('获取我的工单失败');
    console.error('Failed to fetch my instances:', error);
    instances.value = [];
    totalItems.value = 0;
  } finally {
    loading.value = false;
  }
};

// 获取超时工单
const fetchOverdueInstances = async () => {
  try {
    loading.value = true;
    const response = await getOverdueInstances({
      page: currentPage.value,
      page_size: pageSize.value
    });
    
    if (response && Array.isArray(response)) {
      instances.value = response;
      totalItems.value = response.length;
    } else if (response && typeof response === 'object' && Array.isArray(response.data)) {
      instances.value = response.data;
      totalItems.value = response.total || response.data.length;
    } else {
      instances.value = [];
      totalItems.value = 0;
    }
    
    calculateStatistics();
  } catch (error) {
    message.error('获取超时工单失败');
    console.error('Failed to fetch overdue instances:', error);
    instances.value = [];
    totalItems.value = 0;
  } finally {
    loading.value = false;
  }
};

const calculateStatistics = () => {
  const stats = instances.value.reduce(
    (acc, instance) => {
      acc.total_count++;
      switch (instance.status) {
        case InstanceStatus.PROCESSING:
          acc.processing_count++;
          break;
        case InstanceStatus.COMPLETED:
          acc.completed_count++;
          break;
        case InstanceStatus.REJECTED:
          acc.rejected_count++;
          break;
        case InstanceStatus.CANCELLED:
          acc.canceled_count++;
          break;
      }
      return acc;
    },
    {
      total_count: 0,
      completed_count: 0,
      processing_count: 0,
      canceled_count: 0,
      rejected_count: 0
    }
  );
  
  statistics.value = stats;
};

const fetchInstanceDetail = async (id: number) => {
  try {
    const response = await detailInstance(id);
    if (!response) {
      message.error('获取工单详情失败: 无响应数据');
      return;
    }
    
    detailDialog.instance = response;
    
    // 解析表单数据
    let parsedFormData: any = null;
    
    if (typeof response.form_data === 'string') {
      try {
        parsedFormData = JSON.parse(response.form_data);
      } catch (e) {
        parsedFormData = {};
        console.error('解析表单数据失败:', e);
      }
    } else if (response.form_data && typeof response.form_data === 'object') {
      parsedFormData = response.form_data;
    }
    
    // 处理表单数据结构
    if (parsedFormData) {
      // 如果有 fields 属性，说明这是字段定义
      if (parsedFormData.fields && Array.isArray(parsedFormData.fields)) {
        formFieldDefinitions.value = parsedFormData.fields;
        
        // 尝试从其他地方获取实际填写的数据
        if (parsedFormData.data) {
          displayFormData.value = parsedFormData.data;
        } else {
          // 如果没有实际数据，显示字段的默认值
          const defaultData: Record<string, any> = {};
          parsedFormData.fields.forEach((field: any) => {
            defaultData[field.name] = field.default_value || '';
          });
          displayFormData.value = defaultData;
        }
      } else {
        // 如果没有 fields 属性，当作普通的键值对数据处理
        displayFormData.value = parsedFormData;
        formFieldDefinitions.value = [];
      }
    } else {
      displayFormData.value = {};
      formFieldDefinitions.value = [];
    }
    
    // 获取流转记录和评论
    await Promise.all([
      fetchInstanceFlows(id),
      fetchInstanceComments(id)
    ]);
  } catch (error) {
    message.error('获取工单详情失败');
    console.error('Failed to fetch instance detail:', error);
  }
};

const fetchInstanceFlows = async (id: number) => {
  try {
    const response = await getInstanceFlows(id);
    instanceFlows.value = Array.isArray(response) ? response : [];
  } catch (error) {
    console.error('Failed to fetch instance flows:', error);
    instanceFlows.value = [];
  }
};

const fetchInstanceComments = async (id: number) => {
  try {
    const response = await getInstanceComments(id);
    instanceComments.value = Array.isArray(response) ? response : [];
  } catch (error) {
    console.error('Failed to fetch instance comments:', error);
    instanceComments.value = [];
  }
};

// 新增辅助函数
const getFieldLabel = (fieldName: string) => {
  if (formFieldDefinitions.value && formFieldDefinitions.value.length > 0) {
    const field = formFieldDefinitions.value.find(f => f.name === fieldName);
    return field ? field.label : fieldName;
  }
  return fieldName;
};

const getFieldValue = (field: any) => {
  if (displayFormData.value && displayFormData.value[field.name] !== undefined) {
    return displayFormData.value[field.name];
  }
  return field.default_value || '';
};

const handleSelectProcess = async (processId: number) => {
  try {
    selectedProcess.value = processes.value.find(p => p.id === processId) || null;
    
    // 获取流程定义
    try {
      const definition = await getProcessDefinition(processId);
      
      // 根据流程定义设置表单字段
      if (definition && definition.form_data && definition.form_data.fields) {
        // 处理从接口返回的字段定义格式
        formFields.value = definition.form_data.fields.map((field: any) => ({
          field: field.name,
          label: field.label,
          type: field.type,
          required: field.required,
          placeholder: field.placeholder,
          options: field.options ? field.options.map((opt: any) => 
            typeof opt === 'object' ? opt.label : opt
          ) : undefined
        }));
      } else {
        // 使用默认字段定义
        formFields.value = [
          { field: 'name', label: '姓名', type: 'text', required: true, placeholder: '请输入姓名' },
          { field: 'department', label: '部门', type: 'select', required: true, options: ['技术部', '市场部', '人力资源部'] },
          { field: 'description', label: '问题描述', type: 'textarea', required: true, placeholder: '请详细描述您的问题' }
        ];
      }
    } catch (error) {
      console.error('Failed to fetch process definition:', error);
      // 使用默认表单字段
      formFields.value = [
        { field: 'name', label: '姓名', type: 'text', required: true, placeholder: '请输入姓名' },
        { field: 'department', label: '部门', type: 'select', required: true, options: ['技术部', '市场部', '人力资源部'] },
        { field: 'description', label: '问题描述', type: 'textarea', required: true, placeholder: '请详细描述您的问题' }
      ];
    }
    
    // 初始化表单数据
    formFields.value.forEach((field: Field) => {
      if (field.type === 'checkbox') {
        formDataValues[field.field] = false;
      } else if (field.type === 'number') {
        formDataValues[field.field] = 0;
      } else if (field.type === 'date') {
        formDataValues[field.field] = null;
      } else {
        formDataValues[field.field] = '';
      }
    });
  } catch (error) {
    message.error('获取流程详情失败');
    console.error('Failed to fetch process detail:', error);
  }
};

const backToProcessSelection = () => {
  selectedProcess.value = null;
  formFields.value = [];
  Object.keys(formDataValues).forEach(key => delete formDataValues[key]);
};

const handleSizeChange = (current: number, size: number) => {
  pageSize.value = size;
  currentPage.value = 1;
};

const handleCurrentChange = (page: number) => {
  currentPage.value = page;
};

const handleSearch = () => {
  currentPage.value = 1;
};

const handleStatusChange = () => {
  currentPage.value = 1;
};

const handleDateRangeChange = () => {
  currentPage.value = 1;
};

const handleCreateInstance = () => {
  instanceDialog.isEdit = false;
  instanceDialog.instance = null;
  selectedProcess.value = null;
  
  // 重置新实例数据
  Object.assign(newInstance, {
    title: '',
    process_id: 0,
    description: '',
    priority: Priority.NORMAL,
    tags: []
  });
  
  // 清空表单数据
  Object.keys(formDataValues).forEach(key => delete formDataValues[key]);
  
  dueDate.value = null;
  formFields.value = [];
  instanceDialog.visible = true;
};

const handleEditInstance = (instance: InstanceItem | InstanceResp) => {
  instanceDialog.isEdit = true;
  instanceDialog.instance = JSON.parse(JSON.stringify(instance)) as InstanceResp;
  
  // 解析表单数据
  let instanceFormData: Record<string, any> = {};
  let fieldDefinitions: any[] = [];
  
  if (typeof instance.form_data === 'string') {
    try {
      const parsed = JSON.parse(instance.form_data);
      if (parsed.fields && Array.isArray(parsed.fields)) {
        fieldDefinitions = parsed.fields;
        instanceFormData = parsed.data || {};
      } else {
        instanceFormData = parsed;
      }
    } catch (e) {
      instanceFormData = {};
      console.error('解析表单数据失败:', e);
    }
  } else if (instance.form_data && typeof instance.form_data === 'object') {
    const formData = instance.form_data as any;
    if (formData.fields && Array.isArray(formData.fields)) {
      fieldDefinitions = formData.fields;
      instanceFormData = formData.data || {};
    } else {
      instanceFormData = formData;
    }
  }
  
  // 设置表单字段
  if (fieldDefinitions.length > 0) {
    formFields.value = fieldDefinitions.map((field: any) => ({
      field: field.name,
      label: field.label,
      type: field.type,
      required: field.required,
      placeholder: field.placeholder,
      options: field.options ? field.options.map((opt: any) => 
        typeof opt === 'object' ? opt.label : opt
      ) : undefined
    }));
  } else {
    fetchProcessFormFields(instance.process_id);
  }
  
  // 清空现有数据
  Object.keys(formDataValues).forEach(key => delete formDataValues[key]);
  
  // 复制数据到编辑对象
  Object.keys(instanceFormData).forEach(key => {
    formDataValues[key] = instanceFormData[key];
  });
  
  instanceDialog.visible = true;
  detailDialog.visible = false;
};

const fetchProcessFormFields = async (processId: number) => {
  try {
    const definition = await getProcessDefinition(processId);
    if (definition && definition.form_data && definition.form_data.fields) {
      formFields.value = definition.form_data.fields.map((field: any) => ({
        field: field.name,
        label: field.label,
        type: field.type,
        required: field.required,
        placeholder: field.placeholder,
        options: field.options ? field.options.map((opt: any) => 
          typeof opt === 'object' ? opt.label : opt
        ) : undefined
      }));
    } else {
      // 模拟获取流程对应的表单字段
      formFields.value = [
        { field: 'name', label: '姓名', type: 'text', required: true, placeholder: '请输入姓名' },
        { field: 'department', label: '部门', type: 'select', required: true, options: ['技术部', '市场部', '人力资源部'] },
        { field: 'description', label: '问题描述', type: 'textarea', required: true, placeholder: '请详细描述您的问题' }
      ];
    }
  } catch (error) {
    console.error('Failed to fetch process form fields:', error);
    formFields.value = [];
  }
};

const handleViewInstance = async (instance: InstanceItem) => {
  await fetchInstanceDetail(instance.id);
  detailDialog.visible = true;
};

const handleCommand = async (command: string, instance: InstanceItem) => {
  switch (command) {
    case 'approve':
      await processInstance(instance, 'approve');
      break;
    case 'reject':
      await processInstance(instance, 'reject');
      break;
    case 'transfer':
      transferDialog.instanceId = instance.id;
      transferDialog.visible = true;
      break;
    case 'cancel':
      await processInstance(instance, 'cancel');
      break;
    case 'delete':
      deleteDialog.instanceId = instance.id;
      deleteDialog.visible = true;
      break;
  }
};

const saveInstance = async () => {
  if (!selectedProcess.value && !instanceDialog.isEdit && !newInstance.process_id) {
    message.error('请选择流程');
    return;
  }
  
  if (!newInstance.title && !instanceDialog.isEdit) {
    message.error('请输入工单标题');
    return;
  }
  
  try {
    if (instanceDialog.isEdit && instanceDialog.instance) {
      const updateData: UpdateInstanceReq = {
        id: instanceDialog.instance.id,
        title: instanceDialog.instance.title,
        priority: instanceDialog.instance.priority,
      };
      
      await updateInstance(instanceDialog.instance.id, updateData);
      message.success('工单更新成功');
    } else {
      const createData: CreateInstanceReq = {
        ...newInstance
      };
      
      if (dueDate.value) {
        createData.due_date = dueDate.value.toISOString();
      }
      
      await createInstance(createData);
      message.success('工单创建成功');
    }
    
    instanceDialog.visible = false;
    fetchInstances();
  } catch (error) {
    message.error('保存工单失败');
    console.error('Failed to save instance:', error);
  }
};

const processInstance = async (instance: InstanceItem | InstanceResp, action: string) => {
  if (!processingComment.value && ['approve', 'reject'].includes(action)) {
    message.warning('请输入处理意见');
    return;
  }
  
  try {
    const flowData: InstanceActionReq = {
      instance_id: instance.id,
      action: action as any,
      comment: processingComment.value,
      step_id: instance.current_step
    };
    
    await processInstanceFlow(instance.id, flowData);
    
    const actionText = {
      approve: '批准',
      reject: '拒绝',
      cancel: '取消',
      revoke: '撤回'
    }[action] || action;
    
    message.success(`工单 #${instance.id} 已${actionText}`);
    detailDialog.visible = false;
    processingComment.value = '';
    fetchInstances();
  } catch (error) {
    message.error('处理工单失败');
    console.error('Failed to process instance:', error);
  }
};

const showTransferDialog = () => {
  if (detailDialog.instance) {
    transferDialog.instanceId = detailDialog.instance.id;
    transferDialog.visible = true;
  }
};

const confirmTransfer = async () => {
  if (!transferDialog.assigneeId) {
    message.warning('请选择转交人');
    return;
  }
  
  try {
    const transferData: TransferInstanceReq = {
      assignee_id: transferDialog.assigneeId,
      comment: transferDialog.comment
    };
    
    await transferInstance(transferDialog.instanceId, transferData);
    message.success('工单转交成功');
    transferDialog.visible = false;
    transferDialog.assigneeId = null;
    transferDialog.comment = '';
    
    if (detailDialog.visible && detailDialog.instance) {
      fetchInstanceDetail(detailDialog.instance.id);
    }
    fetchInstances();
  } catch (error) {
    message.error('工单转交失败');
    console.error('Failed to transfer instance:', error);
  }
};

const addComment = async () => {
  if (!newComment.value || !detailDialog.instance) {
    message.warning('请输入评论内容');
    return;
  }
  
  try {
    const commentData: InstanceCommentReq = {
      instance_id: detailDialog.instance.id,
      content: newComment.value
    };
    
    await commentInstance(detailDialog.instance.id, commentData);
    message.success('评论已添加');
    newComment.value = '';
    
    // 刷新评论列表
    fetchInstanceComments(detailDialog.instance.id);
  } catch (error) {
    message.error('添加评论失败');
    console.error('Failed to add comment:', error);
  }
};

const beforeUpload = (file: any) => {
  return false; // 阻止自动上传
};

const handleUploadChange = (info: any) => {
  uploadFileList.value = info.fileList;
};

const uploadInstanceAttachment = async () => {
  if (!detailDialog.instance || uploadFileList.value.length === 0) {
    return;
  }
  
  try {
    const formData = new FormData();
    uploadFileList.value.forEach(file => {
      formData.append('files', file.originFileObj);
    });
    
    // 这里需要实现上传逻辑
    message.success('附件上传成功');
    uploadFileList.value = [];
  } catch (error) {
    message.error('附件上传失败');
    console.error('Failed to upload attachment:', error);
  }
};

const downloadAttachment = (attachment: InstanceAttachmentResp) => {
  const link = document.createElement('a');
  link.href = attachment.file_path;
  link.download = attachment.file_name;
  link.click();
};

const deleteAttachmentConfirm = async (attachmentId: number) => {
  if (!detailDialog.instance) return;
  
  try {
    // 这里需要实现删除附件的逻辑
    message.success('附件删除成功');
  } catch (error) {
    message.error('附件删除失败');
    console.error('Failed to delete attachment:', error);
  }
};

const confirmDelete = async () => {
  try {
    await deleteInstance(deleteDialog.instanceId);
    message.success(`工单 #${deleteDialog.instanceId} 已删除`);
    deleteDialog.visible = false;
    fetchInstances();
  } catch (error) {
    message.error('删除工单失败');
    console.error('Failed to delete instance:', error);
  }
};

// 辅助方法
const formatDate = (dateStr: string) => {
  if (!dateStr) return '';
  return dayjs(dateStr).format('YYYY-MM-DD');
};

const formatTime = (dateStr: string) => {
  if (!dateStr) return '';
  return dayjs(dateStr).format('HH:mm');
};

const formatFullDateTime = (dateStr: string) => {
  if (!dateStr) return '';
  return dayjs(dateStr).format('YYYY-MM-DD HH:mm:ss');
};

const formatDuration = (minutes: number) => {
  if (minutes < 60) {
    return `${minutes}分钟`;
  } else if (minutes < 1440) {
    return `${Math.floor(minutes / 60)}小时${minutes % 60}分钟`;
  } else {
    const days = Math.floor(minutes / 1440);
    const hours = Math.floor((minutes % 1440) / 60);
    return `${days}天${hours}小时`;
  }
};

const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const getInitials = (name: string) => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getStatusClass = (status: InstanceStatus) => {
  switch (status) {
    case InstanceStatus.DRAFT: return 'status-draft';
    case InstanceStatus.PROCESSING: return 'status-processing';
    case InstanceStatus.COMPLETED: return 'status-completed';
    case InstanceStatus.REJECTED: return 'status-rejected';
    case InstanceStatus.CANCELLED: return 'status-cancelled';
    case InstanceStatus.PENDING: return 'status-pending';
    case InstanceStatus.OVERDUE: return 'status-overdue';
    default: return '';
  }
};

const getStatusColor = (status: InstanceStatus) => {
  switch (status) {
    case InstanceStatus.DRAFT: return 'orange';
    case InstanceStatus.PROCESSING: return 'blue';
    case InstanceStatus.COMPLETED: return 'green';
    case InstanceStatus.REJECTED: return 'red';
    case InstanceStatus.CANCELLED: return 'default';
    case InstanceStatus.PENDING: return 'gold';
    case InstanceStatus.OVERDUE: return 'volcano';
    default: return 'default';
  }
};

const getStatusText = (status: InstanceStatus) => {
  switch (status) {
    case InstanceStatus.DRAFT: return '草稿';
    case InstanceStatus.PROCESSING: return '处理中';
    case InstanceStatus.COMPLETED: return '已完成';
    case InstanceStatus.REJECTED: return '已拒绝';
    case InstanceStatus.CANCELLED: return '已取消';
    case InstanceStatus.PENDING: return '待处理';
    case InstanceStatus.OVERDUE: return '已超时';
    default: return '未知';
  }
};

const getPriorityColor = (priority: Priority) => {
  switch (priority) {
    case Priority.LOW: return 'green';
    case Priority.NORMAL: return 'blue';
    case Priority.HIGH: return 'orange';
    case Priority.URGENT: return 'red';
    case Priority.CRITICAL: return 'volcano';
    default: return 'default';
  }
};

const getPriorityText = (priority: Priority) => {
  switch (priority) {
    case Priority.LOW: return '低';
    case Priority.NORMAL: return '普通';
    case Priority.HIGH: return '高';
    case Priority.URGENT: return '紧急';
    case Priority.CRITICAL: return '严重';
    default: return '未知';
  }
};

const getFlowColor = (action: string) => {
  switch (action) {
    case 'approve': return 'green';
    case 'reject': return 'red';
    case 'transfer': return 'blue';
    case 'cancel': return 'orange';
    case 'revoke': return 'purple';
    default: return 'gray';
  }
};

const getFlowActionText = (action: string) => {
  switch (action) {
    case 'approve': return '批准';
    case 'reject': return '拒绝';
    case 'transfer': return '转交';
    case 'revoke': return '撤回';
    case 'cancel': return '取消';
    default: return action;
  }
};

const getAvatarColor = (name: string) => {
  const colors = [
    '#1890ff', '#52c41a', '#faad14', '#f5222d',
    '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'
  ];

  let hash = 0;
  if (!name) return colors[0];
  
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  return colors[Math.abs(hash) % colors.length];
};

// 数据初始化
const initData = async () => {
  try { 
    // 获取流程列表
    const processesResponse = await listProcess({
      page: 1,
      size: 100,
      status: 1,
    });
    processes.value = processesResponse?.items || [];
    
    // 获取分类列表
    const categoriesResponse = await listCategory({
      page: 1,
      size: 10,
    });
    categories.value = categoriesResponse?.items || [];
    
    // 获取用户列表
    const usersResponse = await getUserList({
      page: 1,
      size: 100,
      search: ''
    });
    users.value = usersResponse.items || [];
    
    // 模板数据暂时使用模拟数据，后续可替换为真实接口
    templates.value = [
      { id: 1, name: '通用模板', description: '通用工单模板' },
      { id: 2, name: '紧急模板', description: '紧急工单模板' }
    ];
  } catch (error) {
    message.error('初始化数据失败');
    console.error('数据初始化失败:', error);
  }
};

// 初始化
onMounted(() => {
  initData();
  fetchInstances();
});
</script>

<style scoped>
.form-instance-container {
  padding: 24px;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.btn-create {
  background: #1890ff;
  border: none;
}

.stats-row {
  margin-bottom: 24px;
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
  gap: 10px;
}

.form-badge {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-draft {
  background-color: #faad14;
}

.status-processing {
  background-color: #1890ff;
}

.status-completed {
  background-color: #52c41a;
}

.status-rejected {
  background-color: #f5222d;
}

.status-cancelled {
  background-color: #d9d9d9;
}

.status-pending {
  background-color: #fadb14;
}

.status-overdue {
  background-color: #ff4d4f;
}

.form-name-text {
  font-weight: 500;
}

.instance-id {
  font-size: 12px;
  color: #8c8c8c;
}

.creator-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.creator-name {
  font-size: 14px;
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

.action-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
  flex-wrap: wrap;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.detail-dialog .instance-details {
  margin-bottom: 20px;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.detail-header h2 {
  margin: 0;
  font-size: 24px;
  color: #1f2937;
}

.form-data-preview,
.flow-records,
.comments-section,
.attachments-section {
  margin-top: 24px;
}

.form-data-preview h3,
.flow-records h3,
.comments-section h3,
.attachments-section h3 {
  margin-bottom: 16px;
  color: #1f2937;
  font-size: 18px;
}

.flow-item {
  padding: 8px 0;
}

.flow-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.flow-node {
  font-weight: 500;
}

.flow-action {
  background-color: #f0f0f0;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.flow-time {
  color: #8c8c8c;
  font-size: 12px;
  margin-left: auto;
}

.flow-operator,
.flow-comment,
.flow-duration {
  font-size: 14px;
  margin-top: 4px;
}

.comment-list,
.attachment-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.comment-item {
  padding: 12px;
  background-color: #f9f9f9;
  border-radius: 8px;
}

.comment-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.comment-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.comment-author {
  font-weight: 500;
}

.comment-time {
  font-size: 12px;
  color: #8c8c8c;
}

.comment-content {
  white-space: pre-line;
}

.attachment-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background-color: #f9f9f9;
  border-radius: 8px;
}

.attachment-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.attachment-name {
  font-weight: 500;
}

.attachment-size,
.attachment-uploader {
  font-size: 12px;
  color: #8c8c8c;
}

.attachment-actions {
  display: flex;
  gap: 8px;
}

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.process-selection {
  margin-bottom: 24px;
}

.instance-form-header {
  display: flex;
  align-items: flex-start;
  margin-bottom: 24px;
}

.back-button {
  margin-right: 16px;
}

.instance-form-title {
  flex: 1;
}

.instance-form h3 {
  margin-bottom: 16px;
  font-size: 18px;
  color: #1f2937;
}

.instance-form p {
  margin-bottom: 24px;
  color: #6b7280;
}

.action-area {
  margin-top: 24px;
  padding: 16px;
  border-radius: 4px;
}

.mt-16 {
  margin-top: 16px;
}

.mt-8 {
  margin-top: 8px;
}

@media (max-width: 768px) {
  .form-instance-container {
    padding: 16px;
  }
  
  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }
  
  .header-actions {
    flex-direction: column;
    gap: 8px;
  }
  
  .action-buttons {
    justify-content: flex-start;
  }
  
  .detail-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>