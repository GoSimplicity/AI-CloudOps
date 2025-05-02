<template>
  <div>
    <!-- 查询和操作工具栏 -->
    <div class="custom-toolbar">
      <div class="search-filters">
        <a-input
          v-model:value="searchText"
          placeholder="请输入值班组名称"
          style="width: 200px"
          allow-clear
          @pressEnter="handleSearch"
        />
        <a-button type="primary" @click="handleSearch">
          <template #icon><SearchOutlined /></template>
          搜索
        </a-button>
        <a-button @click="handleReset">
          <template #icon><ReloadOutlined /></template>
          重置
        </a-button>
      </div>
      <div class="action-buttons">
        <a-button type="primary" @click="showAddModal">新增值班组</a-button>
      </div>
    </div>

    <!-- 值班组记录列表表格 -->
    <a-table
      :columns="columns"
      :data-source="data"
      row-key="id"
      :loading="loading"
      :pagination="false"
    >
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-tooltip title="编辑资源信息">
            <a-button type="link" @click="showEditModal(record)">
              <template #icon><Icon icon="clarity:note-edit-line" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="查看排班表">
            <a-button type="link" @click="viewSchedule(record)">
              <template #icon><Icon icon="clarity:eye-line" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="删除资源">
            <a-button type="link" danger @click="handleDelete(record)">
              <template #icon><Icon icon="ant-design:delete-outlined" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
        </a-space>
      </template>

      <!-- 用户名称列自定义渲染 -->
      <template #user_names="{ text }">
        <a-tag v-for="name in text" :key="name" color="blue">{{ name }}</a-tag>
      </template>

      <!-- 创建时间列格式化 -->
      <template #created_at="{ text }">
        {{ formatDate(text) }}
      </template>
    </a-table>

        <!-- 分页器 -->
        <a-pagination
      v-model:current="current"
      v-model:pageSize="pageSizeRef"
      :page-size-options="pageSizeOptions"
      :total="total"
      show-size-changer
      @change="handlePageChange"
      @showSizeChange="handleSizeChange"
      class="pagination"
    >
      <template #buildOptionText="props">
        <span v-if="props.value !== '50'">{{ props.value }}条/页</span>
        <span v-else>全部</span>
      </template>
    </a-pagination>

    <!-- 新增值班组模态框 -->
    <a-modal
      title="新增值班组"
      v-model:visible="isAddModalVisible"
      @ok="handleAdd"
      @cancel="closeAddModal"
      :confirmLoading="loading"
      :maskClosable="false"
    >
      <a-form :model="addForm" layout="vertical" ref="addFormRef">
        <a-form-item 
          label="名称" 
          name="name"
          :rules="[{ required: true, message: '请输入值班组名称' }]"
        >
          <a-input
            v-model:value="addForm.name"
            placeholder="请输入值班组名称"
            :maxLength="50"
          />
        </a-form-item>

        <a-form-item 
          label="轮班周期（天）"
          name="shiftDays"
          :rules="[{ required: true, message: '请输入轮班周期' }]"
        >
          <a-input-number
            v-model:value="addForm.shiftDays"
            :min="1"
            :max="365"
            style="width: 100%"
          />
        </a-form-item>

        <a-form-item 
          label="用户名称" 
          name="userNames"
          :rules="[{ required: true, message: '请选择至少一个用户' }]"
        >
          <a-select
            mode="multiple"
            v-model:value="addForm.userNames"
            placeholder="请选择用户"
            style="width: 100%"
            :maxTagCount="3"
            :filterOption="filterOption"
          >
            <a-select-option
              v-for="user in availableUsers"
              :key="user"
              :value="user"
            >
              {{ user }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 编辑值班组记录模态框 -->
    <a-modal
      title="编辑值班组"
      v-model:visible="isEditModalVisible"
      @ok="handleUpdate"
      @cancel="closeEditModal"
      :confirmLoading="loading"
      :maskClosable="false"
    >
      <a-form :model="editForm" layout="vertical" ref="editFormRef">
        <a-form-item 
          label="名称" 
          name="name"
          :rules="[{ required: true, message: '请输入值班组名称' }]"
        >
          <a-input
            v-model:value="editForm.name"
            placeholder="请输入值班组名称"
            :maxLength="50"
          />
        </a-form-item>

        <a-form-item 
          label="轮班周期（天）"
          name="shiftDays"
          :rules="[{ required: true, message: '请输入轮班周期' }]"
        >
          <a-input-number
            v-model:value="editForm.shiftDays"
            :min="1"
            :max="365"
            style="width: 100%"
          />
        </a-form-item>

        <a-form-item 
          label="用户名称" 
          name="userNames"
          :rules="[{ required: true, message: '请选择至少一个用户' }]"
        >
          <a-select
            mode="multiple"
            v-model:value="editForm.userNames"
            placeholder="请选择用户"
            style="width: 100%"
            :maxTagCount="3"
            :filterOption="filterOption"
          >
            <a-select-option
              v-for="user in availableUsers"
              :key="user"
              :value="user"
            >
              {{ user }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  getOnDutyListApi,
  createOnDutyApi,
  updateOnDutyApi,
  deleteOnDutyApi,
  getOnDutyTotalApi,
  getUserList
} from '#/api';
import { Icon } from '@iconify/vue';
import {
  SearchOutlined,
  ReloadOutlined,
} from '@ant-design/icons-vue';
import type { OnDutyGroupItem } from '#/api';
import { useRouter } from 'vue-router';
import dayjs from 'dayjs';

const router = useRouter();
const data = ref<OnDutyGroupItem[]>([]);
const searchText = ref('');
const loading = ref(false);
const addFormRef = ref();
const editFormRef = ref();

// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);
const handleReset = () => {
  searchText.value = '';
  fetchOnDutyGroups();
};

// 处理搜索
const handleSearch = () => {
  fetchOnDutyGroups();
};

const handleSizeChange = (page: number, size: number) => {
  current.value = page;
  pageSizeRef.value = size;
  fetchOnDutyGroups();
};

// 处理分页变化
const handlePageChange = (page: number) => {
  current.value = page;
  fetchOnDutyGroups();
};


// 表格列配置
const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    width: 80,
  },
  {
    title: '值班组名称',
    dataIndex: 'name',
    ellipsis: true,
  },
  {
    title: '轮班周期（天）',
    dataIndex: 'shift_days',
    width: 120,
  },
  {
    title: '成员',
    dataIndex: 'user_names',
    slots: { customRender: 'user_names' },
  },
  {
    title: '创建者',
    dataIndex: 'create_user_name',
    width: 100,
  },
  {
    title: '昨日值班用户',
    dataIndex: 'yesterday_normal_duty_user_id',
    width: 100,
  },
  {
    title: '今日值班用户',
    dataIndex: 'today_duty_user',
    width: 100,
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    slots: { customRender: 'created_at' },
    width: 160,
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    width: 200,
    fixed: 'right',
  },
];

const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);

const addForm = reactive({
  name: '',
  shiftDays: 7,
  userNames: [] as string[],
});

const editForm = reactive({
  id: 0,
  name: '',
  shiftDays: 7,
  userNames: [] as string[],
});

const availableUsers = ref<string[]>([]);

// Select 筛选方法
const filterOption = (input: string, option: any) => {
  return option.value.toLowerCase().indexOf(input.toLowerCase()) >= 0;
};

// 格式化日期
const formatDate = (date: string) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss');
};

const showAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = true;
};

const resetAddForm = () => {
  if (addFormRef.value) {
    addFormRef.value.resetFields();
  }
  addForm.name = '';
  addForm.shiftDays = 7;
  addForm.userNames = [];
};

const closeAddModal = () => {
  resetAddForm();
  isAddModalVisible.value = false;
};

const showEditModal = (record: OnDutyGroupItem) => {
  editForm.id = record.id;
  editForm.name = record.name;
  editForm.shiftDays = record.shift_days;
  editForm.userNames = [...record.user_names];
  isEditModalVisible.value = true;
};

const closeEditModal = () => {
  if (editFormRef.value) {
    editFormRef.value.resetFields();
  }
  isEditModalVisible.value = false;
};

const handleAdd = async () => {
  try {
    await addFormRef.value.validate();
    loading.value = true;

    const payload = {
      name: addForm.name.trim(),
      shift_days: addForm.shiftDays,
      user_names: addForm.userNames,
    };

    await createOnDutyApi(payload);
    message.success('新增值班组成功');
    await fetchOnDutyGroups();
    closeAddModal();
  } catch (error: any) {
    console.error('新增值班组失败:', error);
    message.error(error.message || '新增值班组失败');
  } finally {
    loading.value = false;
  }
};

const handleUpdate = async () => {
  try {
    await editFormRef.value.validate();
    loading.value = true;

    const payload = {
      id: editForm.id,
      name: editForm.name.trim(),
      shift_days: editForm.shiftDays,
      user_names: editForm.userNames,
    };

    await updateOnDutyApi(payload);
    message.success('更新值班组成功');
    await fetchOnDutyGroups();
    closeEditModal();
  } catch (error: any) {
    console.error('更新值班组失败:', error);
    message.error(error.message || '更新值班组失败');
  } finally {
    loading.value = false;
  }
};

const handleDelete = (record: OnDutyGroupItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除值班组"${record.name}"吗？此操作不可恢复。`,
    okText: '确认',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      try {
        loading.value = true;
        await deleteOnDutyApi(record.id);
        message.success('删除值班组成功');
        await fetchOnDutyGroups();

      } catch (error: any) {
        console.error('删除值班组失败:', error);
        message.error(error.message || '删除值班组失败');
      } finally {
        loading.value = false;
      }
    },
  });
};

const fetchUserList = async () => {
  try {
    loading.value = true;
    const response = await getUserList();
    availableUsers.value = response.map((user: any) => user.username);
  } catch (error: any) {
    console.error('获取用户列表失败:', error);
    message.error(error.message || '获取用户列表失败');
  } finally {
    loading.value = false;
  }
};

const viewSchedule = (record: OnDutyGroupItem) => {
  router.push({
    name: 'MonitorOnDutyGroupTable',
    query: { id: record.id.toString() }
  });
};

const fetchOnDutyGroups = async () => {
  try {
    loading.value = true;
    const response = await getOnDutyListApi(current.value, pageSizeRef.value, searchText.value.trim());
    data.value = response;
    total.value = await getOnDutyTotalApi();

  } catch (error: any) {
    console.error('获取值班组列表失败:', error);
    message.error(error.message || '获取值班组列表失败');
  } finally {
    loading.value = false;
  }
};

onMounted(async () => {
  await Promise.all([
    fetchOnDutyGroups(),
    fetchUserList()
  ]);
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
