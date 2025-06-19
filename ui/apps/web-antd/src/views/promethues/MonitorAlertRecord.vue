<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">记录规则管理</h2>
      <div class="page-description">管理和监控Prometheus记录规则配置</div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card custom-toolbar">
      <div class="search-filters">
        <a-input 
          v-model:value="searchText" 
          placeholder="请输入记录名称" 
          class="search-input"
        >
          <template #prefix>
            <SearchOutlined class="search-icon" />
          </template>
        </a-input>
        <a-button type="primary" class="action-button" @click="handleSearch">
          <template #icon>
            <SearchOutlined />
          </template>
          搜索
        </a-button>
        <a-button class="action-button reset-button" @click="handleReset">
          <template #icon>
            <ReloadOutlined />
          </template>
          重置
        </a-button>
      </div>
      <div class="action-buttons">
        <a-button type="primary" class="add-button" @click="showAddModal">
          <template #icon>
            <PlusOutlined />
          </template>
          新增记录
        </a-button>
      </div>
    </div>

    <!-- 记录列表表格 -->
    <div class="dashboard-card table-container">
      <a-table 
        :columns="columns" 
        :data-source="data" 
        row-key="id" 
        :loading="loading"
        :pagination="false"
        class="custom-table"
        :scroll="{ x: 1200 }"
      >
        <!-- 标签组列 -->
        <template #labels="{ record }">
          <div class="tag-container">
            <a-tag v-for="label in record.labels" :key="label" class="tech-tag label-tag">
              {{ label }}
            </a-tag>
          </div>
        </template>
        
        <!-- 是否启用列 -->
        <template #enable="{ text }">
          <a-tag :class="text ? 'tech-tag status-enabled' : 'tech-tag status-disabled'">
            {{ text ? '启用' : '禁用' }}
          </a-tag>
        </template>
        
        <!-- 操作列 -->
        <template #action="{ record }">
          <div class="action-column">
            <a-tooltip title="编辑资源信息">
              <a-button type="primary" shape="circle" class="edit-button" @click="showEditModal(record)">
                <template #icon>
                  <Icon icon="clarity:note-edit-line" />
                </template>
              </a-button>
            </a-tooltip>
            <a-tooltip title="删除资源">
              <a-button type="primary" danger shape="circle" class="delete-button" @click="handleDelete(record)">
                <template #icon>
                  <Icon icon="ant-design:delete-outlined" />
                </template>
              </a-button>
            </a-tooltip>
          </div>
        </template>
      </a-table>

      <!-- 分页器 -->
      <div class="pagination-container">
        <a-pagination 
          v-model:current="current" 
          v-model:pageSize="pageSizeRef" 
          :page-size-options="pageSizeOptions"
          :total="total" 
          show-size-changer 
          @change="handlePageChange" 
          @showSizeChange="handleSizeChange" 
          class="custom-pagination"
        >
          <template #buildOptionText="props">
            <span v-if="props.value !== '50'">{{ props.value }}条/页</span>
            <span v-else>全部</span>
          </template>
        </a-pagination>
      </div>
    </div>

    <!-- 新增记录规则模态框 -->
    <a-modal 
      title="新增记录规则" 
      v-model:visible="isAddModalVisible" 
      @ok="handleAdd" 
      @cancel="closeAddModal"
      :confirmLoading="loading"
      :width="700"
      class="custom-modal"
      ok-text="提交"
      cancel-text="取消"
    >
      <a-form ref="addFormRef" :model="addForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="记录名称" name="name" :rules="[{ required: true, message: '请输入记录名称' }]">
                <a-input v-model:value="addForm.name" placeholder="请输入记录名称" />
              </a-form-item>
            </a-col>
          </a-row>
          
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="Prometheus 实例池" name="poolId" :rules="[{ required: true, message: '请选择实例池' }]">
                <a-select
                  v-model:value="addForm.pool_id"
                  placeholder="请选择实例池"
                  class="full-width"
                >
                  <a-select-option
                    v-for="pool in poolOptions"
                    :key="pool.id"
                    :value="pool.id"
                  >
                    {{ pool.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="IP地址" name="ip_address" :rules="[
                { required: true, message: '请输入IP地址' },
                { pattern: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/, message: '请输入有效的IP地址' }
              ]">
                <a-input v-model:value="addForm.ip_address" placeholder="例如: 192.168.1.100" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="端口" name="port" :rules="[
                { required: true, message: '请输入端口号' },
                { pattern: /^([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$/, message: '请输入有效的端口号(1-65535)' }
              ]">
                <a-input v-model:value="addForm.port" placeholder="例如: 9090" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">规则配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="是否启用" name="enable" :rules="[{ required: true, message: '请选择是否启用' }]">
                <a-switch v-model:checked="addForm.enable" class="tech-switch" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="持续时间" name="forTime">
                <a-input v-model:value="addForm.for_time" placeholder="例如: 15s" />
              </a-form-item>
            </a-col>
          </a-row>
          
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="表达式" name="expr">
                <a-input v-model:value="addForm.expr" placeholder="请输入表达式" />
              </a-form-item>
              <a-form-item>
                <a-button type="primary" class="validate-button" @click="validateAddExpression">
                  <template #icon>
                    <Icon icon="mdi:check-circle-outline" />
                  </template>
                  验证表达式
                </a-button>
              </a-form-item>
            </a-col>
          </a-row>
        </div>
      </a-form>
    </a-modal>

    <!-- 编辑记录规则模态框 -->
    <a-modal 
      title="编辑记录规则" 
      v-model:visible="isEditModalVisible" 
      @ok="handleUpdate" 
      @cancel="closeEditModal"
      :confirmLoading="loading"
      :width="700"
      class="custom-modal"
      ok-text="提交"
      cancel-text="取消"
    >
      <a-form ref="editFormRef" :model="editForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="记录名称" name="name" :rules="[{ required: true, message: '请输入记录名称' }]">
                <a-input v-model:value="editForm.name" placeholder="请输入记录名称" />
              </a-form-item>
            </a-col>
          </a-row>
          
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="Prometheus 实例池" name="poolId" :rules="[{ required: true, message: '请选择实例池' }]">
                <a-select
                  v-model:value="editForm.pool_id"
                  placeholder="请选择实例池"
                  class="full-width"
                >
                  <a-select-option
                    v-for="pool in poolOptions"
                    :key="pool.id"
                    :value="pool.id"
                  >
                    {{ pool.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="IP地址" name="ip_address" :rules="[
                { required: true, message: '请输入IP地址' },
                { pattern: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/, message: '请输入有效的IP地址' }
              ]">
                <a-input v-model:value="editForm.ip_address" placeholder="例如: 192.168.1.100" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="端口" name="port" :rules="[
                { required: true, message: '请输入端口号' },
                { pattern: /^([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$/, message: '请输入有效的端口号(1-65535)' }
              ]">
                <a-input v-model:value="editForm.port" placeholder="例如: 9090" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">规则配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="是否启用" name="enable" :rules="[{ required: true, message: '请选择是否启用' }]">
                <a-switch v-model:checked="editForm.enable" class="tech-switch" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="持续时间" name="forTime">
                <a-input v-model:value="editForm.for_time" placeholder="例如: 15s" />
              </a-form-item>
            </a-col>
          </a-row>
          
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="表达式" name="expr">
                <a-input v-model:value="editForm.expr" placeholder="请输入表达式" />
              </a-form-item>
              <a-form-item>
                <a-button type="primary" class="validate-button" @click="validateEditExpression">
                  <template #icon>
                    <Icon icon="mdi:check-circle-outline" />
                  </template>
                  验证表达式
                </a-button>
              </a-form-item>
            </a-col>
          </a-row>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
} from '@ant-design/icons-vue';
import {
  getRecordRulesListApi,
  createRecordRuleApi,
  updateRecordRuleApi,
  deleteRecordRuleApi,
  getRecordRulesTotalApi,
} from '#/api/core/prometheus_alert_record';
import { getAllMonitorScrapePoolApi } from '#/api/core/prometheus_scrape_pool';
import { validateExprApi } from '#/api/core/prometheus_alert_rule';
import { Icon } from '@iconify/vue';
import type { AlertRecordItem } from '#/api/core/prometheus_alert_record';
import type { FormInstance } from 'ant-design-vue';

// 定义 Pool 类型
interface Pool {
  id: number;
  name: string;
}

// 数据源
const data = ref<AlertRecordItem[]>([]);

// 下拉框数据源
const poolOptions = ref<Pool[]>([]);

// 搜索文本
const searchText = ref('');

const handleReset = () => {
  searchText.value = '';
  fetchRecordRules();
};

// 处理搜索
const handleSearch = () => {
  current.value = 1;
  fetchRecordRules();
};

const handleSizeChange = (page: number, size: number) => {
  current.value = page;
  pageSizeRef.value = size;
  fetchRecordRules();
};

// 处理分页变化
const handlePageChange = (page: number) => {
  current.value = page;
  fetchRecordRules();
};

// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);

// 加载状态
const loading = ref(false);

// 表格列配置
const columns = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
    width: 80,
  },
  {
    title: '记录名称',
    dataIndex: 'name',
    key: 'name',
    width: 150,
  },
  {
    title: '关联 Prometheus 实例池',
    dataIndex: 'pool_name',
    key: 'pool_name',
    width: 180,
  },
  {
    title: 'IP地址',
    dataIndex: 'ip_address',
    key: 'ip_address',
    width: 120,
  },
  {
    title: '是否启用',
    dataIndex: 'enable',
    key: 'enable',
    slots: { customRender: 'enable' },
    width: 100,
  },
  {
    title: '持续时间',
    dataIndex: 'for_time',
    key: 'for_time',
    width: 100,
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    key: 'create_user_name',
    width: 120,
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: 180,
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    fixed: 'right',
    width: 120,
  },
];

const addFormRef = ref<FormInstance>();
const editFormRef = ref<FormInstance>();

// 模态框状态和表单
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);

// 新增表单
const addForm = reactive({
  name: '',
  pool_id: null,
  ip_address: '',
  port: '',
  enable: false,
  for_time: '15s',
  expr: '',
  labels: [],
  annotations: [],
});

// 编辑表单
const editForm = reactive({
  id: 0,
  name: '',
  pool_id: null,
  ip_address: '',
  port: '',
  enable: true,
  for_time: '',
  expr: '',
  labels: [],
  annotations: [],
});

// 显示新增模态框
const showAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

// 重置新增表单
const resetAddForm = () => {
  addForm.name = '';
  addForm.pool_id = null;
  addForm.ip_address = '';
  addForm.port = '';
  addForm.enable = false;
  addForm.for_time = '15s';
  addForm.expr = '';
  addForm.labels = [];
  addForm.annotations = [];
};

// 关闭新增模态框
const closeAddModal = () => {
  isAddModalVisible.value = false;
};

// 显示编辑模态框
const showEditModal = (record: AlertRecordItem) => {
  Object.assign(editForm, {
    id: record.id,
    name: record.name,
    pool_id: record.pool_id,
    ip_address: record.ip_address,
    port: record.port,
    enable: record.enable,
    for_time: record.for_time,
    expr: record.expr,
    labels: record.labels,
    annotations: record.annotations,
  });
  isEditModalVisible.value = true;
};

// 关闭编辑模态框
const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 提交新增记录
const handleAdd = async () => {
  try {
    await addFormRef.value?.validate();
    
    const payload = {
      name: addForm.name,
      pool_id: addForm.pool_id,
      ip_address: addForm.ip_address,
      port: addForm.port,
      enable: addForm.enable,
      for_time: addForm.for_time,
      expr: addForm.expr,
      labels: addForm.labels,
      annotations: addForm.annotations,
    };

    loading.value = true;
    await createRecordRuleApi(payload);
    loading.value = false;
    message.success('新增记录成功');
    fetchRecordRules();
    closeAddModal();
  } catch (error: any) {
    loading.value = false;
    message.error(error.message || '新增记录失败，请稍后重试');
    console.error(error);
  }
};

// 提交更新记录
const handleUpdate = async () => {
  try {
    await editFormRef.value?.validate();
    
    const payload = {
      id: editForm.id,
      name: editForm.name,
      pool_id: editForm.pool_id,
      ip_address: editForm.ip_address,
      port: editForm.port,
      enable: editForm.enable,
      for_time: editForm.for_time,
      expr: editForm.expr,
      labels: editForm.labels,
      annotations: editForm.annotations,
    };

    loading.value = true;
    await updateRecordRuleApi(payload);
    loading.value = false;
    message.success('更新记录规则成功');
    fetchRecordRules();
    closeEditModal();
  } catch (error: any) {
    loading.value = false;
    message.error(error.message || '更新记录规则失败，请稍后重试');
    console.error(error);
  }
};

// 处理删除记录规则
const handleDelete = (record: AlertRecordItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除记录规则 "${record.name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await deleteRecordRuleApi(record.id);
        loading.value = false;
        message.success('记录规则已删除');
        fetchRecordRules();
      } catch (error: any) {
        loading.value = false;
        message.error(error.message || '删除记录规则失败，请稍后重试');
        console.error(error);
      }
    },
  });
};

// 获取记录规则数据
const fetchRecordRules = async () => {
  try {
    loading.value = true;
    const response = await getRecordRulesListApi({
      page: current.value,
      size: pageSizeRef.value,
      search: searchText.value,
    });
    data.value = response.items;
    total.value = response.total;
    loading.value = false;
  } catch (error: any) {
    loading.value = false;
    message.error(error.message || '获取记录规则数据失败，请稍后重试');
    console.error(error);
  }
};

// 获取所有实例池数据
const fetchPools = async () => {
  try {
    const response = await getAllMonitorScrapePoolApi();
    poolOptions.value = response.items;
  } catch (error: any) {
    message.error(error.message || '获取实例池数据失败，请稍后重试');
    console.error(error);
  }
};

// 表达式验证（新增）
const validateAddExpression = async () => {
  try {
    if (!addForm.expr) {
      message.error('表达式不能为空');
      return;
    }
    const payload = { promql_expr: addForm.expr };
    await validateExprApi(payload);
    message.success('表达式验证成功');
  } catch (error: any) {
    message.error(error.message || '表达式验证失败，请稍后重试');
    console.error(error);
  }
};

// 表达式验证（编辑）
const validateEditExpression = async () => {
  try {
    if (!editForm.expr) {
      message.error('表达式不能为空');
      return;
    }
    const payload = { promql_expr: editForm.expr };
    await validateExprApi(payload);
    message.success('表达式验证成功');
  } catch (error: any) {
    message.error(error.message || '表达式验证失败，请稍后重试');
    console.error(error);
  }
};

// 在组件加载时获取数据
onMounted(() => {
  fetchRecordRules();
  fetchPools();
});
</script>

<style scoped>
.monitor-page {
  padding: 20px;
  background-color: #f0f2f5;
  min-height: 100vh;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 8px;
}

.page-description {
  color: #666;
  font-size: 14px;
}

.dashboard-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  padding: 20px;
  margin-bottom: 24px;
  transition: all 0.3s;
}

.custom-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-filters {
  display: flex;
  gap: 16px;
  align-items: center;
}

.search-input {
  width: 250px;
  border-radius: 4px;
  transition: all 0.3s;
}

.search-input:hover,
.search-input:focus {
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

.search-icon {
  color: #bfbfbf;
}

.action-button {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  border-radius: 4px;
  transition: all 0.3s;
}

.reset-button {
  background-color: #f5f5f5;
  color: #595959;
  border-color: #d9d9d9;
}

.reset-button:hover {
  background-color: #e6e6e6;
  border-color: #b3b3b3;
}

.add-button {
  background: linear-gradient(45deg, #1890ff, #36bdf4);
  border: none;
  box-shadow: 0 2px 6px rgba(24, 144, 255, 0.4);
}

.add-button:hover {
  background: linear-gradient(45deg, #096dd9, #1890ff);
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(24, 144, 255, 0.5);
}

.table-container {
  overflow: hidden;
}

.custom-table {
  margin-top: 8px;
}

:deep(.ant-table) {
  border-radius: 8px;
  overflow: hidden;
}

:deep(.ant-table-thead > tr > th) {
  background-color: #f7f9fc;
  font-weight: 600;
  color: #1f1f1f;
  padding: 16px 12px;
}

:deep(.ant-table-tbody > tr > td) {
  padding: 12px;
}

:deep(.ant-table-tbody > tr:hover > td) {
  background-color: #f0f7ff;
}

.tag-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tech-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
}

.status-enabled {
  background-color: #f6ffed;
  color: #389e0d;
  border-left: 3px solid #52c41a;
}

.status-disabled {
  background-color: #fff2f0;
  color: #cf1322;
  border-left: 3px solid #ff4d4f;
}

.label-tag {
  background-color: #f6ffed;
  color: #389e0d;
  border-left: 3px solid #52c41a;
}

.action-column {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.edit-button {
  background: #1890ff;
  border: none;
  box-shadow: 0 2px 4px rgba(24, 144, 255, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
}

.edit-button:hover {
  background: #096dd9;
  transform: scale(1.05);
}

.delete-button {
  background: #ff4d4f;
  border: none;
  box-shadow: 0 2px 4px rgba(255, 77, 79, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
}

.delete-button:hover {
  background: #cf1322;
  transform: scale(1.05);
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}

.custom-pagination {
  margin-right: 12px;
}

/* 模态框样式 */
:deep(.custom-modal .ant-modal-content) {
  border-radius: 8px;
  overflow: hidden;
}

:deep(.custom-modal .ant-modal-header) {
  padding: 20px 24px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

:deep(.custom-modal .ant-modal-title) {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
}

:deep(.custom-modal .ant-modal-body) {
  padding: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

:deep(.custom-modal .ant-modal-footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 表单样式 */
.custom-form {
  width: 100%;
}

.form-section {
  margin-bottom: 28px;
  padding: 0;
  position: relative;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 16px;
  padding-left: 12px;
  border-left: 4px solid #1890ff;
}

:deep(.custom-form .ant-form-item-label > label) {
  font-weight: 500;
  color: #333;
}

.full-width {
  width: 100%;
}

:deep(.tech-switch) {
  background-color: rgba(0, 0, 0, 0.25);
}

:deep(.tech-switch.ant-switch-checked) {
  background: linear-gradient(45deg, #1890ff, #36cfc9);
}

.validate-button {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #52c41a;
  border: none;
  box-shadow: 0 2px 6px rgba(82, 196, 26, 0.4);
  transition: all 0.3s;
}

.validate-button:hover {
  background: #389e0d;
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(82, 196, 26, 0.5);
}
</style>