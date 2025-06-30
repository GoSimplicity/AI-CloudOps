<template>
  <div class="category-management-container">
    <div class="page-header">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreateCategory" class="btn-create">
          <template #icon>
            <PlusOutlined />
          </template>
          创建分类
        </a-button>
        <div class="search-filters">
          <a-input-search 
            v-model:value="searchQuery" 
            placeholder="搜索分类..." 
            class="search-input"
            @search="handleSearch"
            allow-clear 
          />
          <a-select 
            v-model:value="statusFilter" 
            placeholder="状态" 
            class="status-filter"
            @change="handleStatusChange"
          >
            <a-select-option :value="undefined">全部</a-select-option>
            <a-select-option :value="1">启用</a-select-option>
            <a-select-option :value="2">禁用</a-select-option>
          </a-select>
        </div>
      </div>
    </div>

    <div class="stats-row">
      <a-row :gutter="16">
        <a-col :span="8">
          <a-card class="stats-card">
            <a-statistic title="总分类数" :value="stats.total" :value-style="{ color: '#3f8600' }">
              <template #prefix>
                <FolderOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="8">
          <a-card class="stats-card">
            <a-statistic title="启用分类" :value="stats.enabled" :value-style="{ color: '#52c41a' }">
              <template #prefix>
                <CheckCircleOutlined />
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="8">
          <a-card class="stats-card">
            <a-statistic title="禁用分类" :value="stats.disabled" :value-style="{ color: '#cf1322' }">
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
          :data-source="categories" 
          :columns="columns" 
          :pagination="paginationConfig"
          :loading="loading" 
          row-key="id"
          bordered
          :scroll="{ x: 1000 }"
          @change="handleTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="category-name-cell">
                <div class="category-badge" :class="getStatusClass(record.status)"></div>
                <span v-if="record.icon" class="category-icon">{{ record.icon }}</span>
                <span class="category-name-text">{{ record.name }}</span>
              </div>
            </template>

            <template v-if="column.key === 'description'">
              <span class="description-text">{{ record.description || '无描述' }}</span>
            </template>

            <template v-if="column.key === 'sort_order'">
              <a-tag color="blue">{{ record.sort_order }}</a-tag>
            </template>

            <template v-if="column.key === 'status'">
              <a-tag :color="record.status === 1 ? 'green' : 'default'">
                {{ record.status === 1 ? '启用' : '禁用' }}
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
                <a-button type="primary" size="small" @click="handleViewCategory(record)">
                  查看
                </a-button>
                <a-button type="default" size="small" @click="handleEditCategory(record)">
                  编辑
                </a-button>
                <a-dropdown>
                  <template #overlay>
                    <a-menu @click="(e: any) => handleMenuClick(e.key, record)">
                      <a-menu-item key="enable" v-if="record.status === 2">
                        <CheckCircleOutlined />
                        启用
                      </a-menu-item>
                      <a-menu-item key="disable" v-if="record.status === 1">
                        <StopOutlined />
                        禁用
                      </a-menu-item>
                      <a-menu-divider />
                      <a-menu-item key="delete" danger>
                        <DeleteOutlined />
                        删除
                      </a-menu-item>
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

    <!-- 分类创建/编辑对话框 -->
    <a-modal 
      :open="categoryDialogVisible" 
      :title="categoryDialog.isEdit ? '编辑分类' : '创建分类'" 
      :width="dialogWidth"
      @ok="saveCategory" 
      @cancel="closeCategoryDialog"
      :destroy-on-close="true"
      :confirm-loading="saveLoading"
      class="responsive-modal"
    >
      <a-form ref="formRef" :model="categoryDialog.form" :rules="categoryRules" layout="vertical">
        <a-form-item label="分类名称" name="name">
          <a-input v-model:value="categoryDialog.form.name" placeholder="请输入分类名称" />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="图标" name="icon">
              <a-input v-model:value="categoryDialog.form.icon" placeholder="请输入图标（如：📁）" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="排序" name="sort_order">
              <a-input-number 
                v-model:value="categoryDialog.form.sort_order" 
                :min="0" 
                :max="999" 
                placeholder="排序值"
                style="width: 100%"
              />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="categoryDialog.form.description" :rows="3" placeholder="请输入分类描述" />
        </a-form-item>

        <a-form-item label="状态" name="status">
          <a-radio-group v-model:value="categoryDialog.form.status">
            <a-radio :value="1">启用</a-radio>
            <a-radio :value="2">禁用</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情对话框 -->
    <a-modal 
      :open="detailDialogVisible" 
      title="分类详情" 
      :width="dialogWidth" 
      :footer="null" 
      @cancel="closeDetailDialog"
      class="detail-dialog responsive-modal"
    >
      <div v-if="detailDialog.category" class="category-details">
        <div class="detail-header">
          <h2>
            <span v-if="detailDialog.category.icon" class="detail-icon">{{ detailDialog.category.icon }}</span>
            {{ detailDialog.category.name }}
          </h2>
          <a-tag :color="detailDialog.category.status === 1 ? 'green' : 'default'">
            {{ detailDialog.category.status === 1 ? '启用' : '禁用' }}
          </a-tag>
        </div>

        <a-descriptions bordered :column="2">
          <a-descriptions-item label="ID">{{ detailDialog.category.id }}</a-descriptions-item>
          <a-descriptions-item label="排序">{{ detailDialog.category.sort_order }}</a-descriptions-item>
          <a-descriptions-item label="创建人">{{ detailDialog.category.creator_name }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatFullDateTime(detailDialog.category.created_at || '') }}</a-descriptions-item>
          <a-descriptions-item label="更新时间" :span="2">{{ formatFullDateTime(detailDialog.category.updated_at || '') }}</a-descriptions-item>
          <a-descriptions-item label="描述" :span="2">{{ detailDialog.category.description || '无描述' }}</a-descriptions-item>
        </a-descriptions>

        <div class="detail-footer">
          <a-button @click="closeDetailDialog">关闭</a-button>
          <a-button type="primary" @click="handleEditCategory(detailDialog.category)">编辑</a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { message, Modal, type FormInstance } from 'ant-design-vue';
import {
  PlusOutlined,
  FolderOutlined,
  CheckCircleOutlined,
  StopOutlined,
  DownOutlined,
  DeleteOutlined
} from '@ant-design/icons-vue';
import {
  listCategory,
  detailCategory,
  createCategory,
  updateCategory,
  deleteCategory,
  type Category,
  type CreateCategoryReq,
  type UpdateCategoryReq,
  type DeleteCategoryReq,
  type ListCategoryReq,
  getCategoryStatistics
} from '#/api/core/workorder_category';

// 响应式数据类型
interface Statistics {
  total: number;
  enabled: number;
  disabled: number;
}

interface CategoryDialogState {
  isEdit: boolean;
  form: CreateCategoryReq & { id?: number; status?: number };
}

interface DetailDialogState {
  category: Category | null;
}

// 列定义
const columns = [
  {
    title: '分类名称',
    dataIndex: 'name',
    key: 'name',
    width: 200,
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
    width: 200,
    ellipsis: true,
  },
  {
    title: '排序',
    dataIndex: 'sort_order',
    key: 'sort_order',
    width: 100,
    align: 'center' as const,
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100,
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

// 状态数据
const loading = ref<boolean>(false);
const statsLoading = ref<boolean>(false);
const saveLoading = ref<boolean>(false);
const searchQuery = ref<string>('');
const statusFilter = ref<number | undefined>(undefined);
const currentPage = ref<number>(1);
const pageSize = ref<number>(10);
const total = ref<number>(0);
const categories = ref<Category[]>([]);

// 表单引用
const formRef = ref<FormInstance>();

// 模态框控制
const categoryDialogVisible = ref<boolean>(false);
const detailDialogVisible = ref<boolean>(false);

// 响应式对话框宽度
const dialogWidth = computed(() => {
  if (typeof window !== 'undefined') {
    const width = window.innerWidth;
    if (width < 768) return '95%';
    if (width < 1024) return '80%';
    return '600px';
  }
  return '600px';
});

// 统计数据
const stats = reactive<Statistics>({
  total: 0,
  enabled: 0,
  disabled: 0
});

// 分页配置
const paginationConfig = computed(() => ({
  current: currentPage.value,
  pageSize: pageSize.value,
  total: total.value,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number) => `共 ${total} 条`,
  pageSizeOptions: ['10', '20', '50', '100'],
}));

// 分类对话框
const categoryDialog = reactive<CategoryDialogState>({
  isEdit: false,
  form: {
    name: '',
    icon: '',
    sort_order: 0,
    description: '',
    status: 1 // 默认启用
  }
});

// 分类验证规则
const categoryRules = {
  name: [
    { required: true, message: '请输入分类名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度应为2到50个字符', trigger: 'blur' }
  ],
  sort_order: [
    { required: true, message: '请输入排序值', trigger: 'blur' },
    { type: 'number', min: 0, max: 999, message: '排序值应在0-999之间', trigger: 'blur' }
  ]
};

// 详情对话框
const detailDialog = reactive<DetailDialogState>({
  category: null
});

// 加载分类列表
const loadCategories = async (): Promise<void> => {
  loading.value = true;
  try {
    const params: ListCategoryReq = {
      page: currentPage.value,
      size: pageSize.value,
      search: searchQuery.value || undefined,
      status: statusFilter.value
    };
    
    const response = await listCategory(params);
    if (response && response.items) {
      categories.value = response.items;
      total.value = response.total || 0;
      // 更新统计数据中的总数
      stats.total = response.total || 0;
    } else {
      categories.value = [];
      total.value = 0;
      stats.total = 0;
    }
  } catch (error) {
    console.error('加载分类列表失败:', error);
    message.error('加载分类列表失败');
    categories.value = [];
    total.value = 0;
    stats.total = 0;
  } finally {
    loading.value = false;
  }
};

// 加载统计数据
const loadStats = async (): Promise<void> => {
  if (statsLoading.value) return;
  
  statsLoading.value = true;
  try {
    // 使用getCategoryStatistics接口获取启用和禁用的分类数量
    const statistics = await getCategoryStatistics();
    
    // 更新统计数据
    // 使用列表加载时已经获取的total，不需要再发请求
    stats.enabled = statistics?.enabled_count || 0;
    stats.disabled = statistics?.disabled_count || 0;
  } catch (error) {
    console.error('加载统计数据失败:', error);
    // 不重置total，只重置其他统计数据
    stats.enabled = 0;
    stats.disabled = 0;
    // 不显示错误消息，因为这是后台统计操作
  } finally {
    statsLoading.value = false;
  }
};

// 表格变化处理
const handleTableChange = (pagination: any): void => {
  if (pagination.current !== currentPage.value) {
    currentPage.value = pagination.current;
  }
  if (pagination.pageSize !== pageSize.value) {
    pageSize.value = pagination.pageSize;
    currentPage.value = 1; // 切换页面大小时重置到第一页
  }
  loadCategories();
};

// 搜索处理
const handleSearch = (): void => {
  currentPage.value = 1; // 搜索时重置到第一页
  loadCategories();
};

// 状态筛选变化
const handleStatusChange = (): void => {
  currentPage.value = 1; // 筛选时重置到第一页
  loadCategories();
};

// 分类操作
const handleCreateCategory = (): void => {
  categoryDialog.isEdit = false;
  categoryDialog.form = {
    name: '',
    icon: '',
    sort_order: 0,
    description: '',
    status: 1 // 默认启用
  };
  categoryDialogVisible.value = true;
};

const handleEditCategory = async (row: Category): Promise<void> => {
  const editLoading = message.loading('加载分类详情...', 0);
  try {
    const response = await detailCategory({ id: row.id });
    if (response) {
      categoryDialog.isEdit = true;
      categoryDialog.form = {
        id: response.id,
        name: response.name,
        icon: response.icon,
        sort_order: response.sort_order,
        description: response.description,
        status: response.status
      };
      categoryDialogVisible.value = true;
      detailDialogVisible.value = false;
    }
  } catch (error) {
    console.error('加载分类详情失败:', error);
    message.error('加载分类详情失败');
  } finally {
    editLoading();
  }
};

const handleViewCategory = async (row: Category): Promise<void> => {
  const viewLoading = message.loading('加载分类详情...', 0);
  try {
    const response = await detailCategory({ id: row.id });
    if (response) {
      detailDialog.category = response;
      detailDialogVisible.value = true;
    }
  } catch (error) {
    console.error('加载分类详情失败:', error);
    message.error('加载分类详情失败');
  } finally {
    viewLoading();
  }
};

const handleMenuClick = (command: string, row: Category): void => {
  switch (command) {
    case 'enable':
      updateCategoryStatus(row, 1);
      break;
    case 'disable':
      updateCategoryStatus(row, 2);
      break;
    case 'delete':
      confirmDelete(row);
      break;
  }
};

// 更新分类状态
const updateCategoryStatus = async (category: Category, status: number): Promise<void> => {
  try {
    const params: UpdateCategoryReq = {
      id: category.id,
      name: category.name,
      icon: category.icon,
      sort_order: category.sort_order,
      description: category.description,
      status: status
    };
    
    await updateCategory(params);
    message.success(`分类 "${category.name}" ${status === 1 ? '已启用' : '已禁用'}`);
    
    // 刷新当前页数据和统计数据
    await Promise.all([loadCategories(), loadStats()]);
  } catch (error) {
    console.error('更新分类状态失败:', error);
    message.error('更新分类状态失败');
  }
};

// 删除分类
const confirmDelete = (category: Category): void => {
  Modal.confirm({
    title: '警告',
    content: `确定要删除分类 "${category.name}" 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      try {
        const params: DeleteCategoryReq = { id: category.id };
        await deleteCategory(params);
        message.success(`分类 "${category.name}" 已删除`);
        
        // 检查当前页是否还有数据，如果删除后当前页没有数据且不是第一页，则回到上一页
        if (categories.value.length === 1 && currentPage.value > 1) {
          currentPage.value = currentPage.value - 1;
        }
        
        // 刷新数据
        await Promise.all([loadCategories(), loadStats()]);
      } catch (error) {
        console.error('删除分类失败:', error);
        message.error('删除分类失败');
      }
    }
  });
};

// 保存分类
const saveCategory = async (): Promise<void> => {
  if (!formRef.value) return;
  
  try {
    await formRef.value.validate();
  } catch (error) {
    return;
  }

  saveLoading.value = true;
  try {
    if (categoryDialog.isEdit) {
      const params: UpdateCategoryReq = {
        id: categoryDialog.form.id!,
        name: categoryDialog.form.name,
        icon: categoryDialog.form.icon,
        sort_order: categoryDialog.form.sort_order,
        description: categoryDialog.form.description,
        status: categoryDialog.form.status || 1
      };
      await updateCategory(params);
      message.success(`分类 "${categoryDialog.form.name}" 已更新`);
    } else {
      const params: CreateCategoryReq = {
        name: categoryDialog.form.name,
        icon: categoryDialog.form.icon,
        sort_order: categoryDialog.form.sort_order,
        description: categoryDialog.form.description,
        status: categoryDialog.form.status || 1
      };
      await createCategory(params);
      message.success(`分类 "${categoryDialog.form.name}" 已创建`);
      
      // 如果是创建新分类，跳转到第一页查看新创建的分类
      currentPage.value = 1;
    }
    
    categoryDialogVisible.value = false;
    
    // 刷新数据
    await Promise.all([loadCategories(), loadStats()]);
  } catch (error) {
    console.error('保存分类失败:', error);
    message.error('保存分类失败');
  } finally {
    saveLoading.value = false;
  }
};

// 对话框控制
const closeCategoryDialog = (): void => {
  categoryDialogVisible.value = false;
  formRef.value?.resetFields();
};

const closeDetailDialog = (): void => {
  detailDialogVisible.value = false;
};

// 辅助方法
const formatDate = (dateStr?: string): string => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const formatTime = (dateStr?: string): string => {
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

const getInitials = (name?: string): string => {
  if (!name) return '';
  return name
    .split('')
    .slice(0, 2)
    .join('')
    .toUpperCase();
};

const getStatusClass = (status: number): string => {
  return status === 1 ? 'status-enabled' : 'status-disabled';
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

// 初始化
onMounted(() => {
  // 并行加载列表数据和统计数据
  Promise.all([loadCategories(), loadStats()]);
});
</script>

<style scoped>
.category-management-container {
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

.status-filter {
  width: 120px;
  min-width: 100px;
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

.category-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.category-badge {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-enabled {
  background-color: #52c41a;
}

.status-disabled {
  background-color: #d9d9d9;
}

.category-icon {
  font-size: 16px;
}

.category-name-text {
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
  gap: 8px;
  justify-content: center;
  flex-wrap: wrap;
}

.detail-dialog .category-details {
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
  display: flex;
  align-items: center;
  gap: 8px;
  word-break: break-all;
}

.detail-icon {
  font-size: 24px;
}

.detail-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-wrap: wrap;
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

/* 移动端适配 */
@media (max-width: 768px) {
  .category-management-container {
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
  
  .status-filter {
    width: 100%;
    min-width: auto;
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
    gap: 4px;
  }
  
  .action-buttons .ant-btn {
    padding: 0 4px;
    font-size: 12px;
  }
  
  .detail-footer {
    justify-content: center;
  }
  
  .detail-footer .ant-btn {
    flex: 1;
    max-width: 120px;
  }
  
  .responsive-modal :deep(.ant-modal-body) {
    padding: 16px;
    max-height: calc(100vh - 160px);
    overflow-y: auto;
  }
}

/* 平板端适配 */
@media (max-width: 1024px) and (min-width: 769px) {
  .category-management-container {
    padding: 16px;
  }
  
  .search-input {
    width: 200px;
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
</style>