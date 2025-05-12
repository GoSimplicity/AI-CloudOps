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
          <a-select-option :value="3">已拒绝</a-select-option>
        </a-select>
        <a-range-picker 
          v-model:value="dateRange" 
          style="width: 240px" 
          @change="handleDateRangeChange" 
          :allowClear="true"
          :placeholder="['开始日期', '结束日期']"
        />
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
                      <a-menu-item key="submit" v-if="record.status === 0">提交</a-menu-item>
                      <a-menu-item key="process" v-if="record.status === 1">处理</a-menu-item>
                      <a-menu-item key="reject" v-if="record.status === 1">拒绝</a-menu-item>
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
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="工单ID">{{ detailDialog.instance.id }}</a-descriptions-item>
          <a-descriptions-item label="流程ID">{{ detailDialog.instance.process_id }}</a-descriptions-item>
          <a-descriptions-item label="当前节点">{{ detailDialog.instance.current_node }}</a-descriptions-item>
          <a-descriptions-item label="流程版本">{{ detailDialog.instance.process_version }}</a-descriptions-item>
          <a-descriptions-item label="创建人">{{ detailDialog.instance.creator_name }}</a-descriptions-item>
          <a-descriptions-item label="提交时间">{{ formatFullDateTime(detailDialog.instance.created_at) }}</a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.assignee_name" label="处理人">
            {{ detailDialog.instance.assignee_name }}
          </a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.completed_at" label="完成时间">
            {{ formatFullDateTime(detailDialog.instance.completed_at) }}
          </a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.due_date" label="截止时间">
            {{ formatFullDateTime(detailDialog.instance.due_date) }}
          </a-descriptions-item>
        </a-descriptions>

        <div class="form-data-preview">
          <h3>表单数据</h3>
          <a-collapse>
            <a-collapse-panel key="1" header="表单内容">
              <a-form layout="vertical">
                <template v-if="formData">
                  <a-form-item v-for="(value, field) in formData" :key="field" :label="field">
                    <a-input v-if="!Array.isArray(value)" v-model:value="formData[field]" :disabled="true" />
                    <span v-else>{{ value.join(', ') }}</span>
                  </a-form-item>
                </template>
              </a-form>
            </a-collapse-panel>
          </a-collapse>
        </div>

        <div v-if="instanceFlows.length > 0" class="flow-records">
          <h3>流转记录</h3>
          <a-timeline>
            <a-timeline-item v-for="flow in instanceFlows" :key="flow.id" :color="getFlowColor(flow.action)">
              <div class="flow-item">
                <div class="flow-header">
                  <span class="flow-node">{{ flow.node_name }}</span>
                  <span class="flow-action">{{ getFlowActionText(flow.action) }}</span>
                  <span class="flow-time">{{ formatFullDateTime(flow.created_at) }}</span>
                </div>
                <div class="flow-operator">
                  操作人: {{ flow.operator_name }}
                </div>
                <div class="flow-comment" v-if="flow.comment">
                  备注: {{ flow.comment }}
                </div>
              </div>
            </a-timeline-item>
          </a-timeline>
        </div>

        <div v-if="instanceComments.length > 0" class="comments-section">
          <h3>评论</h3>
          <div class="comment-list">
            <div v-for="comment in instanceComments" :key="comment.id" class="comment-item">
              <div class="comment-header">
                <a-avatar :style="{ backgroundColor: getAvatarColor(comment.creator_name) }">
                  {{ getInitials(comment.creator_name) }}
                </a-avatar>
                <div class="comment-info">
                  <div class="comment-author">{{ comment.creator_name }}</div>
                  <div class="comment-time">{{ formatFullDateTime(comment.created_at) }}</div>
                </div>
              </div>
              <div class="comment-content">{{ comment.content }}</div>
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
      :destroy-on-close="true"
    >
      <div v-if="!selectedProcess && !instanceDialog.isEdit" class="process-selection">
        <a-form-item label="标题">
          <a-input v-model:value="newInstance.title" placeholder="请输入工单标题" />
        </a-form-item>
        
        <a-form-item label="选择流程">
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
        
        <a-form-item label="优先级">
          <a-select v-model:value="newInstance.priority" placeholder="请选择优先级" style="width: 100%">
            <a-select-option :value="0">低</a-select-option>
            <a-select-option :value="1">中</a-select-option>
            <a-select-option :value="2">高</a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item label="截止日期">
          <a-date-picker v-model:value="dueDate" style="width: 100%" />
        </a-form-item>
      </div>

      <div v-if="selectedProcess || instanceDialog.isEdit" class="instance-form">
        <template v-if="!instanceDialog.isEdit">
          <h3>{{ selectedProcess?.name }}</h3>
          <p>{{ selectedProcess?.description }}</p>
        </template>
        
        <template v-else>
          <h3>编辑: {{ instanceDialog.instance?.title }}</h3>
        </template>

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
                :placeholder="`请输入${field.label}`" 
              />

              <!-- 数字输入 -->
              <a-input-number 
                v-else-if="field.type === 'number'" 
                v-model:value="formDataValues[field.field]"
                style="width: 100%" 
                :placeholder="`请输入${field.label}`" 
              />

              <!-- 日期选择器 -->
              <a-date-picker 
                v-else-if="field.type === 'date'" 
                v-model:value="formDataValues[field.field]"
                style="width: 100%" 
                :placeholder="`请选择${field.label}`" 
              />

              <!-- 日期范围选择器 -->
              <a-range-picker 
                v-else-if="field.type === 'date_range'" 
                v-model:value="formDataValues[field.field]"
                style="width: 100%" 
                :placeholder="['开始日期', '结束日期']" 
              />

              <!-- 下拉选择 -->
              <a-select 
                v-else-if="field.type === 'select'" 
                v-model:value="formDataValues[field.field]" 
                style="width: 100%"
                :placeholder="`请选择${field.label}`"
              >
                <a-select-option value="选项1">选项1</a-select-option>
                <a-select-option value="选项2">选项2</a-select-option>
                <a-select-option value="选项3">选项3</a-select-option>
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
                <a-radio value="选项1">选项1</a-radio>
                <a-radio value="选项2">选项2</a-radio>
                <a-radio value="选项3">选项3</a-radio>
              </a-radio-group>

              <!-- 多行文本 -->
              <a-textarea 
                v-else-if="field.type === 'textarea'" 
                v-model:value="formDataValues[field.field]" 
                :rows="3"
                :placeholder="`请输入${field.label}`" 
              />
            </a-form-item>
          </template>
          <a-alert v-else type="warning" message="未找到表单字段定义" />
        </a-form>
      </div>
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
import { ref, reactive, computed, onMounted, watch } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { 
  PlusOutlined, 
  FileOutlined, 
  CheckCircleOutlined, 
  ClockCircleOutlined,
  CloseCircleOutlined,
  DownOutlined 
} from '@ant-design/icons-vue';
import dayjs from 'dayjs';
import {
  listInstance,
  detailInstance,
  createInstance,
  deleteInstance,
  approveInstance,
  actionInstance,
  commentInstance,
  listProcess,
  detailProcess,
  listFormDesign,
  detailFormDesign,
  getStatisticsOverview,
  type Instance,
  type ListInstanceReq,
  type DetailInstanceReq,
  type InstanceReq,
  type FormData,
  type InstanceFlowReq,
  type InstanceCommentReq,
  type Process,
  type FormDesign,
  type Field,
  type InstanceFlow,
  type InstanceComment,
  type Schema,
  type WorkOrderStatistics
} from '#/api/core/workorder'; // 使用提供的API

// 状态数据
const loading = ref(false);
const searchQuery = ref('');
const statusFilter = ref(null);
const currentPage = ref(1);
const pageSize = ref(10);
const totalItems = ref(0);
const dateRange = ref<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
const processingComment = ref('');
const newComment = ref('');
const dueDate = ref<dayjs.Dayjs | null>(null);

// 数据源
const instances = ref<Instance[]>([]);
const processes = ref<Process[]>([]);
const formDesigns = ref<FormDesign[]>([]);
const instanceFlows = ref<InstanceFlow[]>([]);
const instanceComments = ref<InstanceComment[]>([]);
const statistics = ref<WorkOrderStatistics>({
  id: 0,
  date: '',
  total_count: 0,
  completed_count: 0,
  processing_count: 0,
  canceled_count: 0,
  rejected_count: 0,
  avg_process_time: 0,
  created_at: '',
  updated_at: ''
});

// 表单字段和数据
const formFields = ref<Field[]>([]);
const formDataValues = reactive<Record<string, any>>({});
const formData = ref<Record<string, any> | null>(null);

// 对话框状态
const detailDialog = reactive({
  visible: false,
  instance: null as Instance | null
});

// 实例创建/编辑对话框
const instanceDialog = reactive({
  visible: false,
  isEdit: false,
  instance: null as Instance | null
});

// 删除对话框
const deleteDialog = reactive({
  visible: false,
  instanceId: 0
});

// 新工单实例
const newInstance = reactive<InstanceReq>({
  title: '',
  process_id: 0,
  process_version: 1,
  form_data: {} as FormData,
  current_node: '',
  priority: 1
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
    width: 180,
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
  loading.value = true;
  try {
    const dateRangeParam = dateRange.value ? 
      [dateRange.value[0].format('YYYY-MM-DD'), dateRange.value[1].format('YYYY-MM-DD')] : 
      undefined;
      
    const params: ListInstanceReq = {
      page: currentPage.value,
      page_size: pageSize.value,
      keyword: searchQuery.value || undefined,
      status: statusFilter.value || undefined,
      date_range: dateRangeParam
    };

    const response = await listInstance(params);
    if (response && response.list) {
      instances.value = response.list;
      totalItems.value = response.total || instances.value.length;
    }
  } catch (error) {
    message.error('获取工单列表失败');
    console.error('Failed to fetch instances:', error);
  } finally {
    loading.value = false;
  }
};

const fetchStatistics = async () => {
  try {
    const response = await getStatisticsOverview();
    if (response) {
      statistics.value = response;
    }
  } catch (error) {
    console.error('Failed to fetch statistics:', error);
  }
};

const fetchProcesses = async () => {
  try {
    const response = await listProcess({ page: 1, size: 100, status: 1 }); // 只获取已发布的流程
    if (response && response.list) {
      processes.value = response.list;
    }
  } catch (error) {
    message.error('获取流程列表失败');
    console.error('Failed to fetch processes:', error);
  }
};

const fetchFormDesigns = async () => {
  try {
    const response = await listFormDesign({ page: 1, size: 100, status: 1 }); // 只获取已发布的表单
    if (response && response.list) {
      formDesigns.value = response.list;
    }
  } catch (error) {
    message.error('获取表单设计列表失败');
    console.error('Failed to fetch form designs:', error);
  }
};

const fetchInstanceDetail = async (id: number) => {
  try {
    const response = await detailInstance({ id });
    if (response) {
      detailDialog.instance = response;
      
      // 解析表单数据
      if (typeof response.form_data === 'string') {
        formData.value = JSON.parse(response.form_data);
      } else {
        formData.value = response.form_data;
      }
      
      // 获取流转记录和评论
      fetchInstanceFlows(id);
      fetchInstanceComments(id);
    }
  } catch (error) {
    message.error('获取工单详情失败');
    console.error('Failed to fetch instance detail:', error);
  }
};

const fetchInstanceFlows = async (instanceId: number) => {
  try {
    // 这里应该有一个API来获取工单流转记录
    // 暂时模拟数据
    instanceFlows.value = [];
  } catch (error) {
    console.error('Failed to fetch instance flows:', error);
  }
};

const fetchInstanceComments = async (instanceId: number) => {
  try {
    // 这里应该有一个API来获取工单评论
    // 暂时模拟数据
    instanceComments.value = [];
  } catch (error) {
    console.error('Failed to fetch instance comments:', error);
  }
};

const handleSelectProcess = async (processId: number) => {
  try {
    const response = await detailProcess({ id: processId });
    if (response) {
      selectedProcess.value = response;
      newInstance.process_version = response.version;
      
      // 获取关联的表单设计
      const formDesignResponse = await detailFormDesign({ id: response.form_design_id });
      if (formDesignResponse) {
        // 解析表单字段
        const schema: Schema = typeof formDesignResponse.schema === 'string' 
          ? JSON.parse(formDesignResponse.schema) 
          : formDesignResponse.schema;
        
        formFields.value = schema.fields;
        
        // 初始化表单数据
        formFields.value.forEach((field: { type: string; field: string }) => {
          if (field.type === 'checkbox') {
            formDataValues[field.field] = false;
          } else if (field.type === 'date_range') {
            formDataValues[field.field] = [];
          } else {
            formDataValues[field.field] = '';
          }
        });
      }
    }
  } catch (error) {
    message.error('获取流程详情失败');
    console.error('Failed to fetch process detail:', error);
  }
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
    process_version: 1,
    form_data: {},
    current_node: '',
    priority: 1
  });
  
  // 清空表单数据
  Object.keys(formDataValues).forEach(key => delete formDataValues[key]);
  
  dueDate.value = null;
  instanceDialog.visible = true;
};

const handleEditInstance = (instance: Instance) => {
  instanceDialog.isEdit = true;
  instanceDialog.instance = JSON.parse(JSON.stringify(instance));
  
  // 解析表单数据
  let instanceFormData: Record<string, any> = {};
  if (typeof instance.form_data === 'string') {
    try {
      instanceFormData = JSON.parse(instance.form_data);
    } catch (e) {
      instanceFormData = {};
    }
  } else {
    instanceFormData = instance.form_data as unknown as Record<string, any>;
  }
  
  // 获取表单字段定义
  fetchProcessFormFields(instance.process_id);
  
  // 复制数据到编辑对象
  Object.keys(instanceFormData).forEach(key => {
    formDataValues[key] = instanceFormData[key];
  });
  
  instanceDialog.visible = true;
  detailDialog.visible = false;
};

const fetchProcessFormFields = async (processId: number) => {
  try {
    const response = await detailProcess({ id: processId });
    if (response) {
      const formDesignResponse = await detailFormDesign({ id: response.form_design_id });
      if (formDesignResponse) {
        const schema: Schema = typeof formDesignResponse.schema === 'string' 
          ? JSON.parse(formDesignResponse.schema) 
          : formDesignResponse.schema;
        
        formFields.value = schema.fields;
      }
    }
  } catch (error) {
    message.error('获取表单字段失败');
    console.error('Failed to fetch form fields:', error);
  }
};

const handleViewInstance = async (instance: Instance) => {
  await fetchInstanceDetail(instance.id);
  detailDialog.visible = true;
};

const handleCommand = (command: string, instance: Instance) => {
  switch (command) {
    case 'submit':
      submitInstance(instance);
      break;
    case 'process':
      handleViewInstance(instance);
      break;
    case 'reject':
      handleViewInstance(instance);
      break;
    case 'delete':
      deleteDialog.instanceId = instance.id;
      deleteDialog.visible = true;
      break;
  }
};

const saveInstance = async () => {
  // 验证必填字段
  if (!selectedProcess.value && !instanceDialog.isEdit) {
    message.error('请选择流程');
    return;
  }
  
  if (!newInstance.title) {
    message.error('请输入工单标题');
    return;
  }
  
  // 验证表单字段
  const missingFields = formFields.value
    .filter((field: Field) => field.required && !formDataValues[field.field])
    .map((field: Field) => field.label);

  if (missingFields.length > 0) {
    message.error(`请填写必填字段: ${missingFields.join(', ')}`);
    return;
  }
  
  try {
    if (instanceDialog.isEdit && instanceDialog.instance) {
      // 构建更新请求
      const updateData: InstanceReq = {
        id: instanceDialog.instance.id,
        title: instanceDialog.instance.title,
        process_id: instanceDialog.instance.process_id,
        process_version: instanceDialog.instance.process_version,
        form_data: formDataValues as FormData,
        current_node: instanceDialog.instance.current_node,
        status: instanceDialog.instance.status,
        priority: instanceDialog.instance.priority
      };
      
      if (dueDate.value) {
        updateData.due_date = dueDate.value.format('YYYY-MM-DD HH:mm:ss');
      }
      
      // 调用更新API（这里假设createInstance也可以用于更新）
      const response = await createInstance(updateData);
      if (response) {
        message.success('工单更新成功');
        instanceDialog.visible = false;
        fetchInstances();
      }
    } else {
      // 创建新实例
      newInstance.form_data = formDataValues as FormData;
      
      if (dueDate.value) {
        newInstance.due_date = dueDate.value.format('YYYY-MM-DD HH:mm:ss');
      }
      
      const response = await createInstance(newInstance);
      if (response) {
        message.success('工单创建成功');
        instanceDialog.visible = false;
        fetchInstances();
      }
    }
  } catch (error) {
    message.error('保存工单失败');
    console.error('Failed to save instance:', error);
  }
};

const submitInstance = async (instance: Instance) => {
  try {
    // 这里应该有一个API来提交工单
    // 假设使用actionInstance
    const flowData: InstanceFlowReq = {
      instance_id: instance.id,
      node_id: instance.current_node,
      node_name: '提交',
      action: 'submit',
      operator_id: 0, // 当前用户ID
      operator_name: '当前用户' // 当前用户名
    };
    
    const response = await actionInstance(flowData);
    if (response) {
      message.success(`工单 #${instance.id} 已提交`);
      fetchInstances();
    }
  } catch (error) {
    message.error('提交工单失败');
    console.error('Failed to submit instance:', error);
  }
};

const processInstance = async (instance: Instance, action: string) => {
  if (!processingComment.value) {
    message.warning('请输入处理意见');
    return;
  }
  
  try {
    const flowData: InstanceFlowReq = {
      instance_id: instance.id,
      node_id: instance.current_node,
      node_name: action === 'approve' ? '批准' : '拒绝',
      action: action,
      operator_id: 0, // 当前用户ID
      operator_name: '当前用户', // 当前用户名
      comment: processingComment.value
    };
    
    const response = await approveInstance(flowData);
    if (response) {
      message.success(`工单 #${instance.id} 已${action === 'approve' ? '批准' : '拒绝'}`);
      detailDialog.visible = false;
      processingComment.value = '';
      fetchInstances();
    }
  } catch (error) {
    message.error('处理工单失败');
    console.error('Failed to process instance:', error);
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
      content: newComment.value,
      creator_id: 0, // 当前用户ID
      creator_name: '当前用户' // 当前用户名
    };
    
    const response = await commentInstance(commentData);
    if (response) {
      message.success('评论已添加');
      newComment.value = '';
      fetchInstanceComments(detailDialog.instance.id);
    }
  } catch (error) {
    message.error('添加评论失败');
    console.error('Failed to add comment:', error);
  }
};

const confirmDelete = async () => {
  try {
    const response = await deleteInstance({ id: deleteDialog.instanceId });
    if (response) {
      message.success(`工单 #${deleteDialog.instanceId} 已删除`);
      deleteDialog.visible = false;
      fetchInstances();
    }
  } catch (error) {
    message.error('删除工单失败');
    console.error('Failed to delete instance:', error);
  }
};

// 辅助方法
const formatDate = (dateStr: string) => {
  if (!dateStr) return '';
  const date = dayjs(dateStr);
  return date.format('YYYY-MM-DD');
};

const formatTime = (dateStr: string) => {
  if (!dateStr) return '';
  const date = dayjs(dateStr);
  return date.format('HH:mm');
};

const formatFullDateTime = (dateStr: string) => {
  if (!dateStr) return '';
  const date = dayjs(dateStr);
  return date.format('YYYY-MM-DD HH:mm:ss');
};

const getInitials = (name: string) => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getStatusClass = (status: number) => {
  switch (status) {
    case 0: return 'status-draft';
    case 1: return 'status-processing';
    case 2: return 'status-completed';
    case 3: return 'status-rejected';
    default: return '';
  }
};

const getStatusColor = (status: number) => {
  switch (status) {
    case 0: return 'orange';
    case 1: return 'blue';
    case 2: return 'green';
    case 3: return 'red';
    default: return 'default';
  }
};

const getStatusText = (status: number) => {
  switch (status) {
    case 0: return '草稿';
    case 1: return '处理中';
    case 2: return '已完成';
    case 3: return '已拒绝';
    default: return '未知';
  }
};

const getPriorityColor = (priority: number) => {
  switch (priority) {
    case 0: return 'green';
    case 1: return 'blue';
    case 2: return 'red';
    default: return 'default';
  }
};

const getPriorityText = (priority: number) => {
  switch (priority) {
    case 0: return '低';
    case 1: return '中';
    case 2: return '高';
    default: return '未知';
  }
};

const getFlowColor = (action: string) => {
  switch (action) {
    case 'submit': return 'blue';
    case 'approve': return 'green';
    case 'reject': return 'red';
    default: return 'gray';
  }
};

const getFlowActionText = (action: string) => {
  switch (action) {
    case 'submit': return '提交';
    case 'approve': return '批准';
    case 'reject': return '拒绝';
    default: return action;
  }
};

const getAvatarColor = (name: string) => {
  // 根据名称生成一致的颜色
  const colors = [
    '#1890ff', '#52c41a', '#faad14', '#f5222d',
    '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'
  ];

  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  return colors[Math.abs(hash) % colors.length];
};

// 初始化
onMounted(() => {
  fetchInstances();
  fetchStatistics();
  fetchProcesses();
  fetchFormDesigns();
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

.form-data-preview {
  margin-top: 24px;
}

.form-data-preview h3 {
  margin-bottom: 16px;
  color: #1f2937;
  font-size: 18px;
}

.flow-records, .comments-section {
  margin-top: 24px;
}

.flow-records h3, .comments-section h3 {
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

.flow-operator, .flow-comment {
  font-size: 14px;
  margin-top: 4px;
}

.comment-list {
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

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.process-selection {
  margin-bottom: 24px;
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
</style>