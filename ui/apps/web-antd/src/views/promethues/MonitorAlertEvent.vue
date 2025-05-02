<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input v-model:value="searchText" placeholder="请输入告警事件名称" style="width: 200px" />
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
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="handleBatchSilence" :disabled="data.length === 0">
          批量屏蔽告警
        </a-button>
      </div>
    </div>

    <!-- 告警事件列表表格 -->
    <a-table :columns="columns" :data-source="data" row-key="id" :loading="loading" :pagination="false">
      <!-- 标签组列 -->
      <template #labels="{ record }">
        <a-tag v-for="label in record.labels" :key="label" color="purple">
          {{ label }}
        </a-tag>
      </template>
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-tooltip title="屏蔽告警">
            <a-button type="link" @click="handleSilence(record)">
              <template #icon>
                <Icon icon="mdi:bell-off-outline" style="font-size: 22px" />
              </template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="认领告警">
            <a-button type="link" @click="handleClaim(record)">
              <template #icon>
                <Icon icon="mdi:hand-back-right-outline" style="font-size: 22px" />
              </template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="取消屏蔽">
            <a-button type="link" @click="handleCancelSilence(record)">
              <template #icon>
                <Icon icon="mdi:bell-ring-outline" style="font-size: 22px" />
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
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  SearchOutlined,
  ReloadOutlined
} from '@ant-design/icons-vue';
import { Icon } from '@iconify/vue';
import type { TableColumnsType } from 'ant-design-vue';
import {
  getAlertEventsListApi,
  silenceAlertApi,
  claimAlertApi,
  cancelSilenceAlertApi,
  silenceBatchApi,
  getAlertEventsTotalApi
} from '#/api';
import type { MonitorAlertEventItem } from '#/api/core/prometheus';

// 状态变量
const data = ref<MonitorAlertEventItem[]>([]);
const loading = ref<boolean>(false);
const searchText = ref('');


// 分页相关
const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
const current = ref(1);
const pageSizeRef = ref(10);
const total = ref(0);

// 表格列配置
const columns: TableColumnsType = [
  {
    title: 'id',
    dataIndex: 'id',
    key: 'id',
    width: 80,
    sorter: (a: MonitorAlertEventItem, b: MonitorAlertEventItem) => a.id - b.id,
  },
  {
    title: '告警名称',
    dataIndex: 'alert_name',
    key: 'alert_name',
    width: 200,
    sorter: (a: MonitorAlertEventItem, b: MonitorAlertEventItem) => a.alert_name.localeCompare(b.alert_name),
  },
  {
    title: '告警状态',
    dataIndex: 'status',
    key: 'status',
    width: 120,
    sorter: (a: MonitorAlertEventItem, b: MonitorAlertEventItem) => a.status.localeCompare(b.status),
    filters: [
      { text: '告警中', value: '告警中' },
      { text: '已屏蔽', value: '已屏蔽' },
      { text: '已认领', value: '已认领' },
      { text: '已恢复', value: '已恢复' },
    ],
    onFilter: (value: string | number | boolean, record: MonitorAlertEventItem) => record.status === value.toString(),
  },
  {
    title: '关联发送组',
    dataIndex: 'send_group_id',
    key: 'send_group_id',
    width: 150,
  },
  {
    title: '触发次数',
    dataIndex: 'event_times',
    key: 'event_times',
    width: 100,
    sorter: (a: MonitorAlertEventItem, b: MonitorAlertEventItem) => a.event_times - b.event_times,
  },
  {
    title: '静默id',
    dataIndex: 'silence_id',
    key: 'silence_id',
    width: 120,
  },
  {
    title: '认领用户',
    dataIndex: 'ren_ling_user_id',
    key: 'ren_ling_user_id',
    width: 120,
  },
  {
    title: '标签组',
    dataIndex: 'labels',
    key: 'labels',
    slots: { customRender: 'labels' },
  },
  {
    title: '发送组',
    dataIndex: 'send_group_name',
    key: 'send_group_name',
    width: 120,
  },
  {
    title: '规则名称',
    dataIndex: 'alert_rule_name',
    key: 'alert_rule_name',
    width: 120,
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: 180,
    sorter: (a: MonitorAlertEventItem, b: MonitorAlertEventItem) =>
      new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  },
  {
    title: '操作',
    key: 'action',
    width: 180,
    fixed: 'right',
    slots: { customRender: 'action' },
  },
];

// 搜索处理
const handleSearch = () => {
  current.value = 1;
  fetchResources();
};

const handleReset = () => {
  searchText.value = '';
  fetchResources();
};

// 分页处理
const handlePageChange = (page: number, pageSize: number) => {
  current.value = page;
  pageSizeRef.value = pageSize;
  fetchResources();
};

const handleSizeChange = (_: number, size: number) => {
  pageSizeRef.value = size;
  fetchResources();
};

// 获取告警事件数据
const fetchResources = async () => {
  loading.value = true;
  try {
    const response = await getAlertEventsListApi(
      current.value,
      pageSizeRef.value,
      searchText.value
    );
    data.value = response as unknown as MonitorAlertEventItem[];
    total.value = await getAlertEventsTotalApi();

  } catch (error: any) {
    message.error(error.message || '获取告警事件数据失败，请稍后重试');
    console.error(error);
  } finally {
    loading.value = false;
  }
};

// 批量屏蔽告警
const handleBatchSilence = () => {
  if (data.value.length === 0) {
    message.warning('当前没有可屏蔽的告警');
    return;
  }

  Modal.confirm({
    title: '确认批量屏蔽',
    content: `您确定要屏蔽当前页 ${data.value.length} 个告警吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        const alertIds = data.value.map(item => item.id);
        await silenceBatchApi(alertIds);
        message.success('批量屏蔽告警成功');
        fetchResources();

      } catch (error: any) {
        message.error(error.message || '批量屏蔽告警失败');
        console.error(error);
      } finally {
        loading.value = false;
      }
    },
  });
};

// 处理屏蔽告警
const handleSilence = async (record: MonitorAlertEventItem) => {
  Modal.confirm({
    title: '确认屏蔽',
    content: `您确定要屏蔽告警 "${record.alert_name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await silenceAlertApi(record.id);
        message.success(`屏蔽告警 "${record.alert_name}" 成功`);
        fetchResources();
      } catch (error: any) {
        message.error(error.message || `屏蔽告警 "${record.alert_name}" 失败`);

        console.error(error);
      } finally {
        loading.value = false;
      }
    },
  });
};

// 处理认领告警
const handleClaim = async (record: MonitorAlertEventItem) => {
  Modal.confirm({
    title: '确认认领',
    content: `您确定要认领告警 "${record.alert_name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await claimAlertApi(record.id);
        message.success(`认领告警 "${record.alert_name}" 成功`);
        fetchResources();
      } catch (error: any) {
        message.error(error.message || `认领告警 "${record.alert_name}" 失败`);

        console.error(error);
      } finally {
        loading.value = false;
      }
    },
  });
};

// 处理取消屏蔽告警
const handleCancelSilence = async (record: MonitorAlertEventItem) => {
  Modal.confirm({
    title: '确认取消屏蔽',
    content: `您确定要取消屏蔽告警 "${record.alert_name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await cancelSilenceAlertApi(record.id);
        message.success(`取消屏蔽告警 "${record.alert_name}" 成功`);
        fetchResources();
      } catch (error: any) {
        message.error(error.message || `取消屏蔽告警 "${record.alert_name}" 失败`);

        console.error(error);
      } finally {
        loading.value = false;
      }
    },
  });
};

// 在组件挂载时获取数据
onMounted(() => {
  fetchResources();
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
</style>
