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
        <a-input-search 
          v-model:value="searchQuery" 
          placeholder="搜索表单..." 
          style="width: 250px" 
          @search="handleSearch"
          allow-clear 
        />
        <a-select 
          v-model:value="statusFilter" 
          placeholder="状态" 
          style="width: 120px" 
          @change="handleStatusChange"
        >
          <a-select-option :value="undefined">全部</a-select-option>
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
            <a-statistic title="总表单数" :value="stats.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <FormOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已发布" :value="stats.published" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="草稿" :value="stats.draft" :value-style="{ color: '#faad14' }">
              <template #prefix>
                <EditOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card class="stats-card">
            <a-statistic title="已禁用" :value="stats.disabled" :value-style="{ color: '#cf1322' }">
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
        <a-table 
          :data-source="paginatedForms" 
          :columns="columns" 
          :pagination="false" 
          :loading="loading" 
          row-key="id"
          bordered
        >
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
                <a-avatar size="small" :style="{ backgroundColor: getAvatarColor(record.creator_name || '') }">
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
                <a-button type="primary" size="small" @click="handleViewForm(record)">
                  查看
                </a-button>
                <a-button type="default" size="small" @click="handleEditForm(record)">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleMenuClick(e.key, record)">
                      <a-menu-item key="preview">
                        <EyeOutlined /> 预览
                      </a-menu-item>
                      <a-menu-divider />
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
          <a-pagination 
            v-model:current="currentPage" 
            :total="totalItems" 
            :page-size="pageSize"
            :page-size-options="['10', '20', '50', '100']" 
            :show-size-changer="true" 
            @change="handleCurrentChange"
            @showSizeChange="handleSizeChange" 
            :show-total="(total: number) => `共 ${total} 条`" 
          />
        </div>
      </a-card>
    </div>

    <!-- 表单创建/编辑对话框 -->
    <a-modal 
      :open="formDialogVisible" 
      :title="formDialog.isEdit ? '编辑表单设计' : '创建表单设计'" 
      width="760px"
      @ok="saveForm" 
      @cancel="closeFormDialog"
      :destroy-on-close="true"
    >
      <a-form ref="formRef" :model="formDialog.form" :rules="formRules" layout="vertical">
        <a-form-item label="表单名称" name="name">
          <a-input v-model:value="formDialog.form.name" placeholder="请输入表单名称" />
        </a-form-item>

        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="formDialog.form.description" :rows="3" placeholder="请输入表单描述" />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="24">
            <a-form-item label="分类" name="category_id">
              <a-select v-model:value="formDialog.form.category_id" placeholder="请选择分类" style="width: 100%">
                <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
                  {{ cat.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-divider orientation="left">表单结构</a-divider>

        <div class="schema-editor">
          <div class="field-list">
            <a-collapse>
              <a-collapse-panel 
                v-for="(field, index) in formDialog.form.schema.fields" 
                :key="index"
                :header="getFieldPanelHeader(field, index)"
                :class="getFieldPanelClass(field)"
              >
                <template #extra>
                  <a-button type="text" danger @click.stop="removeField(index)" size="small">
                    <DeleteOutlined />
                  </a-button>
                </template>

                <a-form-item label="字段类型" :required="true">
                  <a-select v-model:value="field.type" style="width: 100%" @change="handleFieldTypeChange(field)">
                    <a-select-option value="text">文本框</a-select-option>
                    <a-select-option value="number">数字</a-select-option>
                    <a-select-option value="date">日期</a-select-option>
                    <a-select-option value="select">下拉选择</a-select-option>
                    <a-select-option value="checkbox">复选框</a-select-option>
                    <a-select-option value="radio">单选框</a-select-option>
                    <a-select-option value="textarea">多行文本</a-select-option>
                  </a-select>
                </a-form-item>

                <a-form-item label="标签名称" :required="true">
                  <a-input 
                    v-model:value="field.label" 
                    placeholder="字段标签" 
                    :status="!field.label.trim() ? 'error' : ''"
                  />
                  <div v-if="!field.label.trim()" class="field-error">
                    字段标签不能为空
                  </div>
                </a-form-item>

                <a-form-item label="字段名称" :required="true">
                  <a-input 
                    v-model:value="field.name" 
                    placeholder="字段名称（英文、数字、下划线）" 
                    :status="getFieldNameStatus(field.name, index)"
                    @blur="validateFieldName(field, index)"
                  />
                  <div v-if="getFieldNameError(field.name, index)" class="field-error">
                    {{ getFieldNameError(field.name, index) }}
                  </div>
                </a-form-item>

                <a-form-item label="是否必填">
                  <a-switch v-model:checked="field.required" />
                </a-form-item>

                <!-- 选项配置 -->
                <template v-if="['select', 'radio', 'checkbox'].includes(field.type)">
                  <a-form-item 
                    label="选项配置" 
                    :required="true"
                    :help="getOptionsHelp(field)"
                  >
                    <div 
                      v-for="(option, optIndex) in field.options" 
                      :key="optIndex" 
                      class="option-item"
                    >
                      <a-input 
                        v-model:value="option.label" 
                        placeholder="选项标签" 
                        style="width: 45%; margin-right: 8px;"
                        :status="!option.label.trim() ? 'error' : ''"
                      />
                      <a-input 
                        v-model:value="option.value" 
                        placeholder="选项值" 
                        style="width: 45%; margin-right: 8px;"
                        :status="!option.value.trim() ? 'error' : ''"
                      />
                      <a-button type="text" danger @click="removeOption(field, optIndex)" size="small">
                        <DeleteOutlined />
                      </a-button>
                    </div>
                    
                    <!-- 选项验证错误提示 -->
                    <div v-if="getOptionsError(field)" class="field-error">
                      {{ getOptionsError(field) }}
                    </div>
                    
                    <a-button 
                      type="dashed" 
                      @click="addOption(field)" 
                      size="small" 
                      style="width: 100%; margin-top: 8px;"
                    >
                      <PlusOutlined /> 添加选项
                    </a-button>
                  </a-form-item>
                </template>

                <!-- 占位符 -->
                <a-form-item label="占位符">
                  <a-input v-model:value="field.placeholder" placeholder="请输入占位符" />
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

        <!-- 表单验证错误汇总 -->
        <div v-if="formValidationErrors.length > 0" class="form-validation-summary">
          <a-alert
            message="表单验证错误"
            :description="formValidationErrors.join('；')"
            type="error"
            show-icon
            style="margin-bottom: 16px"
          />
        </div>
      </a-form>
    </a-modal>

    <!-- 克隆对话框 -->
    <a-modal 
      :open="cloneDialogVisible" 
      title="克隆表单" 
      @ok="confirmClone" 
      @cancel="closeCloneDialog"
      :destroy-on-close="true"
    >
      <a-form :model="cloneDialog.form" layout="vertical">
        <a-form-item label="新表单名称" name="name" :rules="[{ required: true, message: '请输入新表单名称' }]">
          <a-input v-model:value="cloneDialog.form.name" placeholder="请输入新表单名称" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 预览对话框 -->
    <a-modal 
      :open="previewDialogVisible" 
      title="表单预览" 
      width="80%" 
      :footer="null" 
      @cancel="closePreviewDialog"
      class="preview-dialog"
    >
      <div v-if="previewDialog.form" class="form-preview-wrapper">
        <a-spin :spinning="previewLoading">
          <div class="preview-header">
            <h3>{{ previewDialog.form.name }}</h3>
            <p v-if="previewDialog.form.description" class="preview-description">
              {{ previewDialog.form.description }}
            </p>
            <div class="preview-mode-notice">
              <a-alert
                message="预览模式"
                description="您可以查看和选择表单字段，但无法提交表单。"
                type="info"
                show-icon
                banner
              />
            </div>
          </div>
          
          <div class="preview-form">
            <a-form 
              :model="previewFormData" 
              layout="vertical" 
              class="dynamic-form"
            >
              <template v-for="field in parsedPreviewSchema.fields" :key="field.id">
                <a-form-item 
                  :label="field.label" 
                  :name="field.name"
                  :required="field.required"
                  class="form-field"
                >
                  <!-- 文本输入框 -->
                  <a-input 
                    v-if="field.type === 'text'"
                    v-model:value="previewFormData[field.name]"
                    :placeholder="field.placeholder"
                    class="preview-input"
                  />

                  <!-- 数字输入框 -->
                  <a-input-number 
                    v-else-if="field.type === 'number'"
                    v-model:value="previewFormData[field.name]"
                    :placeholder="field.placeholder"
                    style="width: 100%"
                    class="preview-input"
                  />

                  <!-- 日期选择器 -->
                  <a-date-picker 
                    v-else-if="field.type === 'date'"
                    v-model:value="previewFormData[field.name]"
                    :placeholder="field.placeholder"
                    style="width: 100%"
                    class="preview-input"
                  />

                  <!-- 下拉选择 -->
                  <a-select 
                    v-else-if="field.type === 'select'"
                    v-model:value="previewFormData[field.name]"
                    :placeholder="field.placeholder || '请选择'"
                    style="width: 100%"
                    class="preview-input"
                  >
                    <a-select-option 
                      v-for="option in field.options" 
                      :key="option.value" 
                      :value="option.value"
                    >
                      {{ option.label }}
                    </a-select-option>
                  </a-select>

                  <!-- 单选框组 -->
                  <a-radio-group 
                    v-else-if="field.type === 'radio'"
                    v-model:value="previewFormData[field.name]"
                    class="preview-radio-group"
                  >
                    <div class="radio-options">
                      <a-radio 
                        v-for="option in field.options" 
                        :key="option.value" 
                        :value="option.value"
                        class="preview-radio"
                      >
                        {{ option.label }}
                      </a-radio>
                    </div>
                  </a-radio-group>

                  <!-- 复选框组 -->
                  <a-checkbox-group 
                    v-else-if="field.type === 'checkbox'"
                    v-model:value="previewFormData[field.name]"
                    class="preview-checkbox-group"
                  >
                    <div class="checkbox-options">
                      <a-checkbox 
                        v-for="option in field.options" 
                        :key="option.value" 
                        :value="option.value"
                        class="preview-checkbox"
                      >
                        {{ option.label }}
                      </a-checkbox>
                    </div>
                  </a-checkbox-group>

                  <!-- 多行文本 -->
                  <a-textarea 
                    v-else-if="field.type === 'textarea'"
                    v-model:value="previewFormData[field.name]"
                    :placeholder="field.placeholder"
                    :rows="4"
                    class="preview-input"
                  />
                </a-form-item>
              </template>

              <div class="preview-form-actions">
                <a-tooltip title="预览模式下无法提交表单">
                  <a-button type="primary" disabled size="large">
                    提交表单 (预览模式)
                  </a-button>
                </a-tooltip>
                <a-button @click="resetPreviewForm" size="large" style="margin-left: 12px;">
                  重置表单
                </a-button>
              </div>
            </a-form>
          </div>
        </a-spin>
      </div>
    </a-modal>

    <!-- 详情对话框 -->
    <a-modal 
      :open="detailDialogVisible" 
      title="表单详情" 
      width="80%" 
      :footer="null" 
      @cancel="closeDetailDialog"
      class="detail-dialog"
    >
      <div v-if="detailDialog.form" class="form-details">
        <div class="detail-header">
          <h2>{{ detailDialog.form.name }}</h2>
          <a-tag
            :color="detailDialog.form.status === 0 ? 'orange' : detailDialog.form.status === 1 ? 'green' : 'default'"
          >
            {{ detailDialog.form.status === 0 ? '草稿' : detailDialog.form.status === 1 ? '已发布' : '已禁用' }}
          </a-tag>
        </div>

        <a-tabs default-active-key="1" @change="handleTabChange">
          <a-tab-pane key="1" tab="基本信息">
            <a-descriptions bordered :column="2">
              <a-descriptions-item label="ID">{{ detailDialog.form.id }}</a-descriptions-item>
              <a-descriptions-item label="版本">v{{ detailDialog.form.version }}</a-descriptions-item>
              <a-descriptions-item label="创建人">{{ detailDialog.form.creator_name }}</a-descriptions-item>
              <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.form.created_at || '') }}</a-descriptions-item>
              <a-descriptions-item label="描述" :span="2">{{ detailDialog.form.description || '无描述' }}</a-descriptions-item>
            </a-descriptions>

            <div class="schema-preview">
              <h3>表单结构</h3>
              <a-table 
                :data-source="parsedSchema.fields" 
                :columns="schemaColumns" 
                :pagination="false" 
                bordered
                size="small" 
                row-key="name"
              >
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'required'">
                    <a-tag :color="record.required ? 'red' : ''">
                      {{ record.required ? '必填' : '可选' }}
                    </a-tag>
                  </template>
                  <template v-if="column.key === 'type'">
                    {{ getFieldTypeName(record.type) }}
                  </template>
                </template>
              </a-table>
            </div>
          </a-tab-pane>
        </a-tabs>

        <div class="detail-footer">
          <a-button @click="closeDetailDialog">关闭</a-button>
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
  DownOutlined,
  EyeOutlined
} from '@ant-design/icons-vue';
import {
  listFormDesign,
  detailFormDesign,
  createFormDesign,
  updateFormDesign,
  deleteFormDesign,
  publishFormDesign,
  cloneFormDesign,
  previewFormDesign,
  type FormDesignResp,
  type FormDesignItem,
  type FormField,
  type FormSchema,
  type FormDesignReq,
  type ListFormDesignReq,
  type DetailFormDesignReq,
  type PublishFormDesignReq,
  type CloneFormDesignReq,
} from '#/api/core/workorder';
import type { Category } from '#/api/core/workorder_category'
import { listCategory } from '#/api/core/workorder_category'

// 响应式数据类型
interface Statistics {
  total: number;
  published: number;
  draft: number;
  disabled: number;
}

interface FormDialogState {
  isEdit: boolean;
  form: FormDesignReq;
}

interface CloneDialogState {
  form: {
    name: string;
    originalId: number;
  };
}

interface DetailDialogState {
  form: FormDesignResp | null;
}

interface PreviewDialogState {
  form: FormDesignResp | null;
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
    align: 'center' as const,
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 120,
    align: 'center' as const,
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
    align: 'center' as const,
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
    dataIndex: 'name',
    key: 'name',
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
const loading = ref<boolean>(false);
const previewLoading = ref<boolean>(false);
const searchQuery = ref<string>('');
const statusFilter = ref<number | undefined>(undefined);
const currentPage = ref<number>(1);
const pageSize = ref<number>(10);
const formDesigns = ref<FormDesignItem[]>([]);

// 预览表单数据
const previewFormData = ref<Record<string, any>>({});

// 表单验证错误
const formValidationErrors = ref<string[]>([]);

// 模态框控制
const formDialogVisible = ref<boolean>(false);
const cloneDialogVisible = ref<boolean>(false);
const detailDialogVisible = ref<boolean>(false);
const previewDialogVisible = ref<boolean>(false);

// 统计数据
const stats = reactive<Statistics>({
  total: 0,
  published: 0,
  draft: 0,
  disabled: 0
});

// 分类数据 - 修复变量名
const categories = ref<Category[]>([]);

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

  if (statusFilter.value !== undefined) {
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
const formDialog = reactive<FormDialogState>({
  isEdit: false,
  form: {
    name: '',
    description: '',
    schema: {
      fields: []
    },
    category_id: undefined
  }
});

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入表单名称', trigger: 'blur' },
    { min: 3, max: 50, message: '长度应为3到50个字符', trigger: 'blur' }
  ],
  category_id: [
    { required: true, message: '请选择分类', trigger: 'change' }
  ]
};

// 克隆对话框
const cloneDialog = reactive<CloneDialogState>({
  form: {
    name: '',
    originalId: 0
  }
});

// 详情对话框
const detailDialog = reactive<DetailDialogState>({
  form: null
});

// 预览对话框
const previewDialog = reactive<PreviewDialogState>({
  form: null
});

const parsedSchema = computed(() => {
  if (!detailDialog.form || !detailDialog.form.schema) {
    return { fields: [] };
  }
  
  try {
    return typeof detailDialog.form.schema === 'string' 
      ? JSON.parse(detailDialog.form.schema)
      : detailDialog.form.schema;
  } catch (error) {
    console.error('解析schema失败:', error);
    return { fields: [] };
  }
});

const parsedPreviewSchema = computed(() => {
  if (!previewDialog.form || !previewDialog.form.schema) {
    return { fields: [] };
  }
  
  try {
    return typeof previewDialog.form.schema === 'string' 
      ? JSON.parse(previewDialog.form.schema)
      : previewDialog.form.schema;
  } catch (error) {
    console.error('解析预览schema失败:', error);
    return { fields: [] };
  }
});

// 重置预览表单
const resetPreviewForm = (): void => {
  if (previewDialog.form) {
    initPreviewFormData(previewDialog.form.schema);
    message.success('表单已重置');
  }
};

// 字段验证方法
const validateFieldName = (field: FormField, index: number): void => {
  // 触发重新计算验证状态
  field.name = field.name.trim();
};

const getFieldNameStatus = (name: string, index: number): string => {
  const error = getFieldNameError(name, index);
  return error ? 'error' : '';
};

const getFieldNameError = (name: string, currentIndex: number): string => {
  if (!name.trim()) {
    return '字段名称不能为空';
  }
  
  // 验证字段名格式
  const namePattern = /^[a-zA-Z][a-zA-Z0-9_]*$/;
  if (!namePattern.test(name)) {
    return '字段名称必须以字母开头，只能包含字母、数字和下划线';
  }
  
  // 检查重复
  const fieldNames = formDialog.form.schema.fields.map((field, index) => 
    index === currentIndex ? name : field.name
  );
  const duplicateCount = fieldNames.filter(n => n === name).length;
  if (duplicateCount > 1) {
    return '字段名称不能重复';
  }
  
  return '';
};

const getOptionsError = (field: FormField): string => {
  if (!['select', 'radio', 'checkbox'].includes(field.type)) {
    return '';
  }
  
  if (!field.options || field.options.length === 0) {
    return '选项类型字段至少需要一个选项';
  }
  
  // 检查是否有空的选项
  const hasEmptyOption = field.options.some(option => 
    !option.label.trim() || !option.value.trim()
  );
  
  if (hasEmptyOption) {
    return '所有选项的标签和值都不能为空';
  }
  
  // 检查选项值是否重复
  const values = field.options.map(option => option.value);
  const uniqueValues = new Set(values);
  if (values.length !== uniqueValues.size) {
    return '选项值不能重复';
  }
  
  return '';
};

const getOptionsHelp = (field: FormField): string => {
  if (!['select', 'radio', 'checkbox'].includes(field.type)) {
    return '';
  }
  return '至少需要添加一个选项，且所有选项的标签和值都不能为空';
};

const getFieldPanelHeader = (field: FormField, index: number): string => {
  if (field.label) {
    return field.label;
  }
  return `字段 ${index + 1}`;
};

const getFieldPanelClass = (field: FormField): string => {
  const hasErrors = validateField(field);
  return hasErrors ? 'field-panel-error' : '';
};

const validateField = (field: FormField): boolean => {
  // 检查基本字段
  if (!field.label.trim() || !field.name.trim()) {
    return true;
  }
  
  // 检查字段名格式
  const namePattern = /^[a-zA-Z][a-zA-Z0-9_]*$/;
  if (!namePattern.test(field.name)) {
    return true;
  }
  
  // 检查选项类型字段的选项
  if (['select', 'radio', 'checkbox'].includes(field.type)) {
    if (!field.options || field.options.length === 0) {
      return true;
    }
    
    const hasEmptyOption = field.options.some(option => 
      !option.label.trim() || !option.value.trim()
    );
    
    if (hasEmptyOption) {
      return true;
    }
  }
  
  return false;
};

const handleFieldTypeChange = (field: FormField): void => {
  // 当字段类型改变时，初始化选项
  if (['select', 'radio', 'checkbox'].includes(field.type)) {
    if (!field.options || field.options.length === 0) {
      field.options = [{ label: '', value: '' }];
    }
  } else {
    // 非选项类型字段，清空选项
    field.options = [];
  }
};

const validateFormFields = (): string[] => {
  const errors: string[] = [];
  
  if (formDialog.form.schema.fields.length === 0) {
    errors.push('表单至少需要一个字段');
    return errors;
  }
  
  // 收集所有字段名用于重复检查
  const fieldNames = formDialog.form.schema.fields.map(field => field.name);
  const uniqueFieldNames = new Set(fieldNames.filter(name => name.trim()));
  
  if (fieldNames.length !== uniqueFieldNames.size) {
    errors.push('存在重复的字段名称');
  }
  
  formDialog.form.schema.fields.forEach((field, index) => {
    const fieldPrefix = `字段${index + 1}`;
    
    // 验证基本字段
    if (!field.label.trim()) {
      errors.push(`${fieldPrefix}: 标签不能为空`);
    }
    
    if (!field.name.trim()) {
      errors.push(`${fieldPrefix}: 字段名称不能为空`);
    } else {
      // 验证字段名格式
      const namePattern = /^[a-zA-Z][a-zA-Z0-9_]*$/;
      if (!namePattern.test(field.name)) {
        errors.push(`${fieldPrefix}: 字段名称格式不正确（必须以字母开头，只能包含字母、数字和下划线）`);
      }
    }
    
    // 验证选项类型字段
    if (['select', 'radio', 'checkbox'].includes(field.type)) {
      if (!field.options || field.options.length === 0) {
        errors.push(`${fieldPrefix}: 选项类型字段必须包含至少一个选项`);
      } else {
        // 检查选项内容
        const hasEmptyOption = field.options.some(option => 
          !option.label.trim() || !option.value.trim()
        );
        
        if (hasEmptyOption) {
          errors.push(`${fieldPrefix}: 所有选项的标签和值都不能为空`);
        }
        
        // 检查选项值重复
        const values = field.options.map(option => option.value);
        const uniqueValues = new Set(values);
        if (values.length !== uniqueValues.size) {
          errors.push(`${fieldPrefix}: 选项值不能重复`);
        }
      }
    }
  });
  
  return errors;
};

// 对话框控制方法
const closeFormDialog = (): void => {
  formDialogVisible.value = false;
  formValidationErrors.value = [];
};

const closeCloneDialog = (): void => {
  cloneDialogVisible.value = false;
};

const closeDetailDialog = (): void => {
  detailDialogVisible.value = false;
};

const closePreviewDialog = (): void => {
  previewDialogVisible.value = false;
  previewDialog.form = null;
  previewFormData.value = {};
};

// 标签页切换处理
const handleTabChange = async (activeKey: string): Promise<void> => {
  if (activeKey === '2' && detailDialog.form) {
    // 切换到预览标签页时加载预览数据
    await loadPreviewData(detailDialog.form.id);
  }
};

// 处理预览
const handlePreviewForm = async (row: FormDesignItem): Promise<void> => {
  previewLoading.value = true;
  previewDialogVisible.value = true;
  
  try {
    const response = await previewFormDesign({ id: row.id });
    if (response) {
      previewDialog.form = response;
      // 初始化预览表单数据
      initPreviewFormData(response.schema);
    }
  } catch (error) {
    console.error('加载预览数据失败:', error);
    message.error('加载预览数据失败');
  } finally {
    previewLoading.value = false;
  }
};

// 加载预览数据
const loadPreviewData = async (formId: number): Promise<void> => {
  previewLoading.value = true;
  try {
    const response = await previewFormDesign({ id: formId });
    if (response && response.data) {
      // 初始化预览表单数据
      initPreviewFormData(response.data.schema);
    }
  } catch (error) {
    console.error('加载预览数据失败:', error);
    message.error('加载预览数据失败');
  } finally {
    previewLoading.value = false;
  }
};

// 初始化预览表单数据
const initPreviewFormData = (schema: any): void => {
  const data: Record<string, any> = {};
  
  try {
    const schemaObj = typeof schema === 'string' ? JSON.parse(schema) : schema;
    
    if (schemaObj.fields && Array.isArray(schemaObj.fields)) {
      schemaObj.fields.forEach((field: FormField) => {
        // 根据字段类型设置默认值
        switch (field.type) {
          case 'text':
          case 'textarea':
            data[field.name] = field.default_value || '';
            break;
          case 'number':
            data[field.name] = field.default_value || undefined;
            break;
          case 'date':
            data[field.name] = field.default_value || undefined;
            break;
          case 'select':
          case 'radio':
            data[field.name] = field.default_value || undefined;
            break;
          case 'checkbox':
            data[field.name] = field.default_value || [];
            break;
          default:
            data[field.name] = field.default_value || '';
        }
      });
    }
  } catch (error) {
    console.error('初始化预览表单数据失败:', error);
  }
  
  previewFormData.value = data;
};

// 加载表单列表
const loadFormDesigns = async (): Promise<void> => {
  loading.value = true;
  try {
    const params: ListFormDesignReq = {
      page: currentPage.value,
      size: pageSize.value,
      search: searchQuery.value || undefined,
      status: statusFilter.value !== undefined ? statusFilter.value : undefined
    };
    const response = await listFormDesign(params);
    if (response) {
      formDesigns.value = Array.isArray(response.items) ? response.items : [];
      updateStats(response);
    }
  } catch (error) {
    console.error('加载表单列表失败:', error);
    message.error('加载表单列表失败');
  } finally {
    loading.value = false;
  }
};

// 更新统计数据
const updateStats = (data: any): void => {
  if (data.statistics) {
    stats.total = data.statistics.total || 0;
    stats.published = data.statistics.published || 0;
    stats.draft = data.statistics.draft || 0;
    stats.disabled = data.statistics.disabled || 0;
  } else {
    // 从列表数据计算
    stats.total = formDesigns.value.length;
    stats.published = formDesigns.value.filter((form: { status: number }) => form.status === 1).length;
    stats.draft = formDesigns.value.filter((form: { status: number }) => form.status === 0).length;
    stats.disabled = formDesigns.value.filter((form: { status: number }) => form.status === 2).length;
  }
};

// 方法
const handleSizeChange = (current: number, size: number): void => {
  pageSize.value = size;
  currentPage.value = current;
  loadFormDesigns();
};

const handleCurrentChange = (page: number): void => {
  currentPage.value = page;
  loadFormDesigns();
};

const handleSearch = (): void => {
  currentPage.value = 1;
  loadFormDesigns();
};

const handleStatusChange = (): void => {
  currentPage.value = 1;
  loadFormDesigns();
};

const handleCreateForm = (): void => {
  formDialog.isEdit = false;
  formDialog.form = {
    name: '',
    description: '',
    schema: {
      fields: []
    },
    category_id: undefined
  };
  formValidationErrors.value = [];
  formDialogVisible.value = true;
};

const handleEditForm = async (row: FormDesignItem | FormDesignResp): Promise<void> => {
  loading.value = true;
  try {
    const response = await detailFormDesign({ id: row.id });
    if (response) {
      const formData = response;
      formDialog.isEdit = true;
      
      // 解析schema字符串为对象
      let schemaObj: FormSchema;
      try {
        schemaObj = typeof formData.schema === 'string' 
          ? JSON.parse(formData.schema)
          : formData.schema;
      } catch (error) {
        console.error('解析schema失败:', error);
        schemaObj = { fields: [] };
      }
      
      formDialog.form = {
        id: formData.id,
        name: formData.name,
        description: formData.description,
        schema: schemaObj,
        category_id: formData.category_id
      };
      
      formValidationErrors.value = [];
      formDialogVisible.value = true;
      detailDialogVisible.value = false;
    }
  } catch (error) {
    console.error('加载表单详情失败:', error);
    message.error('加载表单详情失败');
  } finally {
    loading.value = false;
  }
};

const handleViewForm = async (row: FormDesignItem): Promise<void> => {
  loading.value = true;
  try {
    const response = await detailFormDesign({ id: row.id });
    if (response) {
      detailDialog.form = response;
      detailDialogVisible.value = true;
    }
  } catch (error) {
    console.error('加载表单详情失败:', error);
    message.error('加载表单详情失败');
  } finally {
    loading.value = false;
  }
};

const handleMenuClick = (command: string, row: FormDesignItem): void => {
  switch (command) {
    case 'preview':
      handlePreviewForm(row);
      break;
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

const publishForm = async (form: FormDesignItem): Promise<void> => {
  try {
    const params: PublishFormDesignReq = { id: form.id };
    const response = await publishFormDesign(params);
    if (response) {
      message.success(`表单 "${form.name}" 已发布`);
      loadFormDesigns();
    }
  } catch (error) {
    console.error('发布表单失败:', error);
    message.error('发布表单失败');
  }
};

const unpublishForm = async (form: FormDesignItem): Promise<void> => {
  try {
    // 先获取详细信息
    const detailResponse = await detailFormDesign({ id: form.id });
    if (detailResponse && detailResponse.data) {
      const formData = detailResponse.data;
      
      const schemaObj = typeof formData.schema === 'string' 
        ? JSON.parse(formData.schema)
        : formData.schema;
        
      const params: FormDesignReq = {
        id: form.id,
        name: form.name,
        description: form.description,
        schema: schemaObj,
        category_id: form.category_id
      };
      
      const response = await updateFormDesign(params);
      if (response) {
        message.success(`表单 "${form.name}" 已取消发布`);
        loadFormDesigns();
      }
    }
  } catch (error) {
    console.error('取消发布表单失败:', error);
    message.error('取消发布表单失败');
  }
};

const showCloneDialog = (form: FormDesignItem): void => {
  cloneDialog.form.name = `${form.name} 的副本`;
  cloneDialog.form.originalId = form.id;
  cloneDialogVisible.value = true;
};

const confirmClone = async (): Promise<void> => {
  if (!cloneDialog.form.name.trim()) {
    message.error('请输入新表单名称');
    return;
  }
  
  try {
    const params: CloneFormDesignReq = {
      id: cloneDialog.form.originalId, 
      name: cloneDialog.form.name    
    };
    
    const response = await cloneFormDesign(params);
    if (response) {
      message.success(`表单已克隆为 "${cloneDialog.form.name}"`);
      cloneDialogVisible.value = false;
      loadFormDesigns();
    }
  } catch (error) {
    console.error('克隆表单失败:', error);
    message.error('克隆表单失败');
  }
};

const confirmDelete = (form: FormDesignItem): void => {
  Modal.confirm({
    title: '警告',
    content: `确定要删除表单 "${form.name}" 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      try {
        const params: DetailFormDesignReq = { id: form.id };
        await deleteFormDesign(params);
        message.success(`表单 "${form.name}" 已删除`);
        loadFormDesigns();
      } catch (error) {
        console.error('删除表单失败:', error);
        message.error('删除表单失败');
      }
    }
  });
};

const addField = (): void => {
  const newField: FormField = {
    id: `field_${Date.now()}`,
    type: 'text',
    label: '',
    name: '',
    required: false,
    options: []
  };
  formDialog.form.schema.fields.push(newField);
};

const removeField = (index: number): void => {
  formDialog.form.schema.fields.splice(index, 1);
  // 重新验证表单
  formValidationErrors.value = validateFormFields();
};

// 添加选项
const addOption = (field: FormField): void => {
  if (!field.options) {
    field.options = [];
  }
  field.options.push({ label: '', value: '' });
};

// 删除选项
const removeOption = (field: FormField, index: number): void => {
  if (field.options) {
    field.options.splice(index, 1);
  }
};

const saveForm = async (): Promise<void> => {
  // 先验证基本信息
  if (formDialog.form.name.trim() === '') {
    message.error('表单名称不能为空');
    return;
  }

  if (!formDialog.form.category_id) {
    message.error('请选择分类');
    return;
  }
  
  // 验证字段
  const fieldErrors = validateFormFields();
  formValidationErrors.value = fieldErrors;
  
  if (fieldErrors.length > 0) {
    message.error('表单验证失败，请检查字段配置');
    return;
  }

  try {
    const formData: FormDesignReq = {
      ...formDialog.form,
      schema: formDialog.form.schema
    };

    if (formDialog.isEdit) {
      await updateFormDesign(formData);
      message.success(`表单 "${formDialog.form.name}" 已更新`);
    } else {
      await createFormDesign(formData);
      message.success(`表单 "${formDialog.form.name}" 已创建`);
    }
    
    formDialogVisible.value = false;
    formValidationErrors.value = [];
    loadFormDesigns();
  } catch (error) {
    console.error('保存表单失败:', error);
    message.error('保存表单失败');
  }
};

// 获取字段类型名称
const getFieldTypeName = (type: string): string => {
  const typeMap: Record<string, string> = {
    'text': '文本框',
    'number': '数字',
    'date': '日期',
    'select': '下拉选择',
    'checkbox': '复选框',
    'radio': '单选框',
    'textarea': '多行文本'
  };
  return typeMap[type] || type;
};

// 辅助方法
const formatDate = (dateStr: string): string => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const formatTime = (dateStr: string): string => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
};

const formatFullDateTime = (dateStr: string): string => {
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

const getInitials = (name: string): string => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getStatusClass = (status: number): string => {
  switch (status) {
    case 0: return 'status-draft';
    case 1: return 'status-published';
    case 2: return 'status-disabled';
    default: return '';
  }
};

const getAvatarColor = (name: string): string => {
  const colors = [
    '#1890ff', '#52c41a', '#faad14', '#f5222d',
    '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'
  ];
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  return colors[Math.abs(hash) % colors.length]!;
};

// 修复加载分类函数
const loadCategories = async (): Promise<void> => {
  try {
    // 请求所有分类数据，不进行分页
    const response = await listCategory({ page: 1, size: 100 });
    if (response && response.items) {
      categories.value = response.items;
      console.log('分类数据加载成功:', categories.value);
    } else {
      console.warn('分类接口返回数据格式异常:', response);
      categories.value = [];
    }
  } catch (error) {
    console.error('加载分类列表失败:', error);
    message.error('加载分类列表失败');
    categories.value = [];
  }
};

// 初始化 - 修复函数调用
onMounted(async () => {
  // 并行加载表单设计列表和分类数据
  await Promise.all([
    loadFormDesigns(),
    loadCategories()
  ]);
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

.option-item {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

/* 字段验证错误样式 */
.field-error {
  color: #ff4d4f;
  font-size: 12px;
  margin-top: 4px;
}

.field-panel-error :deep(.ant-collapse-header) {
  border-left: 3px solid #ff4d4f;
  background-color: #fff2f0;
}

.form-validation-summary {
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

/* 预览表单样式 */
.form-preview-wrapper {
  background: #fafafa;
  border-radius: 8px;
  padding: 24px;
  min-height: 400px;
}

.preview-header {
  text-align: center;
  margin-bottom: 32px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e8e8e8;
}

.preview-header h3 {
  margin: 0 0 8px 0;
  font-size: 24px;
  color: #1f2937;
  font-weight: 600;
}

.preview-description {
  margin: 0 0 16px 0;
  color: #666;
  font-size: 14px;
}

.preview-mode-notice {
  margin-top: 16px;
}

.preview-form {
  background: white;
  border-radius: 8px;
  padding: 32px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  max-width: 600px;
  margin: 0 auto;
}

.dynamic-form .form-field {
  margin-bottom: 24px;
}

.dynamic-form .form-field :deep(.ant-form-item-label) {
  font-weight: 500;
  color: #333;
}

.dynamic-form .form-field :deep(.ant-form-item-required::before) {
  content: '*';
  color: #ff4d4f;
  margin-right: 4px;
}

/* 预览模式下的输入框样式 */
.preview-input {
  transition: all 0.3s ease;
}

.preview-input:hover {
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.2);
  border-color: #40a9ff;
}

.preview-input:focus {
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.3);
  border-color: #1890ff;
}

/* 单选框和复选框组样式 */
.preview-radio-group,
.preview-checkbox-group {
  padding: 8px;
  border-radius: 6px;
  transition: all 0.3s ease;
}

.preview-radio-group:hover,
.preview-checkbox-group:hover {
  background-color: #f5f5f5;
}

.radio-options,
.checkbox-options {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.preview-radio,
.preview-checkbox {
  transition: all 0.3s ease;
  padding: 4px 8px;
  border-radius: 4px;
}

.preview-radio:hover,
.preview-checkbox:hover {
  background-color: #e6f7ff;
}

.preview-form-actions {
  margin-top: 32px;
  text-align: center;
  padding-top: 24px;
  border-top: 1px solid #f0f0f0;
  display: flex;
  justify-content: center;
  gap: 12px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .form-preview-wrapper {
    padding: 16px;
  }
  
  .preview-form {
    padding: 20px;
  }
  
  .preview-header h3 {
    font-size: 20px;
  }
  
  .radio-options,
  .checkbox-options {
    gap: 8px;
  }
  
  .preview-form-actions {
    flex-direction: column;
    align-items: center;
  }
  
  .preview-form-actions .ant-btn {
    width: 100%;
    max-width: 200px;
  }
}

/* 焦点状态增强 */
.preview-input:focus-within,
.preview-radio-group:focus-within,
.preview-checkbox-group:focus-within {
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
  border-radius: 6px;
}

/* 选中状态样式 */
.preview-radio :deep(.ant-radio-checked .ant-radio-inner),
.preview-checkbox :deep(.ant-checkbox-checked .ant-checkbox-inner) {
  background-color: #1890ff;
  border-color: #1890ff;
}

/* 禁用状态的提示样式 */
.preview-form-actions .ant-btn[disabled] {
  opacity: 0.7;
  cursor: not-allowed;
}
</style>