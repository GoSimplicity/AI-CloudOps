<template>
  <div class="api-management-container">
    <!-- 顶部卡片 -->
    <div class="dashboard-card">
      <div class="card-title">
        <Icon icon="material-symbols:api" class="title-icon" />
        <h2>API管理</h2>
      </div>

      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <a-input
            v-model:value="searchText" 
            placeholder="请输入API名称"
            class="search-input"
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
            新增API
          </a-button>
        </div>
      </div>
    </div>

    <!-- API列表表格 -->
    <div class="table-container">
      <a-table 
        :columns="columns" 
        :data-source="filteredApiList" 
        row-key="id" 
        :loading="loading"
        :pagination="{
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total: number) => `共 ${total} 条记录`,
          pageSize: 10
        }"
        class="api-table"
      >
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-tooltip title="编辑API">
              <a-button type="link" @click="handleEdit(record)" class="action-button edit-button">
                <template #icon><Icon icon="clarity:note-edit-line" /></template>
              </a-button>
            </a-tooltip>
            <a-popconfirm
              title="确定要删除这个API吗?"
              ok-text="确定"
              cancel-text="取消"
              placement="left"
              @confirm="handleDelete(record)"
            >
              <a-tooltip title="删除API">
                <a-button type="link" danger class="action-button delete-button">
                  <template #icon><Icon icon="ant-design:delete-outlined" /></template>
                </a-button>
              </a-tooltip>
            </a-popconfirm>
          </a-space>
        </template>

        <!-- 请求方法列 -->
        <template #method="{ record }">
          <a-tag :color="getMethodColor(record.method)" class="method-tag">
            {{ getMethodName(record.method) }}
          </a-tag>
        </template>

        <!-- 公开状态列 -->
        <template #isPublic="{ record }">
          <a-switch
            :checked="record.is_public === 1"
            @change="(checked: boolean) => handlePublicChange(record, checked ? 1 : 0)"
            class="public-switch"
          />
        </template>
        
        <!-- 路径列 -->
        <template #path="{ record }">
          <div class="api-path">{{ record.path }}</div>
        </template>
      </a-table>
    </div>

    <!-- 新增/编辑对话框 -->
    <a-modal
      v-model:visible="modalVisible"
      :title="modalTitle"
      @cancel="handleModalCancel"
      :footer="null"
      class="custom-modal api-modal"
      :maskClosable="false"
      :destroyOnClose="true"
      :width="700"
    >
      <div class="modal-content">
        <div class="modal-header-icon">
          <div class="icon-wrapper" :class="{ 'edit-icon': modalTitle === '编辑API' }">
            <Icon :icon="modalTitle === '新增API' ? 'mdi:api-plus' : 'mdi:api-edit'" />
          </div>
          <div class="header-text">{{ modalTitle === '新增API' ? '创建新接口' : '编辑接口信息' }}</div>
        </div>
        
        <a-form :model="formData" layout="vertical" class="api-form">
          <div class="form-grid">
            <div class="form-section basic-info">
              <div class="section-header">
                <Icon icon="mdi:information-outline" class="section-icon" />
                <span>基本信息</span>
              </div>
              
              <div class="section-content">
                <a-form-item label="API名称" required class="form-item">
                  <a-input 
                    v-model:value="formData.name" 
                    placeholder="请输入API名称" 
                    class="custom-input"
                  />
                </a-form-item>
                
                <a-form-item label="API路径" required class="form-item">
                  <a-input 
                    v-model:value="formData.path" 
                    placeholder="请输入API路径，例如: /api/users" 
                    class="custom-input"
                  >
                    <template #prefix>
                      <Icon icon="mdi:link-variant" class="input-icon" />
                    </template>
                  </a-input>
                </a-form-item>
                
                <a-form-item label="API描述" class="form-item">
                  <a-textarea 
                    v-model:value="formData.description" 
                    placeholder="请输入API描述" 
                    :rows="3"
                    class="custom-textarea"
                  />
                </a-form-item>
              </div>
            </div>
            
            <div class="form-section request-info">
              <div class="section-header">
                <Icon icon="mdi:code-json" class="section-icon" />
                <span>请求配置</span>
              </div>
              
              <div class="section-content">
                <a-form-item label="请求方法" required class="form-item">
                  <!-- 修改为下拉框形式 -->
                  <a-select
                    v-model:value="formData.method"
                    placeholder="请选择请求方法"
                    class="method-select"
                  >
                    <a-select-option v-for="method in methodOptions" :key="method.value" :value="method.value">
                      <div class="method-option-content">
                        <a-tag :color="getMethodColor(method.value)" class="method-badge">
                          {{ method.label }}
                        </a-tag>
                        <span class="method-description">{{ method.description }}</span>
                      </div>
                    </a-select-option>
                  </a-select>
                </a-form-item>
                
                <div class="form-row">
                  <a-form-item label="API版本" class="form-item">
                    <a-input 
                      v-model:value="formData.version" 
                      placeholder="例如: v1, v2.0" 
                      class="custom-input"
                    >
                      <template #prefix>
                        <Icon icon="mdi:tag-outline" class="input-icon" />
                      </template>
                    </a-input>
                  </a-form-item>
                  
                  <a-form-item label="API分类" class="form-item">
                    <a-input-number 
                      v-model:value="formData.category" 
                      placeholder="分类ID" 
                      class="category-input"
                      :min="0"
                    />
                  </a-form-item>
                </div>
                
                <a-form-item label="访问权限" class="public-switch-form-item">
                  <div class="switch-container">
                    <a-switch 
                      v-model:checked="formData.is_public" 
                      :checkedValue="1" 
                      :unCheckedValue="0"
                      class="public-switch"
                    />
                    <span class="switch-label">{{ formData.is_public === 1 ? '公开接口' : '私有接口' }}</span>
                    <div class="switch-hint">
                      {{ formData.is_public === 1 ? '所有用户均可访问' : '需要授权才能访问' }}
                    </div>
                  </div>
                </a-form-item>
              </div>
            </div>
          </div>
          
          <div class="api-preview" v-if="formData.path && formData.method">
            <div class="preview-title">
              <Icon icon="mdi:eye-outline" class="preview-icon" />
              接口预览
            </div>
            <div class="preview-content">
              <div class="preview-method">
                <a-tag :color="getMethodColor(formData.method)" class="preview-method-tag">
                  {{ getMethodName(formData.method) }}
                </a-tag>
              </div>
              <div class="preview-path">{{ formData.path }}</div>
              <div class="preview-access">
                <a-tag :color="formData.is_public === 1 ? '#52c41a' : '#faad14'" class="access-tag">
                  {{ formData.is_public === 1 ? '公开' : '私有' }}
                </a-tag>
              </div>
            </div>
          </div>
        </a-form>
        
        <div class="modal-footer">
          <a-button @click="handleModalCancel" class="cancel-button">
            取消
          </a-button>
          <a-button type="primary" @click="handleModalOk" class="submit-button">
            <Icon icon="mdi:content-save" class="button-icon" />
            保存
          </a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, computed } from 'vue';
import { message } from 'ant-design-vue';
import { listApisApi, createApiApi, updateApiApi, deleteApiApi } from '#/api/core/system';
import type { SystemApi } from '#/api/core/system';
import { Icon } from '@iconify/vue';

// 表格加载状态
const loading = ref(false);

// 搜索文本
const searchText = ref('');

// API列表数据
const apiList = ref<any[]>([]);

// 请求方法选项
const methodOptions = [
  { label: 'GET', value: 1, icon: 'mdi:arrow-down-box', description: '查询数据' },
  { label: 'POST', value: 2, icon: 'mdi:plus-box', description: '创建数据' },
  { label: 'PUT', value: 3, icon: 'mdi:pencil-box', description: '更新数据' },
  { label: 'DELETE', value: 4, icon: 'mdi:delete-box', description: '删除数据' }
];

// 过滤后的API列表
const filteredApiList = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  if (!searchValue) return apiList.value;
  
  return apiList.value.filter(api => 
    api.name.toLowerCase().includes(searchValue) ||
    api.path.toLowerCase().includes(searchValue) ||
    api.description?.toLowerCase().includes(searchValue)
  );
});

// 对话框相关
const modalVisible = ref(false);
const modalTitle = ref('新增API');
const formData = reactive<SystemApi.CreateApiReq>({
  name: '',
  path: '',
  method: 1,
  description: '',
  version: '',
  category: undefined,
  is_public: 0
});

// 获取API列表
const fetchApiList = async () => {
  loading.value = true;
  try {
    const res = await listApisApi({
      page_number: 1,
      page_size: 999
    });
    apiList.value = res.list;
  } catch (error: any) {
    message.error(error.message || '获取API列表失败');
  }
  loading.value = false;
};

// 表格列配置
const columns = [
  {
    title: 'API名称',
    dataIndex: 'name',
    key: 'name',
    width: 180
  },
  {
    title: 'API路径', 
    dataIndex: 'path',
    key: 'path',
    ellipsis: true,
    slots: { customRender: 'path' }
  },
  {
    title: '请求方法',
    dataIndex: 'method',
    key: 'method', 
    slots: { customRender: 'method' },
    width: 100
  },
  {
    title: 'API描述',
    dataIndex: 'description',
    key: 'description',
    ellipsis: true,
  },
  {
    title: 'API版本',
    dataIndex: 'version',
    key: 'version',
    width: 100
  },
  {
    title: '是否公开',
    dataIndex: 'is_public',
    key: 'is_public',
    slots: { customRender: 'isPublic' },
    width: 100,
    align: 'center'
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    width: 120,
    fixed: 'right'
  },
];

// 获取请求方法名称
const getMethodName = (method: number) => {
  const methodMap: Record<number, string> = {
    1: 'GET',
    2: 'POST', 
    3: 'PUT',
    4: 'DELETE'
  };
  return methodMap[method] || '未知';
};

// 获取请求方法颜色
const getMethodColor = (method: number) => {
  const colorMap: Record<number, string> = {
    1: '#1890ff', // 蓝色 - GET
    2: '#52c41a', // 绿色 - POST
    3: '#faad14', // 橙色 - PUT
    4: '#f5222d'  // 红色 - DELETE
  };
  return colorMap[method] || '#d9d9d9';
};

// 处理搜索
const handleSearch = () => {
  // 搜索功能已通过 computed 属性 filteredApiList 实现
  // 不需要额外的处理逻辑
};

// 处理新增
const handleAdd = () => {
  modalTitle.value = '新增API';
  Object.assign(formData, {
    name: '',
    path: '',
    method: 1,
    description: '',
    version: '',
    category: undefined,
    is_public: 0
  });
  modalVisible.value = true;
};

// 处理编辑
const handleEdit = (record: any) => {
  modalTitle.value = '编辑API';
  Object.assign(formData, record);
  modalVisible.value = true;
};

// 处理删除
const handleDelete = async (record: any) => {
  try {
    await deleteApiApi(record.id);
    message.success('删除成功');
    fetchApiList();
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

// 处理公开状态切换
const handlePublicChange = async (record: any, isPublic: number) => {
  try {
    await updateApiApi({
      ...record,
      is_public: isPublic,
    });
    message.success('更新成功');
    fetchApiList();
  } catch (error: any) {
    message.error(error.message || '更新失败');
  }
};

// 处理对话框确认
const handleModalOk = async () => {
  try {
    if (modalTitle.value === '新增API') {
      await createApiApi(formData);
      message.success('新增API成功');
    } else {
      await updateApiApi(formData as SystemApi.UpdateApiReq);
      message.success('编辑API成功');
    }
    modalVisible.value = false;
    fetchApiList();
  } catch (error: any) {
    message.error(error.message || `${modalTitle.value}失败`);
  }
};

// 处理对话框取消
const handleModalCancel = () => {
  modalVisible.value = false;
};

// 页面加载时获取数据
onMounted(() => {
  fetchApiList();
});
</script>

<style scoped>
/* 整体容器样式 */
.api-management-container {
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

.api-table {
  width: 100%;
}

/* API路径样式 */
.api-path {
  font-family: 'Roboto Mono', 'Courier New', monospace;
  color: #595959;
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 请求方法标签样式 */
.method-tag {
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 500;
  font-family: 'Roboto Mono', monospace;
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

/* 开关样式 */
:deep(.public-switch) {
  background-color: rgba(0, 0, 0, 0.25);
}

:deep(.public-switch.ant-switch-checked) {
  background: linear-gradient(90deg, #1890ff, #36cfc9);
}

/* API模态框基础样式 */
:deep(.api-modal .ant-modal-content) {
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

:deep(.api-modal .ant-modal-header) {
  background: #fff;
  padding: 20px 24px 0;
  border-bottom: none;
}

:deep(.api-modal .ant-modal-title) {
  font-size: 18px;
  font-weight: 600;
  color: #1e293b;
}

:deep(.api-modal .ant-modal-body) {
  padding: 0;
}

:deep(.api-modal .ant-modal-footer) {
  border-top: 1px solid #f0f0f0;
  padding: 16px 24px;
}

.modal-content {
  padding: 20px 24px 24px;
}

.modal-header-icon {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 30px;
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

/* API表单样式 */
.api-form {
  margin-top: 0;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.form-section {
  background-color: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  border: 1px solid #f0f0f0;
  overflow: hidden;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background-color: #f9f9f9;
  border-bottom: 1px solid #f0f0f0;
}

.section-icon {
  color: #1890ff;
  font-size: 18px;
}

.section-content {
  padding: 16px;
}

.form-item {
  margin-bottom: 16px;
}

.form-item:last-child {
  margin-bottom: 0;
}

.form-row {
  display: flex;
  gap: 16px;
}

.form-row .form-item {
  flex: 1;
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

:deep(.custom-textarea) {
  border-radius: 6px;
  transition: all 0.3s;
}

:deep(.custom-textarea:hover) {
  border-color: #40a9ff;
}

:deep(.custom-textarea:focus) {
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

/* 方法选择下拉框样式 */
:deep(.method-select .ant-select-selector) {
  border-radius: 6px;
  transition: all 0.3s;
  height: 38px !important;
  padding: 0 11px !important;
}

:deep(.method-select .ant-select-selection-item) {
  line-height: 36px !important;
}

:deep(.method-select:hover .ant-select-selector) {
  border-color: #40a9ff !important;
}

:deep(.method-select.ant-select-focused .ant-select-selector) {
  border-color: #1890ff !important;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2) !important;
}

/* 方法选项内容样式 */
.method-option-content {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
}

.method-badge {
  min-width: 60px;
  text-align: center;
  font-weight: bold;
  font-size: 12px;
  border: none;
}

.method-description {
  color: #595959;
  font-size: 14px;
}

/* 分类输入框样式 */
:deep(.category-input) {
  width: 100%;
  border-radius: 6px;
}

:deep(.category-input .ant-input-number-handler-wrap) {
  border-radius: 0 6px 6px 0;
}

/* 开关容器样式 */
.switch-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.switch-label {
  font-weight: 500;
  color: #1e293b;
}

.switch-hint {
  font-size: 12px;
  color: #8c8c8c;
  margin-left: auto;
}

/* API预览区域 */
.api-preview {
  margin-top: 24px;
  background-color: #fafafa;
  border-radius: 8px;
  border: 1px dashed #d9d9d9;
  overflow: hidden;
}

.preview-title {
  padding: 12px 16px;
  font-weight: 500;
  color: #1e293b;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-icon {
  color: #1890ff;
  font-size: 18px;
}

.preview-content {
  padding: 16px;
  display: flex;
  align-items: center;
  font-family: 'Roboto Mono', monospace;
  background-color: #f5f5f5;
}

.preview-method {
  margin-right: 12px;
}

.preview-method-tag {
  padding: 4px 12px;
  font-size: 14px;
  font-weight: bold;
  border: none;
}

.preview-path {
  flex: 1;
  font-size: 14px;
  color: #1e293b;
  word-break: break-all;
}

.preview-access {
  margin-left: 12px;
}

.access-tag {
  padding: 2px 8px;
  font-size: 12px;
}

/* 模态框页脚 */
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.cancel-button {
  border-radius: 6px;
  border: 1px solid #d9d9d9;
  background-color: white;
  color: #595959;
  padding: 0 16px;
  height: 38px;
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
  height: 38px;
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
  
  .form-grid {
    grid-template-columns: 1fr;
  }
  
  .form-row {
    flex-direction: column;
  }
  
  .preview-content {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  
  .preview-method {
    margin-right: 0;
  }
  
  .preview-access {
    margin-left: 0;
  }
}
</style>