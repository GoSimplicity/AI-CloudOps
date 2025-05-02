<template>
  <div class="form-instance-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateInstance" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建新实例
        </a-button>
        <a-input-search v-model:value="searchQuery" placeholder="搜索实例..." style="width: 250px" @search="handleSearch"
          allow-clear />
        <a-select v-model:value="statusFilter" placeholder="状态" style="width: 120px" @change="handleStatusChange">
          <a-select-option :value="null">全部</a-select-option>
          <a-select-option :value="0">草稿</a-select-option>
          <a-select-option :value="1">已提交</a-select-option>
          <a-select-option :value="2">已处理</a-select-option>
          <a-select-option :value="3">已拒绝</a-select-option>
        </a-select>
      </div>
    </div>

    <div class="stats-row">
      <a-row :gutter="16">
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="总实例数" :value="stats?.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <FileOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已提交" :value="stats?.submitted" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="待处理" :value="stats?.pending" :value-style="{ color: '#faad14' }">
              <template #prefix>
                <ClockCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已处理" :value="stats?.processed" :value-style="{ color: '#1890ff' }">
              <template #prefix>
                <CheckSquareOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <div class="table-container">
      <a-card>
        <a-table :data-source="paginatedInstances" :columns="columns" :pagination="false" :loading="loading"
          row-key="id" bordered>
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'formName'">
              <div class="form-name-cell">
                <div class="form-badge" :class="getStatusClass(record.status)"></div>
                <div>
                  <div class="form-name-text">{{ record.formName }}</div>
                  <div class="instance-id">#{{ record.id }}</div>
                </div>
              </div>
            </template>

            <template v-if="column.key === 'status'">
              <a-tag :color="getStatusColor(record.status)">
                {{ getStatusText(record.status) }}
              </a-tag>
            </template>

            <template v-if="column.key === 'creator'">
              <div class="creator-info">
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.creatorName) }">
                  {{ getInitials(record.creatorName) }}
                </a-avatar>
                <span class="creator-name">{{ record.creatorName }}</span>
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
                <a-button type="primary" size="small" @click="handleViewInstance(record)">
                  查看
                </a-button>
                <a-button type="default" size="small" @click="handleEditInstance(record)"
                  :disabled="record.status !== 0">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="handleCommand(column.key, record)">
                      <a-menu-item key="submit" v-if="record.status === 0">提交</a-menu-item>
                      <a-menu-item key="process" v-if="record.status === 1">处理</a-menu-item>
                      <a-menu-item key="reject" v-if="record.status === 1">拒绝</a-menu-item>
                      <a-menu-item key="clone">克隆</a-menu-item>
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
          <a-pagination v-model:current="currentPage" :total="totalItems" :page-size="pageSize"
            :page-size-options="['10', '20', '50', '100']" :show-size-changer="true" @change="handleCurrentChange"
            @show-size-change="handleSizeChange" :show-total="(total: number) => `共 ${total} 条`" />
        </div>
      </a-card>
    </div>

    <!-- 表单实例详情对话框 -->
    <a-modal v-model:visible="detailDialog.visible" title="表单实例详情" width="70%" :footer="null" class="detail-dialog">
      <div v-if="detailDialog.instance" class="instance-details">
        <div class="detail-header">
          <h2>{{ detailDialog.instance.formName }}</h2>
          <a-tag :color="getStatusColor(detailDialog.instance.status)">
            {{ getStatusText(detailDialog.instance.status) }}
          </a-tag>
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="实例ID">{{ detailDialog.instance.id }}</a-descriptions-item>
          <a-descriptions-item label="表单ID">{{ detailDialog.instance.formId }}</a-descriptions-item>
          <a-descriptions-item label="提交人">{{ detailDialog.instance.creatorName }}</a-descriptions-item>
          <a-descriptions-item label="提交时间">{{ formatFullDateTime(detailDialog.instance.createdAt)
            }}</a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.processedAt" label="处理时间">
            {{ formatFullDateTime(detailDialog.instance.processedAt) }}
          </a-descriptions-item>
          <a-descriptions-item v-if="detailDialog.instance.handlerName" label="处理人">
            {{ detailDialog.instance.handlerName }}
          </a-descriptions-item>
        </a-descriptions>

        <div class="form-data-preview">
          <h3>表单数据</h3>
          <a-collapse>
            <a-collapse-panel key="1" header="表单内容">
              <a-form layout="vertical">
                <a-form-item v-for="(value, field) in detailDialog.instance.data" :key="field"
                  :label="getFieldLabel(field)">
                  <a-input v-model:value="detailDialog.instance.data[field]" :disabled="true" />
                </a-form-item>
              </a-form>
            </a-collapse-panel>
          </a-collapse>
        </div>

        <div v-if="detailDialog.instance.status === 1" class="action-area">
          <a-divider orientation="left">表单处理</a-divider>
          <a-textarea v-model:value="processingComment" :rows="3" placeholder="请输入处理意见..." />
          <div class="action-buttons mt-16">
            <a-button type="primary" @click="processInstance(detailDialog.instance, 2)">
              批准
            </a-button>
            <a-button danger @click="processInstance(detailDialog.instance, 3)">
              拒绝
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

    <!-- 创建/编辑表单实例对话框 -->
    <a-modal v-model:visible="instanceDialog.visible" :title="instanceDialog.isEdit ? '编辑表单实例' : '创建表单实例'" width="760px"
      @ok="saveInstance" :destroy-on-close="true">
      <div v-if="!selectedForm && !instanceDialog.isEdit" class="form-selection">
        <a-form-item label="选择表单">
          <a-select v-model:value="selectedFormId" placeholder="请选择表单" style="width: 100%" @change="handleSelectForm">
            <a-select-option v-for="form in availableForms" :key="form.id" :value="form.id">
              {{ form.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </div>

      <div v-if="selectedForm || instanceDialog.isEdit" class="instance-form">
        <h3>{{ instanceDialog.isEdit ? (instanceDialog.instance?.formName || '') : selectedForm?.name }}</h3>
        <a-form layout="vertical">
          <a-form-item v-for="field in formFields" :key="field.field" :label="field.label" :name="field.field"
            :rules="[{ required: field.required, message: `请输入${field.label}!` }]">
            <!-- 文本框 -->
            <a-input v-if="field.type === 'text'" v-model:value="instanceData[field.field]"
              :placeholder="`请输入${field.label}`" />

            <!-- 数字输入 -->
            <a-input-number v-else-if="field.type === 'number'" v-model:value="instanceData[field.field]"
              style="width: 100%" :placeholder="`请输入${field.label}`" />

            <!-- 日期选择器 -->
            <a-date-picker v-else-if="field.type === 'date'" v-model:value="instanceData[field.field]"
              style="width: 100%" :placeholder="`请选择${field.label}`" />

            <!-- 下拉选择 -->
            <a-select v-else-if="field.type === 'select'" v-model:value="instanceData[field.field]" style="width: 100%"
              :placeholder="`请选择${field.label}`">
              <a-select-option value="选项1">选项1</a-select-option>
              <a-select-option value="选项2">选项2</a-select-option>
              <a-select-option value="选项3">选项3</a-select-option>
            </a-select>

            <!-- 复选框 -->
            <a-checkbox v-else-if="field.type === 'checkbox'" v-model:checked="instanceData[field.field]">
              {{ field.label }}
            </a-checkbox>

            <!-- 单选框组 -->
            <a-radio-group v-else-if="field.type === 'radio'" v-model:value="instanceData[field.field]">
              <a-radio value="选项1">选项1</a-radio>
              <a-radio value="选项2">选项2</a-radio>
              <a-radio value="选项3">选项3</a-radio>
            </a-radio-group>

            <!-- 多行文本 -->
            <a-textarea v-else-if="field.type === 'textarea'" v-model:value="instanceData[field.field]" :rows="3"
              :placeholder="`请输入${field.label}`" />
          </a-form-item>
        </a-form>
      </div>
    </a-modal>

    <!-- 克隆对话框 -->
    <a-modal v-model:visible="cloneDialog.visible" title="克隆表单实例" @ok="confirmClone" :destroy-on-close="true">
      <p>确定要创建此表单实例的副本吗？</p>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  PlusOutlined,
  FileOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  CheckSquareOutlined,
  DownOutlined
} from '@ant-design/icons-vue';

// 基于Golang模型的类型定义
interface Field {
  type: string;
  label: string;
  field: string;
  required: boolean;
}

interface Schema {
  fields: Field[];
}

interface FormDesign {
  id: number;
  name: string;
  description: string;
  schema: Schema;
  version: number;
  status: number; // 0-草稿，1-已发布，2-已禁用
  categoryID: number;
  creatorID: number;
  creatorName: string;
  createdAt: Date;
  updatedAt: Date;
}

interface FormInstance {
  id: number;
  formId: number;
  formName: string;
  creatorID: number;
  creatorName: string;
  status: number; // 0-草稿，1-已提交，2-已处理，3-已拒绝
  data: Record<string, any>;
  createdAt: Date;
  updatedAt: Date;
  processedAt?: Date;
  handlerID?: number;
  handlerName?: string;
  comment?: string;
}

// 列定义
const columns = [
  {
    title: '表单名称',
    dataIndex: 'formName',
    key: 'formName',
    width: 200,
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 120,
    align: 'center',
  },
  {
    title: '提交人',
    dataIndex: 'creatorName',
    key: 'creator',
    width: 150,
  },
  {
    title: '提交时间',
    dataIndex: 'createdAt',
    key: 'createdAt',
    width: 180,
  },
  {
    title: '处理时间',
    dataIndex: 'processedAt',
    key: 'processedAt',
    width: 180,
    customRender: ({ text }: { text: Date }) => text ? formatDate(text) + ' ' + formatTime(text) : '-'
  },
  {
    title: '操作',
    key: 'action',
    width: 200,
    align: 'center',
  },
];

// 状态数据
const loading = ref(false);
const searchQuery = ref('');
const statusFilter = ref(null);
const currentPage = ref(1);
const pageSize = ref(10);
const processingComment = ref('');

// 统计数据
const stats = reactive({
  total: 76,
  submitted: 45,
  pending: 18,
  processed: 13
});

// 模拟可用表单设计
const availableForms = ref<FormDesign[]>([
  {
    id: 1,
    name: '员工入职表单',
    description: '新员工入职流程使用的表单',
    schema: {
      fields: [
        { type: 'text', label: '姓名', field: 'fullName', required: true },
        { type: 'date', label: '入职日期', field: 'startDate', required: true },
        { type: 'select', label: '部门', field: 'department', required: true },
        { type: 'text', label: '职位', field: 'position', required: true },
        { type: 'textarea', label: '备注', field: 'comments', required: false }
      ]
    },
    version: 2,
    status: 1, // 已发布
    categoryID: 1,
    creatorID: 101,
    creatorName: '张三',
    createdAt: new Date('2025-01-15T08:30:00'),
    updatedAt: new Date('2025-02-10T14:45:00')
  },
  {
    id: 2,
    name: '休假申请表',
    description: '员工申请休假使用的表单',
    schema: {
      fields: [
        { type: 'text', label: '员工姓名', field: 'empName', required: true },
        { type: 'date', label: '开始日期', field: 'startDate', required: true },
        { type: 'date', label: '结束日期', field: 'endDate', required: true },
        { type: 'select', label: '休假类型', field: 'vacationType', required: true },
        { type: 'textarea', label: '原因', field: 'reason', required: false }
      ]
    },
    version: 1,
    status: 1, // 已发布
    categoryID: 2,
    creatorID: 102,
    creatorName: '李四',
    createdAt: new Date('2025-01-20T10:15:00'),
    updatedAt: new Date('2025-01-20T10:15:00')
  },
  {
    id: 3,
    name: 'IT支持请求',
    description: '请求IT支持和报告问题使用的表单',
    schema: {
      fields: [
        { type: 'text', label: '申请人姓名', field: 'requesterName', required: true },
        { type: 'select', label: '问题类别', field: 'issueCategory', required: true },
        { type: 'radio', label: '优先级', field: 'priority', required: true },
        { type: 'textarea', label: '描述', field: 'description', required: true },
        { type: 'checkbox', label: '需要后续跟进', field: 'followUp', required: false }
      ]
    },
    version: 3,
    status: 1, // 已发布
    categoryID: 3,
    creatorID: 103,
    creatorName: '王五',
    createdAt: new Date('2025-01-05T09:20:00'),
    updatedAt: new Date('2025-03-15T11:30:00')
  }
]);

// 模拟表单实例数据
const formInstances = ref<FormInstance[]>([
  {
    id: 1001,
    formId: 1,
    formName: '员工入职表单',
    creatorID: 201,
    creatorName: '赵强',
    status: 2, // 已处理
    data: {
      fullName: '赵强',
      startDate: '2025-03-15',
      department: '研发部',
      position: '高级开发工程师',
      comments: '有5年相关经验'
    },
    createdAt: new Date('2025-03-10T09:30:00'),
    updatedAt: new Date('2025-03-15T14:20:00'),
    processedAt: new Date('2025-03-15T14:20:00'),
    handlerID: 101,
    handlerName: '张三',
    comment: '审核通过，相关资料齐全'
  },
  {
    id: 1002,
    formId: 2,
    formName: '休假申请表',
    creatorID: 202,
    creatorName: '孙明',
    status: 1, // 已提交
    data: {
      empName: '孙明',
      startDate: '2025-04-05',
      endDate: '2025-04-12',
      vacationType: '年假',
      reason: '家庭旅行'
    },
    createdAt: new Date('2025-03-20T11:45:00'),
    updatedAt: new Date('2025-03-20T11:45:00')
  },
  {
    id: 1003,
    formId: 3,
    formName: 'IT支持请求',
    creatorID: 203,
    creatorName: '李娜',
    status: 3, // 已拒绝
    data: {
      requesterName: '李娜',
      issueCategory: '软件问题',
      priority: '高',
      description: '无法访问共享文件夹',
      followUp: true
    },
    createdAt: new Date('2025-03-18T15:30:00'),
    updatedAt: new Date('2025-03-19T10:15:00'),
    processedAt: new Date('2025-03-19T10:15:00'),
    handlerID: 103,
    handlerName: '王五',
    comment: '请检查您的网络连接，问题可能是由于网络故障导致的'
  },
  {
    id: 1004,
    formId: 1,
    formName: '员工入职表单',
    creatorID: 204,
    creatorName: '张伟',
    status: 0, // 草稿
    data: {
      fullName: '张伟',
      startDate: '2025-04-01',
      department: '市场部',
      position: '市场专员',
      comments: ''
    },
    createdAt: new Date('2025-03-22T16:20:00'),
    updatedAt: new Date('2025-03-22T16:20:00')
  },
  {
    id: 1005,
    formId: 2,
    formName: '休假申请表',
    creatorID: 205,
    creatorName: '王芳',
    status: 2, // 已处理
    data: {
      empName: '王芳',
      startDate: '2025-03-25',
      endDate: '2025-03-26',
      vacationType: '病假',
      reason: '看医生'
    },
    createdAt: new Date('2025-03-23T09:15:00'),
    updatedAt: new Date('2025-03-23T14:30:00'),
    processedAt: new Date('2025-03-23T14:30:00'),
    handlerID: 102,
    handlerName: '李四',
    comment: '已批准'
  },
  {
    id: 1006,
    formId: 3,
    formName: 'IT支持请求',
    creatorID: 206,
    creatorName: '刘洋',
    status: 1, // 已提交
    data: {
      requesterName: '刘洋',
      issueCategory: '硬件问题',
      priority: '中',
      description: '键盘部分按键失灵',
      followUp: false
    },
    createdAt: new Date('2025-03-24T10:45:00'),
    updatedAt: new Date('2025-03-24T10:45:00')
  },
  {
    id: 1007,
    formId: 1,
    formName: '员工入职表单',
    creatorID: 207,
    creatorName: '周静',
    status: 1, // 已提交
    data: {
      fullName: '周静',
      startDate: '2025-04-15',
      department: '财务部',
      position: '财务助理',
      comments: '本科会计专业毕业'
    },
    createdAt: new Date('2025-03-25T13:20:00'),
    updatedAt: new Date('2025-03-25T13:20:00')
  }
]);

// 详情对话框
const detailDialog = reactive({
  visible: false,
  instance: null as FormInstance | null
});

// 实例创建/编辑对话框
const instanceDialog = reactive({
  visible: false,
  isEdit: false,
  instance: null as FormInstance | null
});

// 克隆对话框
const cloneDialog = reactive({
  visible: false,
  instanceId: 0
});

// 表单选择和实例数据
const selectedFormId = ref<number | null>(null);
const selectedForm = ref<FormDesign | null>(null);
const instanceData = reactive<Record<string, any>>({});

// 过滤和分页
const filteredInstances = computed(() => {
  let result = [...formInstances.value];

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(instance =>
      instance.formName.toLowerCase().includes(query) ||
      instance.creatorName.toLowerCase().includes(query)
    );
  }

  if (statusFilter.value !== null) {
    result = result.filter(instance => instance.status === statusFilter.value);
  }

  return result.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
});

const totalItems = computed(() => filteredInstances.value.length);

const paginatedInstances = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return filteredInstances.value.slice(start, end);
});

const formFields = computed(() => {
  if (instanceDialog.isEdit && instanceDialog.instance) {
    const form = availableForms.value.find(f => f.id === instanceDialog.instance?.formId);
    return form?.schema.fields || [];
  }
  return selectedForm.value?.schema.fields || [];
});

// 方法
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

const handleCreateInstance = () => {
  instanceDialog.isEdit = false;
  instanceDialog.instance = null;
  selectedFormId.value = null;
  selectedForm.value = null;

  // 清空实例数据
  Object.keys(instanceData).forEach(key => delete instanceData[key]);

  instanceDialog.visible = true;
};

const handleSelectForm = (id: number) => {
  selectedForm.value = availableForms.value.find(form => form.id === id) || null;

  // 预设字段初始值
  if (selectedForm.value) {
    selectedForm.value.schema.fields.forEach(field => {
      if (field.type === 'checkbox') {
        instanceData[field.field] = false;
      } else {
        instanceData[field.field] = '';
      }
    });
  }
};

const handleEditInstance = (instance: FormInstance) => {
  instanceDialog.isEdit = true;
  instanceDialog.instance = JSON.parse(JSON.stringify(instance));

  // 复制数据到编辑对象
  Object.keys(instance.data).forEach(key => {
    instanceData[key] = instance.data[key];
  });

  instanceDialog.visible = true;
  detailDialog.visible = false;
};

const handleViewInstance = (instance: FormInstance) => {
  detailDialog.instance = instance;
  detailDialog.visible = true;
};

const handleCommand = (command: string, instance: FormInstance) => {
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
    case 'clone':
      cloneDialog.instanceId = instance.id;
      cloneDialog.visible = true;
      break;
    case 'delete':
      confirmDelete(instance);
      break;
  }
};

const saveInstance = () => {
  // 验证必填字段
  const form = instanceDialog.isEdit ?
    availableForms.value.find(f => f.id === instanceDialog.instance?.formId) :
    selectedForm.value;

  if (!form) {
    message.error('请选择表单');
    return;
  }

  const missingFields = form.schema.fields
    .filter(field => field.required && !instanceData[field.field])
    .map(field => field.label);

  if (missingFields.length > 0) {
    message.error(`请填写必填字段: ${missingFields.join(', ')}`);
    return;
  }

  if (instanceDialog.isEdit && instanceDialog.instance) {
    // 更新实例
    const index = formInstances.value.findIndex(i => i.id === instanceDialog.instance?.id);
    if (index !== -1) {
      const instance = formInstances.value[index];
      if (instance) {
        instance.data = { ...instanceData };
        instance.updatedAt = new Date();
        message.success('表单实例已更新');
      }
    }
  } else {
    // 创建新实例
    if (!selectedForm.value) {
      message.error('请选择表单');
      return;
    }

    const newId = Math.max(...formInstances.value.map(i => i.id)) + 1;
    const newInstance: FormInstance = {
      id: newId,
      formId: selectedForm.value.id,
      formName: selectedForm.value.name,
      creatorID: 201, // 模拟当前用户ID
      creatorName: '当前用户', // 模拟当前用户名
      status: 0, // 草稿
      data: { ...instanceData },
      createdAt: new Date(),
      updatedAt: new Date()
    };

    formInstances.value.push(newInstance);
    message.success('表单实例已创建');
  }

  instanceDialog.visible = false;
};

const submitInstance = (instance: FormInstance) => {
  const index = formInstances.value.findIndex(i => i.id === instance.id);
  if (index !== -1) {
    const instance = formInstances.value[index];
    if (instance) {
      instance.status = 1; // 已提交
      instance.updatedAt = new Date();
      message.success(`表单实例 #${instance.id} 已提交`);
    }
  }
};

const processInstance = (instance: FormInstance, newStatus: number) => {
  const index = formInstances.value.findIndex(i => i.id === instance.id);
  if (index !== -1) {
    const formInstance = formInstances.value[index];
    if (formInstance) {
      formInstance.status = newStatus;
      formInstance.updatedAt = new Date();
      formInstance.processedAt = new Date();
      formInstance.handlerID = 101; // 模拟处理人ID
      formInstance.handlerName = '当前用户'; // 模拟处理人姓名
      formInstance.comment = processingComment.value;

      message.success(`表单实例 #${instance.id} 已${newStatus === 2 ? '批准' : '拒绝'}`);
      detailDialog.visible = false;
      processingComment.value = '';
    }
  }
};

const confirmClone = () => {
  const originalInstance = formInstances.value.find(i => i.id === cloneDialog.instanceId);
  if (originalInstance) {
    const newId = Math.max(...formInstances.value.map(i => i.id)) + 1;
    const clonedInstance: FormInstance = {
      ...JSON.parse(JSON.stringify(originalInstance)),
      id: newId,
      status: 0, // 总是草稿
      createdAt: new Date(),
      updatedAt: new Date(),
      processedAt: undefined,
      handlerID: undefined,
      handlerName: undefined,
      comment: undefined
    };

    formInstances.value.push(clonedInstance);
    cloneDialog.visible = false;
    message.success(`表单实例 #${originalInstance.id} 的副本已创建`);
  }
};

const confirmDelete = (instance: FormInstance) => {
  Modal.confirm({
    title: '警告',
    content: `确定要删除表单实例 #${instance.id} 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      const index = formInstances.value.findIndex(i => i.id === instance.id);
      if (index !== -1) {
        formInstances.value.splice(index, 1);
        message.success(`表单实例 #${instance.id} 已删除`);
      }
    }
  });
};

// 辅助方法
const formatDate = (date: Date) => {
  if (!date) return '';
  const d = new Date(date);
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const formatTime = (date: Date) => {
  if (!date) return '';
  const d = new Date(date);
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
};

const formatFullDateTime = (date: Date) => {
  if (!date) return '';
  const d = new Date(date);
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
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
    case 1: return 'status-submitted';
    case 2: return 'status-processed';
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
    case 1: return '已提交';
    case 2: return '已处理';
    case 3: return '已拒绝';
    default: return '未知';
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

const getFieldLabel = (field: string) => {
  if (!detailDialog.instance) return field;

  const form = availableForms.value.find(f => f.id === detailDialog.instance?.formId);
  const fieldDef = form?.schema.fields.find(f => f.field === field);
  return fieldDef ? fieldDef.label : field;
};

// 模拟初始化
onMounted(() => {
  loading.value = true;
  // 模拟API加载
  setTimeout(() => {
    loading.value = false;
  }, 800);
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

.page-title {
  font-size: 28px;
  color: #1f2937;
  margin: 0;
  background: linear-gradient(90deg, #1890ff 0%, #13c2c2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  font-weight: 700;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.btn-create {
  background: linear-gradient(135deg, #1890ff 0%);
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

.status-submitted {
  background-color: #1890ff;
}

.status-processed {
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

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.form-selection {
  margin-bottom: 24px;
}

.instance-form h3 {
  margin-bottom: 16px;
  font-size: 18px;
  color: #1f2937;
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
