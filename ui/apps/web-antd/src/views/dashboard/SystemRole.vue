<template>
  <div class="role-management-container">
    <!-- 顶部卡片 -->
    <div class="dashboard-card">
      <div class="card-title">
        <Icon icon="material-symbols:badge-outline" class="title-icon" />
        <h2>角色管理</h2>
      </div>

      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <a-input
            v-model:value="searchText" 
            placeholder="请输入角色名称"
            class="search-input"
            @pressEnter="handleSearch"
          >
            <template #prefix>
              <Icon icon="ri:search-line" />
            </template>
          </a-input>
          <a-button type="primary" @click="handleSearch" class="search-button">
            <template #icon><Icon icon="ri:search-line" /></template>
            搜索
          </a-button>
        </div>
        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button type="primary" @click="handleAdd" class="add-button">
            <template #icon><Icon icon="material-symbols:add" /></template>
            创建角色
          </a-button>
        </div>
      </div>
    </div>

    <!-- 角色列表表格 -->
    <div class="table-container">
      <a-table 
        :columns="columns" 
        :data-source="filteredRoleList" 
        row-key="id" 
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        class="role-table"
      >
        <!-- 方法列自定义渲染 -->
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'method'">
            <a-tag :color="getMethodColor(record.method)" class="method-tag">
              {{ record.method }}
            </a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-tooltip title="编辑角色">
                <a-button type="link" @click="handleEdit(record)" class="action-button edit-button">
                  <template #icon><Icon icon="clarity:note-edit-line" /></template>
                </a-button>
              </a-tooltip>
              <a-tooltip title="删除角色">
                <a-popconfirm
                  title="确定要删除这个角色吗?"
                  ok-text="确定"
                  cancel-text="取消"
                  placement="left"
                  @confirm="handleDelete(record)"
                >
                  <a-button type="link" danger class="action-button delete-button">
                    <template #icon><Icon icon="ant-design:delete-outlined" /></template>
                  </a-button>
                </a-popconfirm>
              </a-tooltip>
            </a-space>
          </template>
        </template>
      </a-table>
    </div>

    <!-- 角色表单弹窗 -->
    <a-modal
      v-model:visible="isModalVisible"
      :title="modalTitle"
      @ok="handleModalSubmit"
      @cancel="handleModalCancel"
      :okText="'保存'"
      :cancelText="'取消'"
      width="600px"
      class="custom-modal role-modal"
      :maskClosable="false"
      :destroyOnClose="true"
    >
      <div class="modal-content">
        <div class="modal-header-icon">
          <div class="icon-wrapper" :class="{ 'edit-icon': modalTitle === '编辑角色' }">
            <Icon :icon="modalTitle === '创建角色' ? 'mdi:shield-plus' : 'mdi:shield-edit'" />
          </div>
          <div class="header-text">{{ modalTitle === '创建角色' ? '创建新角色' : '编辑角色信息' }}</div>
        </div>
        
        <a-form :model="formData" layout="vertical" :rules="formRules" class="role-form">
          <div class="form-section">
            <div class="section-title">
              <Icon icon="mdi:information-outline" class="section-icon" />
              <span>基本信息</span>
            </div>
            
            <div class="form-row">
              <a-form-item label="角色名称" name="name" class="form-item">
                <a-input 
                  v-model:value="formData.name" 
                  placeholder="请输入角色名称" 
                  class="custom-input"
                />
              </a-form-item>
              
              <a-form-item label="域ID" name="domain" class="form-item">
                <a-input 
                  v-model:value="formData.domain" 
                  placeholder="请输入域ID" 
                  class="custom-input"
                />
              </a-form-item>
            </div>
          </div>
          
          <div class="form-section">
            <div class="section-title">
              <Icon icon="mdi:web" class="section-icon" />
              <span>API访问权限</span>
            </div>
            
            <div class="form-row">
              <a-form-item label="访问路径" name="path" class="full-width">
                <a-input 
                  v-model:value="formData.path" 
                  placeholder="请输入API路径，例如：/api/users" 
                  class="custom-input"
                  prefix-icon="mdi:link-variant"
                >
                  <template #prefix>
                    <Icon icon="mdi:link-variant" class="input-icon" />
                  </template>
                </a-input>
              </a-form-item>
            </div>
            
            <div class="form-row">
              <a-form-item label="HTTP方法" name="method" class="form-item">
                <a-select 
                  v-model:value="formData.method" 
                  placeholder="请选择HTTP方法" 
                  class="custom-select"
                  dropdown-class-name="method-dropdown"
                >
                  <a-select-option value="GET" class="method-option">
                    <div class="method-option-content">
                      <div class="method-badge get">GET</div>
                      <span class="method-description">查询数据</span>
                    </div>
                  </a-select-option>
                  <a-select-option value="POST" class="method-option">
                    <div class="method-option-content">
                      <div class="method-badge post">POST</div>
                      <span class="method-description">创建数据</span>
                    </div>
                  </a-select-option>
                  <a-select-option value="PUT" class="method-option">
                    <div class="method-option-content">
                      <div class="method-badge put">PUT</div>
                      <span class="method-description">更新数据</span>
                    </div>
                  </a-select-option>
                  <a-select-option value="DELETE" class="method-option">
                    <div class="method-option-content">
                      <div class="method-badge delete">DELETE</div>
                      <span class="method-description">删除数据</span>
                    </div>
                  </a-select-option>
                  <a-select-option value="*" class="method-option">
                    <div class="method-option-content">
                      <div class="method-badge all">ALL</div>
                      <span class="method-description">所有方法</span>
                    </div>
                  </a-select-option>
                </a-select>
              </a-form-item>
            </div>
          </div>
          
          <div class="role-preview" v-if="formData.name && formData.path && formData.method">
            <div class="preview-title">角色预览</div>
            <div class="preview-content">
              <div class="preview-item">
                <span class="preview-label">角色名称:</span>
                <span class="preview-value">{{ formData.name }}</span>
              </div>
              <div class="preview-item">
                <span class="preview-label">权限范围:</span>
                <span class="preview-value permission-value">
                  <a-tag :color="getMethodColor(formData.method)" class="preview-method-tag">
                    {{ formData.method }}
                  </a-tag>
                  <span class="path-value">{{ formData.path || '/' }}</span>
                </span>
              </div>
            </div>
          </div>
        </a-form>
      </div>
      
      <template #footer>
        <div class="modal-footer">
          <a-button @click="handleModalCancel" class="cancel-button">
            取消
          </a-button>
          <a-button type="primary" @click="handleModalSubmit" class="submit-button">
            <Icon icon="mdi:content-save" class="button-icon" />
            保存
          </a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, computed } from 'vue';
import { message } from 'ant-design-vue';
import { 
  listRolesApi, 
  createRoleApi, 
  updateRoleApi, 
  deleteRoleApi,
  listApisApi,
} from '#/api/core/system';
import type { SystemApi } from '#/api/core/system';
import { Icon } from '@iconify/vue';

interface ApiItem {
  id: number;
  name: string;
  path: string;
  method: string;
  description?: string;
  version?: string;
  category?: number;
  is_public: number;
}

// 表格加载状态
const loading = ref(false);

// 搜索文本
const searchText = ref('');

// 角色列表数据
const roleList = ref<any[]>([]);

// API选项
const apiOptions = ref<{label: string, value: number}[]>([]);

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number) => `共 ${total} 条记录`
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  domain: [{ required: true, message: '请输入域ID', trigger: 'blur' }],
  path: [{ required: true, message: '请输入路径', trigger: 'blur' }],
  method: [{ required: true, message: '请选择方法', trigger: 'change' }]
};

// 过滤后的角色列表
const filteredRoleList = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  if (!searchValue) return roleList.value;
  
  return roleList.value.filter(role => 
    role.name.toLowerCase().includes(searchValue)
  );
});

// 模态框相关
const isModalVisible = ref(false);
const modalTitle = ref('创建角色');
const formData = reactive<Partial<SystemApi.Role>>({
  name: '',
  domain: '',
  path: '',
  method: ''
});

// 当前编辑的角色信息（用于更新）
const currentRole = reactive<Partial<SystemApi.Role>>({
  name: '',
  domain: '',
  path: '',
  method: ''
});

// 获取所有API
const fetchApis = async () => {
  try {
    // 获取API列表
    const apiRes = await listApisApi({
      page_number: 1,
      page_size: 1000
    });
    if (apiRes && apiRes.list) {
      apiOptions.value = apiRes.list.map((api: ApiItem) => ({
        label: `${api.name} (${api.path}) [${api.method}]`,
        value: api.id
      }));
    }
  } catch (error: any) {
    message.error(error.message || '获取权限数据失败');
  }
};

// 表格列配置
const columns = [
  {
    title: '角色名称',
    dataIndex: 'name',
    key: 'name',
    width: 180
  },
  {
    title: '域ID',
    dataIndex: 'domain',
    key: 'domain',
    width: 180
  },
  {
    title: '路径',
    dataIndex: 'path',
    key: 'path',
    ellipsis: true,
  },
  {
    title: '方法',
    dataIndex: 'method',
    key: 'method',
    width: 120
  },
  {
    title: '操作',
    key: 'action',
    width: 120,
    fixed: 'right'
  },
];

// 处理表格变化（分页、排序、筛选）
const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  fetchRoleList();
};

// 获取角色列表
const fetchRoleList = async () => {
  loading.value = true;
  try {
    const res = await listRolesApi({
      page_number: pagination.current,
      page_size: pagination.pageSize
    });
    if (res && res.items) {
      roleList.value = res.items;
      pagination.total = res.total || res.items.length;
    } else {
      roleList.value = [];
      pagination.total = 0;
    }
  } catch (error: any) {
    message.error(error.message || '获取角色列表失败');
    roleList.value = [];
    pagination.total = 0;
  } finally {
    loading.value = false;
  }
};

// 处理搜索
const handleSearch = () => {
  pagination.current = 1; // 搜索时重置到第一页
  // 搜索功能已通过 computed 属性 filteredRoleList 实现
};

// 处理添加
const handleAdd = () => {
  modalTitle.value = '创建角色';
  Object.assign(formData, {
    name: '',
    domain: '',
    path: '',
    method: ''
  });
  isModalVisible.value = true;
};

// 处理编辑
const handleEdit = async (record: SystemApi.Role) => {
  modalTitle.value = '编辑角色';
  
  // 保存当前角色信息用于更新
  Object.assign(currentRole, {
    name: record.name,
    domain: record.domain,
    path: record.path,
    method: record.method
  });
  
  // 设置表单数据
  Object.assign(formData, {
    name: record.name,
    domain: record.domain,
    path: record.path,
    method: record.method
  });

  isModalVisible.value = true;
};

// 处理删除
const handleDelete = async (record: SystemApi.Role) => {
  try {
    await deleteRoleApi({
      name: record.name,
      domain: record.domain,
      path: record.path,
      method: record.method
    });
    message.success('删除成功');
    fetchRoleList();
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

// 处理模态框提交
const handleModalSubmit = async () => {
  // 表单验证
  if (!formData.name || !formData.domain || !formData.path || !formData.method) {
    message.warning('请填写完整的角色信息');
    return;
  }
  
  try {
    if (modalTitle.value === '创建角色') {
      await createRoleApi({
        ...formData as SystemApi.CreateRoleReq
      });
    } else {
      await updateRoleApi({
        old_role: {
          ...currentRole as SystemApi.Role
        },
        new_role: {
          ...formData as SystemApi.Role
        }
      });
    }
    message.success(`${modalTitle.value}成功`);
    isModalVisible.value = false;
    fetchRoleList();
  } catch (error: any) {
    message.error(error.message || `${modalTitle.value}失败`);
  }
};

// 处理模态框取消
const handleModalCancel = () => {
  isModalVisible.value = false;
};

// 根据HTTP方法获取颜色
const getMethodColor = (method: string): string => {
  switch (method) {
    case 'GET': return '#1890ff';  // 蓝色
    case 'POST': return '#52c41a';  // 绿色
    case 'PUT': return '#faad14';  // 黄色
    case 'DELETE': return '#f5222d';  // 红色
    case '*': return '#722ed1';  // 紫色
    default: return '#d9d9d9';  // 灰色
  }
};

onMounted(() => {
  fetchRoleList();
  fetchApis();
});
</script>

<style scoped>
/* 整体容器样式 */
.role-management-container {
  padding: 20px;
  background-color: #f0f2f5;
  min-height: 100vh;
  font-family: 'Roboto', 'PingFang SC', 'Microsoft YaHei', sans-serif;
}

/* 顶部卡片样式 */
.dashboard-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  padding: 20px;
  margin-bottom: 20px;
  transition: all 0.3s;
}

.card-title {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.title-icon {
  font-size: 28px;
  margin-right: 10px;
  color: #1890ff;
}

.card-title h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  color: #1e293b;
}

/* 工具栏样式 */
.custom-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.search-filters {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-input {
  width: 280px;
  border-radius: 6px;
  transition: all 0.3s;
}

.search-input:hover, 
.search-input:focus {
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.search-button {
  border-radius: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.add-button {
  border-radius: 6px;
  background: linear-gradient(90deg, #1890ff, #36cfc9);
  border: none;
  box-shadow: 0 2px 6px rgba(24, 144, 255, 0.3);
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.3s;
}

.add-button:hover {
  background: linear-gradient(90deg, #40a9ff, #5cdbd3);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.4);
}

/* 表格容器样式 */
.table-container {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  padding: 20px;
  overflow: hidden;
}

.role-table {
  width: 100%;
}

/* HTTP方法标签样式 */
.method-tag {
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

/* 操作按钮样式 */
.action-button {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s;
}

.action-button:hover {
  background-color: #f0f0f0;
  transform: translateY(-1px);
}

.edit-button {
  color: #1890ff;
}

.delete-button {
  color: #f5222d;
}

/* 角色模态框样式 */
:deep(.role-modal .ant-modal-content) {
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

:deep(.role-modal .ant-modal-header) {
  background: #fff;
  padding: 20px 24px 0;
  border-bottom: none;
}

:deep(.role-modal .ant-modal-title) {
  font-size: 18px;
  font-weight: 600;
  color: #1e293b;
}

:deep(.role-modal .ant-modal-body) {
  padding: 0 24px 24px;
}

:deep(.role-modal .ant-modal-footer) {
  border-top: 1px solid #f0f0f0;
  padding: 16px 24px;
}

/* 模态框内容样式 */
.modal-content {
  padding: 0;
}

.modal-header-icon {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 24px;
  padding-top: 20px;
}

.icon-wrapper {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: linear-gradient(135deg, #1890ff, #36cfc9);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.25);
}

.icon-wrapper svg {
  font-size: 32px;
  color: white;
}

.edit-icon {
  background: linear-gradient(135deg, #52c41a, #13c2c2);
}

.header-text {
  font-size: 16px;
  color: #1e293b;
  font-weight: 500;
}

/* 表单样式 */
.role-form {
  margin-top: 16px;
}

.form-section {
  margin-bottom: 24px;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background-color: #f9f9f9;
  border-bottom: 1px solid #f0f0f0;
  color: #1e293b;
  font-weight: 500;
}

.section-icon {
  color: #1890ff;
  font-size: 18px;
}

.form-row {
  display: flex;
  gap: 16px;
  padding: 16px;
}

.form-item {
  flex: 1;
  margin-bottom: 0;
}

.full-width {
  width: 100%;
}

:deep(.custom-input) {
  border-radius: 6px;
  transition: all 0.3s;
  height: 38px;
}

:deep(.input-icon) {
  color: #8c8c8c;
  margin-right: 8px;
}

:deep(.custom-input:hover) {
  border-color: #40a9ff;
}

:deep(.custom-input:focus) {
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

:deep(.custom-select) {
  width: 100%;
}

:deep(.custom-select .ant-select-selector) {
  border-radius: 6px !important;
  transition: all 0.3s;
  height: 38px !important;
  padding: 0 11px !important;
}

:deep(.custom-select .ant-select-selection-item) {
  line-height: 36px !important;
}

:deep(.custom-select:hover .ant-select-selector) {
  border-color: #40a9ff !important;
}

:deep(.custom-select.ant-select-focused .ant-select-selector) {
  border-color: #1890ff !important;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2) !important;
}

/* 方法选择器下拉样式 */
:deep(.method-dropdown) {
  border-radius: 8px;
  overflow: hidden;
  padding: 4px;
}

:deep(.method-option) {
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 4px;
}

:deep(.method-option:hover) {
  background-color: #f5f5f5;
}

.method-option-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.method-badge {
  font-size: 12px;
  font-weight: bold;
  padding: 2px 8px;
  border-radius: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: white;
  min-width: 60px;
  text-align: center;
}

.method-badge.get {
  background-color: #1890ff;
}

.method-badge.post {
  background-color: #52c41a;
}

.method-badge.put {
  background-color: #faad14;
}

.method-badge.delete {
  background-color: #f5222d;
}

.method-badge.all {
  background-color: #722ed1;
}

.method-description {
  color: #595959;
  font-size: 14px;
}

/* 角色预览区域 */
.role-preview {
  margin-top: 24px;
  background-color: #f9f9f9;
  border-radius: 8px;
  padding: 16px;
  border: 1px dashed #d9d9d9;
}

.preview-title {
  font-weight: 500;
  color: #1e293b;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-title::before {
  content: '';
  display: inline-block;
  width: 4px;
  height: 16px;
  background: linear-gradient(to bottom, #1890ff, #36cfc9);
  border-radius: 2px;
  margin-right: 8px;
}

.preview-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.preview-item {
  display: flex;
  align-items: center;
}

.preview-label {
  min-width: 80px;
  color: #8c8c8c;
  margin-right: 8px;
}

.preview-value {
  font-weight: 500;
  color: #1e293b;
}

.permission-value {
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-method-tag {
  font-size: 12px;
  padding: 0 8px;
  height: 24px;
  line-height: 24px;
  font-weight: bold;
}

.path-value {
  font-family: monospace;
  background-color: #f0f0f0;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 14px;
}

/* 模态框页脚 */
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.cancel-button {
  border-radius: 6px;
  border: 1px solid #d9d9d9;
  background-color: white;
  color: #595959;
  padding: 0 16px;
  height: 36px;
  transition: all 0.3s;
}

.cancel-button:hover {
  color: #1890ff;
  border-color: #1890ff;
}

.submit-button {
  border-radius: 6px;
  border: none;
  background: linear-gradient(90deg, #1890ff, #36cfc9);
  color: white;
  padding: 0 16px;
  height: 36px;
  display: flex;
  align-items: center;
  gap: 6px;
  box-shadow: 0 2px 6px rgba(24, 144, 255, 0.25);
  transition: all 0.3s;
}

.submit-button:hover {
  background: linear-gradient(90deg, #40a9ff, #5cdbd3);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.35);
}

.button-icon {
  font-size: 16px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .custom-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .search-filters {
    width: 100%;
    margin-bottom: 12px;
  }
  
  .search-input {
    width: 100%;
  }
  
  .form-row {
    flex-direction: column;
  }
}
</style>