<template>
  <div class="api-management">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>API管理</h1>
      <div class="header-actions">
        <a-button @click="handleRefresh">
          <Icon icon="material-symbols:refresh" />
          刷新
        </a-button>
        <a-button type="primary" @click="handleAdd">
          <Icon icon="material-symbols:add" />
          新建API
        </a-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-number">{{ paginationConfig.total }}</div>
        <div class="stat-label">总API数</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ apiStatistics.public_count || 0 }}</div>
        <div class="stat-label">公开API</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ apiStatistics.private_count || 0 }}</div>
        <div class="stat-label">私有API</div>
      </div>
    </div>

    <!-- 搜索筛选区域 -->
    <div class="search-section">
      <div class="search-left">
        <a-input
          v-model:value="searchParams.search"
          placeholder="搜索API名称或路径"
          allowClear
          @pressEnter="handleSearch"
          class="search-input"
        >
          <template #prefix>
            <Icon icon="material-symbols:search" />
          </template>
        </a-input>
        
        <a-select
          v-model:value="searchParams.method"
          placeholder="请求方法"
          allowClear
          class="method-select"
        >
          <a-select-option :value="1">GET</a-select-option>
          <a-select-option :value="2">POST</a-select-option>
          <a-select-option :value="3">PUT</a-select-option>
          <a-select-option :value="4">DELETE</a-select-option>
        </a-select>

        <a-select
          v-model:value="searchParams.is_public"
          placeholder="访问权限"
          allowClear
          class="access-select"
        >
          <a-select-option :value="1">公开</a-select-option>
          <a-select-option :value="2">私有</a-select-option>
        </a-select>
      </div>
      
      <div class="search-right">
        <a-button type="primary" @click="handleSearch">搜索</a-button>
        <a-button @click="handleReset">重置</a-button>
      </div>
    </div>

    <!-- API表格 -->
    <div class="table-container">
      <a-table
        :columns="tableColumns"
        :data-source="apiList"
        :pagination="paginationConfig"
        :loading="loading"
        row-key="id"
        size="middle"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'api'">
            <div class="api-info">
              <div class="api-method">
                <a-tag :color="getMethodColor(record.method)" class="method-tag">
                  {{ getMethodName(record.method) }}
                </a-tag>
              </div>
              <div>
                <div class="api-name">{{ record.name }}</div>
                <div class="api-path">{{ record.path }}</div>
              </div>
            </div>
          </template>
          
          <template v-if="column.key === 'version'">
            <a-tag v-if="record.version" color="blue">{{ record.version }}</a-tag>
            <span v-else class="no-version">-</span>
          </template>
          
          <template v-if="column.key === 'access'">
            <a-switch 
              :checked="record.is_public === 1" 
              @change="(checked: boolean) => handleAccessChange(record, checked ? 1 : 2)"
              size="small"
            />
          </template>
          
          <template v-if="column.key === 'category'">
            <span v-if="record.category">{{ record.category }}</span>
            <span v-else class="no-category">-</span>
          </template>
          
          <template v-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          
          <template v-if="column.key === 'actions'">
            <div class="action-buttons">
              <a-button type="text" size="small" @click="handleView(record)">查看</a-button>
              <a-button type="text" size="small" @click="handleEdit(record)">编辑</a-button>
              <a-popconfirm title="确定要删除吗？" @confirm="handleDelete(record)">
                <a-button type="text" size="small" danger>删除</a-button>
              </a-popconfirm>
            </div>
          </template>
        </template>
      </a-table>
    </div>

    <!-- 查看API详情 -->
    <a-modal v-model:open="viewModalVisible" title="API详情" width="700px" :footer="null">
      <div v-if="viewApiData" class="api-detail">
        <div class="detail-section">
          <h3>基本信息</h3>
          <div class="detail-grid">
            <div class="detail-item">
              <label>API名称</label>
              <span>{{ viewApiData.name }}</span>
            </div>
            <div class="detail-item">
              <label>API路径</label>
              <span class="api-path-detail">{{ viewApiData.path }}</span>
            </div>
            <div class="detail-item">
              <label>请求方法</label>
              <a-tag :color="getMethodColor(viewApiData.method)">
                {{ getMethodName(viewApiData.method) }}
              </a-tag>
            </div>
            <div class="detail-item">
              <label>API版本</label>
              <span>{{ viewApiData.version || '-' }}</span>
            </div>
            <div class="detail-item">
              <label>访问权限</label>
              <a-tag :color="viewApiData.is_public === 1 ? 'green' : 'orange'">
                {{ viewApiData.is_public === 1 ? '公开' : '私有' }}
              </a-tag>
            </div>
            <div class="detail-item">
              <label>API分类</label>
              <span>{{ viewApiData.category || '-' }}</span>
            </div>
            <div class="detail-item" v-if="viewApiData.description">
              <label>API描述</label>
              <span>{{ viewApiData.description }}</span>
            </div>
            <div class="detail-item">
              <label>创建时间</label>
              <span>{{ formatTime(viewApiData.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>
    </a-modal>

    <!-- 编辑API -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      width="800px"
      @ok="handleSubmit"
      :confirm-loading="submitLoading"
    >
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="API名称" name="name">
              <a-input v-model:value="formData.name" placeholder="请输入API名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="API路径" name="path">
              <a-input v-model:value="formData.path" placeholder="例如: /api/users" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="请求方法" name="method">
              <a-select v-model:value="formData.method" placeholder="请选择请求方法">
                <a-select-option :value="1">GET</a-select-option>
                <a-select-option :value="2">POST</a-select-option>
                <a-select-option :value="3">PUT</a-select-option>
                <a-select-option :value="4">DELETE</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="API版本" name="version">
              <a-input v-model:value="formData.version" placeholder="例如: v1.0" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="API分类" name="category">
              <a-input-number 
                v-model:value="formData.category" 
                placeholder="分类ID" 
                :min="0"
                style="width: 100%"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="访问权限" name="is_public">
              <a-select v-model:value="formData.is_public">
                <a-select-option :value="1">公开</a-select-option>
                <a-select-option :value="2">私有</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="API描述" name="description">
          <a-textarea v-model:value="formData.description" :rows="3" placeholder="请输入API描述信息" />
        </a-form-item>

        <div class="api-preview" v-if="formData.path && formData.method">
          <div class="preview-title">
            <Icon icon="material-symbols:preview" />
            接口预览
          </div>
          <div class="preview-content">
            <a-tag :color="getMethodColor(formData.method)" class="preview-method-tag">
              {{ getMethodName(formData.method) }}
            </a-tag>
            <span class="preview-path">{{ formData.path }}</span>
            <a-tag :color="formData.is_public === 1 ? 'green' : 'orange'" class="preview-access-tag">
              {{ formData.is_public === 1 ? '公开' : '私有' }}
            </a-tag>
          </div>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import { Icon } from '@iconify/vue';
import type { FormInstance } from 'ant-design-vue';

import { 
  listApisApi, 
  createApiApi, 
  updateApiApi, 
  deleteApiApi,
  getApiDetailApi,
  getApiStatisticsApi,
  type CreateApiReq,
  type UpdateApiReq
} from '#/api/core/api';

interface ApiInfo {
  id: number;
  name: string;
  path: string;
  method: number;
  description?: string;
  version?: string;
  category?: number;
  is_public: number;
  created_at?: any;
}

// 表单引用
const formRef = ref<FormInstance>();

// 表格列配置
const tableColumns = [
  { title: 'API信息', key: 'api', width: 300, fixed: 'left' },
  { title: 'API描述', key: 'description', dataIndex: 'description', ellipsis: true },
  { title: '版本', key: 'version', width: 100, align: 'center' },
  { title: '分类', key: 'category', width: 100, align: 'center' },
  { title: '访问权限', key: 'access', width: 100, align: 'center' },
  { title: '创建时间', key: 'created_at', width: 120 },
  { title: '操作', key: 'actions', width: 160, fixed: 'right' }
];

// 状态管理
const loading = ref(false);
const submitLoading = ref(false);
const modalVisible = ref(false);
const viewModalVisible = ref(false);
const modalTitle = ref('');

// 数据
const apiList = ref<ApiInfo[]>([]);
const viewApiData = ref<ApiInfo | null>(null);
const apiStatistics = ref<any>({
  public_count: 0,
  private_count: 0
});

// 搜索参数
const searchParams = reactive({
  search: '',
  method: undefined as number | undefined,
  is_public: undefined as number | undefined
});

// 分页配置
const paginationConfig = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  pageSizeOptions: ['10', '20', '50', '100'],
  showTotal: (total: number, range: [number, number]) => 
    `第 ${range[0]}-${range[1]} 条，共 ${total} 条`
});

// 表单数据初始化
const initFormData = () => ({
  name: '',
  path: '',
  method: 1,
  description: '',
  version: '',
  category: undefined as number | undefined,
  is_public: 2, // 默认为私有（禁用状态为2）
  id: 0
});

const formData = reactive(initFormData());

// 表单验证规则
const formRules = {
  name: [{ required: true, message: '请输入API名称', trigger: 'blur' }],
  path: [{ required: true, message: '请输入API路径', trigger: 'blur' }],
  method: [{ required: true, message: '请选择请求方法', trigger: 'change' }]
};

// 工具函数
const formatTime = (timestamp: any) => {
  if (!timestamp) return '-';
  return new Date(typeof timestamp === 'number' ? timestamp * 1000 : timestamp)
    .toLocaleDateString('zh-CN');
};

const getMethodName = (method: number) => {
  const methodMap: Record<number, string> = {
    1: 'GET',
    2: 'POST',
    3: 'PUT',
    4: 'DELETE'
  };
  return methodMap[method] || '未知';
};

const getMethodColor = (method: number) => {
  const colorMap: Record<number, string> = {
    1: '#1890ff', // 蓝色 - GET
    2: '#52c41a', // 绿色 - POST
    3: '#faad14', // 橙色 - PUT
    4: '#f5222d'  // 红色 - DELETE
  };
  return colorMap[method] || '#d9d9d9';
};

// API 调用
const fetchApiList = async () => {
  loading.value = true;
  try {
    const params: any = {
      page: paginationConfig.current,
      size: paginationConfig.pageSize,
      search: searchParams.search || ''
    };

    if (searchParams.method) {
      params.method = searchParams.method;
    }

    if (searchParams.is_public === 1 || searchParams.is_public === 2) {
      params.is_public = searchParams.is_public;
    }

    const response = await listApisApi(params);
    
    apiList.value = response.items || [];
    paginationConfig.total = response.total || 0;
    
  } catch (error: any) {
    message.error(error.message || '获取API列表失败');
    apiList.value = [];
    paginationConfig.total = 0;
  } finally {
    loading.value = false;
  }
};

const fetchApiStatistics = async () => {
  try {
    const response = await getApiStatisticsApi();
    apiStatistics.value = response;
  } catch (error: any) {
    console.error('获取统计数据失败:', error);
    // 不显示错误信息，因为这是后台请求
  }
};

// 事件处理
const handleSearch = () => {
  paginationConfig.current = 1;
  fetchApiList();
};

const handleReset = () => {
  searchParams.search = '';
  searchParams.method = undefined;
  searchParams.is_public = undefined;
  paginationConfig.current = 1;
  fetchApiList();
};

const handleRefresh = async () => {
  await Promise.all([
    fetchApiList(),
    fetchApiStatistics()
  ]);
};

const handleTableChange = (pagination: any) => {
  paginationConfig.current = pagination.current;
  paginationConfig.pageSize = pagination.pageSize;
  fetchApiList();
};

const handleAdd = () => {
  modalTitle.value = '新建API';
  Object.assign(formData, initFormData());
  modalVisible.value = true;
};

const handleEdit = async (api: ApiInfo) => {
  try {
    modalTitle.value = '编辑API';
    const response = await getApiDetailApi(api.id);
    Object.assign(formData, {
      id: response.id,
      name: response.name,
      path: response.path,
      method: response.method,
      description: response.description || '',
      version: response.version || '',
      category: response.category,
      is_public: response.is_public
    });
    modalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取API详情失败');
  }
};

const handleView = async (api: ApiInfo) => {
  try {
    const response = await getApiDetailApi(api.id);
    viewApiData.value = response;
    viewModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取API详情失败');
  }
};

const handleDelete = async (api: ApiInfo) => {
  try {
    await deleteApiApi(api.id);
    message.success('删除成功');
    if (apiList.value.length === 1 && paginationConfig.current > 1) {
      paginationConfig.current--;
    }
    await Promise.all([
      fetchApiList(),
      fetchApiStatistics()
    ]);
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

const handleAccessChange = async (api: ApiInfo, newAccess: number) => {
  const originalAccess = api.is_public;
  
  try {
    // 乐观更新
    api.is_public = newAccess;
    
    const updateData: UpdateApiReq = {
      id: api.id,
      name: api.name,
      path: api.path,
      method: api.method,
      description: api.description || '',
      version: api.version || '',
      category: api.category,
      is_public: newAccess as 1 | 2
    };
    
    await updateApiApi(updateData);
    message.success('访问权限更新成功');
    
    // 重新获取统计数据
    await fetchApiStatistics();
  } catch (error: any) {
    // 发生错误时，恢复原来的状态
    api.is_public = originalAccess;
    message.error(error.message || '访问权限更新失败');
  }
};

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
    submitLoading.value = true;
    
    if (modalTitle.value === '新建API') {
      const createData: CreateApiReq = {
        name: formData.name,
        path: formData.path,
        method: formData.method,
        description: formData.description,
        version: formData.version,
        category: formData.category,
        is_public: formData.is_public as 1 | 2
      };
      
      await createApiApi(createData);
      message.success('创建成功');
    } else {
      const updateData: UpdateApiReq = {
        id: formData.id,
        name: formData.name,
        path: formData.path,
        method: formData.method,
        description: formData.description,
        version: formData.version,
        category: formData.category,
        is_public: formData.is_public as 1 | 2
      };
      await updateApiApi(updateData);
      message.success('更新成功');
    }
    
    modalVisible.value = false;
    await Promise.all([
      fetchApiList(),
      fetchApiStatistics()
    ]);
  } catch (error: any) {
    if (!error.errorFields) {
      message.error(error.message || '操作失败');
    }
  } finally {
    submitLoading.value = false;
  }
};

// 初始化
onMounted(() => {
  fetchApiList();
  fetchApiStatistics();
});
</script>

<style scoped>
/* 继承用户页面的样式风格 */
.api-management {
  padding: 20px;
  background: #f5f5f5;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 20px;
  background: white;
  border-radius: 8px;
  border: 1px solid #d9d9d9;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: #262626;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.header-actions .ant-btn {
  display: flex;
  align-items: center;
  gap: 4px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  background: white;
  padding: 20px;
  border-radius: 8px;
  text-align: center;
  border: 1px solid #d9d9d9;
}

.stat-number {
  font-size: 28px;
  font-weight: 600;
  color: #1890ff;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #8c8c8c;
}

.search-section {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 16px;
  margin-bottom: 20px;
  padding: 20px;
  background: white;
  border-radius: 8px;
  border: 1px solid #d9d9d9;
}

.search-left {
  display: flex;
  gap: 12px;
  flex: 1;
  align-items: flex-end;
}

.search-input {
  flex: 1;
  max-width: 300px;
}

.method-select,
.access-select {
  width: 140px;
}

.search-right {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.table-container {
  background: white;
  border-radius: 8px;
  border: 1px solid #d9d9d9;
  overflow: hidden;
}

.table-container :deep(.ant-table-thead > tr > th) {
  background: #fafafa;
  font-weight: 600;
  color: #262626;
  border-bottom: 1px solid #e8e8e8;
}

.table-container :deep(.ant-table-tbody > tr:hover > td) {
  background: #f5f5f5;
}

.api-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.api-method {
  flex-shrink: 0;
}

.method-tag {
  font-family: 'Roboto Mono', monospace;
  font-weight: 600;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
  border: none;
}

.api-name {
  font-weight: 600;
  color: #262626;
  margin-bottom: 2px;
}

.api-path {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Roboto Mono', monospace;
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
}

.api-path-detail {
  font-family: 'Roboto Mono', monospace;
  background: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
  color: #595959;
}

.no-version,
.no-category {
  color: #bfbfbf;
  font-style: italic;
}

.action-buttons {
  display: flex;
  gap: 4px;
}

.api-detail {
  padding: 8px 0;
}

.detail-section {
  margin-bottom: 24px;
}

.detail-section h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: #262626;
  padding-bottom: 8px;
  border-bottom: 1px solid #e8e8e8;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.detail-item {
  padding: 12px;
  background: #fafafa;
  border-radius: 6px;
  border: 1px solid #e8e8e8;
}

.detail-item label {
  display: block;
  font-size: 12px;
  color: #8c8c8c;
  font-weight: 600;
  margin-bottom: 4px;
}

.detail-item span {
  font-size: 14px;
  color: #262626;
}

.api-preview {
  margin-top: 24px;
  padding: 16px;
  background: #fafafa;
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
}

.preview-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #262626;
  margin-bottom: 12px;
}

.preview-content {
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: 'Roboto Mono', monospace;
}

.preview-method-tag {
  font-weight: 600;
  border: none;
}

.preview-path {
  flex: 1;
  font-size: 14px;
  color: #262626;
  background: #fff;
  padding: 4px 8px;
  border-radius: 4px;
  border: 1px solid #e8e8e8;
}

.preview-access-tag {
  font-size: 12px;
}

.table-container :deep(.ant-btn-text) {
  color: #1890ff;
}

.table-container :deep(.ant-btn-text:hover) {
  color: #40a9ff;
  background: #f0f9ff;
}

.table-container :deep(.ant-btn-text.ant-btn-dangerous) {
  color: #ff4d4f;
}

.table-container :deep(.ant-btn-text.ant-btn-dangerous:hover) {
  color: #ff7875;
  background: #fff2f0;
}

@media (max-width: 1200px) {
  .search-section {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }
  
  .search-left {
    flex-direction: column;
    gap: 12px;
  }
  
  .search-input {
    max-width: none;
  }
  
  .method-select,
  .access-select {
    width: 100%;
  }
  
  .search-right {
    justify-content: flex-end;
  }
}

@media (max-width: 768px) {
  .api-management {
    padding: 12px;
  }
  
  .page-header {
    flex-direction: column;
    gap: 16px;
    text-align: center;
  }
  
  .header-actions {
    width: 100%;
    justify-content: center;
  }
  
  .stats-grid {
    grid-template-columns: 1fr;
  }
  
  .search-right {
    flex-direction: column;
    gap: 8px;
  }
  
  .detail-grid {
    grid-template-columns: 1fr;
  }
  
  .action-buttons {
    flex-direction: column;
    width: 100%;
  }
  
  .api-info {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  
  .preview-content {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>