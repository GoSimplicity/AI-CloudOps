<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">发送组管理</h2>
      <div class="page-description">管理和配置告警发送组及其相关设置</div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card custom-toolbar">
      <div class="search-filters">
        <a-input 
          v-model:value="searchText" 
          placeholder="请输入发送组名称" 
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
          新增发送组
        </a-button>
      </div>
    </div>

    <!-- 发送组列表表格 -->
    <div class="dashboard-card table-container">
      <a-spin :spinning="loading">
        <a-table 
          :columns="columns" 
          :data-source="data"
          row-key="id" 
          :pagination="false"
          class="custom-table"
          :scroll="{ x: 1200 }"
        >
          <template #enable="{ record }">
            {{ record.enable ? '启用' : '禁用' }}
          </template>
          <template #sendResolved="{ record }">
            {{ record.send_resolved ? '是' : '否' }}
          </template>
          <template #upgradeUsers="{ record }">
            {{ record.first_user_names.join(', ') }}
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
      </a-spin>

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

    <!-- 新增/编辑模态框 -->
    <a-modal
      v-model:visible="isModalVisible"
      :title="form.id === 0 ? '新增发送组' : '编辑发送组'"
      @ok="handleSubmit"
      @cancel="resetForm"
      :width="700"
      class="custom-modal"
    >
      <a-form ref="formRef" :model="form" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="发送组名称" name="name" :rules="[{ required: true, message: '请输入发送组名称' }]">
                <a-input v-model:value="form.name" placeholder="请输入发送组名称" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="发送组中文名称" name="name_zh" :rules="[{ required: true, message: '请输入发送组中文名称' }]">
                <a-input v-model:value="form.name_zh" placeholder="请输入发送组中文名称" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>
        
        <div class="form-section">
          <div class="section-title">关联配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="关联采集池" name="pool_id">
                <a-select v-model:value="form.pool_id" placeholder="请选择采集池" class="full-width">
                  <a-select-option v-for="pool in scrapePools" :key="pool.id" :value="pool.id">
                    {{ pool.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="关联值班组" name="on_duty_group_id">
                <a-select v-model:value="form.on_duty_group_id" placeholder="请选择值班组" class="full-width">
                  <a-select-option v-for="group in onDutyGroups" :key="group.id" :value="group.id">
                    {{ group.name }}
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
        </div>
        
        <div class="form-section">
          <div class="section-title">告警配置</div>
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="是否启用" name="enable">
                <a-switch v-model:checked="form.enable" class="tech-switch" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="是否发送恢复消息" name="send_resolved">
                <a-switch v-model:checked="form.send_resolved" class="tech-switch" />
              </a-form-item>
            </a-col>
          </a-row>
          
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12">
              <a-form-item label="重复发送时间" name="repeat_interval">
                <a-input v-model:value="form.repeat_interval" placeholder="默认30s" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-form-item label="需要升级" name="need_upgrade">
                <a-switch v-model:checked="form.need_upgrade" class="tech-switch" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>
        
        <div class="form-section">
          <div class="section-title">通知配置</div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="飞书群机器人 Token" name="fei_shu_qun_robot_token">
                <a-input v-model:value="form.fei_shu_qun_robot_token" placeholder="请输入飞书群机器人Token" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { SearchOutlined, ReloadOutlined, PlusOutlined} from '@ant-design/icons-vue';
import { Icon } from '@iconify/vue';
import {
  getMonitorSendGroupListApi,
  createMonitorSendGroupApi,
  updateMonitorSendGroupApi,
  deleteMonitorSendGroupApi,
} from '#/api/core/prometheus_send_group';
import { getUserList } from '#/api/core/user';
import { getAllMonitorScrapePoolApi } from '#/api/core/prometheus_scrape_pool';
import { getAllOnDutyGroupApi } from '#/api/core/prometheus_onduty';
import type { SendGroupItem } from '#/api/core/prometheus_send_group';
import type { MonitorScrapePoolItem } from '#/api/core/prometheus_scrape_pool';
import type { OnDutyGroupItem } from '#/api/core/prometheus_onduty';
import type { FormInstance } from 'ant-design-vue';

// 分页相关
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const loading = ref(false);
const formRef = ref<FormInstance>();

const columns = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
    width: 80,
  },
  {
    title: '发送组名称',
    dataIndex: 'name',
    key: 'name',
    width: 150,
  },
  {
    title: '发送组中文名称',
    dataIndex: 'name_zh',
    key: 'name_zh',
    width: 150,
  },
  {
    title: '关联采集池ID',
    dataIndex: 'pool_id',
    key: 'pool_id',
    width: 120,
  },
  {
    title: '关联值班组ID',
    dataIndex: 'on_duty_group_id',
    key: 'on_duty_group_id',
    width: 120,
  },
  {
    title: '是否启用',
    dataIndex: 'enable',
    key: 'enable',
    width: 100,
    customRender: ({ record }: { record: SendGroupItem }) =>
      record.enable ? '启用' : '禁用',
  },
  {
    title: '重复发送时间',
    dataIndex: 'repeat_interval',
    key: 'repeat_interval',
    width: 120,
  },
  {
    title: '是否发送恢复消息',
    dataIndex: 'send_resolved',
    key: 'send_resolved',
    width: 150,
    customRender: ({ record }: { record: SendGroupItem }) =>
      record.send_resolved ? '是' : '否',
  },
  {
    title: '需要升级',
    dataIndex: 'need_upgrade',
    key: 'need_upgrade',
    width: 100,
    customRender: ({ record }: { record: SendGroupItem }) =>
      record.need_upgrade ? '需要' : '不需要',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    fixed: 'right',
    width: 120,
  },
];

const data = reactive<SendGroupItem[]>([]); 
const searchText = ref('');
const scrapePools = ref<MonitorScrapePoolItem[]>([]);
const onDutyGroups = ref<OnDutyGroupItem[]>([]);
const isModalVisible = ref(false);
const form = reactive({
  id: 0,
  name: '',
  name_zh: '',
  enable: false,
  pool_id: null,
  on_duty_group_id: null,
  static_receive_users: [] as any[],
  fei_shu_qun_robot_token: '',
  repeat_interval: '',
  send_resolved: false,
  notify_methods: [] as string[],
  need_upgrade: false,
  first_upgrade_users: [] as any[],
  upgrade_minutes: 0,
  second_upgrade_users: [] as any[],
});

const userOptions = ref([]);

// 搜索功能
const handleSearch = () => {
  current.value = 1;
  fetchSendGroups();
};

const handleReset = () => {
  searchText.value = '';
  current.value = 1;
  fetchSendGroups();
};

// 分页功能
const handlePageChange = (page: number) => {
  current.value = page;
  fetchSendGroups();
};

const handleSizeChange = (page: number, size: number) => {
  current.value = page;
  pageSizeRef.value = size;
  fetchSendGroups();
};

// 获取用户列表
const fetchUsers = async () => {
  try {
    const response = await getUserList({
      page: 1,
      size: 100,
      search: ''
    });
    userOptions.value = response.items.map((user: { username: any; id: any }) => ({
      label: user.username,
      value: user.id,
    }));
  } catch (error: any) {
    message.error(error.message || '获取用户列表失败');
    console.error(error);
  }
};

const showAddModal = () => {
  resetForm();
  form.repeat_interval = '30s';
  isModalVisible.value = true;
};

const showEditModal = (record: SendGroupItem) => {
  Object.assign(form, {
    ...record,
    enable: record.enable,
    send_resolved: record.send_resolved,
    need_upgrade: record.need_upgrade,
    static_receive_users: record.static_receive_users || [],
    notify_methods: record.notify_methods || [],
    first_upgrade_users: record.first_upgrade_users || [],
    second_upgrade_users: record.second_upgrade_users || [],
  });

  isModalVisible.value = true;
};

const resetForm = () => {
  Object.assign(form, {
    id: 0,
    name: '',
    name_zh: '',
    enable: false,
    pool_id: null,
    on_duty_group_id: null,
    repeat_interval: '30s',
    send_resolved: false,
    need_upgrade: false,
    fei_shu_qun_robot_token: '',
    static_receive_users: [],
    notify_methods: [],
    first_upgrade_users: [],
    second_upgrade_users: [],
    upgrade_minutes: 0,
  });
  isModalVisible.value = false;
};

const handleSubmit = async () => {
  try {
    const submitData = {
      id: form.id,
      name: form.name,
      name_zh: form.name_zh,
      enable: form.enable,
      pool_id: form.pool_id,
      send_resolved: form.send_resolved,
      on_duty_group_id: form.on_duty_group_id,
      need_upgrade: form.need_upgrade,
      repeat_interval: form.repeat_interval,
      fei_shu_qun_robot_token: form.fei_shu_qun_robot_token,
      static_receive_users: form.static_receive_users,
      notify_methods: form.notify_methods,
      first_upgrade_users: form.first_upgrade_users,
      upgrade_minutes: form.upgrade_minutes,
      second_upgrade_users: form.second_upgrade_users,
    };

    if (form.id === 0) {
      await createMonitorSendGroupApi(submitData as any);
      message.success('新增发送组成功');
    } else {
      await updateMonitorSendGroupApi(submitData as any);
      message.success('编辑发送组成功');
    }

    resetForm();
    fetchSendGroups();

  } catch (error: any) {
    message.error(error.message || '提交失败，请重试');
    console.error(error);
  }
};

const handleDelete = (record: SendGroupItem) => {
  Modal.confirm({
    title: '确认删除',
    content: `您确定要删除发送组 "${record.name_zh}" 吗？`,
    onOk: async () => {
      try {
        await deleteMonitorSendGroupApi(record.id);
        message.success('发送组已删除');
        fetchSendGroups();

      } catch (error: any) {
        message.error(error.message || '删除失败，请重试');
        console.error(error);
      }
    },
  });
};

// 获取发送组列表
const fetchSendGroups = async () => {
  loading.value = true;
  try {
    const response = await getMonitorSendGroupListApi({
      page: current.value,
      size: pageSizeRef.value,
      search: searchText.value,
    });
    data.splice(0, data.length, ...response.items);
    total.value = response.total;
  } catch (error: any) {
    message.error(error.message || '获取发送组数据失败');
    console.error(error);
  } finally {
    loading.value = false;
  }
};

// 获取采集池列表
const fetchScrapePools = async () => {
  try {
    const response = await getAllMonitorScrapePoolApi();
    scrapePools.value = response.items;
  } catch (error: any) {
    message.error(error.message || '获取采集池列表失败');
    console.error(error);
  }
};

// 获取值班组列表 
const fetchOnDutyGroups = async () => {
  try {
    const response = await getAllOnDutyGroupApi();
    onDutyGroups.value = response.items;
  } catch (error: any) {
    message.error(error.message || '获取值班组列表失败');
    console.error(error); 
  }
};

onMounted(() => {
  fetchUsers();
  fetchSendGroups();
  fetchScrapePools();
  fetchOnDutyGroups();
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
</style>