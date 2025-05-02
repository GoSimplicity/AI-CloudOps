<template>
  <div>
    <div class="custom-toolbar">
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input v-model:value="searchText" placeholder="请输入定时任务名称" style="width: 200px; margin-right: 16px;" />
        <!-- 搜索按钮 -->
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <div>
        <a-button type="primary" @click="handleAdd">新增</a-button>
      </div>
    </div>
    <a-table :columns="columns" :data-source="filteredData">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'name'">
          <a>
            {{ record.name }}
          </a>
        </template>
        <template v-else-if="column.key === 'tags'">
          <span>
            <a-tag v-for="tag in record.tags" :key="tag"
              :color="tag === 'loser' ? 'volcano' : tag.length > 5 ? 'geekblue' : 'green'">
              {{ tag.toUpperCase() }}
            </a-tag>
          </span>
        </template>
        <template v-else-if="column.key === 'action'">
          <span>
            <a-button type="link" size="small" @click="() => handleEdit(record)">编辑</a-button>
            <a-button type="link" size="small" danger @click="() => handleDelete(record)">删除</a-button>
          </span>
        </template>
      </template>
    </a-table>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import { message } from 'ant-design-vue';

interface DataItem {
  name: string;
  cluster: string;
  app: string;
  namespace: string;
  schedule: string;
  status: string;
  lastSchedule: string;
  tags: string[];
}

// 搜索文本
const searchText = ref('');
const filteredData = ref<DataItem[]>([]);

const columns = [
  {
    title: '名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '所属集群',
    dataIndex: 'cluster',
    key: 'cluster',
  },
  {
    title: '所属应用',
    dataIndex: 'app',
    key: 'app',
  },
  {
    title: '命名空间',
    dataIndex: 'namespace',
    key: 'namespace',
  },
  {
    title: '调度规则',
    key: 'schedule',
    dataIndex: 'schedule',
  },
  {
    title: '状态',
    key: 'status',
    dataIndex: 'status',
  },
  {
    title: '最近执行时间',
    key: 'lastSchedule',
    dataIndex: 'lastSchedule',
  },
  {
    title: '操作',
    key: 'action',
  }
];

const data: DataItem[] = [
  {
    name: 'backup-job',
    cluster: 'k8s-cluster-1',
    app: 'backup-service',
    namespace: 'default',
    schedule: '0 2 * * *',
    status: 'Active',
    lastSchedule: '2023-10-01 02:00:00',
    tags: ['backup', 'daily'],
  },
  {
    name: 'cleanup-job',
    cluster: 'k8s-cluster-2',
    app: 'maintenance',
    namespace: 'system',
    schedule: '0 0 * * 0',
    status: 'Active',
    lastSchedule: '2023-10-01 00:00:00',
    tags: ['cleanup', 'weekly'],
  },
  {
    name: 'report-job',
    cluster: 'k8s-cluster-1',
    app: 'reporting',
    namespace: 'business',
    schedule: '0 8 1 * *',
    status: 'Suspended',
    lastSchedule: '2023-10-01 08:00:00',
    tags: ['report', 'monthly'],
  }
];

// 搜索按钮
const handleSearch = () => {
  if (searchText.value.trim() === '') {
    filteredData.value = data;
  } else {
    filteredData.value = data.filter(item => item.name.includes(searchText.value));
  }
};

// 编辑和删除操作
const handleEdit = (_: DataItem) => {
  message.success('编辑成功');
};

const handleDelete = (_: DataItem) => {
  message.success('删除成功');
};

const handleAdd = () => {
  message.success('新增成功');
};

onMounted(() => {
  filteredData.value = data;
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

.action-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 16px;
}
</style>
