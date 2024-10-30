<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model="searchText"
          placeholder="请输入AlertManager实例名称"
          style="width: 200px; margin-right: 16px;"
        />
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="showAddModal">新增AlertManager实例池</a-button>
      </div>
    </div>

    <!-- AlertManager 实例池列表表格 -->
    <a-table :columns="columns" :data-source="filteredData" row-key="key">
      <!-- AlertManager实例列 -->
      <template #alertManagerInstances="{ record }">
        <a-tag v-for="instance in record.alertManagerInstances" :key="instance">{{ instance }}</a-tag>
      </template>
      <!-- 分组标签列 -->
      <template #groupBy="{ record }">
        <a-tag v-for="group in record.groupBy" :key="group">{{ group }}</a-tag>
      </template>
      <!-- 告警规则列 -->
      <template #alertRules="{ record }">
        <a-tag v-for="rule in record.alertRules" :key="rule">{{ rule }}</a-tag>
      </template>
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="link" @click="showEditModal(record)">编辑实例池</a-button>
          <a-button type="link" danger @click="handleDelete(record)">删除实例池</a-button>
        </a-space>
      </template>
    </a-table>

    <!-- 新增AlertManager实例池模态框 -->
    <a-modal
      title="新增AlertManager实例池"
      v-model:visible="isAddModalVisible"
      @ok="handleAdd"
      @cancel="closeAddModal"
    >
      <a-form :model="addForm" layout="vertical">
        <a-form-item label="实例池名称" name="name" :rules="[{ required: true, message: '请输入实例池名称' }]">
          <a-input v-model:value="addForm.name" placeholder="请输入实例池名称" />
        </a-form-item>
        <a-form-item label="AlertManager实例" name="alertManagerInstances" :rules="[{ required: true, message: '请输入至少一个AlertManager实例' }]">
          <a-select mode="tags" v-model:value="addForm.alertManagerInstances" placeholder="请输入AlertManager实例地址">
          </a-select>
        </a-form-item>
        <a-form-item label="默认恢复时间" name="resolveTimeout">
          <a-input v-model:value="addForm.resolveTimeout" placeholder="例如: 5s" />
        </a-form-item>
        <a-form-item label="默认分组第一次等待时间" name="groupWait">
          <a-input v-model:value="addForm.groupWait" placeholder="例如: 5s" />
        </a-form-item>
        <a-form-item label="默认分组等待间隔" name="groupInterval">
          <a-input v-model:value="addForm.groupInterval" placeholder="例如: 5s" />
        </a-form-item>
        <a-form-item label="默认重复发送时间" name="repeatInterval">
          <a-input v-model:value="addForm.repeatInterval" placeholder="例如: 5s" />
        </a-form-item>
        <a-form-item label="分组标签" name="groupBy">
          <a-select mode="tags" v-model:value="addForm.groupBy" placeholder="请输入分组标签">
          </a-select>
        </a-form-item>
        <a-form-item label="兜底接收者" name="receiver">
          <a-input v-model:value="addForm.receiver" placeholder="请输入兜底接收者" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 编辑AlertManager实例池模态框 -->
    <a-modal
      title="编辑AlertManager实例池"
      v-model:visible="isEditModalVisible"
      @ok="handleEdit"
      @cancel="closeEditModal"
    >
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="实例池名称" name="name" :rules="[{ required: true, message: '请输入实例池名称' }]">
          <a-input v-model:value="editForm.name" placeholder="请输入实例池名称" />
        </a-form-item>
        <a-form-item label="AlertManager实例" name="alertManagerInstances" :rules="[{ required: true, message: '请输入至少一个AlertManager实例' }]">
          <a-select mode="tags" v-model:value="editForm.alertManagerInstances" placeholder="请输入AlertManager实例地址">
          </a-select>
        </a-form-item>
        <a-form-item label="默认恢复时间" name="resolveTimeout">
          <a-input v-model:value="editForm.resolveTimeout" placeholder="例如: 5s" />
        </a-form-item>
        <a-form-item label="默认分组第一次等待时间" name="groupWait">
          <a-input v-model:value="editForm.groupWait" placeholder="例如: 5s" />
        </a-form-item>
        <a-form-item label="默认分组等待间隔" name="groupInterval">
          <a-input v-model:value="editForm.groupInterval" placeholder="例如: 5s" />
        </a-form-item>
        <a-form-item label="默认重复发送时间" name="repeatInterval">
          <a-input v-model:value="editForm.repeatInterval" placeholder="例如: 5s" />
        </a-form-item>
        <a-form-item label="分组标签" name="groupBy">
          <a-select mode="tags" v-model:value="editForm.groupBy" placeholder="请输入分组标签">
          </a-select>
        </a-form-item>
        <a-form-item label="兜底接收者" name="receiver">
          <a-input v-model:value="editForm.receiver" placeholder="请输入兜底接收者" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, computed, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  getAlertManagerPoolsApi, 
  createAlertManagerPoolApi, 
  updateAlertManagerPoolApi, 
  deleteAlertManagerPoolApi,
} from '#/api'; 

// 定义数据类型
interface AlertManagerPool {
  ID: number;
  name: string; 
  alertManagerInstances: string[]; 
  resolveTimeout: string; 
  groupWait: string; 
  groupInterval: string; 
  repeatInterval: string;
  groupBy: string[];
  receiver: string; 
  userId: number;
  CreatedAt: string; 
}

// 数据源
const data = ref<AlertManagerPool[]>([]);

// 搜索文本
const searchText = ref('');

// 过滤后的数据
const filteredData = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  return data.value.filter(item => item.name.toLowerCase().includes(searchValue));
});

// 表格列配置
const columns = [
{
    title: 'ID',
    dataIndex: 'ID',
    key: 'ID',
  },
  {
    title: '实例池名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: 'AlertManager实例',
    dataIndex: 'alertManagerInstances',
    key: 'alertManagerInstances',
    slots: { customRender: 'alertManagerInstances' }, // 使用自定义插槽来渲染 AlertManager 实例
  },
  {
    title: '默认恢复时间',
    dataIndex: 'resolveTimeout',
    key: 'resolveTimeout',
  },
  {
    title: '默认分组第一次等待时间',
    dataIndex: 'groupWait',
    key: 'groupWait',
  },
  {
    title: '默认分组等待间隔',
    dataIndex: 'groupInterval',
    key: 'groupInterval',
  },
  {
    title: '默认重复发送时间',
    dataIndex: 'repeatInterval',
    key: 'repeatInterval',
  },
  {
    title: '分组标签',
    dataIndex: 'groupBy',
    key: 'groupBy',
    slots: { customRender: 'groupBy' }, // 使用自定义插槽来渲染分组标签
  },
  {
    title: '兜底接收者',
    dataIndex: 'receiver',
    key: 'receiver',
  },
  {
    title: '创建者',
    dataIndex: 'userId',
    key: 'userId',
  },
  {
    title: '创建时间',
    dataIndex: 'CreatedAt',
    key: 'CreatedAt',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' }, // 使用自定义插槽来渲染操作按钮
  },
];

// 新增模态框状态和表单
const isAddModalVisible = ref(false);
const addForm = reactive({
  name: '',
  alertManagerInstances: [] as string[],
  resolveTimeout: '5s',
  groupWait: '5s',
  groupInterval: '5s',
  repeatInterval: '5s',
  groupBy: [] as string[],
  receiver: 'admin',
});

// 编辑模态框状态和表单
const isEditModalVisible = ref(false);
const editForm = reactive({
  ID: 0,
  key: '',
  name: '',
  alertManagerInstances: [] as string[],
  resolveTimeout: '5s',
  groupWait: '5s',
  groupInterval: '5s',
  repeatInterval: '5s',
  groupBy: [] as string[],
  receiver: 'admin',
});

// 显示新增模态框
const showAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

// 重置新增表单
const resetAddForm = () => {
  addForm.name = '';
  addForm.alertManagerInstances = [];
  addForm.resolveTimeout = '5s';
  addForm.groupWait = '5s';
  addForm.groupInterval = '5s';
  addForm.repeatInterval = '5s';
  addForm.groupBy = [];
  addForm.receiver = 'admin';
};

// 关闭新增模态框
const closeAddModal = () => {
  isAddModalVisible.value = false;
};

// 提交新增AlertManager实例池
const handleAdd = async () => {
  try {
    const payload = {
      name: addForm.name,
      alertManagerInstances: addForm.alertManagerInstances,
      resolveTimeout: addForm.resolveTimeout,
      groupWait: addForm.groupWait,
      groupInterval: addForm.groupInterval,
      repeatInterval: addForm.repeatInterval,
      groupBy: addForm.groupBy,
      receiver: addForm.receiver,
    };
    await createAlertManagerPoolApi(payload); // 调用创建 API
    message.success('新增实例池成功');
    fetchAlertManagerPools(); // 重新获取数据
    closeAddModal();
  } catch (error) {
    message.error('新增实例池失败，请稍后重试');
    console.error(error);
  }
};

// 显示编辑模态框
const showEditModal = (record: AlertManagerPool) => {
  // 预填充表单数据
  editForm.ID = record.ID;
  editForm.name = record.name;
  editForm.alertManagerInstances = record.alertManagerInstances;
  editForm.resolveTimeout = record.resolveTimeout;
  editForm.groupWait = record.groupWait;
  editForm.groupInterval = record.groupInterval;
  editForm.repeatInterval = record.repeatInterval;
  editForm.groupBy = record.groupBy;
  editForm.receiver = record.receiver;
  console.log("record:::::", record.ID)
  isEditModalVisible.value = true;
};

// 关闭编辑模态框
const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 提交更新AlertManager实例池
const handleEdit = async () => {
  try {
    const payload = {
      ID: editForm.ID,
      name: editForm.name,
      alertManagerInstances: editForm.alertManagerInstances,
      resolveTimeout: editForm.resolveTimeout,
      groupWait: editForm.groupWait,
      groupInterval: editForm.groupInterval,
      repeatInterval: editForm.repeatInterval,
      groupBy: editForm.groupBy,
      receiver: editForm.receiver,
    };
    await updateAlertManagerPoolApi(payload); // 调用更新 API
    message.success('更新实例池成功');
    fetchAlertManagerPools(); // 重新获取数据
    closeEditModal();
  } catch (error) {
    message.error('更新实例池失败，请稍后重试');
    console.error(error);
  }
};

// 处理删除实例池
const handleDelete = (record: AlertManagerPool) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除实例池 "${record.name}" 吗？`,
    onOk: async () => {
      try {
        await deleteAlertManagerPoolApi(record.ID); // 调用删除 API
        message.success('实例池已删除');
        fetchAlertManagerPools(); // 重新获取数据
      } catch (error) {
        message.error('删除实例池失败，请稍后重试');
        console.error(error);
      }
    },
  });
};

// 获取AlertManager实例池数据
const fetchAlertManagerPools = async () => {
  try {
    const response = await getAlertManagerPoolsApi(); // 调用获取数据 API
    data.value = response;
  } catch (error) {
    message.error('获取实例池数据失败，请稍后重试');
    console.error(error);
  }
};

// 在组件加载时获取数据
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
}

.search-filters {
  display: flex;
  align-items: center;
}
</style>
