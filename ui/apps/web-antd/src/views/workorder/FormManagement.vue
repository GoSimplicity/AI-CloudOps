<template>
    <div class="form-management-container">
      <div class="page-header">
        <div class="header-actions">
          <a-button type="primary" @click="handleCreateForm" class="btn-create">
            <template #icon>
              <PlusOutlined />
            </template>
            <span class="btn-text">创建新表单</span>
          </a-button>
          <div class="search-filters">
            <a-input-search 
              v-model:value="searchQuery" 
              placeholder="搜索表单..." 
              class="search-input"
              @search="handleSearch"
              @change="handleSearchChange"
              allow-clear 
            />
            <a-select 
              v-model:value="categoryFilter" 
              placeholder="选择分类" 
              class="category-filter"
              @change="handleCategoryChange"
              allow-clear
            >
              <a-select-option :value="undefined">全部分类</a-select-option>
              <a-select-option v-for="category in categories" :key="category.id" :value="category.id">
                {{ category.name }}
              </a-select-option>
            </a-select>
            <a-select 
              v-model:value="statusFilter" 
              placeholder="状态" 
              class="status-filter"
              @change="handleStatusChange"
              allow-clear
            >
              <a-select-option :value="undefined">全部状态</a-select-option>
              <a-select-option :value="1">草稿</a-select-option>
              <a-select-option :value="2">已发布</a-select-option>
              <a-select-option :value="3">已禁用</a-select-option>
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
              <a-statistic title="总表单数" :value="stats.total" :value-style="{ color: '#3f8600' }">
                <template #prefix>
                  <FormOutlined />
                </template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :xs="12" :sm="12" :md="6" :lg="6">
            <a-card class="stats-card">
              <a-statistic title="已发布" :value="stats.published" :value-style="{ color: '#52c41a' }">
                <template #prefix>
                  <CheckCircleOutlined />
                </template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :xs="12" :sm="12" :md="6" :lg="6">
            <a-card class="stats-card">
              <a-statistic title="草稿" :value="stats.draft" :value-style="{ color: '#faad14' }">
                <template #prefix>
                  <EditOutlined />
                </template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :xs="12" :sm="12" :md="6" :lg="6">
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
            :data-source="formDesigns" 
            :columns="columns" 
            :pagination="paginationConfig" 
            :loading="loading" 
            row-key="id"
            bordered
            :scroll="{ x: 1200 }"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <div class="form-name-cell">
                  <div class="form-badge" :class="getStatusClass(record.status)"></div>
                  <span class="form-name-text">{{ record.name }}</span>
                </div>
              </template>
  
              <template v-if="column.key === 'category'">
                <a-tag v-if="record.category_name" color="blue">
                  {{ record.category_name }}
                </a-tag>
                <span v-else class="text-gray">未分类</span>
              </template>
  
              <template v-if="column.key === 'description'">
                <span class="description-text">{{ record.description || '无描述' }}</span>
              </template>
  
              <template v-if="column.key === 'version'">
                <a-tag color="blue">v{{ record.version }}</a-tag>
              </template>
  
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">
                  {{ getStatusText(record.status) }}
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
                        <a-menu-item key="publish" v-if="record.status === 1">发布</a-menu-item>
                        <a-menu-item key="unpublish" v-if="record.status === 2">取消发布</a-menu-item>
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
        </a-card>
      </div>
  
      <!-- 基础表单创建/编辑对话框 - 简化版 -->
      <a-modal 
        :open="formDialogVisible" 
        :title="formDialog.isEdit ? '编辑表单信息' : '创建表单'" 
        :width="dialogWidth"
        @ok="saveForm" 
        @cancel="closeFormDialog"
        :destroy-on-close="true"
        class="responsive-modal"
      >
        <a-form ref="formRef" :model="formDialog.form" :rules="formRules" layout="vertical">
          <a-form-item label="表单名称" name="name">
            <a-input v-model:value="formDialog.form.name" placeholder="请输入表单名称" />
          </a-form-item>
  
          <a-form-item label="描述" name="description">
            <a-textarea v-model:value="formDialog.form.description" :rows="3" placeholder="请输入表单描述" />
          </a-form-item>
  
          <a-form-item label="分类" name="category_id">
            <a-select v-model:value="formDialog.form.category_id" placeholder="请选择分类" style="width: 100%">
              <a-select-option v-for="cat in categories" :key="cat.id" :value="cat.id">
                {{ cat.name }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="状态" name="status" v-if="formDialog.isEdit">
            <a-select v-model:value="formDialog.form.status" placeholder="请选择状态" style="width: 100%">
              <a-select-option :value="1">草稿</a-select-option>
              <a-select-option :value="2">已发布</a-select-option>
              <a-select-option :value="3">已禁用</a-select-option>
            </a-select>
          </a-form-item>

          <a-alert
            v-if="!formDialog.isEdit"
            message="提示"
            description="表单创建后需要单独设计表单结构，请前往表单设计页面完善表单字段。"
            type="info"
            show-icon
            style="margin-bottom: 16px;"
          />
        </a-form>
      </a-modal>
  
      <!-- 克隆对话框 -->
      <a-modal 
        :open="cloneDialogVisible" 
        title="克隆表单" 
        :width="dialogWidth"
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
        :width="previewDialogWidth"
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
                <template v-for="field in previewDialog.form.schema.fields" :key="field.id">
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
        :width="previewDialogWidth"
        :footer="null" 
        @cancel="closeDetailDialog"
        class="detail-dialog"
      >
        <div v-if="detailDialog.form" class="form-details">
          <div class="detail-header">
            <h2>{{ detailDialog.form.name }}</h2>
            <a-tag :color="getStatusColor(detailDialog.form.status)">
              {{ getStatusText(detailDialog.form.status) }}
            </a-tag>
          </div>
  
          <a-descriptions bordered :column="1" :labelStyle="{ width: '120px' }">
            <a-descriptions-item label="ID">{{ detailDialog.form.id }}</a-descriptions-item>
            <a-descriptions-item label="版本">v{{ detailDialog.form.version }}</a-descriptions-item>
            <a-descriptions-item label="分类">
              <a-tag v-if="detailDialog.form.category" color="blue">
                {{ detailDialog.form.category.name }}
              </a-tag>
              <span v-else class="text-gray">未分类</span>
            </a-descriptions-item>
            <a-descriptions-item label="创建人">{{ detailDialog.form.creator_name }}</a-descriptions-item>
            <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.form.created_at || '') }}</a-descriptions-item>
            <a-descriptions-item label="描述">{{ detailDialog.form.description || '无描述' }}</a-descriptions-item>
          </a-descriptions>
  
          <div class="schema-preview">
            <h3>表单结构</h3>
            <a-table 
              :data-source="detailDialog.form.schema.fields" 
              :columns="schemaColumns" 
              :pagination="false" 
              bordered
              size="small" 
              row-key="name"
              :scroll="{ x: 600 }"
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
  
          <div class="detail-footer">
            <a-button @click="closeDetailDialog">关闭</a-button>
            <a-button type="primary" @click="handleEditForm(detailDialog.form)">编辑</a-button>
            <a-button type="default" @click="handleDesignForm(detailDialog.form)">设计表单</a-button>
          </div>
        </div>
      </a-modal>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, reactive, onMounted, computed, watch } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  import { useRouter } from 'vue-router';
  import {
    PlusOutlined,
    FormOutlined,
    CheckCircleOutlined,
    EditOutlined,
    StopOutlined,
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
    type FormField,
    type FormSchema,
    type ListFormDesignReq,
    type CreateFormDesignReq,
    type UpdateFormDesignReq,
  } from '#/api/core/workorder_form_design';
  import type { Category } from '#/api/core/workorder_category';
  import { listCategory } from '#/api/core/workorder_category';
  
  const router = useRouter();
  
  // 响应式对话框宽度
  const dialogWidth = computed(() => {
    if (typeof window !== 'undefined') {
      const width = window.innerWidth;
      if (width < 768) return '95%';
      if (width < 1024) return '80%';
      return '600px'; // 简化后宽度更小
    }
    return '600px';
  });
  
  const previewDialogWidth = computed(() => {
    if (typeof window !== 'undefined') {
      const width = window.innerWidth;
      if (width < 768) return '95%';
      if (width < 1024) return '90%';
      return '80%';
    }
    return '80%';
  });
  
  // 列定义
  const columns = [
    { title: '表单名称', dataIndex: 'name', key: 'name', width: 180, fixed: 'left' },
    { title: '分类', dataIndex: 'category_name', key: 'category', width: 120, align: 'center' as const },
    { title: '描述', dataIndex: 'description', key: 'description', width: 200, ellipsis: true },
    { title: '版本', dataIndex: 'version', key: 'version', width: 100, align: 'center' as const },
    { title: '状态', dataIndex: 'status', key: 'status', width: 120, align: 'center' as const },
    { title: '创建人', dataIndex: 'creator_name', key: 'creator', width: 150 },
    { title: '创建时间', dataIndex: 'created_at', key: 'createdAt', width: 180 },
    { title: '操作', key: 'action', width: 200, align: 'center' as const, fixed: 'right' }
  ];
  
  const schemaColumns = [
    { title: '类型', dataIndex: 'type', key: 'type', width: 120 },
    { title: '标签', dataIndex: 'label', key: 'label', width: 180 },
    { title: '字段名', dataIndex: 'name', key: 'name', width: 180 },
    { title: '是否必填', dataIndex: 'required', key: 'required', width: 100 }
  ];
  
  // 状态数据
  const loading = ref(false);
  const previewLoading = ref(false);
  const searchQuery = ref('');
  const categoryFilter = ref<number | undefined>(undefined);
  const statusFilter = ref<number | undefined>(undefined);
  const formDesigns = ref<FormDesignResp[]>([]);
  const categories = ref<Category[]>([]);
  const previewFormData = ref<Record<string, any>>({});
  
  // 防抖处理
  let searchTimeout: any = null;
  
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
  
  // 统计数据
  const stats = reactive({
    total: 0,
    published: 0,
    draft: 0,
    disabled: 0
  });
  
  // 对话框状态
  const formDialogVisible = ref(false);
  const cloneDialogVisible = ref(false);
  const detailDialogVisible = ref(false);
  const previewDialogVisible = ref(false);
  
  // 简化的表单对话框数据 - 移除schema相关
  const formDialog = reactive({
    isEdit: false,
    form: {
      id: undefined as number | undefined,
      name: '',
      description: '',
      category_id: undefined as number | undefined,
      status: 1 as number // 新增状态字段
    }
  });
  
  // 克隆对话框数据
  const cloneDialog = reactive({
    form: {
      name: '',
      originalId: 0
    }
  });
  
  // 详情对话框数据
  const detailDialog = reactive({
    form: null as FormDesignResp | null
  });
  
  // 预览对话框数据
  const previewDialog = reactive({
    form: null as FormDesignResp | null
  });
  
  // 简化的表单验证规则
  const formRules = {
    name: [
      { required: true, message: '请输入表单名称', trigger: 'blur' },
      { min: 3, max: 50, message: '长度应为3到50个字符', trigger: 'blur' }
    ],
    category_id: [
      { required: true, message: '请选择分类', trigger: 'change' }
    ]
  };
  
  // 辅助方法
  const getStatusColor = (status: number): string => {
    const colorMap = { 1: 'orange', 2: 'green', 3: 'default' };
    return colorMap[status as keyof typeof colorMap] || 'default';
  };
  
  const getStatusText = (status: number): string => {
    const textMap = { 1: '草稿', 2: '已发布', 3: '已禁用' };
    return textMap[status as keyof typeof textMap] || '未知';
  };
  
  const getStatusClass = (status: number): string => {
    const classMap = { 1: 'status-draft', 2: 'status-published', 3: 'status-disabled' };
    return classMap[status as keyof typeof classMap] || '';
  };
  
  const getFieldTypeName = (type: string): string => {
    const typeMap: Record<string, string> = {
      text: '文本框', number: '数字', date: '日期', select: '下拉选择',
      checkbox: '复选框', radio: '单选框', textarea: '多行文本'
    };
    return typeMap[type] || type;
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
  
  // 数据加载
  const loadFormDesigns = async (): Promise<void> => {
    loading.value = true;
    try {
      const params: ListFormDesignReq = {
        page: paginationConfig.current,
        size: paginationConfig.pageSize,
        search: searchQuery.value || undefined,
        status: statusFilter.value,
        category_id: categoryFilter.value
      };
      
      const response = await listFormDesign(params);
      if (response) {
        formDesigns.value = response.items || [];
        paginationConfig.total = response.total || 0;
        updateStats();
      }
    } catch (error) {
      console.error('加载表单列表失败:', error);
      message.error('加载表单列表失败');
    } finally {
      loading.value = false;
    }
  };
  
  const loadCategories = async (): Promise<void> => {
    try {
      const response = await listCategory({ page: 1, size: 100 });
      categories.value = response.items || [];
    } catch (error) {
      console.error('加载分类列表失败:', error);
      message.error('加载分类列表失败');
    }
  };
  
  const updateStats = (): void => {
    stats.total = formDesigns.value.length;
    stats.published = formDesigns.value.filter(form => form.status === 2).length;
    stats.draft = formDesigns.value.filter(form => form.status === 1).length;
    stats.disabled = formDesigns.value.filter(form => form.status === 3).length;
  };
  
  // 事件处理
  const handleTableChange = (pagination: any): void => {
    paginationConfig.current = pagination.current;
    paginationConfig.pageSize = pagination.pageSize;
    loadFormDesigns();
  };
  
  const handleSearch = (): void => {
    paginationConfig.current = 1;
    loadFormDesigns();
  };
  
  const handleSearchChange = (): void => {
    if (searchTimeout) {
      clearTimeout(searchTimeout);
    }
    searchTimeout = setTimeout(() => {
      paginationConfig.current = 1;
      loadFormDesigns();
    }, 500);
  };
  
  const handleCategoryChange = (): void => {
    paginationConfig.current = 1;
    loadFormDesigns();
  };
  
  const handleStatusChange = (): void => {
    paginationConfig.current = 1;
    loadFormDesigns();
  };
  
  const handleResetFilters = (): void => {
    searchQuery.value = '';
    categoryFilter.value = undefined;
    statusFilter.value = undefined;
    paginationConfig.current = 1;
    loadFormDesigns();
    message.success('过滤条件已重置');
  };
  
  const handleCreateForm = (): void => {
    formDialog.isEdit = false;
    formDialog.form = {
      id: undefined,
      name: '',
      description: '',
      category_id: undefined,
      status: 1
    };
    formDialogVisible.value = true;
  };
  
  const handleEditForm = async (record: FormDesignResp): Promise<void> => {
    try {
      const response = await detailFormDesign({ id: record.id });
      if (response) {
        formDialog.isEdit = true;
        formDialog.form = {
          id: response.id,
          name: response.name,
          description: response.description,
          category_id: response.category_id,
          status: response.status
        };
        formDialogVisible.value = true;
        detailDialogVisible.value = false;
      }
    } catch (error) {
      console.error('加载表单详情失败:', error);
      message.error('加载表单详情失败');
    }
  };
  
  const handleViewForm = async (record: FormDesignResp): Promise<void> => {
    try {
      const response = await detailFormDesign({ id: record.id });
      if (response) {
        detailDialog.form = response;
        detailDialogVisible.value = true;
      }
    } catch (error) {
      console.error('加载表单详情失败:', error);
      message.error('加载表单详情失败');
    }
  };

  // 新增：跳转到表单设计页面
  const handleDesignForm = (record: FormDesignResp): void => {
    // 这里可以跳转到表单设计页面，传递表单ID
    router.push({
      name: 'FormDesign', // 假设表单设计页面的路由名称
      params: { id: record.id },
      query: { mode: 'design' }
    });
    detailDialogVisible.value = false;
  };
  
  const handleMenuClick = (command: string, record: FormDesignResp): void => {
    switch (command) {
      case 'preview':
        handlePreviewForm(record);
        break;
      case 'publish':
        publishForm(record);
        break;
      case 'unpublish':
        unpublishForm(record);
        break;
      case 'clone':
        showCloneDialog(record);
        break;
      case 'delete':
        confirmDelete(record);
        break;
    }
  };
  
  const handlePreviewForm = async (record: FormDesignResp): Promise<void> => {
    previewLoading.value = true;
    previewDialogVisible.value = true;
    
    try {
      const response = await previewFormDesign({ id: record.id });
      if (response) {
        previewDialog.form = response;
        initPreviewFormData(response.schema);
      }
    } catch (error) {
      console.error('加载预览数据失败:', error);
      message.error('加载预览数据失败');
    } finally {
      previewLoading.value = false;
    }
  };
  
  const publishForm = async (record: FormDesignResp): Promise<void> => {
    try {
      await publishFormDesign({ id: record.id });
      message.success(`表单 "${record.name}" 已发布`);
      loadFormDesigns();
    } catch (error) {
      console.error('发布表单失败:', error);
      message.error('发布表单失败');
    }
  };
  
  const unpublishForm = async (record: FormDesignResp): Promise<void> => {
    try {
      const updateData: UpdateFormDesignReq = {
        id: record.id,
        name: record.name,
        description: record.description,
        schema: record.schema,
        category_id: record.category_id,
        status: 1
      };
      await updateFormDesign(updateData);
      message.success(`表单 "${record.name}" 已取消发布`);
      loadFormDesigns();
    } catch (error) {
      console.error('取消发布表单失败:', error);
      message.error('取消发布表单失败');
    }
  };
  
  const showCloneDialog = (record: FormDesignResp): void => {
    cloneDialog.form.name = `${record.name} 的副本`;
    cloneDialog.form.originalId = record.id;
    cloneDialogVisible.value = true;
  };
  
  const confirmClone = async (): Promise<void> => {
    if (!cloneDialog.form.name.trim()) {
      message.error('请输入新表单名称');
      return;
    }
    
    try {
      await cloneFormDesign({
        id: cloneDialog.form.originalId,
        name: cloneDialog.form.name
      });
      message.success(`表单已克隆为 "${cloneDialog.form.name}"`);
      cloneDialogVisible.value = false;
      loadFormDesigns();
    } catch (error) {
      console.error('克隆表单失败:', error);
      message.error('克隆表单失败');
    }
  };
  
  const confirmDelete = (record: FormDesignResp): void => {
    Modal.confirm({
      title: '警告',
      content: `确定要删除表单 "${record.name}" 吗？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      async onOk() {
        try {
          await deleteFormDesign({ id: record.id });
          message.success(`表单 "${record.name}" 已删除`);
          loadFormDesigns();
        } catch (error) {
          console.error('删除表单失败:', error);
          message.error('删除表单失败');
        }
      }
    });
  };
  
  // 简化的表单保存 - 不再处理schema
  const saveForm = async (): Promise<void> => {
    if (!formDialog.form.name.trim()) {
      message.error('表单名称不能为空');
      return;
    }
  
    if (!formDialog.form.category_id) {
      message.error('请选择分类');
      return;
    }
  
    try {
      if (formDialog.isEdit && formDialog.form.id) {
        // 编辑时需要保留原有的schema
        const existingForm = await detailFormDesign({ id: formDialog.form.id });
        if (existingForm) {
          const updateData: UpdateFormDesignReq = {
            id: formDialog.form.id,
            name: formDialog.form.name,
            description: formDialog.form.description,
            schema: existingForm.schema, // 保留原有架构
            category_id: formDialog.form.category_id,
            status: formDialog.form.status
          };
          await updateFormDesign(updateData);
          message.success(`表单 "${formDialog.form.name}" 已更新`);
        }
      } else {
        // 创建时使用空的schema
        const createData: CreateFormDesignReq = {
          name: formDialog.form.name,
          description: formDialog.form.description,
          schema: { fields: [] }, // 空的schema，需要后续设计
          category_id: formDialog.form.category_id
        };
        await createFormDesign(createData);
        message.success(`表单 "${formDialog.form.name}" 已创建，请前往设计页面完善表单结构`);
      }
      
      formDialogVisible.value = false;
      loadFormDesigns();
    } catch (error) {
      console.error('保存表单失败:', error);
      message.error('保存表单失败');
    }
  };
  
  // 预览表单数据初始化
  const initPreviewFormData = (schema: FormSchema): void => {
    const data: Record<string, any> = {};
    
    schema.fields.forEach((field: FormField) => {
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
    
    previewFormData.value = data;
  };
  
  const resetPreviewForm = (): void => {
    if (previewDialog.form) {
      initPreviewFormData(previewDialog.form.schema);
      message.success('表单已重置');
    }
  };
  
  // 对话框关闭
  const closeFormDialog = (): void => {
    formDialogVisible.value = false;
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
  
  // 监听过滤条件变化，自动刷新数据
  watch([categoryFilter, statusFilter], () => {
    if (paginationConfig.current !== 1) {
      paginationConfig.current = 1;
    }
    loadFormDesigns();
  });
  
  // 组件销毁时清理定时器
  onMounted(async () => {
    await Promise.all([loadFormDesigns(), loadCategories()]);
  });
  
  // 清理定时器
  const cleanup = () => {
    if (searchTimeout) {
      clearTimeout(searchTimeout);
    }
  };
  
  // 在组件卸载时清理
  onMounted(() => {
    return cleanup;
  });
  </script>
  
  <style scoped>
  .form-management-container {
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
  
  .category-filter,
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
    gap: 10px;
  }
  
  .form-badge {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
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
    word-break: break-all;
  }
  
  .description-text {
    color: #606266;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    overflow: hidden;
    word-break: break-all;
  }
  
  .text-gray {
    color: #999;
    font-style: italic;
  }
  
  .creator-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .creator-name {
    font-size: 14px;
    word-break: break-all;
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
    gap: 4px;
    justify-content: center;
    flex-wrap: wrap;
  }
  
  .detail-dialog .form-details {
    margin-bottom: 20px;
  }
  
  .detail-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 20px;
    flex-wrap: wrap;
  }
  
  .detail-header h2 {
    margin: 0;
    font-size: 24px;
    color: #1f2937;
    word-break: break-all;
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
    flex-wrap: wrap;
  }
  
  .form-preview-wrapper {
    background: #fafafa;
    border-radius: 8px;
    padding: 16px;
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
    word-break: break-all;
  }
  
  .preview-description {
    margin: 0 0 16px 0;
    color: #666;
    font-size: 14px;
    word-break: break-all;
  }
  
  .preview-mode-notice {
    margin-top: 16px;
  }
  
  .preview-form {
    background: white;
    border-radius: 8px;
    padding: 24px;
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
    flex-wrap: wrap;
  }
  
  .responsive-modal :deep(.ant-modal-content) {
    margin: 0;
  }
  
  /* 移动端适配 */
  @media (max-width: 768px) {
    .form-management-container {
      padding: 8px;
    }
    
    .header-actions {
      flex-direction: column;
      align-items: stretch;
    }
    
    .search-filters {
      width: 100%;
    }
    
    .search-input {
      width: 100%;
      min-width: auto;
    }
    
    .category-filter,
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
    
    .form-preview-wrapper {
      padding: 12px;
    }
    
    .preview-form {
      padding: 16px;
    }
    
    .preview-header h3 {
      font-size: 18px;
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
    
    .detail-footer {
      justify-content: center;
    }
    
    .detail-footer .ant-btn {
      flex: 1;
      max-width: 120px;
    }
  }
  
  /* 平板端适配 */
  @media (max-width: 1024px) and (min-width: 769px) {
    .form-management-container {
      padding: 16px;
    }
    
    .search-input {
      width: 200px;
    }
    
    .preview-form {
      padding: 20px;
    }
  }
  
  /* 超小屏幕适配 */
  @media (max-width: 480px) {
    .header-actions {
      gap: 8px;
    }
    
    .stats-card {
      text-align: center;
    }
    
    .creator-info {
      flex-direction: column;
      gap: 4px;
      align-items: center;
    }
    
    .creator-name {
      font-size: 12px;
    }
    
    .date-info {
      text-align: center;
    }
    
    .date {
      font-size: 12px;
    }
    
    .time {
      font-size: 10px;
    }
  }
  
  /* 表格滚动优化 */
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
      padding: 16px;
      max-height: calc(100vh - 160px);
      overflow-y: auto;
    }
  }
  
  /* 焦点状态优化 */
  .preview-input:focus-within,
  .preview-radio-group:focus-within,
  .preview-checkbox-group:focus-within {
    box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
    border-radius: 6px;
  }
  
  .preview-radio :deep(.ant-radio-checked .ant-radio-inner),
  .preview-checkbox :deep(.ant-checkbox-checked .ant-checkbox-inner) {
    background-color: #1890ff;
    border-color: #1890ff;
  }
  
  .preview-form-actions .ant-btn[disabled] {
    opacity: 0.7;
    cursor: not-allowed;
  }
  </style>