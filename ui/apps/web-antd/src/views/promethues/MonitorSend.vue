<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input 
          v-model:value="searchText" 
          placeholder="请输入发送组名称" 
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
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="showAddModal">
          <template #icon><PlusOutlined /></template>
          新增发送组
        </a-button>
      </div>
    </div>

    <!-- 数据加载状态 -->
    <a-spin :spinning="loading">
      <!-- 发送组列表表格 -->
      <a-table 
        :columns="columns" 
        :data-source="data"
        :pagination="false"
        row-key="id"
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
        <a-space>
          <a-tooltip title="编辑资源信息">
            <a-button type="link" @click="showEditModal(record)">
              <template #icon><Icon icon="clarity:note-edit-line" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="删除资源">
            <a-button type="link" danger @click="handleDelete(record)">
              <template #icon><Icon icon="ant-design:delete-outlined" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
        </a-space>
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
    </a-spin>

    <!-- 新增/编辑模态框 -->
    <a-modal
      v-model:visible="isModalVisible"
      title="发送组"
      @ok="handleSubmit"
      @cancel="resetForm"
    >
      <a-form :model="form" :rules="rules">
        <a-form-item
          label="发送组名称"
          name="name"
          :rules="[{ required: true, message: '请输入发送组名称' }]"
        >
          <a-input v-model:value="form.name" />
        </a-form-item>
        <a-form-item
          label="发送组中文名称"
          name="name_zh"
          :rules="[{ required: true, message: '请输入发送组中文名称' }]"
        >
          <a-input v-model:value="form.name_zh" />
        </a-form-item>
        <a-form-item label="是否启用" name="enable">
          <a-switch v-model:checked="form.enable" :checked-children="'启用'" :un-checked-children="'禁用'" />
        </a-form-item>
        <a-form-item label="关联采集池" name="pool_id">
          <a-select v-model:value="form.pool_id">
            <a-select-option v-for="pool in scrapePools" :key="pool.id" :value="pool.id">
              {{ pool.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="关联值班组" name="on_duty_group_id">
          <a-select v-model:value="form.on_duty_group_id">
            <a-select-option v-for="group in onDutyGroups" :key="group.id" :value="group.id">
              {{ group.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="重复发送时间" name="repeat_interval">
          <a-input v-model:value="form.repeat_interval" placeholder="默认30s" />
        </a-form-item>
        <a-form-item label="是否发送恢复消息" name="send_resolved">
          <a-select v-model:value="form.send_resolved">
            <a-select-option :value="true">发送</a-select-option>
            <a-select-option :value="false">不发送</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="需要升级" name="need_upgrade">
          <a-select v-model:value="form.need_upgrade">
            <a-select-option :value="true">需要</a-select-option>
            <a-select-option :value="false">不需要</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="飞书群机器人 Token" name="fei_shu_qun_robot_token">
          <a-input v-model:value="form.fei_shu_qun_robot_token" />
        </a-form-item>
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
  getUserList,
  getMonitorSendGroupTotalApi,
  getAllMonitorScrapePoolApi,
  getAllOnDutyGroupApi,
} from '#/api';
import type { SendGroupItem, MonitorScrapePoolItem, OnDutyGroupItem } from '#/api';

// 分页相关
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);
const pageSizeOptions = ['10', '20', '30', '50'];
const loading = ref(false);

const columns = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
  },
  {
    title: '发送组名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '发送组中文名称',
    dataIndex: 'name_zh',
    key: 'name_zh',
  },
  {
    title: '关联采集池ID',
    dataIndex: 'pool_id',
    key: 'pool_id',
  },
  {
    title: '关联值班组ID',
    dataIndex: 'on_duty_group_id',
    key: 'on_duty_group_id',
  },
  {
    title: '是否启用',
    dataIndex: 'enable',
    key: 'enable',
    customRender: ({ record }: { record: SendGroupItem }) =>
      record.enable ? '启用' : '禁用',
  },
  {
    title: '重复发送时间',
    dataIndex: 'repeat_interval',
    key: 'repeat_interval',
  },
  {
    title: '是否发送恢复消息',
    dataIndex: 'send_resolved',
    key: 'send_resolved',
    customRender: ({ record }: { record: SendGroupItem }) =>
      record.send_resolved ? '是' : '否',
  },
  {
    title: '需要升级',
    dataIndex: 'need_upgrade',
    key: 'need_upgrade',
    customRender: ({ record }: { record: SendGroupItem }) =>
      record.need_upgrade ? '需要' : '不需要',
  },
  {
    title: '操作',
    dataIndex: 'action',
    key: 'action',
    slots: { customRender: 'action' },
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

const rules = {
  name: [{ required: true, message: '请输入发送组名称' }],
  name_zh: [{ required: true, message: '请输入发送组中文名称' }],
};

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
const handlePageChange = (page: number, pageSize: number) => {
  current.value = page;
  pageSizeRef.value = pageSize;
  fetchSendGroups();
};

const handleSizeChange = (current: number, size: number) => {
  pageSizeRef.value = size;
  fetchSendGroups();
};

// 获取用户列表
const fetchUsers = async () => {
  try {
    const users = await getUserList();
    userOptions.value = users.map((user: { username: any; id: any }) => ({
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
    repeat_interval: '30s',
    send_resolved: false,
    need_upgrade: false,
    static_receive_users: [],
    notify_methods: [],
    first_upgrade_users: [],
    second_upgrade_users: [],
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
    const response = await getMonitorSendGroupListApi(
      current.value,
      pageSizeRef.value,
      searchText.value,
    );
    data.splice(0, data.length, ...response);
    const totalCount = await getMonitorSendGroupTotalApi();
    total.value = totalCount;
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
    scrapePools.value = response;
  } catch (error: any) {
    message.error(error.message || '获取采集池列表失败');

    console.error(error);
  }
};

// 获取值班组列表 
const fetchOnDutyGroups = async () => {
  try {
    const response = await getAllOnDutyGroupApi();
    onDutyGroups.value = response;
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

