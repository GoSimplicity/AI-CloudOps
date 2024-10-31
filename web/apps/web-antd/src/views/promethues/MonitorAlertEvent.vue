<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model="searchText"
          placeholder="请输入告警事件名称"
          style="width: 200px; margin-right: 16px;"
        />
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button
          type="primary"
          @click="handleBatchSilence"
          :disabled="filteredData.length === 0"
        >
          批量屏蔽告警
        </a-button>
      </div>
    </div>

    <!-- 告警事件列表表格 -->
    <a-table
      :columns="columns"
      :data-source="filteredData"
      row-key="ID"
      :loading="loading"
      pagination={{
        pageSize: 10,
        showSizeChanger: true,
        pageSizeOptions: [10, 20, 50],
      }}
    >
      <!-- 标签组列 -->
      <template #labels="{ record }">
        <a-tag v-for="label in record.labels" :key="label" color="purple">
          {{ label }}
        </a-tag>
      </template>
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="link" @click="handleSilence(record)">屏蔽告警</a-button>
          <a-button type="link" @click="handleClaim(record)">认领告警</a-button>
          <a-button type="link" @click="handleCancelSilence(record)">取消屏蔽</a-button>
        </a-space>
      </template>
    </a-table>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  getAlertEventsApi,
  silenceAlertApi,
  claimAlertApi,
  cancelSilenceAlertApi,
  silenceBatchApi,
} from '#/api'; // 请根据实际路径调整

// 定义数据类型
interface AlertEvent {
  ID: number; // 唯一标识符，用于区分不同的告警事件
  alertName: string; // 告警名称
  fingerprint: string; // 告警唯一ID
  status: string; // 告警状态，如 "告警中"、"已屏蔽"、"已认领"、"已恢复"
  sendGroupId: string; // 关联的发送组名称
  eventTimes: number; // 触发次数
  renLingUserId: string; // 认领告警的用户名
  labels: string[]; // 标签组，格式为 key=v
  CreatedAt: string; // 创建时间
  silenceId: string; // 静默ID
}

// 状态变量
const data = ref<AlertEvent[]>([]);
const loading = ref<boolean>(false);
const searchText = ref('');

// 过滤后的数据，通过 computed 属性动态计算
const filteredData = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  return data.value.filter(item =>
    item.alertName.toLowerCase().includes(searchValue)
  );
});

// 表格列配置
const columns = [
  {
    title: 'ID',
    dataIndex: 'ID',
    key: 'ID',
    sorter: (a: AlertEvent, b: AlertEvent) => a.ID - b.ID,
  },
  {
    title: '告警名称',
    dataIndex: 'alertName',
    key: 'alertName',
    sorter: (a: AlertEvent, b: AlertEvent) => a.alertName.localeCompare(b.alertName),
  },
  {
    title: '告警状态',
    dataIndex: 'status',
    key: 'status',
    sorter: (a: AlertEvent, b: AlertEvent) => a.status.localeCompare(b.status),
    filters: [
      { text: '告警中', value: '告警中' },
      { text: '已屏蔽', value: '已屏蔽' },
      { text: '已认领', value: '已认领' },
      { text: '已恢复', value: '已恢复' },
    ],
    onFilter: (value: string, record: AlertEvent) => record.status === value,
  },
  {
    title: '关联发送组',
    dataIndex: 'sendGroupId',
    key: 'sendGroupId',
    sorter: (a: AlertEvent, b: AlertEvent) => a.sendGroupId.localeCompare(b.sendGroupId),
  },
  {
    title: '触发次数',
    dataIndex: 'eventTimes',
    key: 'eventTimes',
    sorter: (a: AlertEvent, b: AlertEvent) => a.eventTimes - b.eventTimes,
  },
  {
    title: '静默ID',
    dataIndex: 'silenceId',
    key: 'silenceId',
    sorter: (a: AlertEvent, b: AlertEvent) => a.silenceId.localeCompare(b.silenceId),
  },
  {
    title: '认领用户',
    dataIndex: 'renLingUserId',
    key: 'renLingUserId',
    sorter: (a: AlertEvent, b: AlertEvent) => a.renLingUserId.localeCompare(b.renLingUserId),
  },
  {
    title: '标签组',
    dataIndex: 'labels',
    key: 'labels',
    slots: { customRender: 'labels' },
  },
  {
    title: '创建时间',
    dataIndex: 'CreatedAt',
    key: 'CreatedAt',
    sorter: (a: AlertEvent, b: AlertEvent) => new Date(a.CreatedAt).getTime() - new Date(b.CreatedAt).getTime(),
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 获取告警事件数据的函数
const fetchResources = async () => {
  loading.value = true;
  try {
    const response = await getAlertEventsApi();
    data.value = response; // 假设后端返回的数据格式与 AlertEvent[] 匹配
  } catch (error) {
    message.error('获取告警事件数据失败，请稍后重试');
    console.error(error);
  } finally {
    loading.value = false;
  }
};

// 批量屏蔽告警
const handleBatchSilence = () => {
  if (filteredData.value.length === 0) {
    message.warning('当前没有可屏蔽的告警');
    return;
  }

  Modal.confirm({
    title: '确认批量屏蔽',
    content: `您确定要屏蔽当前过滤的 ${filteredData.value.length} 个告警吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        const alertIds = filteredData.value.map(item => item.ID);
        await silenceBatchApi(alertIds); // 替换为实际的 API 接口
        message.success('批量屏蔽告警成功');
        fetchResources(); // 刷新数据
      } catch (error) {
        message.error('批量屏蔽告警失败');
        console.error(error);
      } finally {
        loading.value = false;
      }
    },
  });
};

// 处理屏蔽告警
const handleSilence = async (record: AlertEvent) => {
  Modal.confirm({
    title: '确认屏蔽',
    content: `您确定要屏蔽告警 "${record.alertName}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await silenceAlertApi(record.ID); // 替换为实际的 API 接口
        message.success(`屏蔽告警 "${record.alertName}" 成功`);
        fetchResources(); // 刷新数据
      } catch (error) {
        message.error(`屏蔽告警 "${record.alertName}" 失败`);
        console.error(error);
      } finally {
        loading.value = false;
      }
    },
  });
};

// 处理认领告警
const handleClaim = async (record: AlertEvent) => {
  Modal.confirm({
    title: '确认认领',
    content: `您确定要认领告警 "${record.alertName}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await claimAlertApi(record.ID); // 替换为实际的 API 接口
        message.success(`认领告警 "${record.alertName}" 成功`);
        fetchResources(); // 刷新数据
      } catch (error) {
        message.error(`认领告警 "${record.alertName}" 失败`);
        console.error(error);
      } finally {
        loading.value = false;
      }
    },
  });
};

// 处理取消屏蔽告警
const handleCancelSilence = async (record: AlertEvent) => {
  Modal.confirm({
    title: '确认取消屏蔽',
    content: `您确定要取消屏蔽告警 "${record.alertName}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        loading.value = true;
        await cancelSilenceAlertApi(record.ID); // 替换为实际的 API 接口
        message.success(`取消屏蔽告警 "${record.alertName}" 成功`);
        fetchResources(); // 刷新数据
      } catch (error) {
        message.error(`取消屏蔽告警 "${record.alertName}" 失败`);
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
  padding: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-filters {
  display: flex;
  align-items: center;
}

.action-buttons {
  display: flex;
  align-items: center;
}

a-form-item {
  margin-bottom: 16px;
}
</style>
