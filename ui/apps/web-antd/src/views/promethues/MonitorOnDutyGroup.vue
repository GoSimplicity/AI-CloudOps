<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">值班组管理</h2>
      <div class="page-description">管理和监控值班组及其相关配置</div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card custom-toolbar">
      <div class="search-filters">
        <a-input 
          v-model:value="searchText" 
          placeholder="请输入值班组名称" 
          class="search-input"
          allow-clear
          @pressEnter="handleSearch"
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
          新增值班组
        </a-button>
      </div>
    </div>

    <!-- 值班组列表表格 -->
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
        <!-- 用户名称列自定义渲染 -->
        <template #user_names="{ text }">
          <div class="tag-container">
            <a-tag v-for="name in text" :key="name" class="tech-tag prometheus-tag">
              {{ name }}
            </a-tag>
          </div>
        </template>

        <!-- 创建时间列格式化 -->
        <template #created_at="{ text }">
          {{ formatDate(text) }}
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
            <a-tooltip title="查看排班表">
              <a-button type="primary" shape="circle" class="view-button" @click="viewSchedule(record)">
                <template #icon>
                  <Icon icon="clarity:eye-line" />
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

    <!-- 新增值班组模态框 -->
    <a-modal 
      title="新增值班组" 
      v-model:visible="isAddModalVisible" 
      @ok="handleAdd" 
      @cancel="closeAddModal"
      :confirmLoading="loading"
      :maskClosable="false"
      :width="700"
      class="custom-modal"
    >
      <a-form :model="addForm" layout="vertical" ref="addFormRef" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item 
                label="值班组名称" 
                name="name"
                :rules="[{ required: true, message: '请输入值班组名称' }]"
              >
                <a-input
                  v-model:value="addForm.name"
                  placeholder="请输入值班组名称"
                  :maxLength="50"
                />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item 
                label="轮班周期（天）"
                name="shiftDays"
                :rules="[{ required: true, message: '请输入轮班周期' }]"
              >
                <a-input-number
                  v-model:value="addForm.shiftDays"
                  :min="1"
                  :max="365"
                  class="full-width"
                />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">值班人员</div>
          <a-row :gutter="16">
            <a-col :span="24">
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
            </a-col>
          </a-row>
        </div>
      </a-form>
    </a-modal>

    <!-- 编辑值班组模态框 -->
    <a-modal 
      title="编辑值班组" 
      v-model:visible="isEditModalVisible" 
      @ok="handleUpdate" 
      @cancel="closeEditModal"
      :confirmLoading="loading"
      :maskClosable="false"
      :width="700"
      class="custom-modal"
    >
      <a-form :model="editForm" layout="vertical" ref="editFormRef" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item 
                label="值班组名称" 
                name="name"
                :rules="[{ required: true, message: '请输入值班组名称' }]"
              >
                <a-input
                  v-model:value="editForm.name"
                  placeholder="请输入值班组名称"
                  :maxLength="50"
                />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item 
                label="轮班周期（天）"
                name="shiftDays"
                :rules="[{ required: true, message: '请输入轮班周期' }]"
              >
                <a-input-number
                  v-model:value="editForm.shiftDays"
                  :min="1"
                  :max="365"
                  class="full-width"
                />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">值班人员</div>
          <a-row :gutter="16">
            <a-col :span="24">
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
  getOnDutyListApi,
  createOnDutyApi,
  updateOnDutyApi,
  deleteOnDutyApi,
  getOnDutyTotalApi,
} from '#/api/core/prometheus_onduty';
import { getUserList } from '#/api/core/user';
import { Icon } from '@iconify/vue';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined
} from '@ant-design/icons-vue';
import type { OnDutyGroupItem } from '#/api/core/prometheus_onduty';
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
  current.value = 1;
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
    const response = await getUserList({
      page: 1,
      size: 100,
      search: ''
    });
    availableUsers.value = response.items.map((user: any) => user.username);
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
    const response = await getOnDutyListApi({
      page: current.value,
      size: pageSizeRef.value,
      search: searchText.value.trim(),
    });
    data.value = response.items;
    total.value = response.total;

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

.view-button {
  background: #52c41a;
  border: none;
  box-shadow: 0 2px 4px rgba(82, 196, 26, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
}

.view-button:hover {
  background: #389e0d;
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
</style>