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
    <a-modal v-model:visible="templateDialog.visible" :title="templateDialog.isEdit ? '编辑模板' : '创建模板'" width="760px"
      @ok="saveTemplate" :destroy-on-close="true">
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

        <a-form-item label="状态" name="status">
          <a-radio-group v-model:value="templateDialog.form.status">
            <a-radio :value="1">启用</a-radio>
            <a-radio :value="0">禁用</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-divider orientation="left">默认值设置</a-divider>

        <a-form-item label="审核人" name="approver">
          <a-input v-model:value="templateDialog.form.default_values.approver" placeholder="请输入默认审核人" />
        </a-form-item>

        <a-form-item label="截止日期" name="deadline">
          <a-date-picker 
            v-model:value="templateDialog.form.default_values.deadline" 
            style="width: 100%" 
            @change="handleDeadlineChange"
          />
        </a-form-item>

        <a-form-item label="排序" name="sort_order">
          <a-input-number v-model:value="templateDialog.form.sort_order" :min="0" style="width: 100%" />
        </a-form-item>

        <a-form-item label="图标" name="icon">
          <a-input v-model:value="templateDialog.form.icon" placeholder="请输入图标类名" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 克隆对话框 -->
    <a-modal v-model:visible="cloneDialog.visible" title="克隆模板" @ok="confirmClone" :destroy-on-close="true">
      <a-form :model="cloneDialog.form" layout="vertical">
        <a-form-item label="新模板名称" name="name">
          <a-input v-model:value="cloneDialog.form.name" placeholder="请输入新模板名称" />
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="cloneDialog.form.description" :rows="3" placeholder="请输入模板描述" />
        </a-form-item>
        <a-form-item label="分类" name="category_id">
          <a-select v-model:value="cloneDialog.form.category_id" placeholder="请选择分类" style="width: 100%">
            <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
              {{ cat.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 预览对话框 -->
    <a-modal v-model:visible="previewDialog.visible" title="模板预览" width="80%" footer={null} class="preview-dialog">
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
import { ref, reactive, computed, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import dayjs from 'dayjs';
import {
  PlusOutlined,
  FileOutlined,
  FileTextOutlined,
  SettingOutlined,
  PlusCircleOutlined,
  DeleteOutlined,
  DownOutlined
} from '@ant-design/icons-vue';
import {
  type Template,
  type TemplateReq,
  type DetailTemplateReq,
  type DeleteTemplateReq,
  type ListTemplateReq,
  type DefaultValues,
  createTemplate,
  updateTemplate,
  deleteTemplate,
  listTemplate,
  detailTemplate,
  listProcess,
  type Process
} from '#/api/core/workorder';

// 类型定义
interface Category {
  id: number;
  name: string;
  color: string;
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
const searchQuery = ref('');
const categoryFilter = ref(null);
const statusFilter = ref(null);
const currentPage = ref(1);
const pageSize = ref(10);
const totalItems = ref(0);
const templates = ref<Template[]>([]);
const processes = ref<Process[]>([]);

// 统计数据
const stats = reactive({
  total: 0,
  regular: 0,
  system: 0,
  recentAdded: 0
});

// 分类数据
const categories = ref<Category[]>([
  { id: 1, name: '邮件模板', color: '#1890ff' },
  { id: 2, name: '通知模板', color: '#52c41a' },
  { id: 3, name: '报表模板', color: '#722ed1' },
  { id: 4, name: '文档模板', color: '#fa8c16' },
  { id: 5, name: '销售模板', color: '#eb2f96' }
]);

// 模板对话框
const templateDialog = reactive({
  visible: false,
  isEdit: false,
  form: {
    id: undefined,
    name: '',
    description: '',
    process_id: 0,
    default_values: {
      approver: '',
      deadline: ''
    } as DefaultValues,
    deadline_value: null as any,
    icon: '',
    status: 1,
    sort_order: 0,
    category_id: undefined,
    creator_id: undefined,
    creator_name: ''
  } as TemplateReq
});

// 克隆对话框
const cloneDialog = reactive({
  visible: false,
  form: {
    name: '',
    description: '',
    category_id: undefined as number | undefined,
    originalId: 0
  }
});

// 预览对话框
const previewDialog = reactive({
  visible: false,
  template: null as Template | null
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

// 方法
const loadTemplates = async () => {
  loading.value = true;
  try {
    const params: ListTemplateReq = {
      page: currentPage.value,
      size: pageSize.value,
      search: searchQuery.value || undefined,
      status: statusFilter.value || undefined
    };

    const response = await listTemplate(params);
    if (response && response) {
      templates.value = response.list || [];
      totalItems.value = response.total || 0;
      
      // 更新统计数据
      updateStats();
    } else {
      message.error('获取模板列表失败');
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
      size: 100,  
      status: 1     // 只获取已启用的流程
    });
    if (response && response) {
      processes.value = response.list || [];
    }
  } catch (error) {
    console.error('加载流程列表失败:', error);
    message.error('加载流程列表失败');
  }
};

const updateStats = () => {
  // 计算统计数据
  stats.total = totalItems.value;
  
  // 计算常规模板数量 (不是系统模板的)
  stats.regular = templates.value.filter((t: Template) => !isSystemTemplate(t)).length;
  
  // 计算系统模板数量 (假设creator_id为1的是系统模板)
  stats.system = templates.value.filter((t: Template) => isSystemTemplate(t)).length;
  
  // 计算最近7天新增的模板数量
  const sevenDaysAgo = new Date();
  sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 7);
  stats.recentAdded = templates.value.filter((t: Template) => {
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
    name: '',
    description: '',
    process_id: 0,
    default_values: {
      approver: '',
      deadline: ''
    },
    icon: '',
    status: 1,
    sort_order: 0,
    category_id: undefined,
    creator_id: undefined,
    creator_name: ''
  };
  templateDialog.visible = true;
};

const handleEditTemplate = async (template: Template) => {
  if (isSystemTemplate(template)) {
    message.warning('系统模板不可编辑');
    return;
  }

  loading.value = true;
  try {
    const res = await detailTemplate({ id: template.id });
    if (res && res) {
      const templateData = res;
      templateDialog.isEdit = true;
      templateDialog.form = {
        id: templateData.id,
        name: templateData.name,
        description: templateData.description,
        process_id: templateData.process_id,
        default_values: JSON.parse(templateData.default_values || '{}'),
        icon: templateData.icon || '',
        status: templateData.status,
        sort_order: templateData.sort_order,
        category_id: templateData.category_id,
        creator_id: templateData.creator_id,
        creator_name: templateData.creator_name
      };
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

const handlePreviewTemplate = async (template: Template) => {
  loading.value = true;
  try {
    const res = await detailTemplate({ id: template.id });
    if (res && res) {
      previewDialog.template = res;
      previewDialog.visible = true;
    }
  } catch (error) {
    console.error('获取模板详情失败:', error);
    message.error('获取模板详情失败');
  } finally {
    loading.value = false;
  }
};

const handleCommand = (command: string, template: Template) => {
  switch (command) {
    case 'enable':
      enableTemplate(template);
      break;
    case 'disable':
      disableTemplate(template);
      break;
    case 'clone':
      showCloneDialog(template);
      break;
    case 'delete':
      confirmDelete(template);
      break;
  }
};

const enableTemplate = async (template: Template) => {
  try {
    const templateReq: TemplateReq = {
      id: template.id,
      name: template.name,
      description: template.description,
      process_id: template.process_id,
      default_values: JSON.parse(template.default_values || '{}'),
      icon: template.icon,
      status: 1,
      sort_order: template.sort_order,
      category_id: template.category_id,
      creator_id: template.creator_id,
      creator_name: template.creator_name
    };
    
    await updateTemplate(templateReq);
    message.success(`模板 "${template.name}" 已启用`);
    loadTemplates();
  } catch (error) {
    console.error('启用模板失败:', error);
    message.error('启用模板失败');
  }
};

const disableTemplate = async (template: Template) => {
  try {
    const templateReq: TemplateReq = {
      id: template.id,
      name: template.name,
      description: template.description,
      process_id: template.process_id,
      default_values: JSON.parse(template.default_values || '{}'),
      icon: template.icon,
      status: 0,
      sort_order: template.sort_order,
      category_id: template.category_id,
      creator_id: template.creator_id,
      creator_name: template.creator_name
    };
    
    await updateTemplate(templateReq);
    message.success(`模板 "${template.name}" 已禁用`);
    loadTemplates();
  } catch (error) {
    console.error('禁用模板失败:', error);
    message.error('禁用模板失败');
  }
};

const showCloneDialog = (template: Template) => {
  cloneDialog.form.name = `${template.name} 副本`;
  cloneDialog.form.description = template.description;
  cloneDialog.form.category_id = template.category_id;
  cloneDialog.form.originalId = template.id;
  cloneDialog.visible = true;
};

const confirmClone = async () => {
  try {
    // 首先获取原模板的详细信息
    const res = await detailTemplate({ id: cloneDialog.form.originalId });
    if (res && res) {
      const originalTemplate = res;
      
      // 创建新模板请求对象
      const newTemplate: TemplateReq = {
        name: cloneDialog.form.name,
        description: cloneDialog.form.description,
        process_id: originalTemplate.process_id,
        default_values: JSON.parse(originalTemplate.default_values || '{}'),
        icon: originalTemplate.icon,
        status: 1, // 默认启用
        sort_order: originalTemplate.sort_order,
        category_id: cloneDialog.form.category_id || originalTemplate.category_id
      };
      
      // 创建克隆模板
      await createTemplate(newTemplate);
      message.success(`模板 "${originalTemplate.name}" 已克隆为 "${cloneDialog.form.name}"`);
      cloneDialog.visible = false;
      loadTemplates();
    }
  } catch (error) {
    console.error('克隆模板失败:', error);
    message.error('克隆模板失败');
  }
};

const confirmDelete = (template: Template) => {
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
        await deleteTemplate({ id: template.id });
        message.success(`模板 "${template.name}" 已删除`);
        loadTemplates();
      } catch (error) {
        console.error('删除模板失败:', error);
        message.error('删除模板失败');
      }
    }
  });
};

const handleDeadlineChange = (value: any) => {
  if (value) {
    templateDialog.form.default_values.deadline = value.format('YYYY-MM-DD');
  } else {
    templateDialog.form.default_values.deadline = '';
  }
};

const saveTemplate = async () => {
  if (templateDialog.form.name.trim() === '') {
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

  try {
    if (templateDialog.isEdit) {
      // 更新现有模板
      await updateTemplate(templateDialog.form);
      message.success(`模板 "${templateDialog.form.name}" 已更新`);
    } else {
      // 创建新模板
      await createTemplate(templateDialog.form);
      message.success(`模板 "${templateDialog.form.name}" 已创建`);
    }
    templateDialog.visible = false;
    loadTemplates();
  } catch (error) {
    console.error('保存模板失败:', error);
    message.error('保存模板失败');
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

const formatDefaultValues = (jsonStr: string) => {
  try {
    if (!jsonStr) return '{}';
    const obj = JSON.parse(jsonStr);
    return JSON.stringify(obj, null, 2);
  } catch (e) {
    return jsonStr;
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

const getCategoryName = (categoryId?: number) => {
  if (!categoryId) return '未分类';
  const category = categories.value.find(c => c.id === categoryId);
  return category ? category.name : '未分类';
};

const getCategoryColor = (categoryId?: number) => {
  if (!categoryId) return '';
  const category = categories.value.find(c => c.id === categoryId);
  return category ? category.color : '';
};

// 判断是否是系统模板 (假设creator_id为1的是系统模板)
const isSystemTemplate = (template: Template) => {
  return template.creator_id === 1;
};

// 初始化
onMounted(() => {
  loadTemplates();
  loadProcesses();
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
</style>