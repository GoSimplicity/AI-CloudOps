<template>
  <div class="monitor-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">告警事件管理</h2>
      <div class="page-description">管理和监控Prometheus告警事件及处理状态</div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card custom-toolbar">
      <div class="search-filters">
        <a-input 
          v-model:value="searchText" 
          placeholder="请输入告警事件名称" 
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
        <a-button type="primary" class="add-button" @click="handleBatchSilence" :disabled="data.length === 0">
          批量屏蔽告警
        </a-button>
      </div>
    </div>

    <!-- 告警事件列表表格 -->
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
        
        <!-- 操作列 -->
        <template #action="{ record }">
          <div class="action-column">
            <a-tooltip title="屏蔽告警">
              <a-button type="primary" shape="circle" class="edit-button" @click="handleSilence(record)">
                <template #icon>
                  <Icon icon="mdi:bell-off-outline" />
                </template>
              </a-button>
            </a-tooltip>
            <a-tooltip title="认领告警">
              <a-button type="primary" shape="circle" class="claim-button" @click="handleClaim(record)">
                <template #icon>
                  <Icon icon="mdi:hand-back-right-outline" />
                </template>
              </a-button>
            </a-tooltip>
            <a-tooltip title="取消屏蔽">
              <a-button type="primary" danger shape="circle" class="delete-button" @click="handleCancelSilence(record)">
                <template #icon>
                  <Icon icon="mdi:bell-ring-outline" />
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
} from '#/api/core/prometheus_alert_event';
import type { MonitorAlertEventItem } from '#/api/core/prometheus_alert_event';

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
    const response = await getAlertEventsListApi({
      page: current.value,
      size: pageSizeRef.value,
      search: searchText.value,
    });
    data.value = response.items;
    total.value = response.total;

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

.claim-button {
  background: #52c41a;
  border: none;
  box-shadow: 0 2px 4px rgba(82, 196, 26, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
}

.claim-button:hover {
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
</style>