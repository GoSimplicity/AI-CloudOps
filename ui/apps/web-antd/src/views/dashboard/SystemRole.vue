<template>
  <div class="role-management">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>角色管理</h1>
      <div class="header-actions">
        <a-button @click="handleRefresh">
          <Icon icon="material-symbols:refresh" />
          刷新
        </a-button>
        <a-button type="primary" @click="handleAdd">
          <Icon icon="material-symbols:add" />
          新建角色
        </a-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-number">{{ roleList.length }}</div>
        <div class="stat-label">总角色数</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ activeRoles }}</div>
        <div class="stat-label">启用角色</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ systemRoles }}</div>
        <div class="stat-label">系统角色</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ totalUsers }}</div>
        <div class="stat-label">关联用户</div>
      </div>
    </div>

    <!-- 搜索筛选区域 -->
    <div class="search-section">
      <div class="search-left">
        <a-input
          v-model:value="searchParams.search"
          placeholder="搜索角色名称或编码"
          allowClear
          @pressEnter="handleSearch"
          class="search-input"
        >
          <template #prefix>
            <Icon icon="material-symbols:search" />
          </template>
        </a-input>
        
        <a-select
          v-model:value="searchParams.status"
          placeholder="状态筛选"
          allowClear
          class="status-select"
        >
          <a-select-option :value="1">启用</a-select-option>
          <a-select-option :value="0">禁用</a-select-option>
        </a-select>

        <a-select
          v-model:value="typeFilter"
          placeholder="类型筛选"
          allowClear
          class="type-select"
        >
          <a-select-option :value="1">系统角色</a-select-option>
          <a-select-option :value="0">自定义角色</a-select-option>
        </a-select>
      </div>
      
      <div class="search-right">
        <a-button type="primary" @click="handleSearch">搜索</a-button>
        <a-button @click="handleReset">重置</a-button>
      </div>
    </div>

    <!-- 角色表格 -->
    <div class="table-container">
      <a-table
        :columns="tableColumns"
        :data-source="filteredRoles"
        :pagination="paginationConfig"
        :loading="loading"
        row-key="id"
        size="middle"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'role'">
            <div class="role-info">
              <div class="role-icon">
                <Icon 
                  :icon="record.is_system === 1 ? 'material-symbols:admin-panel-settings' : 'material-symbols:badge-outline'" 
                  :style="{ color: record.is_system === 1 ? '#faad14' : '#1890ff' }"
                />
              </div>
              <div>
                <div class="role-name">{{ record.name }}</div>
                <div class="role-code">{{ record.code }}</div>
              </div>
            </div>
          </template>
          
          <template v-if="column.key === 'type'">
            <a-tag v-if="record.is_system === 1" color="orange">系统角色</a-tag>
            <a-tag v-else color="blue">自定义角色</a-tag>
          </template>
          
          <template v-if="column.key === 'status'">
            <a-switch 
              :checked="record.status === 1" 
              @change="(checked: boolean) => handleStatusChange(record, checked ? 1 : 0)"
              size="small"
              :disabled="record.is_system === 1"
            />
          </template>
          
          <template v-if="column.key === 'description'">
            <div class="description-text">{{ record.description || '暂无描述' }}</div>
          </template>
          
          <template v-if="column.key === 'apis'">
            {{ record.apis?.length || 0 }}
          </template>
          
          <template v-if="column.key === 'users'">
            {{ record.users?.length || 0 }}
          </template>
          
          <template v-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          
          <template v-if="column.key === 'actions'">
            <div class="action-buttons">
              <a-button type="text" size="small" @click="handleView(record)">查看</a-button>
              <a-button type="text" size="small" @click="handleEdit(record)">编辑</a-button>
              <a-button type="text" size="small" @click="handlePermission(record)">权限</a-button>
              <a-popconfirm 
                title="确定要删除吗？" 
                @confirm="handleDelete(record)"
                :disabled="record.is_system === 1"
              >
                <a-button 
                  type="text" 
                  size="small" 
                  danger 
                  :disabled="record.is_system === 1"
                >
                  删除
                </a-button>
              </a-popconfirm>
            </div>
          </template>
        </template>
      </a-table>
    </div>

    <!-- 查看角色详情 -->
    <a-modal v-model:open="viewModalVisible" title="角色详情" width="700px" :footer="null">
      <div v-if="viewRoleData" class="role-detail">
        <div class="detail-section">
          <h3>基本信息</h3>
          <div class="detail-grid">
            <div class="detail-item">
              <label>角色名称</label>
              <span>{{ viewRoleData.name }}</span>
            </div>
            <div class="detail-item">
              <label>角色编码</label>
              <span>{{ viewRoleData.code }}</span>
            </div>
            <div class="detail-item">
              <label>角色类型</label>
              <a-tag v-if="viewRoleData.is_system === 1" color="orange">
                <Icon icon="material-symbols:admin-panel-settings" />
                系统角色
              </a-tag>
              <a-tag v-else color="blue">
                <Icon icon="material-symbols:badge-outline" />
                自定义角色
              </a-tag>
            </div>
            <div class="detail-item">
              <label>状态</label>
              <a-tag :color="viewRoleData.status === 1 ? 'green' : 'red'">
                {{ viewRoleData.status === 1 ? '启用' : '禁用' }}
              </a-tag>
            </div>
            <div class="detail-item" v-if="viewRoleData.description">
              <label>角色描述</label>
              <span>{{ viewRoleData.description }}</span>
            </div>
            <div class="detail-item">
              <label>创建时间</label>
              <span>{{ formatTime(viewRoleData.created_at) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="viewRoleData.apis?.length">
          <h3>权限信息</h3>
          <div class="apis-list">
            <div 
              v-for="api in viewRoleData.apis" 
              :key="api.id"
              class="api-item"
            >
              <div class="api-method" :class="getMethodClass(api.method)">
                {{ formatMethod(api.method) }}
              </div>
              <div class="api-info">
                <div class="api-name">{{ api.name }}</div>
                <div class="api-path">{{ api.path }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="viewRoleData.users?.length">
          <h3>关联用户</h3>
          <div class="users-list">
            <a-tag 
              v-for="user in viewRoleData.users" 
              :key="user.id"
              :color="user.enable === 1 ? 'blue' : 'default'"
            >
              {{ user.real_name || user.username }}
            </a-tag>
          </div>
        </div>
      </div>
    </a-modal>

    <!-- 编辑角色 -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      width="900px"
      @ok="handleSubmit"
      :confirm-loading="submitLoading"
    >
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="角色名称" name="name">
              <a-input v-model:value="formData.name" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="角色编码" name="code">
              <a-input v-model:value="formData.code" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="角色描述" name="description">
          <a-textarea v-model:value="formData.description" :rows="3" placeholder="请输入角色描述信息" />
        </a-form-item>

        <a-form-item label="状态" name="status">
          <a-select v-model:value="formData.status">
            <a-select-option :value="1">启用</a-select-option>
            <a-select-option :value="0">禁用</a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="关联API权限" name="api_ids">
          <div class="api-selector">
            <!-- API搜索框 -->
            <div class="api-search-container">
              <a-input
                v-model:value="apiSearchParams.search"
                placeholder="搜索API名称"
                allowClear
                @pressEnter="handleApiSearch"
                class="api-search-input"
              >
                <template #prefix>
                  <Icon icon="material-symbols:search" />
                </template>
                <template #suffix>
                  <a-spin v-if="apiLoading" size="small" />
                </template>
              </a-input>
              <a-button type="primary" @click="handleApiSearch" :loading="apiLoading">
                搜索
              </a-button>
            </div>

            <!-- 已选择的API -->
            <div v-if="selectedApis.length > 0" class="selected-apis">
              <div class="selected-header">
                <span>已选择 {{ selectedApis.length }} 个API</span>
                <a-button type="text" size="small" @click="clearSelectedApis">清空</a-button>
              </div>
              <div class="selected-list">
                <div 
                  v-for="api in selectedApis" 
                  :key="api.id"
                  class="selected-api-item"
                >
                  <div class="api-method" :class="getMethodClass(api.method)">
                    {{ formatMethod(api.method) }}
                  </div>
                  <div class="api-info">
                    <div class="api-name">{{ api.name }}</div>
                    <div class="api-path">{{ api.path }}</div>
                  </div>
                  <a-button 
                    type="text" 
                    size="small" 
                    @click="removeSelectedApi(api)"
                    class="remove-btn"
                  >
                    <Icon icon="material-symbols:close" />
                  </a-button>
                </div>
              </div>
            </div>

            <!-- API列表 -->
            <div class="api-list-container">
              <div class="api-list-header">
                <span>选择API权限 ({{ apiPagination.total }} 个)</span>
                <div class="api-actions">
                  <a-button 
                    type="text" 
                    size="small" 
                    @click="selectCurrentPageApis"
                    :disabled="currentPageApis.length === 0"
                  >
                    选择当前页 ({{ currentPageApis.length }})
                  </a-button>
                  <a-button 
                    type="text" 
                    size="small" 
                    @click="selectAllSearchedApis"
                    :disabled="apiPagination.total === 0"
                    :loading="selectAllLoading"
                  >
                    选择全部搜索结果 ({{ apiPagination.total }})
                  </a-button>
                </div>
              </div>
              
              <div class="api-list" v-if="modalApiList.length > 0">
                <div 
                  v-for="api in modalApiList" 
                  :key="api.id"
                  class="api-list-item"
                  :class="{ 
                    'selected': isApiSelected(api.id),
                    'disabled': isApiSelected(api.id)
                  }"
                  @click="toggleApiSelection(api)"
                >
                  <a-checkbox 
                    :checked="isApiSelected(api.id)"
                    @change="() => toggleApiSelection(api)"
                    @click.stop
                  />
                  <div class="api-method" :class="getMethodClass(api.method)">
                    {{ formatMethod(api.method) }}
                  </div>
                  <div class="api-info">
                    <div class="api-name">{{ api.name }}</div>
                    <div class="api-path">{{ api.path }}</div>
                  </div>
                </div>
              </div>

              <div v-else-if="!apiLoading" class="empty-api-list">
                <Icon icon="material-symbols:search-off" />
                <span>{{ apiSearchParams.search ? '未找到匹配的API' : '暂无API数据' }}</span>
              </div>

              <div v-if="apiLoading" class="loading-api-list">
                <a-spin size="large" />
                <span>加载中...</span>
              </div>

              <!-- API分页 -->
              <div class="api-pagination" v-if="apiPagination.total > 0">
                <a-pagination
                  v-model:current="apiPagination.current"
                  v-model:page-size="apiPagination.pageSize"
                  :total="apiPagination.total"
                  :show-size-changer="true"
                  :page-size-options="['10', '20', '50']"
                  :show-quick-jumper="true"
                  :show-total="(total: number, range: [number, number]) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`"
                  size="small"
                  @change="handleApiPageChange"
                  @showSizeChange="handleApiPageSizeChange"
                />
              </div>
            </div>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 权限管理 -->
    <a-modal v-model:open="permissionModalVisible" title="权限管理" width="800px" :footer="null">
      <div v-if="currentRole">
        <div class="permission-header">
          <h4>{{ currentRole.name }} 的权限</h4>
        </div>
        
        <a-tabs>
          <a-tab-pane key="assigned" tab="已分配权限">
            <div class="permission-list">
              <div v-if="assignedApis.length === 0" class="empty-state">
                暂无已分配的权限
              </div>
              <div v-for="api in assignedApis" :key="api.id" class="permission-item">
                <div class="api-method" :class="getMethodClass(api.method)">
                  {{ formatMethod(api.method) }}
                </div>
                <div class="permission-info">
                  <div class="permission-name">{{ api.name }}</div>
                  <div class="permission-path">{{ api.path }}</div>
                </div>
                <a-button type="text" danger size="small" @click="handleRevokeApi(api)">移除</a-button>
              </div>
            </div>
          </a-tab-pane>
          
          <a-tab-pane key="available" tab="可分配权限">
            <div class="permission-list">
              <div v-if="availableApis.length === 0" class="empty-state">
                暂无可分配的权限
              </div>
              <div v-for="api in availableApis" :key="api.id" class="permission-item">
                <div class="api-method" :class="getMethodClass(api.method)">
                  {{ formatMethod(api.method) }}
                </div>
                <div class="permission-info">
                  <div class="permission-name">{{ api.name }}</div>
                  <div class="permission-path">{{ api.path }}</div>
                </div>
                <a-button type="primary" size="small" @click="handleAssignApi(api)">分配</a-button>
              </div>
            </div>
          </a-tab-pane>
        </a-tabs>
      </div>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, computed, onMounted, watch } from 'vue';
import { message } from 'ant-design-vue';
import { Icon } from '@iconify/vue';
import type { FormInstance } from 'ant-design-vue';
import { debounce } from 'lodash-es';

import { 
  listRolesApi, 
  createRoleApi, 
  updateRoleApi, 
  deleteRoleApi,
  getRoleDetailApi,
  assignApisToRoleApi,
  revokeApisFromRoleApi,
  getRoleApisApi,
  type Role,
  type ListRolesReq,
} from '#/api/core/system';

import { listApisApi } from '#/api/core/api';

// 表单引用
const formRef = ref<FormInstance>();

// 表格列配置
const tableColumns = [
  { title: '角色', key: 'role', width: 200, fixed: 'left' },
  { title: '类型', key: 'type', width: 100, align: 'center' },
  { title: '状态', key: 'status', width: 80, align: 'center' },
  { title: '描述', key: 'description', width: 200 },
  { title: '权限数', key: 'apis', width: 80, align: 'center' },
  { title: '用户数', key: 'users', width: 80, align: 'center' },
  { title: '创建时间', key: 'created_at', width: 120 },
  { title: '操作', key: 'actions', width: 160, fixed: 'right' }
];

// 状态管理
const loading = ref(false);
const submitLoading = ref(false);
const modalVisible = ref(false);
const permissionModalVisible = ref(false);
const viewModalVisible = ref(false);
const modalTitle = ref('');
const apiLoading = ref(false);
const selectAllLoading = ref(false);

// 数据
const roleList = ref<Role[]>([]);
const apiList = ref<any[]>([]);
const modalApiList = ref<any[]>([]);
const selectedApis = ref<any[]>([]);
const currentRole = ref<Role | null>(null);
const assignedApis = ref<any[]>([]);
const viewRoleData = ref<Role | null>(null);
const typeFilter = ref<number | undefined>(undefined);

// 搜索参数
const searchParams = reactive<ListRolesReq>({
  page: 1,
  size: 20,
  search: '',
  status: undefined
});

// API搜索参数
const apiSearchParams = reactive({
  search: ''
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

// API分页配置
const apiPagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0
});

// 表单数据初始化
const initFormData = () => ({
  id: undefined as number | undefined,
  name: '',
  code: '',
  description: '',
  status: 1 as 0 | 1,
  api_ids: [] as number[]
});

const formData = reactive(initFormData());

// 表单验证规则
const formRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入角色编码', trigger: 'blur' }],
  status: [{ required: true, message: '请选择状态', trigger: 'change' }]
};

// 计算属性
const activeRoles = computed(() => 
  roleList.value.filter((role: Role) => role.status === 1).length
);

const systemRoles = computed(() => 
  roleList.value.filter((role: Role) => role.is_system === 1).length
);

const totalUsers = computed(() => 
  roleList.value.reduce((total: number, role: Role) => total + (role.users?.length || 0), 0)
);

const filteredRoles = computed(() => {
  let filtered = roleList.value;
  
  if (searchParams.search) {
    const searchText = searchParams.search.toLowerCase();
    filtered = filtered.filter((role: Role) => 
      role.name.toLowerCase().includes(searchText) ||
      role.code.toLowerCase().includes(searchText)
    );
  }
  
  if (searchParams.status !== undefined) {
    filtered = filtered.filter((role: Role) => role.status === searchParams.status);
  }

  if (typeFilter.value !== undefined) {
    filtered = filtered.filter((role: Role) => role.is_system === typeFilter.value);
  }
  
  return filtered;
});

const availableApis = computed(() => {
  if (!currentRole.value) return [];
  
  const assignedIds = assignedApis.value.map(api => api.id);
  return apiList.value.filter(api => !assignedIds.includes(api.id));
});

const currentPageApis = computed(() => {
  return modalApiList.value.filter(api => !isApiSelected(api.id));
});

// 工具函数
const formatTime = (timestamp: any) => {
  if (!timestamp) return '-';
  return new Date(typeof timestamp === 'number' ? timestamp * 1000 : timestamp)
    .toLocaleDateString('zh-CN');
};

const formatMethod = (method: any): string => {
  if (typeof method === 'string') {
    return method.toUpperCase();
  }
  return String(method || 'UNKNOWN').toUpperCase();
};

const getMethodClass = (method: any): string => {
  const methodStr = formatMethod(method).toLowerCase();
  return ['get', 'post', 'put', 'delete', 'patch'].includes(methodStr) ? methodStr : 'unknown';
};

// API选择相关函数
const isApiSelected = (apiId: number): boolean => {
  return selectedApis.value.some(api => api.id === apiId);
};

const toggleApiSelection = (api: any) => {
  const index = selectedApis.value.findIndex(item => item.id === api.id);
  if (index >= 0) {
    selectedApis.value.splice(index, 1);
  } else {
    selectedApis.value.push(api);
  }
  updateFormApiIds();
};

const removeSelectedApi = (api: any) => {
  const index = selectedApis.value.findIndex(item => item.id === api.id);
  if (index >= 0) {
    selectedApis.value.splice(index, 1);
    updateFormApiIds();
  }
};

const clearSelectedApis = () => {
  selectedApis.value = [];
  updateFormApiIds();
};

const selectCurrentPageApis = () => {
  currentPageApis.value.forEach(api => {
    if (!isApiSelected(api.id)) {
      selectedApis.value.push(api);
    }
  });
  updateFormApiIds();
};

const selectAllSearchedApis = async () => {
  if (selectAllLoading.value) return;
  
  try {
    selectAllLoading.value = true;
    
    // 获取所有搜索结果
    const response = await listApisApi({
      search: apiSearchParams.search || undefined,
      page: 1,
      size: apiPagination.total
    });
    
    const allSearchedApis = (response.items || []).map((api: any) => ({
      ...api,
      method: formatMethod(api.method),
      name: api.name || '未命名API',
      path: api.path || '/'
    }));

    // 添加未选择的API
    let addedCount = 0;
    allSearchedApis.forEach((api: any) => {
      if (!isApiSelected(api.id)) {
        selectedApis.value.push(api);
        addedCount++;
      }
    });
    
    updateFormApiIds();
    message.success(`已添加 ${addedCount} 个API到选择列表`);
  } catch (error: any) {
    message.error(error.message || '获取API列表失败');
  } finally {
    selectAllLoading.value = false;
  }
};

const updateFormApiIds = () => {
  formData.api_ids = selectedApis.value.map((api: any) => api.id);
};

// API 调用
const fetchRoleList = async () => {
  loading.value = true;
  try {
    const response = await listRolesApi({
      ...searchParams,
      page: paginationConfig.current,
      size: paginationConfig.pageSize
    });
    
    roleList.value = response.items || [];
    paginationConfig.total = response.total || 0;
    
  } catch (error: any) {
    message.error(error.message || '获取角色列表失败');
    roleList.value = [];
    paginationConfig.total = 0;
  } finally {
    loading.value = false;
  }
};

const fetchApiList = async () => {
  try {
    const response = await listApisApi({
      page: 1,
      size: 100
    });
    
    const safeApiList = (response.items || []).map((api: any) => ({
      ...api,
      method: formatMethod(api.method),
      name: api.name || '未命名API',
      path: api.path || '/'
    }));
    
    apiList.value = safeApiList;
  } catch (error: any) {
    message.error(error.message || '获取API列表失败');
  }
};

const fetchModalApiList = async () => {
  apiLoading.value = true;
  try {
    const response = await listApisApi({
      search: apiSearchParams.search || undefined,
      page: apiPagination.current,
      size: apiPagination.pageSize
    });
    
    const safeApiList = (response.items || []).map((api: any) => ({
      ...api,
      method: formatMethod(api.method),
      name: api.name || '未命名API',
      path: api.path || '/'
    }));
    
    modalApiList.value = safeApiList;
    apiPagination.total = response.total || 0;
  } catch (error: any) {
    message.error(error.message || '获取API列表失败');
    modalApiList.value = [];
    apiPagination.total = 0;
  } finally {
    apiLoading.value = false;
  }
};

const fetchRoleApis = async (roleId: number) => {
  try {
    const response = await getRoleApisApi(roleId);
    
    const safeAssignedApis = (response.items || []).map((api: any) => ({
      ...api,
      method: formatMethod(api.method),
      name: api.name || '未命名API',
      path: api.path || '/'
    }));
    
    assignedApis.value = safeAssignedApis;
  } catch (error: any) {
    message.error(error.message || '获取角色权限失败');
  }
};

// 搜索相关
const handleApiSearch = () => {
  apiPagination.current = 1;
  fetchModalApiList();
};

// 使用防抖处理搜索输入
const debouncedApiSearch = debounce(() => {
  apiPagination.current = 1;
  fetchModalApiList();
}, 300);

const handleApiPageChange = (page: number, size: number) => {
  apiPagination.current = page;
  apiPagination.pageSize = size;
  fetchModalApiList();
};

const handleApiPageSizeChange = (current: number, size: number) => {
  apiPagination.current = 1;
  apiPagination.pageSize = size;
  fetchModalApiList();
};

// 事件处理
const handleSearch = () => {
  paginationConfig.current = 1;
  fetchRoleList();
};

const handleReset = () => {
  searchParams.search = '';
  searchParams.status = undefined;
  typeFilter.value = undefined;
  paginationConfig.current = 1;
  fetchRoleList();
};

const handleRefresh = () => {
  fetchRoleList();
};

const handleTableChange = (pagination: any) => {
  paginationConfig.current = pagination.current;
  paginationConfig.pageSize = pagination.pageSize;
  fetchRoleList();
};

const handleAdd = () => {
  modalTitle.value = '新建角色';
  Object.assign(formData, initFormData());
  selectedApis.value = [];
  apiSearchParams.search = '';
  apiPagination.current = 1;
  fetchModalApiList();
  modalVisible.value = true;
};

const handleEdit = async (role: Role) => {
  try {
    modalTitle.value = '编辑角色';
    const response = await getRoleDetailApi(role.id);
    
    // 设置已选择的APIs
    const roleApis = (response.apis || []).map((api: any) => ({
      ...api,
      method: formatMethod(api.method),
      name: api.name || '未命名API',
      path: api.path || '/'
    }));
    
    selectedApis.value = roleApis;
    
    Object.assign(formData, {
      id: response.id,
      name: response.name,
      code: response.code,
      description: response.description,
      status: response.status,
      api_ids: roleApis.map((api: any) => api.id)
    });
    
    // 重置API搜索
    apiSearchParams.search = '';
    apiPagination.current = 1;
    fetchModalApiList();
    modalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取角色详情失败');
  }
};

const handleView = async (role: Role) => {
  try {
    const response = await getRoleDetailApi(role.id);
    viewRoleData.value = response;
    viewModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取角色详情失败');
  }
};

const handlePermission = async (role: Role) => {
  currentRole.value = role;
  await fetchRoleApis(role.id);
  permissionModalVisible.value = true;
};

const handleDelete = async (role: Role) => {
  try {
    await deleteRoleApi({ id: role.id });
    message.success('删除成功');
    if (filteredRoles.value.length === 1 && paginationConfig.current > 1) {
      paginationConfig.current--;
    }
    await fetchRoleList();
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

const handleStatusChange = async (role: Role, newStatus: 0 | 1) => {
  const originalStatus = role.status;
  
  try {
    role.status = newStatus;
    
    await updateRoleApi({
      id: role.id,
      name: role.name,
      code: role.code,
      description: role.description,
      status: newStatus,
      api_ids: role.apis?.map((api: any) => api.id) || []
    });
    message.success('状态更新成功');
  } catch (error: any) {
    role.status = originalStatus;
    message.error(error.message || '状态更新失败');
  }
};

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
    submitLoading.value = true;
    
    if (formData.id) {
      await updateRoleApi({
        id: formData.id,
        name: formData.name,
        code: formData.code,
        description: formData.description,
        status: formData.status,
        api_ids: formData.api_ids
      });
      message.success('更新成功');
    } else {
      await createRoleApi({
        name: formData.name,
        code: formData.code,
        description: formData.description,
        status: formData.status,
        api_ids: formData.api_ids
      });
      message.success('创建成功');
    }
    
    modalVisible.value = false;
    await fetchRoleList();
  } catch (error: any) {
    if (!error.errorFields) {
      message.error(error.message || '操作失败');
    }
  } finally {
    submitLoading.value = false;
  }
};

const handleAssignApi = async (api: any) => {
  if (!currentRole.value) return;
  
  try {
    await assignApisToRoleApi({
      role_id: currentRole.value.id,
      api_ids: [api.id]
    });
    message.success('权限分配成功');
    await fetchRoleApis(currentRole.value.id);
    await fetchRoleList();
  } catch (error: any) {
    message.error(error.message || '权限分配失败');
  }
};

const handleRevokeApi = async (api: any) => {
  if (!currentRole.value) return;
  
  try {
    await revokeApisFromRoleApi({
      role_id: currentRole.value.id,
      api_ids: [api.id]
    });
    message.success('权限移除成功');
    await fetchRoleApis(currentRole.value.id);
    await fetchRoleList();
  } catch (error: any) {
    message.error(error.message || '权限移除失败');
  }
};

// 监听API搜索输入变化
watch(() => apiSearchParams.search, () => {
  debouncedApiSearch();
});

// 监听模态框关闭，清理数据
watch(modalVisible, (newVal) => {
  if (!newVal) {
    selectedApis.value = [];
    apiSearchParams.search = '';
    apiPagination.current = 1;
    modalApiList.value = [];
  }
});

// 初始化
onMounted(() => {
  fetchRoleList();
  fetchApiList();
});
</script>

<style scoped>
/* 保持原有样式不变，只添加新的样式 */
.role-management {
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

.status-select,
.type-select {
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

.role-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.role-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
  border: 1px solid rgba(0, 0, 0, 0.1);
}

.role-name {
  font-weight: 600;
  color: #262626;
  margin-bottom: 2px;
}

.role-code {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
}

.description-text {
  color: #595959;
  line-height: 1.4;
  max-width: 200px;
  word-break: break-word;
}

.action-buttons {
  display: flex;
  gap: 4px;
}

.role-detail {
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

.apis-list, .users-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.api-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #fafafa;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  margin-bottom: 8px;
}

.api-method {
  font-size: 10px;
  font-weight: bold;
  padding: 3px 8px;
  border-radius: 4px;
  color: white;
  min-width: 50px;
  text-align: center;
  text-transform: uppercase;
  line-height: 1;
  flex-shrink: 0;
}

.api-method.get {
  background: #1890ff;
}

.api-method.post {
  background: #52c41a;
}

.api-method.put {
  background: #faad14;
}

.api-method.delete {
  background: #f5222d;
}

.api-method.patch {
  background: #722ed1;
}

.api-info {
  flex: 1;
}

.api-name {
  font-weight: 500;
  color: #262626;
  font-size: 14px;
  line-height: 1.4;
}

.api-path {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  line-height: 1.4;
}

.permission-header {
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e8e8e8;
}

.permission-header h4 {
  margin: 0;
  font-size: 16px;
  color: #262626;
  font-weight: 600;
}

.permission-list {
  max-height: 400px;
  overflow-y: auto;
}

.empty-state {
  text-align: center;
  color: #8c8c8c;
  padding: 40px 0;
  background: #fafafa;
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
}

.permission-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: #fafafa;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  margin-bottom: 8px;
}

.permission-item:hover {
  background: #f0f0f0;
}

.permission-info {
  flex: 1;
  margin-left: 12px;
}

.permission-name {
  font-weight: 600;
  color: #262626;
  margin-bottom: 4px;
}

.permission-path {
  font-size: 12px;
  color: #8c8c8c;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

/* API选择器样式 */
.api-selector {
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  background: #fafafa;
}

.api-search-container {
  padding: 12px;
  border-bottom: 1px solid #e8e8e8;
  display: flex;
  gap: 8px;
}

.api-search-input {
  flex: 1;
}

.selected-apis {
  padding: 12px;
  border-bottom: 1px solid #e8e8e8;
  background: #f0f9ff;
}

.selected-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #1890ff;
}

.selected-list {
  max-height: 200px;
  overflow-y: auto;
}

.selected-api-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  background: white;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  margin-bottom: 6px;
}

.selected-api-item:last-child {
  margin-bottom: 0;
}

.remove-btn {
  padding: 2px;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ff4d4f;
}

.api-list-container {
  background: white;
}

.api-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid #e8e8e8;
  background: #fafafa;
  font-weight: 600;
  color: #262626;
}

.api-actions {
  display: flex;
  gap: 8px;
}

.api-list {
  max-height: 300px;
  overflow-y: auto;
  padding: 8px 12px;
}

.api-list-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  margin-bottom: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.api-list-item:hover {
  background: #f5f5f5;
  border-color: #1890ff;
}

.api-list-item.selected {
  background: #e6f7ff;
  border-color: #1890ff;
}

.api-list-item.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.empty-api-list {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 40px;
  color: #8c8c8c;
  text-align: center;
}

.empty-api-list .iconify {
  font-size: 32px;
  opacity: 0.5;
}

.loading-api-list {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 40px;
  color: #8c8c8c;
  text-align: center;
}

.api-pagination {
  padding: 12px;
  border-top: 1px solid #e8e8e8;
  background: #fafafa;
  display: flex;
  justify-content: center;
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

/* 滚动条样式 */
.selected-list::-webkit-scrollbar,
.api-list::-webkit-scrollbar,
.permission-list::-webkit-scrollbar {
  width: 6px;
}

.selected-list::-webkit-scrollbar-track,
.api-list::-webkit-scrollbar-track,
.permission-list::-webkit-scrollbar-track {
  background: #f0f0f0;
  border-radius: 3px;
}

.selected-list::-webkit-scrollbar-thumb,
.api-list::-webkit-scrollbar-thumb,
.permission-list::-webkit-scrollbar-thumb {
  background: #d9d9d9;
  border-radius: 3px;
}

.selected-list::-webkit-scrollbar-thumb:hover,
.api-list::-webkit-scrollbar-thumb:hover,
.permission-list::-webkit-scrollbar-thumb:hover {
  background: #bfbfbf;
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
  
  .status-select,
  .type-select {
    width: 100%;
  }
  
  .search-right {
    justify-content: flex-end;
  }
}

@media (max-width: 768px) {
  .role-management {
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
  
  .role-info {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  
  .role-icon {
    width: 32px;
    height: 32px;
  }

  .api-list-header {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }
  
  .api-actions {
    width: 100%;
    justify-content: flex-end;
  }
  
  .selected-header {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }

  .api-search-container {
    flex-direction: column;
    gap: 8px;
  }
}
</style>