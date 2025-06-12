<template>
  <div class="template-manager-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateTemplate" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建新模板
        </a-button>
        <a-input-search v-model:value="searchQuery" placeholder="搜索模板..." style="width: 250px" @search="handleSearch"
          allow-clear />
        <a-select v-model:value="categoryFilter" placeholder="分类" style="width: 120px" @change="handleCategoryChange">
          <a-select-option :value="null">全部分类</a-select-option>
          <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
            {{ cat.name }}
          </a-select-option>
        </a-select>
        <a-select v-model:value="statusFilter" placeholder="状态" style="width: 120px" @change="handleStatusChange">
          <a-select-option :value="null">全部状态</a-select-option>
          <a-select-option :value="1">启用</a-select-option>
          <a-select-option :value="0">禁用</a-select-option>
        </a-select>
      </div>
    </div>

    <div class="stats-row">
      <a-row :gutter="16">
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="模板总数" :value="stats.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <FileOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="常规模板" :value="stats.regular" :value-style="{ color: '#1890ff' }">
              <template #prefix>
                <FileTextOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="系统模板" :value="stats.system" :value-style="{ color: '#722ed1' }">
              <template #prefix>
                <SettingOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="近7天新增" :value="stats.recentAdded" :value-style="{ color: '#fa8c16' }">
              <template #prefix>
                <PlusCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <div class="table-container">
      <a-card>
        <a-table :data-source="templates" :columns="columns" :pagination="false" :loading="loading"
          row-key="id" bordered>
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="template-name-cell">
                <div class="template-badge" :class="record.status ? 'status-enabled' : 'status-disabled'"></div>
                <span class="template-name-text">{{ record.name }}</span>
                <a-tag v-if="isSystemTemplate(record)" color="purple" size="small">系统</a-tag>
              </div>
            </template>

            <template v-if="column.key === 'description'">
              <span class="description-text">{{ record.description || '无描述' }}</span>
            </template>

            <template v-if="column.key === 'category'">
              <a-tag :color="getCategoryColor(record.category_id)">{{ getCategoryName(record.category_id) }}</a-tag>
            </template>

            <template v-if="column.key === 'status'">
              <a-switch :checked="record.status === 1" disabled />
            </template>

            <template v-if="column.key === 'creator'">
              <div class="creator-info">
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.creator_name) }">
                  {{ getInitials(record.creator_name) }}
                </a-avatar>
                <span class="creator-name">{{ record.creator_name }}</span>
              </div>
            </template>

            <template v-if="column.key === 'createdAt'">
              <div class="date-info">
                <span class="date">{{ formatDate(record.created_at) }}</span>
                <span class="time">{{ formatTime(record.created_at) }}</span>
              </div>
            </template>

            <template v-if="column.key === 'action'">
              <div class="action-buttons">
                <a-button type="primary" size="small" @click="handlePreviewTemplate(record)">
                  预览
                </a-button>
                <a-button type="default" size="small" @click="handleEditTemplate(record)" :disabled="isSystemTemplate(record)">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleCommand(e.key, record)">
                      <a-menu-item key="enable" v-if="record.status === 0">启用</a-menu-item>
                      <a-menu-item key="disable" v-if="record.status === 1">禁用</a-menu-item>
                      <a-menu-item key="clone">克隆</a-menu-item>
                      <a-menu-divider />
                      <a-menu-item key="delete" danger :disabled="isSystemTemplate(record)">删除</a-menu-item>
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
          <a-pagination v-model:current="currentPage" :total="totalItems" :pageSize="pageSize"
            :pageSizeOptions="['10', '20', '50', '100']" :showSizeChanger="true" @change="handlePageChange"
            @showSizeChange="handleSizeChange" :showTotal="(total: number) => `共 ${total} 条`" />
        </div>
      </a-card>
    </div>

    <!-- 模板创建/编辑对话框 -->
    <a-modal v-model:visible="templateDialog.visible" :title="templateDialog.isEdit ? '编辑模板' : '创建模板'" width="800px"
      @ok="saveTemplate" :destroy-on-close="true" :confirm-loading="templateDialog.loading">
      <a-form ref="formRef" :model="templateDialog.form" :rules="formRules" layout="vertical">
        <a-form-item label="模板名称" name="name">
          <a-input v-model:value="templateDialog.form.name" placeholder="请输入模板名称" />
        </a-form-item>

        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="templateDialog.form.description" :rows="3" placeholder="请输入模板描述" />
        </a-form-item>

        <a-form-item label="分类" name="category_id">
          <a-select v-model:value="templateDialog.form.category_id" placeholder="请选择分类" style="width: 100%">
            <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
              {{ cat.name }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="关联流程" name="process_id">
          <a-select v-model:value="templateDialog.form.process_id" placeholder="请选择流程" style="width: 100%">
            <a-select-option v-for="process in processes" :key="process.id" :value="process.id">
              {{ process.name }}
            </a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="状态" name="status" v-if="templateDialog.isEdit">
          <a-radio-group v-model:value="templateDialog.form.status">
            <a-radio :value="1">启用</a-radio>
            <a-radio :value="0">禁用</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-divider orientation="left">默认值设置</a-divider>

        <!-- 结构化默认值配置 -->
        <a-card title="默认值配置" size="small" class="default-values-card">
          
          <!-- 优先级设置 -->
          <a-form-item label="默认优先级" style="margin-bottom: 16px;">
            <a-select v-model:value="templateDialog.defaultValues.priority" placeholder="请选择优先级" style="width: 100%">
              <a-select-option :value="1">低</a-select-option>
              <a-select-option :value="2">中</a-select-option>
              <a-select-option :value="3">高</a-select-option>
              <a-select-option :value="4">紧急</a-select-option>
            </a-select>
          </a-form-item>

          <!-- 审批人设置 -->
          <a-form-item label="默认审批人" style="margin-bottom: 16px;">
            <a-select 
              v-model:value="templateDialog.defaultValues.approvers" 
              mode="multiple"
              placeholder="请选择审批人" 
              style="width: 100%"
              :loading="loadingUsers"
            >
              <a-select-option v-for="user in users" :key="user.id" :value="user.id">
                {{ user.name }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <!-- 处理时限设置 -->
          <a-form-item label="默认处理时限（小时）" style="margin-bottom: 16px;">
            <a-input-number 
              v-model:value="templateDialog.defaultValues.due_hours" 
              :min="1" 
              :max="720"
              placeholder="请输入小时数"
              style="width: 100%" 
            />
          </a-form-item>

          <!-- 自定义字段设置 -->
          <a-form-item label="自定义字段" style="margin-bottom: 16px;">
            <div class="custom-fields-section">
              <div v-for="(field, index) in templateDialog.defaultValues.customFields" :key="index" class="field-row">
                <a-input 
                  v-model:value="field.key" 
                  placeholder="字段名"
                  style="width: 200px; margin-right: 8px;"
                />
                <a-input 
                  v-model:value="field.value" 
                  placeholder="默认值"
                  style="width: 200px; margin-right: 8px;"
                />
                <a-button type="text" danger @click="removeCustomField(index)">
                  <template #icon>
                    <DeleteOutlined />
                  </template>
                </a-button>
              </div>
              <a-button type="dashed" @click="addCustomField" style="width: 100%; margin-top: 8px;">
                <template #icon>
                  <PlusOutlined />
                </template>
                添加自定义字段
              </a-button>
            </div>
          </a-form-item>

          <!-- JSON预览 -->
          <a-form-item label="JSON预览">
            <a-textarea 
              :value="generateDefaultValuesJSON()"
              :rows="6" 
              readonly
              class="json-preview"
            />
          </a-form-item>
        </a-card>

        <a-form-item label="排序" name="sort_order" style="margin-top: 16px;">
          <a-input-number v-model:value="templateDialog.form.sort_order" :min="0" style="width: 100%" />
        </a-form-item>

        <a-form-item label="图标" name="icon">
          <a-input v-model:value="templateDialog.form.icon" placeholder="请输入图标类名" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 克隆对话框 -->
    <a-modal v-model:visible="cloneDialog.visible" title="克隆模板" @ok="confirmClone" :destroy-on-close="true" :confirm-loading="cloneDialog.loading">
      <a-form :model="cloneDialog.form" layout="vertical">
        <a-form-item label="新模板名称" name="name">
          <a-input v-model:value="cloneDialog.form.name" placeholder="请输入新模板名称" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 预览对话框 -->
    <a-modal v-model:visible="previewDialog.visible" title="模板预览" width="80%" :footer="null" class="preview-dialog">
      <div v-if="previewDialog.template" class="template-details">
        <div class="detail-header">
          <h2>{{ previewDialog.template.name }}</h2>
          <a-tag :color="previewDialog.template.status ? 'green' : 'default'">
            {{ previewDialog.template.status ? '启用' : '禁用' }}
          </a-tag>
          <a-tag v-if="isSystemTemplate(previewDialog.template)" color="purple">系统</a-tag>
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="ID">{{ previewDialog.template.id }}</a-descriptions-item>
          <a-descriptions-item label="分类">{{ getCategoryName(previewDialog.template.category_id) }}</a-descriptions-item>
          <a-descriptions-item label="创建人">{{ previewDialog.template.creator_name }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatFullDateTime(previewDialog.template.created_at || '')
          }}</a-descriptions-item>
          <a-descriptions-item label="描述" :span="2">{{ previewDialog.template.description || '无描述'
          }}</a-descriptions-item>
        </a-descriptions>

        <div class="template-content-preview">
          <a-tabs>
            <a-tab-pane key="details" tab="详细信息">
              <a-descriptions bordered :column="1">
                <a-descriptions-item label="流程ID">{{ previewDialog.template.process_id }}</a-descriptions-item>
                <a-descriptions-item label="排序">{{ previewDialog.template.sort_order }}</a-descriptions-item>
                <a-descriptions-item label="图标">{{ previewDialog.template.icon || '无' }}</a-descriptions-item>
                <a-descriptions-item label="默认值">
                  <pre>{{ formatDefaultValues(previewDialog.template.default_values) }}</pre>
                </a-descriptions-item>
              </a-descriptions>
            </a-tab-pane>
          </a-tabs>
        </div>

        <div class="detail-footer">
          <a-button @click="previewDialog.visible = false">关闭</a-button>
          <a-button type="primary" @click="handleEditTemplate(previewDialog.template)"
            :disabled="isSystemTemplate(previewDialog.template)">编辑</a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  PlusOutlined,
  FileOutlined,
  FileTextOutlined,
  SettingOutlined,
  PlusCircleOutlined,
  DownOutlined,
  DeleteOutlined
} from '@ant-design/icons-vue';
import {
  type TemplateItem,
  type CreateTemplateReq,
  type UpdateTemplateReq,
  type CloneTemplateReq,
  type ListTemplateReq,
  createTemplate,
  updateTemplate,
  deleteTemplate,
  listTemplate,
  detailTemplate,
  enableTemplate,
  disableTemplate,
  cloneTemplate
} from '#/api/core/workorder_template';
import { listProcess } from '#/api/core/workorder_process';
import { listCategory, type CategoryResp } from '#/api/core/workorder_category';
import { getUserList } from '#/api/core/user';

// 类型定义
interface Category {
  id: number;
  name: string;
  color: string;
}

interface Process {
  id: number;
  name: string;
}

interface User {
  id: number;
  name: string;
}

interface CustomField {
  key: string;
  value: string;
}

interface DefaultValues {
  priority: number | null;
  approvers: number[];
  due_hours: number | null;
  customFields: CustomField[];
}

// 表格列定义
const columns = [
  {
    title: '模板名称',
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
    title: '分类',
    dataIndex: 'category_id',
    key: 'category',
    width: 120,
    align: 'center',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100,
    align: 'center',
  },
  {
    title: '创建人',
    dataIndex: 'creator_name',
    key: 'creator',
    width: 150,
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
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

// 状态数据
const loading = ref(false);
const loadingUsers = ref(false);
const searchQuery = ref('');
const categoryFilter = ref<number | null>(null);
const statusFilter = ref<0 | 1 | null>(null);
const currentPage = ref(1);
const pageSize = ref(10);
const totalItems = ref(0);
const templates = ref<TemplateItem[]>([]);
const processes = ref<Process[]>([]);
const users = ref<User[]>([]);

// 统计数据
const stats = reactive({
  total: 0,
  regular: 0,
  system: 0,
  recentAdded: 0
});

// 分类数据
const categories = ref<Category[]>([]);

// 模板对话框
const templateDialog = reactive({
  visible: false,
  isEdit: false,
  loading: false,
  form: {
    id: undefined as number | undefined,
    name: '',
    description: '',
    process_id: 0,
    default_values: {} as any,
    icon: '',
    status: 1 as 0 | 1,
    sort_order: 0,
    category_id: undefined as number | undefined
  },
  defaultValues: {
    priority: null,
    approvers: [],
    due_hours: null,
    customFields: []
  } as DefaultValues
});

// 克隆对话框
const cloneDialog = reactive({
  visible: false,
  loading: false,
  form: {
    id: 0,
    name: ''
  }
});

// 预览对话框
const previewDialog = reactive({
  visible: false,
  template: null as TemplateItem | null
});

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入模板名称', trigger: 'blur' },
    { min: 3, max: 50, message: '长度应为3到50个字符', trigger: 'blur' }
  ],
  category_id: [
    { required: true, message: '请选择分类', trigger: 'change' }
  ],
  process_id: [
    { required: true, message: '请选择关联流程', trigger: 'change' }
  ]
};

// 表单引用
const formRef = ref();

// 默认值配置相关方法
const addCustomField = () => {
  templateDialog.defaultValues.customFields.push({
    key: '',
    value: ''
  });
};

const removeCustomField = (index: number) => {
  templateDialog.defaultValues.customFields.splice(index, 1);
};

const generateDefaultValuesJSON = () => {
  const defaultValues: any = {};

  // 构建 fields 对象
  const fields: any = {};
  templateDialog.defaultValues.customFields.forEach(field => {
    if (field.key && field.value) {
      fields[field.key] = field.value;
    }
  });
  
  if (Object.keys(fields).length > 0) {
    defaultValues.fields = fields;
  }

  // 添加其他字段
  if (templateDialog.defaultValues.priority !== null) {
    defaultValues.priority = templateDialog.defaultValues.priority;
  }

  if (templateDialog.defaultValues.approvers.length > 0) {
    defaultValues.approvers = templateDialog.defaultValues.approvers;
  }

  if (templateDialog.defaultValues.due_hours !== null) {
    defaultValues.due_hours = templateDialog.defaultValues.due_hours;
  }

  return JSON.stringify(defaultValues, null, 2);
};

const parseDefaultValues = (jsonStr: string | object) => {
  try {
    let parsed: any = {};
    
    if (typeof jsonStr === 'string') {
      parsed = JSON.parse(jsonStr);
    } else if (jsonStr && typeof jsonStr === 'object') {
      parsed = jsonStr;
    }

    // 重置默认值
    templateDialog.defaultValues = {
      priority: parsed.priority || null,
      approvers: parsed.approvers || [],
      due_hours: parsed.due_hours || null,
      customFields: []
    };

    // 解析自定义字段
    if (parsed.fields && typeof parsed.fields === 'object') {
      templateDialog.defaultValues.customFields = Object.entries(parsed.fields).map(([key, value]) => ({
        key,
        value: String(value)
      }));
    }
  } catch (error) {
    console.warn('解析默认值失败:', error);
    // 重置为默认值
    templateDialog.defaultValues = {
      priority: null,
      approvers: [],
      due_hours: null,
      customFields: []
    };
  }
};

// 加载用户列表
const loadUsers = async () => {
  loadingUsers.value = true;
  try {
    const response = await getUserList({
      page: 1,
      size: 100,
      search: ''
    });
    users.value = response.items || [];
  } catch (error) {
    console.error('加载用户列表失败:', error);
    message.error('加载用户列表失败');
  } finally {
    loadingUsers.value = false;
  }
};

// 加载分类数据
const loadCategories = async () => {
  try {
    const response = await listCategory({
      page: 1,
      size: 100, 
      status: 1 
    });
    
    if (response?.items) {
      categories.value = response.items.map((item: CategoryResp, index: number) => ({
        id: item.id,
        name: item.name,
        color: generateCategoryColor(index)
      }));
    }
  } catch (error) {
    console.error('加载分类列表失败:', error);
    message.error('加载分类列表失败');
  }
};

// 生成分类颜色
const generateCategoryColor = (index: number) => {
  const colors = [
    'blue', 'green', 'orange', 'red', 
    'purple', 'cyan', 'magenta', 'lime',
    'pink', 'yellow', 'volcano', 'geekblue'
  ];
  return colors[index % colors.length];
};

// 加载模板列表
const loadTemplates = async () => {
  loading.value = true;
  try {
    const params: ListTemplateReq = {
      page: currentPage.value,
      size: pageSize.value
    };

    if (searchQuery.value) {
      params.name = searchQuery.value;
    }
    if (categoryFilter.value !== null) {
      params.category_id = categoryFilter.value;
    }
    if (statusFilter.value !== null) {
      params.status = statusFilter.value;
    }

    const response = await listTemplate(params);
    if (response) {
      templates.value = response.items || [];
      totalItems.value = response.total || 0;
      updateStats();
    }
  } catch (error) {
    console.error('加载模板列表失败:', error);
    message.error('加载模板列表失败');
  } finally {
    loading.value = false;
  }
};

const loadProcesses = async () => {
  try {
    const response = await listProcess({
      page: 1,
      size: 100
    });
    if (response) {
      processes.value = response.items || [];
    }
  } catch (error) {
    console.error('加载流程列表失败:', error);
    message.error('加载流程列表失败');
  }
};

const updateStats = () => {
  stats.total = totalItems.value;
  stats.regular = templates.value.filter((t: TemplateItem) => !isSystemTemplate(t)).length;
  stats.system = templates.value.filter((t: TemplateItem) => isSystemTemplate(t)).length;
  
  const sevenDaysAgo = new Date();
  sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 7);
  stats.recentAdded = templates.value.filter((t: TemplateItem) => {
    const createdAt = new Date(t.created_at || '');
    return createdAt >= sevenDaysAgo;
  }).length;
};

const handleSizeChange = (current: number, size: number) => {
  pageSize.value = size;
  currentPage.value = 1;
  loadTemplates();
};

const handlePageChange = (page: number) => {
  currentPage.value = page;
  loadTemplates();
};

const handleSearch = () => {
  currentPage.value = 1;
  loadTemplates();
};

const handleCategoryChange = () => {
  currentPage.value = 1;
  loadTemplates();
};

const handleStatusChange = () => {
  currentPage.value = 1;
  loadTemplates();
};

const handleCreateTemplate = () => {
  templateDialog.isEdit = false;
  templateDialog.form = {
    id: undefined,
    name: '',
    description: '',
    process_id: 0,
    default_values: {},
    icon: '',
    status: 1,
    sort_order: 0,
    category_id: undefined
  };
  
  // 重置默认值配置
  templateDialog.defaultValues = {
    priority: null,
    approvers: [],
    due_hours: null,
    customFields: []
  };
  
  templateDialog.visible = true;
};

const handleEditTemplate = async (template: TemplateItem) => {
  if (isSystemTemplate(template)) {
    message.warning('系统模板不可编辑');
    return;
  }

  loading.value = true;
  try {
    const response = await detailTemplate(template.id);
    if (response) {
      const templateData = response;
      templateDialog.isEdit = true;
      
      templateDialog.form = {
        id: templateData.id,
        name: templateData.name,
        description: templateData.description || '',
        process_id: templateData.process_id,
        default_values: templateData.default_values || {},
        icon: templateData.icon || '',
        status: templateData.status,
        sort_order: templateData.sort_order || 0,
        category_id: templateData.category_id
      };

      // 解析默认值到结构化表单
      parseDefaultValues(templateData.default_values);
      
      templateDialog.visible = true;
      previewDialog.visible = false;
    }
  } catch (error) {
    console.error('获取模板详情失败:', error);
    message.error('获取模板详情失败');
  } finally {
    loading.value = false;
  }
};

const handlePreviewTemplate = async (template: TemplateItem) => {
  loading.value = true;
  try {
    const response = await detailTemplate(template.id);
    if (response) {
      previewDialog.template = response;
      previewDialog.visible = true;
    }
  } catch (error) {
    console.error('获取模板详情失败:', error);
    message.error('获取模板详情失败');
  } finally {
    loading.value = false;
  }
};

const handleCommand = (command: string, template: TemplateItem) => {
  switch (command) {
    case 'enable':
      handleEnableTemplate(template);
      break;
    case 'disable':
      handleDisableTemplate(template);
      break;
    case 'clone':
      showCloneDialog(template);
      break;
    case 'delete':
      confirmDelete(template);
      break;
  }
};

const handleEnableTemplate = async (template: TemplateItem) => {
  try {
    await enableTemplate(template.id);
    message.success(`模板 "${template.name}" 已启用`);
    loadTemplates();
  } catch (error) {
    console.error('启用模板失败:', error);
    message.error('启用模板失败');
  }
};

const handleDisableTemplate = async (template: TemplateItem) => {
  try {
    await disableTemplate(template.id);
    message.success(`模板 "${template.name}" 已禁用`);
    loadTemplates();
  } catch (error) {
    console.error('禁用模板失败:', error);
    message.error('禁用模板失败');
  }
};

const showCloneDialog = (template: TemplateItem) => {
  cloneDialog.form.id = template.id;
  cloneDialog.form.name = `${template.name} 副本`;
  cloneDialog.visible = true;
};

const confirmClone = async () => {
  if (!cloneDialog.form.name.trim()) {
    message.error('请输入新模板名称');
    return;
  }

  cloneDialog.loading = true;
  try {
    const data: CloneTemplateReq = {
      id: cloneDialog.form.id,
      name: cloneDialog.form.name
    };
    
    await cloneTemplate(data);
    message.success(`模板已克隆为 "${cloneDialog.form.name}"`);
    cloneDialog.visible = false;
    loadTemplates();
  } catch (error) {
    console.error('克隆模板失败:', error);
    message.error('克隆模板失败');
  } finally {
    cloneDialog.loading = false;
  }
};

const confirmDelete = (template: TemplateItem) => {
  if (isSystemTemplate(template)) {
    message.warning('系统模板不可删除');
    return;
  }

  Modal.confirm({
    title: '警告',
    content: `确定要删除模板 "${template.name}" 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      try {
        await deleteTemplate(template.id);
        message.success(`模板 "${template.name}" 已删除`);
        loadTemplates();
      } catch (error) {
        console.error('删除模板失败:', error);
        message.error('删除模板失败');
      }
    }
  });
};

const saveTemplate = async () => {
  // 先验证表单
  try {
    await formRef.value?.validate();
  } catch (error) {
    console.log('表单验证失败:', error);
    return;
  }

  // 验证必填字段
  if (!templateDialog.form.name.trim()) {
    message.error('模板名称不能为空');
    return;
  }

  if (!templateDialog.form.category_id) {
    message.error('请选择分类');
    return;
  }
  
  if (!templateDialog.form.process_id) {
    message.error('请选择关联流程');
    return;
  }

  // 构建默认值对象
  let defaultValues: any = {};

  // 构建 fields 对象
  const fields: any = {};
  templateDialog.defaultValues.customFields.forEach(field => {
    if (field.key && field.value) {
      fields[field.key] = field.value;
    }
  });
  
  if (Object.keys(fields).length > 0) {
    defaultValues.fields = fields;
  }

  // 添加其他字段
  if (templateDialog.defaultValues.priority !== null) {
    defaultValues.priority = templateDialog.defaultValues.priority;
  }

  if (templateDialog.defaultValues.approvers.length > 0) {
    defaultValues.approvers = templateDialog.defaultValues.approvers;
  }

  if (templateDialog.defaultValues.due_hours !== null) {
    defaultValues.due_hours = templateDialog.defaultValues.due_hours;
  }

  templateDialog.loading = true;
  try {
    if (templateDialog.isEdit && templateDialog.form.id) {
      const data: UpdateTemplateReq = {
        id: templateDialog.form.id,
        name: templateDialog.form.name,
        description: templateDialog.form.description || '',
        process_id: templateDialog.form.process_id,
        default_values: defaultValues,
        icon: templateDialog.form.icon || '',
        category_id: templateDialog.form.category_id,
        sort_order: templateDialog.form.sort_order || 0,
        status: templateDialog.form.status
      };
      await updateTemplate(data);
      message.success(`模板 "${templateDialog.form.name}" 已更新`);
    } else {
      const data: CreateTemplateReq = {
        name: templateDialog.form.name,
        description: templateDialog.form.description || '',
        process_id: templateDialog.form.process_id,
        default_values: defaultValues,
        icon: templateDialog.form.icon || '',
        category_id: templateDialog.form.category_id,
        sort_order: templateDialog.form.sort_order || 0
      };
      await createTemplate(data);
      message.success(`模板 "${templateDialog.form.name}" 已创建`);
    }
    templateDialog.visible = false;
    loadTemplates();
  } catch (error) {
    console.error('保存模板失败:', error);
    message.error('保存模板失败');
  } finally {
    templateDialog.loading = false;
  }
};

// 辅助方法
const formatDate = (dateStr: string) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const formatTime = (dateStr: string) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
};

const formatFullDateTime = (dateStr: string) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
};

const formatDefaultValues = (jsonStr: string | object) => {
  try {
    if (!jsonStr) return '{}';
    if (typeof jsonStr === 'string') {
      const obj = JSON.parse(jsonStr);
      return JSON.stringify(obj, null, 2);
    } else {
      return JSON.stringify(jsonStr, null, 2);
    }
  } catch (e) {
    return typeof jsonStr === 'string' ? jsonStr : JSON.stringify(jsonStr);
  }
};

const getInitials = (name: string) => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getAvatarColor = (name: string) => {
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

const getCategoryName = (categoryId?: number) => {
  if (!categoryId) return '未分类';
  const category = categories.value.find(c => c.id === categoryId);
  return category ? category.name : '未分类';
};

const getCategoryColor = (categoryId?: number) => {
  if (!categoryId) return 'default';
  const category = categories.value.find(c => c.id === categoryId);
  return category ? category.color : 'default';
};

const isSystemTemplate = (template: TemplateItem) => {
  return template.creator_id === 1;
};

// 初始化
onMounted(() => {
  loadCategories();
  loadTemplates();
  loadProcesses();
  loadUsers();
});
</script>

<style scoped>
.template-manager-container {
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

.template-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.template-badge {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-enabled {
  background-color: #52c41a;
}

.status-disabled {
  background-color: #d9d9d9;
}

.template-name-text {
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

.template-content-preview {
  margin-top: 24px;
}

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.default-values-card {
  margin-bottom: 16px;
}

.custom-fields-section {
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  padding: 16px;
  background-color: #fafafa;
}

.field-row {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.field-row:last-child {
  margin-bottom: 0;
}

.json-preview {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  background-color: #f6f8fa;
  border: 1px solid #e1e4e8;
}
</style>