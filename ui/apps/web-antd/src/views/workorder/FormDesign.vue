<template>
  <div class="form-design-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateForm" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建新表单
        </a-button>
        <a-input-search v-model:value="searchQuery" placeholder="搜索表单..." style="width: 250px" @search="handleSearch"
          allow-clear />
        <a-select v-model:value="statusFilter" placeholder="状态" style="width: 120px" @change="handleStatusChange">
          <a-select-option :value="null">全部</a-select-option>
          <a-select-option :value="0">草稿</a-select-option>
          <a-select-option :value="1">已发布</a-select-option>
          <a-select-option :value="2">已禁用</a-select-option>
        </a-select>
      </div>
    </div>

    <div class="stats-row">
      <a-row :gutter="16">
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="总表单数" :value="stats?.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <FormOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已发布" :value="stats?.published" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="草稿" :value="stats?.draft" :value-style="{ color: '#faad14' }">
              <template #prefix>
                <EditOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已禁用" :value="stats?.disabled" :value-style="{ color: '#cf1322' }">
              <template #prefix>
                <StopOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <div class="table-container">
      <a-card>
        <a-table :data-source="paginatedForms" :columns="columns" :pagination="false" :loading="loading" row-key="id"
          bordered>
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="form-name-cell">
                <div class="form-badge" :class="getStatusClass(record.status)"></div>
                <span class="form-name-text">{{ record.name }}</span>
              </div>
            </template>

            <template v-if="column.key === 'description'">
              <span class="description-text">{{ record.description || '无描述' }}</span>
            </template>

            <template v-if="column.key === 'version'">
              <a-tag color="blue">v{{ record.version }}</a-tag>
            </template>

            <template v-if="column.key === 'status'">
              <a-tag :color="record.status === 0 ? 'orange' : record.status === 1 ? 'green' : 'default'">
                {{ record.status === 0 ? '草稿' : record.status === 1 ? '已发布' : '已禁用' }}
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
                <a-button type="primary" size="small" @click="handleViewForm(record)">
                  查看
                </a-button>
                <a-button type="default" size="small" @click="handleEditForm(record)">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="handleCommand">
                      <a-menu-item key="publish" v-if="record.status === 0">发布</a-menu-item>
                      <a-menu-item key="unpublish" v-if="record.status === 1">取消发布</a-menu-item>
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

    <!-- 表单创建/编辑对话框 -->
    <a-modal v-model:visible="formDialog.visible" :title="formDialog.isEdit ? '编辑表单设计' : '创建表单设计'" width="760px"
      @ok="saveForm" :destroy-on-close="true">
      <a-form ref="formRef" :model="formDialog.form" :rules="formRules" layout="vertical">
        <a-form-item label="表单名称" name="name">
          <a-input v-model:value="formDialog.form.name" placeholder="请输入表单名称" />
        </a-form-item>

        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="formDialog.form.description" :rows="3" placeholder="请输入表单描述" />
        </a-form-item>

        <a-form-item label="分类" name="categoryID">
          <a-select v-model:value="formDialog.form.categoryID" placeholder="请选择分类" style="width: 100%">
            <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
              {{ cat.name }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="状态" name="status">
          <a-radio-group v-model:value="formDialog.form.status">
            <a-radio :value="0">草稿</a-radio>
            <a-radio :value="1">已发布</a-radio>
            <a-radio :value="2">已禁用</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-divider orientation="left">表单结构</a-divider>

        <div class="schema-editor">
          <div class="field-list">
            <a-collapse>
              <a-collapse-panel v-for="(field, index) in formDialog.form.schema.fields" :key="index"
                :header="field.label || `字段 ${index + 1}`">
                <template #extra>
                  <a-button type="text" danger @click.stop="removeField(index)" size="small">
                    <DeleteOutlined />
                  </a-button>
                </template>

                <a-form-item label="字段类型">
                  <a-select v-model:value="field.type" style="width: 100%">
                    <a-select-option value="text">文本框</a-select-option>
                    <a-select-option value="number">数字</a-select-option>
                    <a-select-option value="date">日期</a-select-option>
                    <a-select-option value="select">下拉选择</a-select-option>
                    <a-select-option value="checkbox">复选框</a-select-option>
                    <a-select-option value="radio">单选框</a-select-option>
                    <a-select-option value="textarea">多行文本</a-select-option>
                  </a-select>
                </a-form-item>

                <a-form-item label="标签名称">
                  <a-input v-model:value="field.label" placeholder="字段标签" />
                </a-form-item>

                <a-form-item label="字段名称">
                  <a-input v-model:value="field.field" placeholder="字段名称" />
                </a-form-item>

                <a-form-item label="是否必填">
                  <a-switch v-model:checked="field.required" />
                </a-form-item>
              </a-collapse-panel>
            </a-collapse>

            <div class="add-field-button">
              <a-button type="dashed" block @click="addField" style="margin-top: 16px">
                <PlusOutlined /> 添加字段
              </a-button>
            </div>
          </div>
        </div>
      </a-form>
    </a-modal>

    <!-- 克隆对话框 -->
    <a-modal v-model:visible="cloneDialog.visible" title="克隆表单" @ok="confirmClone" :destroy-on-close="true">
      <a-form :model="cloneDialog.form" layout="vertical">
        <a-form-item label="新表单名称" name="name">
          <a-input v-model:value="cloneDialog.form.name" placeholder="请输入新表单名称" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情对话框 -->
    <a-modal v-model:visible="detailDialog.visible" title="表单详情" width="70%" :footer="null" class="detail-dialog">
      <div v-if="detailDialog.form" class="form-details">
        <div class="detail-header">
          <h2>{{ detailDialog.form.name }}</h2>
          <a-tag
            :color="detailDialog.form.status === 0 ? 'orange' : detailDialog.form.status === 1 ? 'green' : 'default'">
            {{ detailDialog.form.status === 0 ? '草稿' : detailDialog.form.status === 1 ? '已发布' : '已禁用' }}
          </a-tag>
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="ID">{{ detailDialog.form.id }}</a-descriptions-item>
          <a-descriptions-item label="版本">v{{ detailDialog.form.version }}</a-descriptions-item>
          <a-descriptions-item label="创建人">{{ detailDialog.form.creatorName }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.form.createdAt) }}</a-descriptions-item>
          <a-descriptions-item label="描述" :span="2">{{ detailDialog.form.description || '无描述' }}</a-descriptions-item>
        </a-descriptions>

        <div class="schema-preview">
          <h3>表单结构</h3>
          <a-table :data-source="detailDialog.form.schema.fields" :columns="schemaColumns" :pagination="false" bordered
            size="small" row-key="field">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'required'">
                <a-tag :color="record.required ? 'red' : ''">
                  {{ record.required ? '必填' : '可选' }}
                </a-tag>
              </template>
            </template>
          </a-table>
        </div>

        <div class="detail-footer">
          <a-button @click="detailDialog.visible = false">关闭</a-button>
          <a-button type="primary" @click="handleEditForm(detailDialog.form)">编辑</a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  PlusOutlined,
  FormOutlined,
  CheckCircleOutlined,
  EditOutlined,
  StopOutlined,
  DeleteOutlined,
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

interface Category {
  id: number;
  name: string;
}

// 列定义
const columns = [
  {
    title: '表单名称',
    dataIndex: 'name',
    key: 'name',
    width: 180,
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
    width: 200,
    ellipsis: true,
  },
  {
    title: '版本',
    dataIndex: 'version',
    key: 'version',
    width: 100,
    align: 'center',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 120,
    align: 'center',
  },
  {
    title: '创建人',
    dataIndex: 'creatorName',
    key: 'creator',
    width: 150,
  },
  {
    title: '创建时间',
    dataIndex: 'createdAt',
    key: 'createdAt',
    width: 180,
  },
  {
    title: '操作',
    key: 'action',
    width: 200,
    align: 'center',
  },
];

// 表单结构列定义
const schemaColumns = [
  {
    title: '类型',
    dataIndex: 'type',
    key: 'type',
    width: 120,
  },
  {
    title: '标签',
    dataIndex: 'label',
    key: 'label',
    width: 180,
  },
  {
    title: '字段名',
    dataIndex: 'field',
    key: 'field',
    width: 180,
  },
  {
    title: '是否必填',
    dataIndex: 'required',
    key: 'required',
    width: 100,
  },
];

// 状态数据
const loading = ref(false);
const searchQuery = ref('');
const statusFilter = ref(null);
const currentPage = ref(1);
const pageSize = ref(10);

// 统计数据
const stats = reactive({
  total: 48,
  published: 32,
  draft: 12,
  disabled: 4
});

// 模拟表单设计数据
const formDesigns = ref<FormDesign[]>([
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
  },
  {
    id: 4,
    name: '设备采购申请',
    description: '申请采购新设备的表单',
    schema: {
      fields: [
        { type: 'text', label: '申请人姓名', field: 'requesterName', required: true },
        { type: 'text', label: '物品描述', field: 'itemDescription', required: true },
        { type: 'number', label: '预计费用', field: 'estimatedCost', required: true },
        { type: 'select', label: '部门', field: 'department', required: true },
        { type: 'textarea', label: '申请理由', field: 'justification', required: true }
      ]
    },
    version: 1,
    status: 0, // 草稿
    categoryID: 4,
    creatorID: 104,
    creatorName: '赵六',
    createdAt: new Date('2025-03-10T16:45:00'),
    updatedAt: new Date('2025-03-10T16:45:00')
  },
  {
    id: 5,
    name: '差旅报销单',
    description: '提交差旅费用报销的表单',
    schema: {
      fields: [
        { type: 'text', label: '员工姓名', field: 'empName', required: true },
        { type: 'date', label: '出行日期', field: 'travelDate', required: true },
        { type: 'text', label: '目的地', field: 'destination', required: true },
        { type: 'number', label: '总金额', field: 'totalAmount', required: true },
        { type: 'textarea', label: '行程概要', field: 'tripSummary', required: true }
      ]
    },
    version: 2,
    status: 1, // 已发布
    categoryID: 2,
    creatorID: 105,
    creatorName: '钱七',
    createdAt: new Date('2025-02-05T13:20:00'),
    updatedAt: new Date('2025-03-01T09:10:00')
  },
  {
    id: 6,
    name: '项目提案表',
    description: '提交新项目提案的表单',
    schema: {
      fields: [
        { type: 'text', label: '项目标题', field: 'projectTitle', required: true },
        { type: 'text', label: '项目负责人', field: 'projectLead', required: true },
        { type: 'date', label: '预计开始日期', field: 'startDate', required: true },
        { type: 'number', label: '预算估计', field: 'budget', required: true },
        { type: 'textarea', label: '项目描述', field: 'description', required: true },
        { type: 'select', label: '部门', field: 'department', required: true }
      ]
    },
    version: 1,
    status: 0, // 草稿
    categoryID: 5,
    creatorID: 106,
    creatorName: '孙八',
    createdAt: new Date('2025-03-15T11:00:00'),
    updatedAt: new Date('2025-03-15T11:00:00')
  },
  {
    id: 7,
    name: '绩效评估表',
    description: '年度员工绩效评估表单',
    schema: {
      fields: [
        { type: 'text', label: '员工姓名', field: 'empName', required: true },
        { type: 'text', label: '经理姓名', field: 'managerName', required: true },
        { type: 'date', label: '评估周期开始', field: 'periodStart', required: true },
        { type: 'date', label: '评估周期结束', field: 'periodEnd', required: true },
        { type: 'select', label: '总体评级', field: 'overallRating', required: true },
        { type: 'textarea', label: '优势', field: 'strengths', required: true },
        { type: 'textarea', label: '需改进方面', field: 'improvements', required: true }
      ]
    },
    version: 3,
    status: 1, // 已发布
    categoryID: 1,
    creatorID: 107,
    creatorName: '周九',
    createdAt: new Date('2024-11-20T14:30:00'),
    updatedAt: new Date('2025-02-25T09:45:00')
  },
  {
    id: 8,
    name: '客户反馈调查',
    description: '收集客户反馈的表单',
    schema: {
      fields: [
        { type: 'text', label: '客户名称', field: 'customerName', required: false },
        { type: 'radio', label: '整体满意度', field: 'satisfaction', required: true },
        { type: 'checkbox', label: '使用的服务', field: 'services', required: true },
        { type: 'textarea', label: '评论', field: 'comments', required: false },
        { type: 'checkbox', label: '是否需要后续联系', field: 'followUp', required: false }
      ]
    },
    version: 2,
    status: 2, // 已禁用
    categoryID: 6,
    creatorID: 108,
    creatorName: '吴十',
    createdAt: new Date('2025-01-12T15:20:00'),
    updatedAt: new Date('2025-03-05T16:30:00')
  },
  {
    id: 9,
    name: '供应商注册表',
    description: '注册新供应商的表单',
    schema: {
      fields: [
        { type: 'text', label: '公司名称', field: 'companyName', required: true },
        { type: 'text', label: '联系人', field: 'contactPerson', required: true },
        { type: 'text', label: '电子邮件', field: 'email', required: true },
        { type: 'text', label: '电话', field: 'phone', required: true },
        { type: 'textarea', label: '公司简介', field: 'description', required: true },
        { type: 'select', label: '行业', field: 'industry', required: true }
      ]
    },
    version: 1,
    status: 1, // 已发布
    categoryID: 7,
    creatorID: 109,
    creatorName: '郑十一',
    createdAt: new Date('2025-02-18T13:40:00'),
    updatedAt: new Date('2025-02-18T13:40:00')
  },
  {
    id: 10,
    name: '培训申请表',
    description: '申请员工培训项目的表单',
    schema: {
      fields: [
        { type: 'text', label: '员工姓名', field: 'empName', required: true },
        { type: 'select', label: '培训类型', field: 'trainingType', required: true },
        { type: 'date', label: '申请日期', field: 'requestedDate', required: true },
        { type: 'number', label: '预计费用', field: 'estimatedCost', required: true },
        { type: 'textarea', label: '申请理由', field: 'justification', required: true }
      ]
    },
    version: 1,
    status: 0, // 草稿
    categoryID: 1,
    creatorID: 110,
    creatorName: '刘十二',
    createdAt: new Date('2025-03-20T10:30:00'),
    updatedAt: new Date('2025-03-20T10:30:00')
  }
]);

// 模拟分类数据
const categories = ref<Category[]>([
  { id: 1, name: '人力资源' },
  { id: 2, name: '财务部门' },
  { id: 3, name: 'IT部门' },
  { id: 4, name: '运营部门' },
  { id: 5, name: '项目管理' },
  { id: 6, name: '客户服务' },
  { id: 7, name: '采购部门' }
]);

// 过滤和分页
const filteredForms = computed(() => {
  let result = [...formDesigns.value];

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(form =>
      form.name.toLowerCase().includes(query) ||
      (form.description && form.description.toLowerCase().includes(query))
    );
  }

  if (statusFilter.value !== null) {
    result = result.filter(form => form.status === statusFilter.value);
  }

  return result;
});

const totalItems = computed(() => filteredForms.value.length);

const paginatedForms = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return filteredForms.value.slice(start, end);
});

// 表单对话框
const formDialog = reactive({
  visible: false,
  isEdit: false,
  form: {
    id: 0,
    name: '',
    description: '',
    schema: {
      fields: [] as Field[]
    },
    version: 1,
    status: 0,
    categoryID: null as number | null,
    creatorID: 101, // 模拟用户ID
    creatorName: '当前用户', // 模拟用户名
    createdAt: new Date(),
    updatedAt: new Date()
  }
});

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入表单名称', trigger: 'blur' },
    { min: 3, max: 50, message: '长度应为3到50个字符', trigger: 'blur' }
  ],
  categoryID: [
    { required: true, message: '请选择分类', trigger: 'change' }
  ]
};

// 克隆对话框
const cloneDialog = reactive({
  visible: false,
  form: {
    name: '',
    originalId: 0
  }
});

// 详情对话框
const detailDialog = reactive({
  visible: false,
  form: null as FormDesign | null
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

const handleCreateForm = () => {
  formDialog.isEdit = false;
  formDialog.form = {
    id: 0,
    name: '',
    description: '',
    schema: {
      fields: []
    },
    version: 1,
    status: 0,
    categoryID: null,
    creatorID: 101,
    creatorName: '当前用户',
    createdAt: new Date(),
    updatedAt: new Date()
  };
  formDialog.visible = true;
};

const handleEditForm = (row: FormDesign) => {
  formDialog.isEdit = true;
  formDialog.form = JSON.parse(JSON.stringify(row));
  formDialog.visible = true;
  detailDialog.visible = false;
};

const handleViewForm = (row: FormDesign) => {
  detailDialog.form = row;
  detailDialog.visible = true;
};

const handleCommand = (command: string, row: FormDesign) => {
  switch (command) {
    case 'publish':
      publishForm(row);
      break;
    case 'unpublish':
      unpublishForm(row);
      break;
    case 'clone':
      showCloneDialog(row);
      break;
    case 'delete':
      confirmDelete(row);
      break;
  }
};

const publishForm = (form: FormDesign) => {
  const index = formDesigns.value.findIndex(f => f.id === form.id);
  if (index !== -1) {
    const formDesign = formDesigns.value[index];
    if (formDesign) {
      formDesign.status = 1;
      formDesign.updatedAt = new Date();
      message.success(`表单 "${form.name}" 已发布`);
    }
  }
};

const unpublishForm = (form: FormDesign) => {
  const index = formDesigns.value.findIndex(f => f.id === form.id);
  if (index !== -1) {
    const formDesign = formDesigns.value[index];
    if (formDesign) {
      formDesign.status = 0;
      formDesign.updatedAt = new Date();
      message.success(`表单 "${form.name}" 已取消发布`);
    }
  }
};

const showCloneDialog = (form: FormDesign) => {
  cloneDialog.form.name = `${form.name} 的副本`;
  cloneDialog.form.originalId = form.id;
  cloneDialog.visible = true;
};

const confirmClone = () => {
  const originalForm = formDesigns.value.find(f => f.id === cloneDialog.form.originalId);
  if (originalForm) {
    const newId = Math.max(...formDesigns.value.map(f => f.id)) + 1;
    const clonedForm: FormDesign = {
      ...JSON.parse(JSON.stringify(originalForm)),
      id: newId,
      name: cloneDialog.form.name,
      status: 0, // 总是草稿
      version: 1,
      createdAt: new Date(),
      updatedAt: new Date()
    };

    formDesigns.value.push(clonedForm);
    cloneDialog.visible = false;
    message.success(`表单 "${originalForm.name}" 已克隆为 "${cloneDialog.form.name}"`);
  }
};

const confirmDelete = (form: FormDesign) => {
  Modal.confirm({
    title: '警告',
    content: `确定要删除表单 "${form.name}" 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk() {
      const index = formDesigns.value.findIndex(f => f.id === form.id);
      if (index !== -1) {
        formDesigns.value.splice(index, 1);
        message.success(`表单 "${form.name}" 已删除`);
      }
    }
  });
};

const addField = () => {
  formDialog.form.schema.fields.push({
    type: 'text',
    label: '',
    field: '',
    required: false
  });
};

const removeField = (index: number) => {
  formDialog.form.schema.fields.splice(index, 1);
};

const saveForm = () => {
  if (formDialog.form.name.trim() === '') {
    message.error('表单名称不能为空');
    return;
  }

  if (formDialog.form.categoryID === null) {
    message.error('请选择分类');
    return;
  }

  if (formDialog.isEdit) {
    // 更新现有表单
    const index = formDesigns.value.findIndex(f => f.id === formDialog.form.id);
    if (index !== -1) {
      formDialog.form.updatedAt = new Date();
      formDesigns.value[index] = { ...formDialog.form } as FormDesign;
      message.success(`表单 "${formDialog.form.name}" 已更新`);
    }
  } else {
    // 创建新表单
    const newId = Math.max(...formDesigns.value.map(f => f.id)) + 1;
    formDialog.form.id = newId;
    formDesigns.value.push({ ...formDialog.form } as FormDesign);
    message.success(`表单 "${formDialog.form.name}" 已创建`);
  }
  formDialog.visible = false;
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
    case 1: return 'status-published';
    case 2: return 'status-disabled';
    default: return '';
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
.form-design-container {
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
  background: linear-gradient(90deg, #1890ff 0%, #52c41a 100%);
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

.status-published {
  background-color: #52c41a;
}

.status-disabled {
  background-color: #d9d9d9;
}

.form-name-text {
  font-weight: 500;
}

.description-text {
  color: #606266;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
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

.schema-editor {
  border-radius: 4px;
  padding: 16px;
  margin-bottom: 20px;
}

.field-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.add-field-button {
  text-align: center;
  margin-top: 16px;
}

.detail-dialog .form-details {
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

.schema-preview {
  margin-top: 24px;
}

.schema-preview h3 {
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
</style>
