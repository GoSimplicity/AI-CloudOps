<template>
  <div>
    <!-- 查询和操作工具栏 -->
    <div class="custom-toolbar">
      <div class="search-filters">
        <a-input v-model:value="searchText" placeholder="请输入AlertManager实例名称" style="width: 200px" />
        <a-button type="primary" size="middle" @click="handleSearch">
          <template #icon>
            <SearchOutlined />
          </template>
          搜索
        </a-button>
        <a-button @click="handleReset">
          <template #icon>
            <ReloadOutlined />
          </template>
          重置
        </a-button>
      </div>
      <div class="action-buttons">
        <a-button type="primary" @click="showAddModal">新增AlertManager实例池</a-button>
      </div>
    </div>

    <!-- 数据表格 -->
    <a-table :columns="columns" :data-source="data" row-key="id" :pagination="false">
      <template #alert_manager_instances="{ record }">
        <a-tag v-for="instance in record.alert_manager_instances" :key="instance">
          {{ instance }}
        </a-tag>
      </template>

      <template #group_by="{ record }">
        <template v-if="record.group_by && record.group_by.length && record.group_by[0] !== ''">
          <a-tag v-for="label in record.group_by" :key="label">
            {{ label.split(',')[0] }}: {{ label.split(',')[1] }}
          </a-tag>
        </template>
        <a-tag v-else color="default">无标签</a-tag>
      </template>

      <template #alertRules="{ record }">
        <a-tag v-for="rule in record.alertRules" :key="rule">
          {{ rule }}
        </a-tag>
      </template>

      <template #action="{ record }">
        <a-space>
          <a-tooltip title="编辑资源信息">
            <a-button type="link" @click="showEditModal(record)">
              <template #icon>
                <Icon icon="clarity:note-edit-line" style="font-size: 22px" />
              </template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="删除资源">
            <a-button type="link" danger @click="handleDelete(record)">
              <template #icon>
                <Icon icon="ant-design:delete-outlined" style="font-size: 22px" />
              </template>
            </a-button>
          </a-tooltip>
        </a-space>
      </template>
    </a-table>

    <!-- 分页器 -->
    <a-pagination v-model:current="current" v-model:pageSize="pageSizeRef" :page-size-options="pageSizeOptions"
      :total="total" show-size-changer @change="handlePageChange" @showSizeChange="handleSizeChange" class="pagination">
      <template #buildOptionText="props">
        <span v-if="props.value !== '50'">{{ props.value }}条/页</span>
        <span v-else>全部</span>
      </template>
    </a-pagination>

    <!-- 新增模态框 -->
    <a-modal title="新增AlertManager实例池" v-model:visible="isAddModalVisible" @ok="handleAdd" @cancel="closeAddModal">
      <a-form ref="addFormRef" :model="addForm" layout="vertical">
        <a-form-item label="实例池名称" name="name" :rules="[{ required: true, message: '请输入实例池名称' }]">
          <a-input v-model:value="addForm.name" placeholder="请输入实例池名称" />
        </a-form-item>

        <!-- 动态AlertManager实例表单项 -->
        <a-form-item v-for="(instance, index) in addForm.alert_manager_instances" :key="instance.key"
          :label="index === 0 ? 'AlertManager实例' : ''" :name="['alert_manager_instances', index, 'value']"
          :rules="{ required: true, message: '请输入AlertManager实例IP' }">
          <a-input v-model:value="instance.value" placeholder="请输入AlertManager实例IP"
            style="width: 60%; margin-right: 8px" />
          <MinusCircleOutlined v-if="addForm.alert_manager_instances.length > 1" class="dynamic-delete-button"
            @click="removeAlertManagerInstance(instance)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addAlertManagerInstance">
            <PlusOutlined />
            添加AlertManager实例
          </a-button>
        </a-form-item>

        <!-- 动态标签表单项 -->
        <a-form-item v-for="(label, index) in addForm.group_by" :key="label.key" :label="index === 0 ? '分组标签' : ''">
          <a-input v-model:value="label.labelKey" placeholder="标签名" style="width: 40%; margin-right: 8px" />
          <a-input v-model:value="label.labelValue" placeholder="标签值" style="width: 40%; margin-right: 8px" />
          <MinusCircleOutlined class="dynamic-delete-button" @click="removeLabel(label)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addLabel">
            <PlusOutlined />
            添加标签
          </a-button>
        </a-form-item>

        <a-form-item label="默认恢复时间" name="resolve_timeout">
          <a-input v-model:value="addForm.resolve_timeout" placeholder="例如: 5s" />
        </a-form-item>

        <a-form-item label="默认分组第一次等待时间" name="group_wait">
          <a-input v-model:value="addForm.group_wait" placeholder="例如: 5s" />
        </a-form-item>

        <a-form-item label="默认分组等待间隔" name="group_interval">
          <a-input v-model:value="addForm.group_interval" placeholder="例如: 5s" />
        </a-form-item>

        <a-form-item label="默认重复发送时间" name="repeat_interval">
          <a-input v-model:value="addForm.repeat_interval" placeholder="例如: 5s" />
        </a-form-item>

        <a-form-item label="兜底接收者" name="receiver">
          <a-input v-model:value="addForm.receiver" placeholder="请输入兜底接收者" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 编辑模态框 -->
    <a-modal title="编辑AlertManager实例池" v-model:visible="isEditModalVisible" @ok="handleEdit" @cancel="closeEditModal">
      <a-form ref="editFormRef" :model="editForm" layout="vertical">
        <a-form-item label="实例池名称" name="name" :rules="[{ required: true, message: '请输入实例池名称' }]">
          <a-input v-model:value="editForm.name" placeholder="请输入实例池名称" />
        </a-form-item>

        <!-- 动态AlertManager实例表单项 -->
        <a-form-item v-for="(instance, index) in editForm.alert_manager_instances" :key="instance.key"
          :label="index === 0 ? 'AlertManager实例' : ''" :name="['alert_manager_instances', index, 'value']"
          :rules="{ required: true, message: '请输入AlertManager实例IP' }">
          <a-input v-model:value="instance.value" placeholder="请输入AlertManager实例IP"
            style="width: 60%; margin-right: 8px" />
          <MinusCircleOutlined v-if="editForm.alert_manager_instances.length > 1" class="dynamic-delete-button"
            @click="removeEditAlertManagerInstance(instance)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addEditAlertManagerInstance">
            <PlusOutlined />
            添加AlertManager实例
          </a-button>
        </a-form-item>

        <!-- 动态标签表单项 -->
        <a-form-item v-for="(label, index) in editForm.group_by" :key="label.key" :label="index === 0 ? '分组标签' : ''">
          <a-input v-model:value="label.labelKey" placeholder="标签名" style="width: 40%; margin-right: 8px" />
          <a-input v-model:value="label.labelValue" placeholder="标签值" style="width: 40%; margin-right: 8px" />
          <MinusCircleOutlined class="dynamic-delete-button" @click="removeEditLabel(label)" />
        </a-form-item>
        <a-form-item>
          <a-button type="dashed" style="width: 60%" @click="addEditLabel">
            <PlusOutlined />
            添加标签
          </a-button>
        </a-form-item>

        <a-form-item label="默认恢复时间" name="resolve_timeout">
          <a-input v-model:value="editForm.resolve_timeout" placeholder="例如: 5s" />
        </a-form-item>

        <a-form-item label="默认分组第一次等待时间" name="group_wait">
          <a-input v-model:value="editForm.group_wait" placeholder="例如: 5s" />
        </a-form-item>

        <a-form-item label="默认分组等待间隔" name="group_interval">
          <a-input v-model:value="editForm.group_interval" placeholder="例如: 5s" />
        </a-form-item>

        <a-form-item label="默认重复发送时间" name="repeat_interval">
          <a-input v-model:value="editForm.repeat_interval" placeholder="例如: 5s" />
        </a-form-item>

        <a-form-item label="兜底接收者" name="receiver">
          <a-input v-model:value="editForm.receiver" placeholder="请输入兜底接收者" />
        </a-form-item>
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
  getAlertManagerPoolTotalApi,
} from '#/api';
import { Icon } from '@iconify/vue';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  MinusCircleOutlined
} from '@ant-design/icons-vue';
import type { MonitorAlertPoolItem } from '#/api/core/prometheus';

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
  },
  {
    title: '名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: 'AlertManager实例',
    dataIndex: 'alert_manager_instances',
    key: 'alert_manager_instances',
    slots: { customRender: 'alert_manager_instances' },
  },
  {
    title: '恢复时间',
    dataIndex: 'resolve_timeout',
    key: 'resolve_timeout',
  },
  {
    title: '分组等待时间',
    dataIndex: 'group_wait',
    key: 'group_wait',
  },
  {
    title: '分组等待间隔',
    dataIndex: 'group_interval',
    key: 'group_interval',
  },
  {
    title: '重复发送时间',
    dataIndex: 'repeat_interval',
    key: 'repeat_interval',
  },
  {
    title: '分组标签',
    dataIndex: 'group_by',
    key: 'group_by',
    slots: { customRender: 'group_by' },
  },
  {
    title: '兜底接收者',
    dataIndex: 'receiver',
    key: 'receiver',
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    key: 'create_user_name',
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
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
    const response = await getAlertManagerPoolListApi(
      current.value,
      pageSizeRef.value,
      searchText.value
    );
    data.value = response;
    total.value = await getAlertManagerPoolTotalApi();

  } catch (error: any) {
    message.error(error.message || '获取实例池数据失败');
  }
};

onMounted(() => {
  fetchAlertManagerPools();
});
</script>

<style scoped>
.custom-toolbar {
  padding: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.search-filters {
  display: flex;
  gap: 16px;
  align-items: center;
}

.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
}

.pagination {
  margin-top: 16px;
  text-align: right;
  margin-right: 12px;
}

.dynamic-delete-button {
  cursor: pointer;
  position: relative;
  top: 4px;
  font-size: 24px;
  color: #999;
  transition: all 0.3s;
}

.dynamic-delete-button:hover {
  color: #777;
}

.dynamic-delete-button[disabled] {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
