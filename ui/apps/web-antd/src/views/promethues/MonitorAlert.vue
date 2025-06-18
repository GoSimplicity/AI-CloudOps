<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">AlertManager实例池管理</h2>
      <div class="page-description">管理和监控AlertManager实例池及其相关配置</div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card custom-toolbar">
      <div class="search-filters">
        <a-input 
          v-model:value="searchText" 
          placeholder="请输入AlertManager实例名称" 
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
          新增AlertManager实例池
        </a-button>
      </div>
    </div>

    <!-- AlertManager实例池列表表格 -->
    <div class="dashboard-card table-container">
      <a-table 
        :columns="columns" 
        :data-source="data" 
        row-key="id" 
        :pagination="false"
        class="custom-table"
        :scroll="{ x: 1200 }"
      >
        <template #alert_manager_instances="{ record }">
          <div class="tag-container">
            <a-tag v-for="instance in record.alert_manager_instances" :key="instance" class="tech-tag alert-tag">
              {{ instance }}
            </a-tag>
          </div>
        </template>

        <template #group_by="{ record }">
          <div class="tag-container">
            <template v-if="record.group_by && record.group_by.length && record.group_by[0] !== ''">
              <a-tag v-for="label in record.group_by" :key="label" class="tech-tag label-tag">
                <span class="label-key">{{ label.split(',')[0] }}</span>
                <span class="label-separator">:</span>
                <span class="label-value">{{ label.split(',')[1] }}</span>
              </a-tag>
            </template>
            <a-tag v-else class="tech-tag empty-tag">无标签</a-tag>
          </div>
        </template>

        <template #alertRules="{ record }">
          <div class="tag-container">
            <a-tag v-for="rule in record.alertRules" :key="rule" class="tech-tag prometheus-tag">
              {{ rule }}
            </a-tag>
          </div>
        </template>

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

    <!-- 新增AlertManager实例池模态框 -->
    <a-modal 
      title="新增AlertManager实例池" 
      v-model:visible="isAddModalVisible" 
      @ok="handleAdd" 
      @cancel="closeAddModal"
      :width="700"
      class="custom-modal"
    >
      <a-form ref="addFormRef" :model="addForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="实例池名称" name="name" :rules="[{ required: true, message: '请输入实例池名称' }]">
                <a-input v-model:value="addForm.name" placeholder="请输入实例池名称" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">实例配置</div>
          <!-- 动态AlertManager实例表单项 -->
          <a-form-item v-for="(instance, index) in addForm.alert_manager_instances" :key="instance.key"
            :label="index === 0 ? 'AlertManager实例' : ''" :name="['alert_manager_instances', index, 'value']"
            :rules="{ required: true, message: '请输入AlertManager实例IP' }">
            <div class="dynamic-input-container">
              <a-input v-model:value="instance.value" placeholder="请输入AlertManager实例IP" class="dynamic-input" />
              <MinusCircleOutlined v-if="addForm.alert_manager_instances.length > 1" class="dynamic-delete-button"
                @click="removeAlertManagerInstance(instance)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addAlertManagerInstance">
              <PlusOutlined />
              添加AlertManager实例
            </a-button>
          </a-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">标签配置</div>
          <!-- 动态标签表单项 -->
          <a-form-item v-for="(label, index) in addForm.group_by" :key="label.key"
            :label="index === 0 ? '分组标签' : ''">
            <div class="label-input-group">
              <a-input v-model:value="label.labelKey" placeholder="请输入标签Key" class="label-key-input" />
              <div class="label-separator">:</div>
              <a-input v-model:value="label.labelValue" placeholder="请输入标签Value" class="label-value-input" />
              <MinusCircleOutlined v-if="addForm.group_by.length > 1" class="dynamic-delete-button"
                @click="removeLabel(label)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addLabel">
              <PlusOutlined />
              添加标签
            </a-button>
          </a-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">告警配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="默认恢复时间" name="resolve_timeout">
                <a-input v-model:value="addForm.resolve_timeout" placeholder="例如: 5s" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="默认分组第一次等待时间" name="group_wait">
                <a-input v-model:value="addForm.group_wait" placeholder="例如: 5s" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="默认分组等待间隔" name="group_interval">
                <a-input v-model:value="addForm.group_interval" placeholder="例如: 5s" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="默认重复发送时间" name="repeat_interval">
                <a-input v-model:value="addForm.repeat_interval" placeholder="例如: 5s" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="兜底接收者" name="receiver">
                <a-input v-model:value="addForm.receiver" placeholder="请输入兜底接收者" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>
      </a-form>
    </a-modal>

    <!-- 编辑AlertManager实例池模态框 -->
    <a-modal 
      title="编辑AlertManager实例池" 
      v-model:visible="isEditModalVisible" 
      @ok="handleEdit" 
      @cancel="closeEditModal"
      :width="700"
      class="custom-modal"
    >
      <a-form ref="editFormRef" :model="editForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="实例池名称" name="name" :rules="[{ required: true, message: '请输入实例池名称' }]">
                <a-input v-model:value="editForm.name" placeholder="请输入实例池名称" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">实例配置</div>
          <!-- 动态AlertManager实例表单项 -->
          <a-form-item v-for="(instance, index) in editForm.alert_manager_instances" :key="instance.key"
            :label="index === 0 ? 'AlertManager实例' : ''" :name="['alert_manager_instances', index, 'value']"
            :rules="{ required: true, message: '请输入AlertManager实例IP' }">
            <div class="dynamic-input-container">
              <a-input v-model:value="instance.value" placeholder="请输入AlertManager实例IP" class="dynamic-input" />
              <MinusCircleOutlined v-if="editForm.alert_manager_instances.length > 1" class="dynamic-delete-button"
                @click="removeEditAlertManagerInstance(instance)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addEditAlertManagerInstance">
              <PlusOutlined />
              添加AlertManager实例
            </a-button>
          </a-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">标签配置</div>
          <!-- 动态标签表单项 -->
          <a-form-item v-for="(label, index) in editForm.group_by" :key="label.key"
            :label="index === 0 ? '分组标签' : ''">
            <div class="label-input-group">
              <a-input v-model:value="label.labelKey" placeholder="请输入标签Key" class="label-key-input" />
              <div class="label-separator">:</div>
              <a-input v-model:value="label.labelValue" placeholder="请输入标签Value" class="label-value-input" />
              <MinusCircleOutlined v-if="editForm.group_by.length > 1" class="dynamic-delete-button"
                @click="removeEditLabel(label)" />
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addEditLabel">
              <PlusOutlined />
              添加标签
            </a-button>
          </a-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">告警配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="默认恢复时间" name="resolve_timeout">
                <a-input v-model:value="editForm.resolve_timeout" placeholder="例如: 5s" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="默认分组第一次等待时间" name="group_wait">
                <a-input v-model:value="editForm.group_wait" placeholder="例如: 5s" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="默认分组等待间隔" name="group_interval">
                <a-input v-model:value="editForm.group_interval" placeholder="例如: 5s" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="默认重复发送时间" name="repeat_interval">
                <a-input v-model:value="editForm.repeat_interval" placeholder="例如: 5s" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="兜底接收者" name="receiver">
                <a-input v-model:value="editForm.receiver" placeholder="请输入兜底接收者" />
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
  getAlertManagerPoolListApi,
  createAlertManagerPoolApi,
  updateAlertManagerPoolApi,
  deleteAlertManagerPoolApi,
} from '#/api/core/prometheus_alert_pool';
import { Icon } from '@iconify/vue';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  MinusCircleOutlined
} from '@ant-design/icons-vue';
import type { MonitorAlertPoolItem } from '#/api/core/prometheus_alert_pool';

// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);

// 数据源
const data = ref<MonitorAlertPoolItem[]>([]);

// 搜索文本
const searchText = ref('');

// 分页处理
const handlePageChange = (page: number) => {
  current.value = page;
  fetchAlertManagerPools();
};

const handleSizeChange = (page: number, size: number) => {
  current.value = page;
  pageSizeRef.value = size;
  fetchAlertManagerPools();
};

// 搜索处理
const handleSearch = () => {
  current.value = 1;
  fetchAlertManagerPools();
};

// 表格列配置
const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
    width: 80,
  },
  {
    title: '名称',
    dataIndex: 'name',
    key: 'name',
    width: 150,
  },
  {
    title: 'AlertManager实例',
    dataIndex: 'alert_manager_instances',
    key: 'alert_manager_instances',
    slots: { customRender: 'alert_manager_instances' },
    width: 200,
  },
  {
    title: '恢复时间',
    dataIndex: 'resolve_timeout',
    key: 'resolve_timeout',
    width: 120,
  },
  {
    title: '分组等待时间',
    dataIndex: 'group_wait',
    key: 'group_wait',
    width: 120,
  },
  {
    title: '分组等待间隔',
    dataIndex: 'group_interval',
    key: 'group_interval',
    width: 120,
  },
  {
    title: '重复发送时间',
    dataIndex: 'repeat_interval',
    key: 'repeat_interval',
    width: 120,
  },
  {
    title: '分组标签',
    dataIndex: 'group_by',
    key: 'group_by',
    slots: { customRender: 'group_by' },
    width: 200,
  },
  {
    title: '兜底接收者',
    dataIndex: 'receiver',
    key: 'receiver',
    width: 120,
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

// 新增相关
const isAddModalVisible = ref(false);
const addFormRef = ref();
const addForm = reactive({
  name: '',
  alert_manager_instances: [{ value: '', key: Date.now() }],
  resolve_timeout: '',
  group_wait: '',
  group_interval: '',
  repeat_interval: '',
  group_by: [{ labelKey: '', labelValue: '', key: Date.now() }],
  receiver: '',
});

const showAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

const resetAddForm = () => {
  Object.assign(addForm, {
    name: '',
    alert_manager_instances: [{ value: '', key: Date.now() }],
    resolve_timeout: '',
    group_wait: '',
    group_interval: '',
    repeat_interval: '',
    group_by: [{ labelKey: '', labelValue: '', key: Date.now() }],
    receiver: '',
  });
};

const closeAddModal = () => {
  isAddModalVisible.value = false;
};

// AlertManager实例动态表单项操作
const addAlertManagerInstance = () => {
  addForm.alert_manager_instances.push({
    value: '',
    key: Date.now()
  });
};

const removeAlertManagerInstance = (instance: any) => {
  const index = addForm.alert_manager_instances.indexOf(instance);
  if (index !== -1) {
    addForm.alert_manager_instances.splice(index, 1);
  }
};

// 标签动态表单项操作
const addLabel = () => {
  addForm.group_by.push({
    labelKey: '',
    labelValue: '',
    key: Date.now()
  });
};

const removeLabel = (label: any) => {
  const index = addForm.group_by.indexOf(label);
  if (index !== -1) {
    addForm.group_by.splice(index, 1);
  }
};

const handleReset = () => {
  searchText.value = '';
  fetchAlertManagerPools();
};

const handleAdd = async () => {
  try {
    await addFormRef.value?.validate();
    const formData = {
      ...addForm,
      alert_manager_instances: addForm.alert_manager_instances.map(item => item.value),
      group_by: addForm.group_by.map(item => `${item.labelKey},${item.labelValue}`),
    };
    await createAlertManagerPoolApi(formData);
    message.success('新增实例池成功');
    await fetchAlertManagerPools();
    closeAddModal();
  } catch (error: any) {
    message.error(error.message || '新增实例池失败');
  }
};

// 编辑相关
const isEditModalVisible = ref(false);
const editFormRef = ref();
const editForm = reactive({
  id: 0,
  name: '',
  alert_manager_instances: [{ value: '', key: Date.now() }],
  resolve_timeout: '',
  group_wait: '',
  group_interval: '',
  repeat_interval: '',
  group_by: [{ labelKey: '', labelValue: '', key: Date.now() }],
  receiver: '',
});

const showEditModal = (record: MonitorAlertPoolItem) => {
  editForm.id = record.id;
  editForm.name = record.name;
  editForm.alert_manager_instances = record.alert_manager_instances.map(value => ({
    value,
    key: Date.now()
  }));
  editForm.resolve_timeout = record.resolve_timeout;
  editForm.group_wait = record.group_wait;
  editForm.group_interval = record.group_interval;
  editForm.repeat_interval = record.repeat_interval;
  editForm.group_by = record.group_by ?
    record.group_by.map((value: string) => {
      const [labelKey, labelValue] = value.split(',');
      return {
        labelKey: labelKey || '',
        labelValue: labelValue || '',
        key: Date.now()
      };
    }) : [];
  editForm.receiver = record.receiver;
  isEditModalVisible.value = true;
};

const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 编辑模态框动态表单项操作
const addEditAlertManagerInstance = () => {
  editForm.alert_manager_instances.push({
    value: '',
    key: Date.now()
  });
};

const removeEditAlertManagerInstance = (instance: any) => {
  const index = editForm.alert_manager_instances.indexOf(instance);
  if (index !== -1) {
    editForm.alert_manager_instances.splice(index, 1);
  }
};

const addEditLabel = () => {
  editForm.group_by.push({
    labelKey: '',
    labelValue: '',
    key: Date.now()
  });
};

const removeEditLabel = (label: any) => {
  const index = editForm.group_by.indexOf(label);
  if (index !== -1) {
    editForm.group_by.splice(index, 1);
  }
};

const handleEdit = async () => {
  try {
    await editFormRef.value?.validate();
    const formData = {
      ...editForm,
      alert_manager_instances: editForm.alert_manager_instances.map(item => item.value),
      group_by: editForm.group_by.map(item => `${item.labelKey},${item.labelValue}`),
    };
    await updateAlertManagerPoolApi(formData);
    message.success('更新实例池成功');
    await fetchAlertManagerPools();
    closeEditModal();
  } catch (error: any) {
    message.error(error.message || '更新实例池失败');
  }
};

// 删除处理
const handleDelete = (record: MonitorAlertPoolItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除实例池 "${record.name}" 吗？`,
    async onOk() {
      try {
        await deleteAlertManagerPoolApi(record.id);
        message.success('实例池已删除');
        await fetchAlertManagerPools();

      } catch (error: any) {
        message.error(error.message || '删除实例池失败');
      }
    },
  });
};

// 获取数据
const fetchAlertManagerPools = async () => {
  try {
    const response = await getAlertManagerPoolListApi({
      page: current.value,
      size: pageSizeRef.value,
      search: searchText.value,
    });
    data.value = response.items;
    total.value = response.total;

  } catch (error: any) {
    message.error(error.message || '获取实例池数据失败');
  }
};

onMounted(() => {
  fetchAlertManagerPools();
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

.prometheus-tag {
  background-color: #e6f7ff;
  color: #0958d9;
  border-left: 3px solid #1890ff;
}

.alert-tag {
  background-color: #fff7e6;
  color: #d46b08;
  border-left: 3px solid #fa8c16;
}

.label-tag {
  background-color: #f6ffed;
  color: #389e0d;
  border-left: 3px solid #52c41a;
}

.empty-tag {
  background-color: #f5f5f5;
  color: #8c8c8c;
}

.label-key {
  font-weight: 600;
}

.label-separator {
  margin: 0 4px;
  color: #8c8c8c;
}

.label-value {
  color: #555;
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

.dynamic-input-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.dynamic-input {
  width: 100%;
}

.dynamic-delete-button {
  cursor: pointer;
  color: #ff4d4f;
  font-size: 18px;
  transition: all 0.3s;
}

.dynamic-delete-button:hover {
  color: #cf1322;
  transform: scale(1.1);
}

.add-dynamic-button {
  width: 100%;
  margin-top: 8px;
  background: #f5f5f5;
  border: 1px dashed #d9d9d9;
  color: #595959;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.add-dynamic-button:hover {
  color: #1890ff;
  border-color: #1890ff;
  background: #f0f7ff;
}

.label-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.label-key-input,
.label-value-input {
  flex: 1;
}

.label-separator {
  font-weight: bold;
  color: #8c8c8c;
}
</style>